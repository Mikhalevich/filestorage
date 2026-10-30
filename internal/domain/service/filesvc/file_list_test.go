package filesvc_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func writeFile(t *testing.T, name string, content string) {
	t.Helper()

	require.NoError(t, os.WriteFile(name, []byte(content), 0o600))
}

const testUser = "user"

// newService creates file service with dataDir linked as testUser's directory.
func newService(t *testing.T, dataDir string) *filesvc.Service {
	t.Helper()

	storageDir := t.TempDir()
	require.NoError(t, os.Symlink(dataDir, filepath.Join(storageDir, testUser)))

	return filesvc.New(storageDir)
}

// atRoot addresses p relative to testUser's directory.
func atRoot(p string) resource.Resource {
	return resource.Resource{User: testUser, Path: p}
}

func makeDir(t *testing.T, name string) {
	t.Helper()

	require.NoError(t, os.Mkdir(name, 0o700))
}

func setupRoot(t *testing.T) string {
	t.Helper()

	rootDir := t.TempDir()

	writeFile(t, filepath.Join(rootDir, "a.txt"), "aaa")
	writeFile(t, filepath.Join(rootDir, "b.txt"), "bbbbb")
	makeDir(t, filepath.Join(rootDir, "sub"))
	writeFile(t, filepath.Join(rootDir, "sub", "c.txt"), "c")
	makeDir(t, filepath.Join(rootDir, "empty"))

	return rootDir
}

// entrySizes maps entry names to their sizes, directory names get a trailing slash.
func entrySizes(t *testing.T, files []file.File) map[string]int64 {
	t.Helper()

	sizes := make(map[string]int64, len(files))
	for _, f := range files {
		assert.False(t, f.ModTime.IsZero(), "file %q: zero mod time", f.Name)

		name := f.Name
		if f.IsDir() {
			name += "/"
		}

		sizes[name] = f.Size
	}

	return sizes
}

func TestFileList(t *testing.T) {
	t.Parallel()

	svc := newService(t, setupRoot(t))

	tests := []struct {
		name string
		path string
		want map[string]int64
	}{
		{name: "root empty path", path: "", want: map[string]int64{"a.txt": 3, "b.txt": 5, "empty/": 0, "sub/": 0}},
		{name: "root slash", path: "/", want: map[string]int64{"a.txt": 3, "b.txt": 5, "empty/": 0, "sub/": 0}},
		{name: "root dot", path: ".", want: map[string]int64{"a.txt": 3, "b.txt": 5, "empty/": 0, "sub/": 0}},
		{name: "subdir", path: "sub", want: map[string]int64{"c.txt": 1}},
		{name: "subdir leading slash", path: "/sub", want: map[string]int64{"c.txt": 1}},
		{name: "subdir trailing slash", path: "sub/", want: map[string]int64{"c.txt": 1}},
		{name: "empty dir", path: "empty", want: map[string]int64{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			files, err := svc.FileList(atRoot(tc.path))
			require.NoError(t, err)

			assert.Equal(t, tc.want, entrySizes(t, files))
		})
	}
}

func TestFileListNotExist(t *testing.T) {
	t.Parallel()

	svc := newService(t, setupRoot(t))

	_, err := svc.FileList(atRoot("missing"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.True(t, perror.IsType(err, perror.TypeNotFound))
}

func TestFileListNotDir(t *testing.T) {
	t.Parallel()

	svc := newService(t, setupRoot(t))

	_, err := svc.FileList(atRoot("a.txt"))
	require.Error(t, err)
	assert.True(t, perror.IsType(err, perror.TypeInvalidArgument))
}

func TestFileListEscapeRoot(t *testing.T) {
	t.Parallel()

	svc := newService(t, filepath.Join(setupRoot(t), "sub"))

	for _, p := range []string{"..", "../", "/../", "../sub"} {
		_, err := svc.FileList(atRoot(p))
		assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "path %q: %v", p, err)
	}
}

func TestFileListSortedByModTime(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	for name, offset := range map[string]time.Duration{
		"a.txt": 2 * time.Hour,
		"b.txt": 0,
		"c.txt": time.Hour,
		"d.txt": time.Hour,
	} {
		fullName := filepath.Join(rootDir, name)
		writeFile(t, fullName, name)

		modTime := base.Add(offset)
		require.NoError(t, os.Chtimes(fullName, modTime, modTime))
	}

	files, err := newService(t, rootDir).FileList(atRoot(""))
	require.NoError(t, err)

	got := make([]string, 0, len(files))
	for _, f := range files {
		got = append(got, f.Name)
	}

	assert.Equal(t, []string{"b.txt", "c.txt", "d.txt", "a.txt"}, got)
}

func TestFileListDirsFirst(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	for name, offset := range map[string]time.Duration{
		"b.txt":  time.Hour,
		"a.txt":  2 * time.Hour,
		"z-dir/": 0,
		"a-dir/": 3 * time.Hour,
	} {
		fullName := filepath.Join(rootDir, name)
		if strings.HasSuffix(name, "/") {
			makeDir(t, fullName)
		} else {
			writeFile(t, fullName, name)
		}

		modTime := base.Add(offset)
		require.NoError(t, os.Chtimes(fullName, modTime, modTime))
	}

	files, err := newService(t, rootDir).FileList(atRoot(""))
	require.NoError(t, err)

	got := make([]string, 0, len(files))
	for _, f := range files {
		got = append(got, f.Name)
	}

	assert.Equal(t, []string{"a-dir", "z-dir", "b.txt", "a.txt"}, got)
}

func TestFileListSkipsSymlinks(t *testing.T) {
	t.Parallel()

	rootDir := setupRoot(t)
	require.NoError(t, os.Symlink("a.txt", filepath.Join(rootDir, "link.txt")))
	require.NoError(t, os.Symlink("sub", filepath.Join(rootDir, "link-dir")))

	files, err := newService(t, rootDir).FileList(atRoot(""))
	require.NoError(t, err)

	assert.Equal(t, map[string]int64{"a.txt": 3, "b.txt": 5, "empty/": 0, "sub/": 0}, entrySizes(t, files))
}

func TestFileListUserRoot(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	makeDir(t, filepath.Join(rootDir, "alice"))
	makeDir(t, filepath.Join(rootDir, "bob"))
	writeFile(t, filepath.Join(rootDir, "alice", "a.txt"), "a")
	writeFile(t, filepath.Join(rootDir, "bob", "b.txt"), "b")

	svc := filesvc.New(rootDir)

	files, err := svc.FileList(resource.Resource{User: "alice"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "a.txt", files[0].Name)

	_, err = svc.FileList(resource.Resource{User: "alice", Path: "../bob"})
	assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "error: %v", err)
}

func TestFileListCreatesUserDir(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()

	files, err := filesvc.New(rootDir).FileList(resource.Resource{User: "carol"})
	require.NoError(t, err)
	assert.Empty(t, files)
	assert.DirExists(t, filepath.Join(rootDir, "carol"))
}

func TestInvalidUser(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	makeDir(t, filepath.Join(rootDir, "alice"))
	writeFile(t, filepath.Join(rootDir, "alice", "a.txt"), "a")

	svc := filesvc.New(filepath.Join(rootDir, "storage"))
	makeDir(t, filepath.Join(rootDir, "storage"))

	for _, user := range []string{"", ".", "..", "../alice", "alice/sub", "/alice", `..\alice`} {
		t.Run(user, func(t *testing.T) {
			t.Parallel()

			_, err := svc.FileList(resource.Resource{User: user})
			assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "error: %v", err)

			err = svc.CreateDirectory(resource.Resource{User: user, Path: "new"})
			assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "error: %v", err)
		})
	}

	entries, err := os.ReadDir(filepath.Join(rootDir, "storage"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}
