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
)

const (
	maxRenameAttempts = 1000
	filePerm          = 0o600
)

// SaveFile stores content as fileName inside dirPath directory relative to the service root.
// If fileName is already taken, a numeric suffix is added before the extension (a.txt -> a_1.txt).
// Returns info about the stored file, its name may differ from fileName.
func (s *Service) SaveFile(
	dirPath string,
	fileName string,
	content io.Reader,
) (file.File, error) {
	root, err := os.OpenRoot(s.rootDir)
	if err != nil {
		return file.File{}, fmt.Errorf("open root: %w", err)
	}

	defer root.Close()

	dir := toFSPath(dirPath)

	// root.OpenFile rejects paths escaping the root with an unexported error, so validate them upfront.
	if !fs.ValidPath(dir) {
		return file.File{}, fmt.Errorf("validate dir path %q: %w", dir, perror.InvalidArgument("invalid directory path"))
	}

	if !isValidFileName(fileName) {
		return file.File{}, fmt.Errorf("validate file name %q: %w", fileName, perror.InvalidArgument("invalid file name"))
	}

	osFile, filePath, err := createUniqueFile(root, dir, fileName)
	if err != nil {
		return file.File{}, fmt.Errorf("create file: %w", err)
	}

	if err := writeContent(osFile, content); err != nil {
		if removeErr := root.Remove(filePath); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove partial file: %w", removeErr))
		}

		return file.File{}, fmt.Errorf("write file %q: %w", filePath, err)
	}

	info, err := root.Stat(filePath)
	if err != nil {
		return file.File{}, fmt.Errorf("stat file %q: %w", filePath, err)
	}

	return file.File{
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// isValidFileName reports whether name is a single path element ("", ".." are rejected by fs.ValidPath).
func isValidFileName(name string) bool {
	return name != "." && !strings.Contains(name, "/") && fs.ValidPath(name)
}

// createUniqueFile creates a new file in dir, adding a numeric suffix to fileName until a free name is found.
// O_EXCL guarantees that an existing file is never overwritten, even by concurrent uploads.
func createUniqueFile(root *os.Root, dir string, fileName string) (*os.File, string, error) {
	for attempt := range maxRenameAttempts {
		filePath := path.Join(dir, candidateName(fileName, attempt))

		osFile, err := root.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePerm)
		if err == nil {
			return osFile, filePath, nil
		}

		if !errors.Is(err, fs.ErrExist) {
			return nil, "", fmt.Errorf("open file %q: %w", filePath, createFileError(err))
		}
	}

	return nil, "", fmt.Errorf("no free name for %q after %d attempts", fileName, maxRenameAttempts)
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

func writeContent(osFile *os.File, content io.Reader) error {
	if _, err := io.Copy(osFile, content); err != nil {
		osFile.Close()

		return fmt.Errorf("copy content: %w", err)
	}

	if err := osFile.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

// createFileError attaches perror describing the failure to the original error.
func createFileError(err error) error {
	switch {
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("not a directory"), err)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %w", perror.NotFound("directory not found"), err)
	case errors.Is(err, fs.ErrInvalid):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("invalid directory path"), err)
	}

	return err
}
