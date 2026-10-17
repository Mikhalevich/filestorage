package filesvc

import (
	"fmt"
	"os"
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
// The root is closed when rootFn returns, so nothing opened from it may outlive the call.
func (s *Service) withRoot[T any](
	relativePath string,
	rootFn func(root *os.Root, fsPath string) (T, error),
) (T, error) {
	root, err := os.OpenRoot(s.rootDir)
	if err != nil {
		var zero T

		return zero, fmt.Errorf("open root: %w", err)
	}

	defer root.Close()

	return rootFn(root, toFSPath(relativePath))
}
