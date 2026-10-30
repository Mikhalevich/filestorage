package filesvc

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

const (
	maxRenameAttempts = 1000
	filePerm          = 0o600
)

// SaveFile stores content as fileName inside dirPath directory relative to the service root.
// If fileName is already taken, a numeric suffix is added before the extension (a.txt -> a_1.txt).
// Returns info about the stored file, its name may differ from fileName.
func (s *Service) SaveFile(
	res resource.Resource,
	fileName string,
	content io.Reader,
) (file.File, error) {
	info, err := s.withRoot(
		res,
		func(root *os.Root, dir string) (os.FileInfo, error) {
			// root.OpenFile rejects paths escaping the root with an unexported error, so validate them upfront.
			if !fs.ValidPath(dir) {
				return nil, fmt.Errorf("validate dir path %q: %w", dir, perror.InvalidArgument("invalid directory path"))
			}

			if !isValidFileName(fileName) {
				return nil, fmt.Errorf("validate file name %q: %w", fileName, perror.InvalidArgument("invalid file name"))
			}

			filePath, err := writeUniqueFile(
				root,
				dir,
				fileName,
				func(osFile *os.File) error {
					if _, err := io.Copy(osFile, content); err != nil {
						return fmt.Errorf("copy content: %w", err)
					}

					return nil
				},
			)
			if err != nil {
				return nil, err
			}

			info, err := root.Stat(filePath)
			if err != nil {
				return nil, fmt.Errorf("stat file %q: %w", filePath, err)
			}

			return info, nil
		},
	)
	if err != nil {
		return file.File{}, err
	}

	return file.File{
		Name:    info.Name(),
		Type:    file.TypeFile,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// isValidFileName reports whether name is a single path element ("", ".." are rejected by fs.ValidPath).
func isValidFileName(name string) bool {
	return name != "." && !strings.Contains(name, "/") && fs.ValidPath(name)
}

// writeUniqueFile creates a new file in dir and fills it with writeFn,
// adding a numeric suffix to fileName until a free name is found.
// O_EXCL guarantees that an existing file is never overwritten, even by concurrent uploads.
// Returns path of the written file relative to root.
func writeUniqueFile(
	root *os.Root,
	dir string,
	fileName string,
	writeFn func(osFile *os.File) error,
) (string, error) {
	for attempt := range maxRenameAttempts {
		filePath := path.Join(dir, candidateName(fileName, attempt))

		if err := createAndWriteFile(root, filePath, writeFn); err != nil {
			if !perror.IsType(err, perror.TypeAlreadyExists) {
				return "", fmt.Errorf("write file %q: %w", filePath, err)
			}

			continue
		}

		return filePath, nil
	}

	return "", fmt.Errorf("no free name for %q after %d attempts", fileName, maxRenameAttempts)
}

// createAndWriteFile creates filePath (failing if it exists), fills it with writeFn and closes it.
// On write or close failure the partial file is removed.
func createAndWriteFile(
	root *os.Root,
	filePath string,
	writeFn func(osFile *os.File) error,
) error {
	osFile, err := root.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
	if err != nil {
		return fmt.Errorf("open file: %w", createFileError(err))
	}

	writeErr := writeFn(osFile)

	if closeErr := osFile.Close(); closeErr != nil {
		writeErr = errors.Join(writeErr, fmt.Errorf("close file: %w", closeErr))
	}

	if writeErr == nil {
		return nil
	}

	if removeErr := root.Remove(filePath); removeErr != nil {
		return errors.Join(writeErr, fmt.Errorf("remove partial file: %w", removeErr))
	}

	return writeErr
}

// candidateName returns name for the given attempt: a.txt, a_1.txt, a_2.txt...
// Names without base part (.env) get the suffix at the end: .env_1.
func candidateName(name string, attempt int) string {
	if attempt == 0 {
		return name
	}

	ext := path.Ext(name)

	base := strings.TrimSuffix(name, ext)
	if base == "" {
		base, ext = name, ""
	}

	return fmt.Sprintf("%s_%d%s", base, attempt, ext)
}

// createFileError attaches perror describing the failure to the original error.
func createFileError(err error) error {
	switch {
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("not a directory"), err)
	case errors.Is(err, fs.ErrExist):
		return fmt.Errorf("%w: %w", perror.AlreadyExists("file already exists"), err)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %w", perror.NotFound("directory not found"), err)
	case errors.Is(err, fs.ErrInvalid):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("invalid directory path"), err)
	}

	return err
}
