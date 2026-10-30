package filesvc

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

type openFileResult struct {
	info os.FileInfo
	file *os.File
}

// OpenFile opens the regular file located at filePath relative to the service root.
// The caller must close the returned reader.
func (s *Service) OpenFile(res resource.Resource) (file.File, io.ReadSeekCloser, error) {
	result, err := s.withRoot(
		res,
		func(root *os.Root, fsPath string) (openFileResult, error) {
			// root.Open rejects paths escaping the root with an unexported error, so validate them upfront.
			if !fs.ValidPath(fsPath) {
				return openFileResult{}, fmt.Errorf("validate path %q: %w", fsPath, perror.InvalidArgument("invalid file path"))
			}

			osFile, err := root.Open(fsPath)
			if err != nil {
				return openFileResult{}, fmt.Errorf("open file %q: %w", fsPath, openFileError(err))
			}

			info, err := osFile.Stat()
			if err != nil {
				osFile.Close()

				return openFileResult{}, fmt.Errorf("stat file %q: %w", fsPath, err)
			}

			if !info.Mode().IsRegular() {
				osFile.Close()

				return openFileResult{}, fmt.Errorf("check file %q: %w", fsPath, perror.InvalidArgument("not a regular file"))
			}

			// closing the root does not close files opened through it.
			return openFileResult{
				info: info,
				file: osFile,
			}, nil
		},
	)
	if err != nil {
		return file.File{}, nil, err
	}

	return file.File{
		Name:    result.info.Name(),
		Type:    file.TypeFile,
		Size:    result.info.Size(),
		ModTime: result.info.ModTime(),
	}, result.file, nil
}

// openFileError attaches perror describing the failure to the original error.
func openFileError(err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %w", perror.NotFound("file not found"), err)
	case errors.Is(err, fs.ErrInvalid):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("invalid file path"), err)
	}

	return err
}
