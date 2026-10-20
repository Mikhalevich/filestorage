package filesvc_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
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

			require.NoError(t, filesvc.New(rootDir).CreateDirectory(tc.path))

			info, err := os.Stat(filepath.Join(rootDir, tc.wantDir))
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestCreateDirectoryAlreadyExists(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	for _, p := range []string{"sub", "empty/", "a.txt"} {
		err := svc.CreateDirectory(p)
		require.ErrorIs(t, err, fs.ErrExist, "path %q", p)
		assert.True(t, perror.IsType(err, perror.TypeAlreadyExists), "path %q: %v", p, err)
	}
}

func TestCreateDirectoryParentNotExist(t *testing.T) {
	t.Parallel()

	err := filesvc.New(setupRoot(t)).CreateDirectory("missing/new")
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.True(t, perror.IsType(err, perror.TypeNotFound), "%v", err)
}

func TestCreateDirectoryParentNotDir(t *testing.T) {
	t.Parallel()

	err := filesvc.New(setupRoot(t)).CreateDirectory("a.txt/new")
	assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "%v", err)
}

func TestCreateDirectoryInvalidPath(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	for _, p := range []string{"", "/", "."} {
		err := svc.CreateDirectory(p)
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}
}

func TestCreateDirectoryEscapeRoot(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	svc := filesvc.New(filepath.Join(rootDir, "sub"))

	for _, p := range []string{"../new", "/../new"} {
		err := svc.CreateDirectory(p)
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}

	_, err := os.Stat(filepath.Join(rootDir, "new"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}
