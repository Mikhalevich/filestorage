package handler_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

type formPart struct {
	field    string
	fileName string
	content  string
}

func multipartBody(t *testing.T, parts []formPart) (io.Reader, string) {
	t.Helper()

	var (
		body   bytes.Buffer
		writer = multipart.NewWriter(&body)
	)

	for _, p := range parts {
		if p.fileName == "" {
			require.NoError(t, writer.WriteField(p.field, p.content))

			continue
		}

		partWriter, err := writer.CreateFormFile(p.field, p.fileName)
		require.NoError(t, err)

		_, err = io.WriteString(partWriter, p.content)
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	return &body, writer.FormDataContentType()
}

func callFileUpload(
	t *testing.T,
	svc *fakeFileService,
	dirPath string,
	body io.Reader,
	contentType string,
) (*httptest.ResponseRecorder, error) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/upload/"+dirPath, body)
	req.SetPathValue(handler.PathValueDirPath, dirPath)
	req.Header.Set("Content-Type", contentType)

	rec := httptest.NewRecorder()

	//nolint:wrapcheck
	return rec, handler.New(svc).FileUpload(rec, req)
}

func TestFileUploadSuccess(t *testing.T) {
	t.Parallel()

	svc := &fakeFileService{}
	body, contentType := multipartBody(t, []formPart{
		{field: "file", fileName: "a.txt", content: "aaa"},
		{field: "comment", content: "ignored"},
		{field: "file", fileName: "b.txt", content: "bbbbb"},
	})

	rec, err := callFileUpload(t, svc, "sub/dir", body, contentType)
	require.NoError(t, err)

	assert.Equal(t, "sub/dir", svc.gotPath)
	assert.Equal(t, []savedFile{
		{name: "a.txt", content: "aaa"},
		{name: "b.txt", content: "bbbbb"},
	}, svc.saved)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `[
		{"name":"stored_a.txt","size":3,"mod_time":"2026-01-01T00:00:00Z"},
		{"name":"stored_b.txt","size":5,"mod_time":"2026-01-01T00:00:00Z"}
	]`, rec.Body.String())
}

func TestFileUploadBadRequest(t *testing.T) {
	t.Parallel()

	noFilesBody, noFilesContentType := multipartBody(t, []formPart{{field: "comment", content: "text"}})
	emptyBody, emptyContentType := multipartBody(t, nil)

	tests := []struct {
		name        string
		body        io.Reader
		contentType string
		wantMsg     string
	}{
		{
			name:        "not multipart",
			body:        strings.NewReader("raw content"),
			contentType: "text/plain",
			wantMsg:     "multipart/form-data body expected",
		},
		{
			name:        "malformed part header",
			body:        strings.NewReader("--xyz\r\nbad header\r\n\r\ntext\r\n--xyz--\r\n"),
			contentType: "multipart/form-data; boundary=xyz",
			wantMsg:     "invalid multipart body",
		},
		{
			name:        "no file parts",
			body:        noFilesBody,
			contentType: noFilesContentType,
			wantMsg:     "no files in request",
		},
		{
			name:        "empty multipart",
			body:        emptyBody,
			contentType: emptyContentType,
			wantMsg:     "no files in request",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeFileService{}

			_, err := callFileUpload(t, svc, "dir", tc.body, tc.contentType)

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, http.StatusBadRequest, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
			assert.Empty(t, svc.saved)
		})
	}
}

func TestFileUploadServiceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "not found",
			svcErr:     fmt.Errorf("create file: %w", perror.NotFound("directory not found")),
			wantStatus: http.StatusNotFound,
			wantMsg:    "directory not found",
		},
		{
			name:       "invalid argument",
			svcErr:     fmt.Errorf("validate file name: %w", perror.InvalidArgument("invalid file name")),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid file name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body, contentType := multipartBody(t, []formPart{{field: "file", fileName: "a.txt", content: "aaa"}})

			_, err := callFileUpload(t, &fakeFileService{err: tc.svcErr}, "dir", body, contentType)

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
		})
	}
}

func TestFileUploadInternalError(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("boom")
	body, contentType := multipartBody(t, []formPart{{field: "file", fileName: "a.txt", content: "aaa"}})

	rec, err := callFileUpload(t, &fakeFileService{err: svcErr}, "dir", body, contentType)
	require.ErrorIs(t, err, svcErr)

	var httpErr *httperror.HTTPError
	assert.NotErrorAs(t, err, &httpErr)
	assert.Empty(t, rec.Body.String())
}
