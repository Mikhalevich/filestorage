package filesvc

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

// OpenFile opens the regular file located at filePath relative to the service root.
// The caller must close the returned reader.
func (s *Service) OpenFile(filePath string) (file.File, io.ReadSeekCloser, error) {
	root, err := os.OpenRoot(s.rootDir)
	if err != nil {
		return file.File{}, nil, fmt.Errorf("open root: %w", err)
	}

	// closing the root does not close files opened through it.
	defer root.Close()

	fsPath := toFSPath(filePath)

	// root.Open rejects paths escaping the root with an unexported error, so validate them upfront.
	if !fs.ValidPath(fsPath) {
		return file.File{}, nil, fmt.Errorf("validate path %q: %w", fsPath, perror.InvalidArgument("invalid file path"))
	}

	osFile, err := root.Open(fsPath)
	if err != nil {
		return file.File{}, nil, fmt.Errorf("open file %q: %w", fsPath, openFileError(err))
	}

	info, err := osFile.Stat()
	if err != nil {
		osFile.Close()

		return file.File{}, nil, fmt.Errorf("stat file %q: %w", fsPath, err)
	}

	if !info.Mode().IsRegular() {
		osFile.Close()

		return file.File{}, nil, fmt.Errorf("check file %q: %w", fsPath, perror.InvalidArgument("not a regular file"))
	}

	return file.File{
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, osFile, nil
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
