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

func (s *Service) withRoot[T any](
	relativePath string,
	rootFn func(root *os.Root, fsPath string) (T, error),
) (T, error) {
	var result T

	root, err := os.OpenRoot(s.rootDir)
	if err != nil {
		return result, fmt.Errorf("open root: %w", err)
	}

	defer root.Close()

	relativeFSPath := toFSPath(relativePath)

	result, err = rootFn(root, relativeFSPath)
	if err != nil {
		return result, fmt.Errorf("fn: %w", err)
	}

	return result, nil
}
