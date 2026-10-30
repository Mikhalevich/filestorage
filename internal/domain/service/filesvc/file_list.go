package filesvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

// FileList returns regular files and subdirectories located in dirPath directory relative to the service root.
// Other entries (symlinks, devices...) are skipped. Subdirectories come first sorted by name and have zero size,
// files follow sorted by modification time (oldest first), ties are sorted by name.
func (s *Service) FileList(res resource.Resource) ([]file.File, error) {
	files, err := s.withRoot(
		res,
		func(root *os.Root, fsPath string) ([]file.File, error) {
			entries, err := fs.ReadDir(root.FS(), fsPath)
			if err != nil {
				return nil, fmt.Errorf("read dir %q: %w", fsPath, readDirError(err))
			}

			files := make([]file.File, 0, len(entries))

			for _, entry := range entries {
				if !entry.IsDir() && !entry.Type().IsRegular() {
					continue
				}

				info, err := entry.Info()
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						continue
					}

					return nil, fmt.Errorf("file info %q: %w", entry.Name(), err)
				}

				entryFile := file.File{
					Name:    entry.Name(),
					Type:    createFileType(entry.IsDir()),
					ModTime: info.ModTime(),
				}

				if !entryFile.IsDir() {
					entryFile.Size = info.Size()
				}

				files = append(files, entryFile)
			}

			return files, nil
		},
	)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(files, compareEntries)

	return files, nil
}

func createFileType(isDir bool) file.Type {
	if isDir {
		return file.TypeDirectory
	}

	return file.TypeFile
}

// compareEntries orders directories before files: directories by name,
// files by modification time and then by name.
func compareEntries(lhs, rhs file.File) int {
	if lhs.IsDir() != rhs.IsDir() {
		if lhs.IsDir() {
			return -1
		}

		return 1
	}

	if !lhs.IsDir() {
		if c := lhs.ModTime.Compare(rhs.ModTime); c != 0 {
			return c
		}
	}

	return strings.Compare(lhs.Name, rhs.Name)
}

func toFSPath(dirPath string) string {
	cleaned := path.Clean(strings.TrimPrefix(dirPath, "/"))
	if cleaned == "" {
		return "."
	}

	return cleaned
}

// readDirError attaches perror describing the failure to the original error.
func readDirError(err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %w", perror.NotFound("directory not found"), err)
	case errors.Is(err, fs.ErrInvalid):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("invalid directory path"), err)
	case errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %w", perror.InvalidArgument("not a directory"), err)
	}

	return err
}
