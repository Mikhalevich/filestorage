package handler_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler/httperror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

func callDirectoryCreate(t *testing.T, svc *fakeFileService, dirPath string) (*httptest.ResponseRecorder, error) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/dirs/"+url.PathEscape(dirPath), nil)
	req.SetPathValue(handler.PathValueDirPath, dirPath)

	rec := httptest.NewRecorder()

	//nolint:wrapcheck
	return rec, handler.New(svc).DirectoryCreate(rec, req)
}

func TestDirectoryCreateSuccess(t *testing.T) {
	t.Parallel()

	svc := &fakeFileService{}

	rec, err := callDirectoryCreate(t, svc, "sub/new")
	require.NoError(t, err)

	assert.Equal(t, "sub/new", svc.gotPath)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestDirectoryCreateErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "already exists",
			svcErr:     fmt.Errorf("create directory: %w", perror.AlreadyExists("directory already exists")),
			wantStatus: http.StatusConflict,
			wantMsg:    "directory already exists",
		},
		{
			name:       "not found",
			svcErr:     fmt.Errorf("create directory: %w", perror.NotFound("parent directory not found")),
			wantStatus: http.StatusNotFound,
			wantMsg:    "parent directory not found",
		},
		{
			name:       "invalid argument",
			svcErr:     fmt.Errorf("validate dir path: %w", perror.InvalidArgument("invalid directory path")),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "invalid directory path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callDirectoryCreate(t, &fakeFileService{err: tc.svcErr}, "new")

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
		})
	}
}

func TestDirectoryCreateInternalError(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("boom")

	rec, err := callDirectoryCreate(t, &fakeFileService{err: svcErr}, "new")
	require.ErrorIs(t, err, svcErr)

	var httpErr *httperror.HTTPError
	assert.NotErrorAs(t, err, &httpErr)
	assert.Empty(t, rec.Body.String())
}
