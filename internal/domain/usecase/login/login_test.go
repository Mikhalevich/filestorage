package login_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
	"github.com/Mikhalevich/filestorage/internal/domain/usecase/login"
)

type mocks struct {
	userSvc  *login.MockUserService
	tokenSvc *login.MockTokenService
	usecase  *login.Usecase
}

func newMocks(t *testing.T) mocks {
	t.Helper()

	var (
		ctrl = gomock.NewController(t)
		mks  = mocks{
			userSvc:  login.NewMockUserService(ctrl),
			tokenSvc: login.NewMockTokenService(ctrl),
		}
	)

	mks.usecase = login.New(mks.userSvc, mks.tokenSvc)

	return mks
}

func TestLoginSuccess(t *testing.T) {
	t.Parallel()

	mks := newMocks(t)

	gomock.InOrder(
		mks.userSvc.EXPECT().UserByNameAndPwd(t.Context(), "alice", "secret123").Return(user.User{Name: "alice"}, nil),
		mks.tokenSvc.EXPECT().IssueToken("alice").Return("token", nil),
	)

	token, err := mks.usecase.Login(t.Context(), "alice", "secret123")
	require.NoError(t, err)
	assert.Equal(t, "token", token)
}

func TestLoginUnauthorized(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		svcErr error
	}{
		{
			name:   "unknown user",
			svcErr: fmt.Errorf("get user: %w", perror.NotFound("user not found")),
		},
		{
			name:   "wrong password",
			svcErr: fmt.Errorf("compare password: %w", perror.Unauthorized("invalid credentials")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mks := newMocks(t)

			mks.userSvc.EXPECT().UserByNameAndPwd(t.Context(), "alice", "secret123").Return(user.User{}, tc.svcErr)
			// no IssueToken expectation: the token must not be issued.

			token, err := mks.usecase.Login(t.Context(), "alice", "secret123")
			assert.Empty(t, token)

			// unknown user and wrong password look the same to the client.
			perr, ok := perror.ParseError(err)
			require.True(t, ok, err)
			assert.Equal(t, perror.TypeUnauthorized, perr.Type)
			assert.Equal(t, "invalid credentials", perr.Message)
		})
	}
}

func TestLoginErrors(t *testing.T) {
	t.Parallel()

	t.Run("user service error", func(t *testing.T) {
		t.Parallel()

		var (
			mks    = newMocks(t)
			svcErr = errors.New("connection refused")
		)

		mks.userSvc.EXPECT().UserByNameAndPwd(t.Context(), "alice", "secret123").Return(user.User{}, svcErr)

		token, err := mks.usecase.Login(t.Context(), "alice", "secret123")
		require.ErrorIs(t, err, svcErr)
		assert.False(t, perror.IsType(err, perror.TypeUnauthorized))
		assert.Empty(t, token)
	})

	t.Run("issue token error", func(t *testing.T) {
		t.Parallel()

		var (
			mks      = newMocks(t)
			issueErr = errors.New("sign error")
		)

		mks.userSvc.EXPECT().UserByNameAndPwd(t.Context(), "alice", "secret123").Return(user.User{Name: "alice"}, nil)
		mks.tokenSvc.EXPECT().IssueToken("alice").Return("", issueErr)

		token, err := mks.usecase.Login(t.Context(), "alice", "secret123")
		require.ErrorIs(t, err, issueErr)
		assert.Empty(t, token)
	})
}
