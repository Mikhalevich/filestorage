package filesvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

const (
	dirPerm = 0o700
)

// CreateDirectory creates dirPath directory relative to the service root.
// Only the last path element is created: the parent directory must exist.
func (s *Service) CreateDirectory(dirPath string) error {
	_, err := s.withRoot(
		dirPath,
		func(root *os.Root, fsPath string) (struct{}, error) {
			// root.Mkdir rejects paths escaping the root with an unexported error, so validate them upfront.
			// "." is the root itself, it always exists and can't be created.
			if !fs.ValidPath(fsPath) || fsPath == "." {
				return struct{}{}, fmt.Errorf("validate dir path %q: %w", fsPath, perror.InvalidArgument("invalid directory path"))
			}

			if err := root.Mkdir(fsPath, dirPerm); err != nil {
				return struct{}{}, fmt.Errorf("create directory %q: %w", fsPath, createDirectoryError(err))
			}

			return struct{}{}, nil
		},
	)

	return err
}

// createDirectoryError attaches perror describing the failure to the original error.
func createDirectoryError(err error) error {
	switch {
	case errors.Is(err, fs.ErrExist):
		return fmt.Errorf("%w: %w", perror.AlreadyExists("directory already exists"), err)
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("not a directory"), err)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %w", perror.NotFound("parent directory not found"), err)
	case errors.Is(err, fs.ErrInvalid):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("invalid directory path"), err)
	}

	return err
}
