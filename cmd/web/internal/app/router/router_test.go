package router_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/handler"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/router"
	"github.com/Mikhalevich/filestorage/internal/domain/model/auth"
	"github.com/Mikhalevich/filestorage/internal/domain/model/file"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
)

type fakeFileService struct {
	gotPath string
}

func (f *fakeFileService) FileList(res resource.Resource) ([]file.File, error) {
	f.gotPath = res.FullPath()

	return nil, nil
}

func (f *fakeFileService) OpenFile(res resource.Resource) (file.File, io.ReadSeekCloser, error) {
	f.gotPath = res.FullPath()

	return file.File{Name: "a.txt", Type: file.TypeFile}, nopCloser{strings.NewReader("abc")}, nil
}

func (f *fakeFileService) SaveFile(res resource.Resource, fileName string, _ io.Reader) (file.File, error) {
	f.gotPath = res.FullPath()

	return file.File{Name: fileName, Type: file.TypeFile}, nil
}

func (f *fakeFileService) DeleteFile(res resource.Resource) error {
	f.gotPath = res.FullPath()

	return nil
}

func (f *fakeFileService) CreateDirectory(res resource.Resource) error {
	f.gotPath = res.FullPath()

	return nil
}

type nopCloser struct {
	io.ReadSeeker
}

func (nopCloser) Close() error {
	return nil
}

// fakeVerifier accepts any non-empty token and uses it as the subject.
type fakeVerifier struct{}

func (fakeVerifier) VerifyToken(token string) (auth.Claims, error) {
	if token == "" {
		return auth.Claims{}, perror.Unauthorized("invalid token")
	}

	return auth.Claims{Subject: token}, nil
}

func TestCreateMux(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		method   string
		target   string
		token    string
		wantCode int
		wantPath string
	}{
		{
			name:     "list own directory",
			method:   http.MethodGet,
			target:   "/files/alice/sub/dir",
			token:    "alice",
			wantCode: http.StatusOK,
			wantPath: "alice/sub/dir",
		},
		{
			name:     "list own root",
			method:   http.MethodGet,
			target:   "/files/alice/",
			token:    "alice",
			wantCode: http.StatusOK,
			wantPath: "alice",
		},
		{
			name:     "download own file",
			method:   http.MethodGet,
			target:   "/download/alice/sub/a.txt",
			token:    "alice",
			wantCode: http.StatusOK,
			wantPath: "alice/sub/a.txt",
		},
		{
			name:     "delete own file",
			method:   http.MethodDelete,
			target:   "/files/alice/a.txt",
			token:    "alice",
			wantCode: http.StatusNoContent,
			wantPath: "alice/a.txt",
		},
		{
			name:     "create own directory",
			method:   http.MethodPost,
			target:   "/dirs/alice/new",
			token:    "alice",
			wantCode: http.StatusCreated,
			wantPath: "alice/new",
		},
		{
			name:     "public without token",
			method:   http.MethodGet,
			target:   "/files/public/sub",
			wantCode: http.StatusOK,
			wantPath: "public/sub",
		},
		{
			name:     "missing token",
			method:   http.MethodGet,
			target:   "/files/alice/",
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "other user",
			method:   http.MethodGet,
			target:   "/files/bob/",
			token:    "alice",
			wantCode: http.StatusForbidden,
		},
		{
			name:     "unknown route",
			method:   http.MethodGet,
			target:   "/unknown/alice/",
			token:    "alice",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				svc = &fakeFileService{}
				mux = router.CreateMux(
					handler.New(svc),
					middleware.Resource(),
					middleware.Auth(fakeVerifier{}),
				)
				rec = httptest.NewRecorder()
				req = httptest.NewRequestWithContext(t.Context(), tc.method, tc.target, nil)
			)

			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			mux.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantCode, rec.Code, rec.Body.String())
			assert.Equal(t, tc.wantPath, svc.gotPath)
		})
	}
}
