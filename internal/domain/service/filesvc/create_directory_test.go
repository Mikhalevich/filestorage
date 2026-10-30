package filesvc_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

func TestCreateDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantDir string
	}{
		{name: "root dir", path: "new", wantDir: "new"},
		{name: "root dir leading slash", path: "/new", wantDir: "new"},
		{name: "subdir", path: "sub/new", wantDir: "sub/new"},
		{name: "subdir trailing slash", path: "empty/new/", wantDir: "empty/new"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rootDir := setupRoot(t)

			require.NoError(t, newService(t, rootDir).CreateDirectory(atRoot(tc.path)))

			info, err := os.Stat(filepath.Join(rootDir, tc.wantDir))
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestCreateDirectoryAlreadyExists(t *testing.T) {
	t.Parallel()

	svc := newService(t, setupRoot(t))

	for _, p := range []string{"sub", "empty/", "a.txt"} {
		err := svc.CreateDirectory(atRoot(p))
		require.ErrorIs(t, err, fs.ErrExist, "path %q", p)
		assert.True(t, perror.IsType(err, perror.TypeAlreadyExists), "path %q: %v", p, err)
	}
}

func TestCreateDirectoryParentNotExist(t *testing.T) {
	t.Parallel()

	err := newService(t, setupRoot(t)).CreateDirectory(atRoot("missing/new"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.True(t, perror.IsType(err, perror.TypeNotFound), "%v", err)
}

func TestCreateDirectoryParentNotDir(t *testing.T) {
	t.Parallel()

	err := newService(t, setupRoot(t)).CreateDirectory(atRoot("a.txt/new"))
	assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "%v", err)
}

func TestCreateDirectoryInvalidPath(t *testing.T) {
	t.Parallel()

	svc := newService(t, setupRoot(t))

	for _, p := range []string{"", "/", "."} {
		err := svc.CreateDirectory(atRoot(p))
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}
}

func TestCreateDirectoryEscapeRoot(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	svc := newService(t, filepath.Join(rootDir, "sub"))

	for _, p := range []string{"../new", "/../new"} {
		err := svc.CreateDirectory(atRoot(p))
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}

	_, err := os.Stat(filepath.Join(rootDir, "new"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}
