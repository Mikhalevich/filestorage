package filesvc

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

type Service struct {
	rootDir string
}

// New creates file service. All paths passed to the service are resolved relative to rootDir.
func New(rootDir string) *Service {
	return &Service{
		rootDir: rootDir,
	}
}

// withRoot opens the service root and calls rootFn with it and relativePath converted to fs path.
// The root is closed when rootFn returns; files opened through it stay valid.
func (s *Service) withRoot[T any](
	res resource.Resource,
	rootFn func(root *os.Root, fsPath string) (T, error),
) (T, error) {
	root, err := os.OpenRoot(filepath.Join(s.rootDir, res.User))
	if err != nil {
		var zero T

		return zero, fmt.Errorf("open root: %w", err)
	}

	defer root.Close()

	return rootFn(root, toFSPath(res.Path))
}

// withRootNoReturn opens the service root and calls rootFn with it and relativePath converted to fs path.
// The root is closed when rootFn returns; files opened through it stay valid.
func (s *Service) withRootNoReturn(
	res resource.Resource,
	rootFn func(root *os.Root, fsPath string) error,
) error {
	if _, err := s.withRoot(
		res,
		func(root *os.Root, fsPath string) (struct{}, error) {
			return struct{}{}, rootFn(root, fsPath)
		},
	); err != nil {
		return fmt.Errorf("with root: %w", err)
	}

	return nil
}
