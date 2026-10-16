package filesvc_test

import (
	"io"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func TestOpenFile(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	tests := []struct {
		name        string
		path        string
		wantName    string
		wantContent string
	}{
		{name: "root file", path: "a.txt", wantName: "a.txt", wantContent: "aaa"},
		{name: "root file leading slash", path: "/b.txt", wantName: "b.txt", wantContent: "bbbbb"},
		{name: "subdir file", path: "sub/c.txt", wantName: "c.txt", wantContent: "c"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			info, content, err := svc.OpenFile(tc.path)
			require.NoError(t, err)

			t.Cleanup(func() { require.NoError(t, content.Close()) })

			assert.Equal(t, tc.wantName, info.Name)
			assert.Equal(t, int64(len(tc.wantContent)), info.Size)
			assert.False(t, info.ModTime.IsZero())

			data, err := io.ReadAll(content)
			require.NoError(t, err)
			assert.Equal(t, tc.wantContent, string(data))
		})
	}
}

func TestOpenFileNotExist(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	_, _, err := svc.OpenFile("missing.txt")
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.True(t, perror.IsType(err, perror.TypeNotFound))
}

func TestOpenFileNotRegular(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	for _, p := range []string{"", "/", ".", "sub", "sub/", "empty"} {
		_, _, err := svc.OpenFile(p)
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}
}

func TestOpenFileEscapeRoot(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(filepath.Join(setupRoot(t), "sub"))

	for _, p := range []string{"../a.txt", "/../a.txt", "../sub/c.txt"} {
		_, _, err := svc.OpenFile(p)
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}
}
