package register_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/usecase/register"
)

type mocks struct {
	userSvc  *register.MockUserService
	tokenSvc *register.MockTokenService
	timeProv *register.MockTimeProvider
	usecase  *register.Usecase
}

func newMocks(t *testing.T) mocks {
	t.Helper()

	var (
		ctrl = gomock.NewController(t)
		mks  = mocks{
			userSvc:  register.NewMockUserService(ctrl),
			tokenSvc: register.NewMockTokenService(ctrl),
			timeProv: register.NewMockTimeProvider(ctrl),
		}
	)

	mks.usecase = register.New(mks.userSvc, mks.tokenSvc, mks.timeProv)

	return mks
}

func TestRegister(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mks := newMocks(t)

		gomock.InOrder(
			mks.timeProv.EXPECT().Now().Return(now),
			mks.userSvc.EXPECT().Create(t.Context(), "alice", "secret123", now).Return(nil),
			mks.tokenSvc.EXPECT().IssueToken("alice").Return("token", nil),
		)

		token, err := mks.usecase.Register(t.Context(), "alice", "secret123")
		require.NoError(t, err)
		assert.Equal(t, "token", token)
	})

	t.Run("create error", func(t *testing.T) {
		t.Parallel()

		var (
			mks       = newMocks(t)
			createErr = perror.AlreadyExists("user already exists")
		)

		mks.timeProv.EXPECT().Now().Return(now)
		mks.userSvc.EXPECT().Create(t.Context(), "alice", "secret123", now).Return(createErr)
		// no IssueToken expectation: the token must not be issued.

		token, err := mks.usecase.Register(t.Context(), "alice", "secret123")
		require.ErrorIs(t, err, createErr)
		assert.True(t, perror.IsType(err, perror.TypeAlreadyExists))
		assert.Empty(t, token)
	})

	t.Run("issue token error", func(t *testing.T) {
		t.Parallel()

		var (
			mks      = newMocks(t)
			issueErr = errors.New("sign error")
		)

		mks.timeProv.EXPECT().Now().Return(now)
		mks.userSvc.EXPECT().Create(t.Context(), "alice", "secret123", now).Return(nil)
		mks.tokenSvc.EXPECT().IssueToken("alice").Return("", issueErr)

		token, err := mks.usecase.Register(t.Context(), "alice", "secret123")
		require.ErrorIs(t, err, issueErr)
		assert.Empty(t, token)
	})
}
