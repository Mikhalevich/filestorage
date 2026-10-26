package filesvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

// DeleteFile removes the regular file located at filePath relative to the service root.
// Directories and other non-regular files are never removed.
func (s *Service) DeleteFile(filePath string) error {
	return s.withRootNoReturn(
		filePath,
		func(root *os.Root, fsPath string) error {
			// root.Lstat rejects paths escaping the root with an unexported error, so validate them upfront.
			if !fs.ValidPath(fsPath) {
				return fmt.Errorf("validate path %q: %w", fsPath, perror.InvalidArgument("invalid file path"))
			}

			info, err := root.Lstat(fsPath)
			if err != nil {
				return fmt.Errorf("stat file %q: %w", fsPath, deleteFileError(err))
			}

			if !info.Mode().IsRegular() {
				return fmt.Errorf("check file %q: %w", fsPath, perror.InvalidArgument("not a regular file"))
			}

			if err := root.Remove(fsPath); err != nil {
				return fmt.Errorf("remove file %q: %w", fsPath, deleteFileError(err))
			}

			return nil
		},
	)
}

// deleteFileError attaches perror describing the failure to the original error.
func deleteFileError(err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist), errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %w", perror.NotFound("file not found"), err)
	case errors.Is(err, fs.ErrInvalid):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("invalid file path"), err)
	}

	return err
}
