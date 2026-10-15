package filesvc_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/service/filesvc"
)

func writeFile(t *testing.T, name string, content string) {
	t.Helper()

	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %q: %v", name, err)
	}
}

func makeDir(t *testing.T, name string) {
	t.Helper()

	if err := os.Mkdir(name, 0o700); err != nil {
		t.Fatalf("make dir %q: %v", name, err)
	}
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

func assertFiles(t *testing.T, got []file.File, want map[string]int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %d: %+v", len(want), len(got), got)
	}

	for _, f := range got {
		size, ok := want[f.Name]
		if !ok {
			t.Errorf("unexpected file %q", f.Name)

			continue
		}

		if f.Size != size {
			t.Errorf("file %q: expected size %d, got %d", f.Name, size, f.Size)
		}

		if f.ModTime.IsZero() {
			t.Errorf("file %q: zero mod time", f.Name)
		}
	}
}

func TestFileList(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	tests := []struct {
		name string
		path string
		want map[string]int64
	}{
		{name: "root empty path", path: "", want: map[string]int64{"a.txt": 3, "b.txt": 5}},
		{name: "root slash", path: "/", want: map[string]int64{"a.txt": 3, "b.txt": 5}},
		{name: "root dot", path: ".", want: map[string]int64{"a.txt": 3, "b.txt": 5}},
		{name: "subdir", path: "sub", want: map[string]int64{"c.txt": 1}},
		{name: "subdir leading slash", path: "/sub", want: map[string]int64{"c.txt": 1}},
		{name: "subdir trailing slash", path: "sub/", want: map[string]int64{"c.txt": 1}},
		{name: "empty dir", path: "empty", want: map[string]int64{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			files, err := svc.FileList(tc.path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertFiles(t, files, tc.want)
		})
	}
}

func TestFileListNotExist(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(setupRoot(t))

	_, err := svc.FileList("missing")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist, got %v", err)
	}
}

func TestFileListEscapeRoot(t *testing.T) {
	t.Parallel()

	svc := filesvc.New(filepath.Join(setupRoot(t), "sub"))

	for _, p := range []string{"..", "../", "/../", "../sub"} {
		if _, err := svc.FileList(p); err == nil {
			t.Errorf("path %q: expected error, got nil", p)
		}
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
		if err := os.Chtimes(fullName, modTime, modTime); err != nil {
			t.Fatalf("chtimes %q: %v", name, err)
		}
	}

	files, err := filesvc.New(rootDir).FileList("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"b.txt", "c.txt", "d.txt", "a.txt"}

	got := make([]string, 0, len(files))
	for _, f := range files {
		got = append(got, f.Name)
	}

	if !slices.Equal(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}
