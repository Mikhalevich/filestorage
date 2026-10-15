package filesvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
)

// FileList returns regular files located in dirPath directory relative to the service root.
// Subdirectories and non-regular files are skipped.
// Files are sorted by modification time (oldest first), ties are sorted by name.
func (s *Service) FileList(dirPath string) ([]file.File, error) {
	root, err := os.OpenRoot(s.rootDir)
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}

	defer root.Close()

	fsPath := toFSPath(dirPath)

	entries, err := fs.ReadDir(root.FS(), fsPath)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", fsPath, err)
	}

	files := make([]file.File, 0, len(entries))

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("file info %q: %w", entry.Name(), err)
		}

		files = append(files, file.File{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	slices.SortFunc(files, func(lhs, rhs file.File) int {
		if c := lhs.ModTime.Compare(rhs.ModTime); c != 0 {
			return c
		}

		return strings.Compare(lhs.Name, rhs.Name)
	})

	return files, nil
}

func toFSPath(dirPath string) string {
	cleaned := path.Clean(strings.TrimPrefix(dirPath, "/"))
	if cleaned == "" {
		return "."
	}

	return cleaned
}
