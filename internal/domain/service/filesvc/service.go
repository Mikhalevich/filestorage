package filesvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
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

// withRoot opens the user's directory as the root and calls rootFn with it and res.Path converted to fs path.
// The user's directory is created on first use.
// The root is closed when rootFn returns; files opened through it stay valid.
func (s *Service) withRoot[T any](
	res resource.Resource,
	rootFn func(root *os.Root, fsPath string) (T, error),
) (T, error) {
	root, err := s.openUserRoot(res.User)
	if err != nil {
		var zero T

		return zero, fmt.Errorf("open user root: %w", err)
	}

	defer root.Close()

	return rootFn(root, toFSPath(res.Path))
}

// withRootNoReturn opens the user's directory as the root and calls rootFn with it and res.Path converted to fs path.
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

// openUserRoot opens the user's directory located in the service root, the directory is created if missing.
func (s *Service) openUserRoot(user string) (*os.Root, error) {
	// the user is joined to the service root directly, so it must be a single element that can't escape the root.
	if !isValidUser(user) {
		return nil, fmt.Errorf("validate user %q: %w", user, perror.InvalidArgument("invalid user"))
	}

	userDir := filepath.Join(s.rootDir, user)

	if err := os.Mkdir(userDir, dirPerm); err != nil && !errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("create user dir: %w", err)
	}

	root, err := os.OpenRoot(userDir)
	if err != nil {
		return nil, fmt.Errorf("open root: %w", err)
	}

	return root, nil
}

// isValidUser reports whether user is a single path element other than "." and "..".
func isValidUser(user string) bool {
	return fs.ValidPath(user) && user != "." && !strings.ContainsAny(user, `/\`)
}
