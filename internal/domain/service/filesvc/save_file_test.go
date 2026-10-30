package filesvc_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func readFile(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(name)
	require.NoError(t, err)

	return string(data)
}

func TestSaveFile(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	svc := filesvc.New(rootDir)

	tests := []struct {
		name     string
		dirPath  string
		fileName string
		wantPath string
	}{
		{name: "root", dirPath: "", fileName: "new.txt", wantPath: "new.txt"},
		{name: "root slash", dirPath: "/", fileName: "new2.txt", wantPath: "new2.txt"},
		{name: "subdir", dirPath: "sub", fileName: "new.txt", wantPath: filepath.Join("sub", "new.txt")},
		{name: "subdir slashes", dirPath: "/empty/", fileName: "new.txt", wantPath: filepath.Join("empty", "new.txt")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			info, err := svc.SaveFile(atRoot(tc.dirPath), tc.fileName, strings.NewReader("content"))
			require.NoError(t, err)

			assert.Equal(t, tc.fileName, info.Name)
			assert.Equal(t, int64(len("content")), info.Size)
			assert.False(t, info.ModTime.IsZero())
			assert.Equal(t, "content", readFile(t, filepath.Join(rootDir, tc.wantPath)))
		})
	}
}

func TestSaveFileRename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fileName  string
		wantNames []string
	}{
		{name: "with extension", fileName: "a.txt", wantNames: []string{"a_1.txt", "a_2.txt"}},
		{name: "without extension", fileName: "b", wantNames: []string{"b", "b_1"}},
		{name: "dotfile", fileName: ".env", wantNames: []string{".env", ".env_1"}},
		{name: "existing dir", fileName: "sub", wantNames: []string{"sub_1", "sub_2"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rootDir := setupRoot(t)
			svc := filesvc.New(rootDir)

			for i, wantName := range tc.wantNames {
				content := wantName + strings.Repeat("!", i)

				info, err := svc.SaveFile(atRoot(""), tc.fileName, strings.NewReader(content))
				require.NoError(t, err)

				assert.Equal(t, wantName, info.Name)
				assert.Equal(t, content, readFile(t, filepath.Join(rootDir, wantName)))
			}
		})
	}
}

func TestSaveFileDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)

	_, err := filesvc.New(rootDir).SaveFile(atRoot(""), "a.txt", strings.NewReader("new"))
	require.NoError(t, err)

	assert.Equal(t, "aaa", readFile(t, filepath.Join(rootDir, "a.txt")))
}

func TestSaveFileErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dirPath  string
		fileName string
		wantType perror.Type
	}{
		{name: "dir not found", dirPath: "missing", fileName: "x.txt", wantType: perror.TypeNotFound},
		{name: "dir is file", dirPath: "c.txt", fileName: "x.txt", wantType: perror.TypeInvalidArgument},
		{name: "escape root", dirPath: "..", fileName: "x.txt", wantType: perror.TypeInvalidArgument},
		{name: "escape root slash", dirPath: "/../", fileName: "x.txt", wantType: perror.TypeInvalidArgument},
		{name: "empty name", dirPath: "", fileName: "", wantType: perror.TypeInvalidArgument},
		{name: "dot name", dirPath: "", fileName: ".", wantType: perror.TypeInvalidArgument},
		{name: "parent dir name", dirPath: "", fileName: "..", wantType: perror.TypeInvalidArgument},
		{name: "name with slash", dirPath: "", fileName: "sub/x.txt", wantType: perror.TypeInvalidArgument},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := filesvc.New(filepath.Join(setupRoot(t), "sub"))

			_, err := svc.SaveFile(atRoot(tc.dirPath), tc.fileName, strings.NewReader("content"))
			assert.True(t, perror.IsType(err, tc.wantType), "error: %v", err)
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestSaveFileReadError(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)

	content := io.MultiReader(strings.NewReader("partial"), failingReader{})

	_, err := filesvc.New(rootDir).SaveFile(atRoot(""), "new.txt", content)
	require.Error(t, err)
	assert.False(t, perror.IsType(err, perror.TypeInvalidArgument))

	assert.NoFileExists(t, filepath.Join(rootDir, "new.txt"))
}
