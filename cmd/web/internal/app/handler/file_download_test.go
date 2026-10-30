package handler_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func newDownloadFileService(name string, content string) *fakeFileService {
	return &fakeFileService{
		info: file.File{
			Name:    name,
			Size:    int64(len(content)),
			ModTime: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		content: &trackedReader{Reader: strings.NewReader(content)},
	}
}

func callFileDownload(
	t *testing.T,
	svc *fakeFileService,
	filePath string,
	headers map[string]string,
) (*httptest.ResponseRecorder, error) {
	t.Helper()

	target := "/download/" + testUser + "/" + url.PathEscape(filePath)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	req = withResource(req, filePath)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()

	//nolint:wrapcheck
	return rec, handler.New(svc).FileDownload(rec, req)
}

func TestFileDownloadSuccess(t *testing.T) {
	t.Parallel()

	svc := newDownloadFileService("a.txt", "hello")

	rec, err := callFileDownload(t, svc, "sub/a.txt", nil)
	require.NoError(t, err)

	assert.Equal(t, testUser+"/sub/a.txt", svc.gotPath)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
	assert.Equal(t, "attachment; filename=\"a.txt\"", rec.Header().Get("Content-Disposition"))
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "5", rec.Header().Get("Content-Length"))
	assert.Equal(t, "Thu, 01 Jan 2026 00:00:00 GMT", rec.Header().Get("Last-Modified"))
	assert.True(t, svc.content.closed)
}

func TestFileDownloadQuotedFilename(t *testing.T) {
	t.Parallel()

	svc := newDownloadFileService("my file.txt", "hello")

	rec, err := callFileDownload(t, svc, "my file.txt", nil)
	require.NoError(t, err)

	assert.Equal(t, `attachment; filename="my file.txt"`, rec.Header().Get("Content-Disposition"))
}

func TestFileDownloadRange(t *testing.T) {
	t.Parallel()

	svc := newDownloadFileService("a.txt", "hello")

	rec, err := callFileDownload(t, svc, "a.txt", map[string]string{"Range": "bytes=1-3"})
	require.NoError(t, err)

	assert.Equal(t, http.StatusPartialContent, rec.Code)
	assert.Equal(t, "ell", rec.Body.String())
	assert.Equal(t, "bytes 1-3/5", rec.Header().Get("Content-Range"))
	assert.True(t, svc.content.closed)
}

func TestFileDownloadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "not found",
			svcErr:     fmt.Errorf("open file: %w", perror.NotFound("file not found")),
			wantStatus: http.StatusNotFound,
			wantMsg:    "file not found",
		},
		{
			name:       "invalid argument",
			svcErr:     fmt.Errorf("check file: %w", perror.InvalidArgument("not a regular file")),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "not a regular file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callFileDownload(t, &fakeFileService{err: tc.svcErr}, "a.txt", nil)

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
		})
	}
}

func TestFileDownloadInternalError(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("boom")

	rec, err := callFileDownload(t, &fakeFileService{err: svcErr}, "a.txt", nil)
	require.ErrorIs(t, err, svcErr)

	var httpErr *httperror.HTTPError
	assert.NotErrorAs(t, err, &httpErr)
	assert.Empty(t, rec.Body.String())
}
