package handler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
)

type fakeFileService struct {
	files   []file.File
	err     error
	gotPath string
}

func (f *fakeFileService) FileList(dirPath string) ([]file.File, error) {
	f.gotPath = dirPath

	return f.files, f.err
}

func callFileList(t *testing.T, svc *fakeFileService, dirPath string) (*httptest.ResponseRecorder, error) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/files/"+dirPath, nil)
	req.SetPathValue(handler.PathValueDirPath, dirPath)

	rec := httptest.NewRecorder()

	//nolint:wrapcheck
	return rec, handler.New(svc).FileList(rec, req)
}

func TestFileListSuccess(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	svc := &fakeFileService{
		files: []file.File{
			{Name: "a.txt", Size: 3, ModTime: modTime},
			{Name: "b.txt", Size: 5, ModTime: modTime.Add(time.Hour)},
		},
	}

	rec, err := callFileList(t, svc, "sub/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if svc.gotPath != "sub/dir" {
		t.Errorf("expected path %q, got %q", "sub/dir", svc.gotPath)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content type application/json, got %q", ct)
	}

	want := `[{"name":"a.txt","size":3,"mod_time":"2026-01-01T00:00:00Z"},` +
		`{"name":"b.txt","size":5,"mod_time":"2026-01-01T01:00:00Z"}]`
	if got := rec.Body.String(); got != want {
		t.Errorf("expected body %s, got %s", want, got)
	}
}

func TestFileListEmpty(t *testing.T) {
	t.Parallel()

	rec, err := callFileList(t, &fakeFileService{}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var body []any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}

	if body == nil || len(body) != 0 {
		t.Errorf("expected empty json array, got %q", rec.Body.String())
	}
}

func TestFileListErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{name: "not exist", svcErr: fmt.Errorf("read dir: %w", fs.ErrNotExist), wantStatus: http.StatusNotFound},
		{name: "invalid path", svcErr: fmt.Errorf("read dir: %w", fs.ErrInvalid), wantStatus: http.StatusBadRequest},
		{name: "not dir", svcErr: fmt.Errorf("read dir: %w", syscall.ENOTDIR), wantStatus: http.StatusBadRequest},
		{name: "other", svcErr: errors.New("boom"), wantStatus: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callFileList(t, &fakeFileService{err: tc.svcErr}, "dir")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			httpErr, ok := errors.AsType[*httperror.HTTPError](err)
			if tc.wantStatus == 0 {
				if ok {
					t.Fatalf("expected generic error, got http error %v", httpErr)
				}

				return
			}

			if !ok {
				t.Fatalf("expected http error, got %v", err)
			}

			if httpErr.Status != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, httpErr.Status)
			}
		})
	}
}
