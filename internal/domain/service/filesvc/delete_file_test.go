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

func TestDeleteFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		wantGone string
		wantKept []string
	}{
		{name: "root file", path: "a.txt", wantGone: "a.txt", wantKept: []string{"b.txt", "sub/c.txt"}},
		{name: "root file leading slash", path: "/b.txt", wantGone: "b.txt", wantKept: []string{"a.txt", "sub/c.txt"}},
		{name: "subdir file", path: "sub/c.txt", wantGone: "sub/c.txt", wantKept: []string{"a.txt", "b.txt", "sub"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rootDir := setupRoot(t)

			require.NoError(t, filesvc.New(rootDir).DeleteFile(atRoot(tc.path)))

			_, err := os.Stat(filepath.Join(rootDir, tc.wantGone))
			require.ErrorIs(t, err, fs.ErrNotExist)

			for _, p := range tc.wantKept {
				_, err := os.Stat(filepath.Join(rootDir, p))
				require.NoError(t, err, "path %q", p)
			}
		})
	}
}

func TestDeleteFileNotExist(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	err := svc.DeleteFile(atRoot("missing.txt"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.True(t, perror.IsType(err, perror.TypeNotFound))

	err = svc.DeleteFile(atRoot("a.txt/x"))
	assert.True(t, perror.IsType(err, perror.TypeNotFound), "%v", err)
}

func TestDeleteFileNotRegular(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	svc := filesvc.New(rootDir)

	for _, p := range []string{"", "/", ".", "sub", "sub/", "empty"} {
		err := svc.DeleteFile(atRoot(p))
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}

	for _, p := range []string{"sub", "empty"} {
		_, err := os.Stat(filepath.Join(rootDir, p))
		require.NoError(t, err, "path %q", p)
	}
}

func TestDeleteFileEscapeRoot(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	svc := filesvc.New(filepath.Join(rootDir, "sub"))

	for _, p := range []string{"../a.txt", "/../a.txt", "../sub/c.txt"} {
		err := svc.DeleteFile(atRoot(p))
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}

	for _, p := range []string{"a.txt", "sub/c.txt"} {
		_, err := os.Stat(filepath.Join(rootDir, p))
		require.NoError(t, err, "path %q", p)
	}
}
