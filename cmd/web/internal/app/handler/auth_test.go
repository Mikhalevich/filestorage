package handler_test

import (
	"context"
	"errors"
	"fmt"
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

type fakeUserService struct {
	err          error
	gotName      string
	gotPassword  string
	gotOperation string
}

func (f *fakeUserService) Register(_ context.Context, name string, password string) (string, error) {
	return f.issue("register", name, password)
}

func (f *fakeUserService) Login(_ context.Context, name string, password string) (string, error) {
	return f.issue("login", name, password)
}

func (f *fakeUserService) issue(operation string, name string, password string) (string, error) {
	f.gotOperation = operation
	f.gotName = name
	f.gotPassword = password

	if f.err != nil {
		return "", f.err
	}

	return "token-" + name, nil
}

type authHandlerFn func(h *handler.Handler) func(http.ResponseWriter, *http.Request) error

func callAuth(
	t *testing.T,
	svc *fakeUserService,
	handlerFn authHandlerFn,
	body string,
) (*httptest.ResponseRecorder, error) {
	t.Helper()

	var (
		req = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/", strings.NewReader(body))
		rec = httptest.NewRecorder()
	)

	return rec, handlerFn(handler.New(nil, svc, svc))(rec, req)
}

func registerFn(h *handler.Handler) func(http.ResponseWriter, *http.Request) error {
	return h.AuthRegister
}

func loginFn(h *handler.Handler) func(http.ResponseWriter, *http.Request) error {
	return h.AuthLogin
}

func TestAuthSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		handlerFn     authHandlerFn
		wantOperation string
		wantStatus    int
	}{
		{
			name:          "register",
			handlerFn:     registerFn,
			wantOperation: "register",
			wantStatus:    http.StatusCreated,
		},
		{
			name:          "login",
			handlerFn:     loginFn,
			wantOperation: "login",
			wantStatus:    http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &fakeUserService{}

			rec, err := callAuth(t, svc, tc.handlerFn, `{"username":"alice","password":"secret123"}`)
			require.NoError(t, err)

			assert.Equal(t, tc.wantOperation, svc.gotOperation)
			assert.Equal(t, "alice", svc.gotName)
			assert.Equal(t, "secret123", svc.gotPassword)
			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			assert.JSONEq(t, `{"token":"token-alice"}`, rec.Body.String())
		})
	}
}

func TestAuthInvalidBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed json",
			body: `{"username":`,
		},
		{
			name: "too large",
			body: `{"username":"` + strings.Repeat("a", 5<<10) + `"}`,
		},
	}

	for _, tc := range tests {
		for _, fn := range []authHandlerFn{registerFn, loginFn} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				svc := &fakeUserService{}

				_, err := callAuth(t, svc, fn, tc.body)

				var httpErr *httperror.HTTPError
				require.ErrorAs(t, err, &httpErr)
				assert.Equal(t, http.StatusBadRequest, httpErr.Status)
				assert.Equal(t, "invalid credentials", httpErr.Msg)
				assert.Empty(t, svc.gotOperation)
			})
		}
	}
}

func TestAuthServiceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handlerFn  authHandlerFn
		svcErr     error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "register invalid argument",
			handlerFn:  registerFn,
			svcErr:     fmt.Errorf("validate name: %w", perror.InvalidArgument("username is reserved")),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "username is reserved",
		},
		{
			name:       "register already exists",
			handlerFn:  registerFn,
			svcErr:     fmt.Errorf("create user: %w", perror.AlreadyExists("user already exists")),
			wantStatus: http.StatusConflict,
			wantMsg:    "user already exists",
		},
		{
			name:       "login unauthorized",
			handlerFn:  loginFn,
			svcErr:     fmt.Errorf("compare password: %w", perror.Unauthorized("invalid credentials")),
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "invalid credentials",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := callAuth(t, &fakeUserService{err: tc.svcErr}, tc.handlerFn, `{"username":"alice","password":"x"}`)

			var httpErr *httperror.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, tc.wantStatus, httpErr.Status)
			assert.Equal(t, tc.wantMsg, httpErr.Msg)
		})
	}
}

func TestAuthInternalError(t *testing.T) {
	t.Parallel()

	for _, fn := range []authHandlerFn{registerFn, loginFn} {
		svcErr := errors.New("boom")

		rec, err := callAuth(t, &fakeUserService{err: svcErr}, fn, `{"username":"alice","password":"secret123"}`)
		require.ErrorIs(t, err, svcErr)

		var httpErr *httperror.HTTPError
		assert.NotErrorAs(t, err, &httpErr)
		assert.Empty(t, rec.Body.String())
	}
}
