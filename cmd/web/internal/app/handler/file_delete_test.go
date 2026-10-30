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

func callFileDelete(t *testing.T, svc *fakeFileService, filePath string) (*httptest.ResponseRecorder, error) {
	t.Helper()

	target := "/files/" + testUser + "/" + url.PathEscape(filePath)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, target, nil)
	req = withResource(req, filePath)

	rec := httptest.NewRecorder()

	//nolint:wrapcheck
	return rec, handler.New(svc).FileDelete(rec, req)
}

func TestFileDeleteSuccess(t *testing.T) {
	t.Parallel()

	svc := &fakeFileService{}

	rec, err := callFileDelete(t, svc, "sub/a.txt")
	require.NoError(t, err)

	assert.Equal(t, testUser+"/sub/a.txt", svc.gotPath)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestFileDeleteErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "not found",
			svcErr:     fmt.Errorf("stat file: %w", perror.NotFound("file not found")),
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

			_, err := callFileDelete(t, &fakeFileService{err: tc.svcErr}, "a.txt")

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
		})
	}
}

func TestFileDeleteInternalError(t *testing.T) {
	t.Parallel()

	svcErr := errors.New("boom")

	rec, err := callFileDelete(t, &fakeFileService{err: svcErr}, "a.txt")
	require.ErrorIs(t, err, svcErr)

	var httpErr *httperror.HTTPError
	assert.NotErrorAs(t, err, &httpErr)
	assert.Empty(t, rec.Body.String())
}
