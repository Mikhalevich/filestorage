package handler_test

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
	require.NoError(t, err)

	assert.Equal(t, "sub/dir", svc.gotPath)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `[
		{"name":"a.txt","size":3,"mod_time":"2026-01-01T00:00:00Z"},
		{"name":"b.txt","size":5,"mod_time":"2026-01-01T01:00:00Z"}
	]`, rec.Body.String())
}

func TestFileListEmpty(t *testing.T) {
	t.Parallel()

	rec, err := callFileList(t, &fakeFileService{}, "")
	require.NoError(t, err)

	assert.JSONEq(t, `[]`, rec.Body.String())
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callFileList(t, &fakeFileService{err: tc.svcErr}, "dir")

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
		})
	}
}

func TestFileListInternalError(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("boom")

	_, err := callFileList(t, &fakeFileService{err: svcErr}, "dir")
	require.ErrorIs(t, err, svcErr)

	var httpErr *httperror.HTTPError
	assert.NotErrorAs(t, err, &httpErr)
}
