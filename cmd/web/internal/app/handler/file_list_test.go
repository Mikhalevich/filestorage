package handler_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

type fakeFileService struct {
	files   []file.File
	info    file.File
	content *trackedReader
	err     error
	gotPath string
	saved   []savedFile
}

type savedFile struct {
	name    string
	content string
}

type trackedReader struct {
	*strings.Reader

	closed bool
}

func (r *trackedReader) Close() error {
	r.closed = true

	return nil
}

func (f *fakeFileService) FileList(dirPath string) ([]file.File, error) {
	f.gotPath = dirPath

	return f.files, f.err
}

func (f *fakeFileService) OpenFile(filePath string) (file.File, io.ReadSeekCloser, error) {
	f.gotPath = filePath

	if f.err != nil {
		return file.File{}, nil, f.err
	}

	return f.info, f.content, nil
}

func (f *fakeFileService) SaveFile(dirPath string, fileName string, content io.Reader) (file.File, error) {
	f.gotPath = dirPath

	if f.err != nil {
		return file.File{}, f.err
	}

	data, err := io.ReadAll(content)
	if err != nil {
		return file.File{}, fmt.Errorf("read content: %w", err)
	}

	f.saved = append(f.saved, savedFile{name: fileName, content: string(data)})

	return file.File{
		Name:    "stored_" + fileName,
		Size:    int64(len(data)),
		ModTime: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (f *fakeFileService) DeleteFile(filePath string) error {
	f.gotPath = filePath

	return f.err
}

func (f *fakeFileService) CreateDirectory(dirPath string) error {
	f.gotPath = dirPath

	return f.err
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
		wantMsg    string
	}{
		{
			name:       "not found",
			svcErr:     fmt.Errorf("read dir: %w", perror.NotFound("directory not found")),
			wantStatus: http.StatusNotFound,
			wantMsg:    "directory not found",
		},
		{
			name:       "invalid argument",
			svcErr:     fmt.Errorf("read dir: %w", perror.InvalidArgument("not a directory")),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "not a directory",
		},
		{
			name:       "already exists",
			svcErr:     fmt.Errorf("write file: %w", perror.AlreadyExists("file already exists")),
			wantStatus: http.StatusConflict,
			wantMsg:    "file already exists",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callFileList(t, &fakeFileService{err: tc.svcErr}, "dir")

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
		})
	}
}

func TestFileListInternalError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		svcErr error
	}{
		{name: "plain error", svcErr: errors.New("boom")},
		{name: "unspecified perror", svcErr: perror.New(perror.TypeUnspecified, "boom")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callFileList(t, &fakeFileService{err: tc.svcErr}, "dir")
			require.ErrorIs(t, err, tc.svcErr)

			var httpErr *httperror.HTTPError
			assert.NotErrorAs(t, err, &httpErr)
		})
	}
}
