package usersvc_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
	"github.com/Mikhalevich/filestorage/internal/domain/service/usersvc"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	var (
		ctrl      = gomock.NewController(t)
		repo      = usersvc.NewMockRepository(ctrl)
		svc       = usersvc.New(repo)
		createdAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		created   user.User
	)

	repo.EXPECT().CreateUser(t.Context(), gomock.Any()).DoAndReturn(
		func(_ context.Context, usr user.User) error {
			created = usr

			return nil
		},
	)

	require.NoError(t, svc.Create(t.Context(), "alice", "secret123", createdAt))

	assert.Equal(t, "alice", created.Name)
	assert.Equal(t, createdAt, created.CreatedAt)
	assert.NotEqual(t, []byte("secret123"), created.PasswordHash)
	require.NoError(t, bcrypt.CompareHashAndPassword(created.PasswordHash, []byte("secret123")))
}

func TestCreateInvalidArgument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
		wantMsg  string
	}{
		{name: "empty name", username: "", password: "secret123"},
		{name: "short name", username: "ab", password: "secret123"},
		{name: "long name", username: strings.Repeat("a", 33), password: "secret123"},
		{name: "uppercase", username: "Alice", password: "secret123"},
		{name: "path separator", username: "al/ice", password: "secret123"},
		{name: "parent dir", username: "..", password: "secret123"},
		{name: "leading dash", username: "-alice", password: "secret123"},
		{name: "reserved", username: "public", password: "secret123", wantMsg: "username is reserved"},
		{name: "empty password", username: "alice", password: ""},
		{name: "long password", username: "alice", password: strings.Repeat("a", 73)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctrl = gomock.NewController(t)
				repo = usersvc.NewMockRepository(ctrl)
			)

			// no CreateUser expectation: the repository must not be called.
			err := usersvc.New(repo).Create(t.Context(), tc.username, tc.password, time.Now())

			perr, ok := perror.ParseError(err)
			require.True(t, ok, err)
			assert.Equal(t, perror.TypeInvalidArgument, perr.Type)

			if tc.wantMsg != "" {
				assert.Equal(t, tc.wantMsg, perr.Message)
			}
		})
	}
}

func TestCreateBoundaryLengths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "min name", username: "abc", password: "secret123"},
		{name: "max name", username: strings.Repeat("a", 32), password: "secret123"},
		{name: "name with dash and underscore", username: "a-b_c", password: "secret123"},
		{name: "min password", username: "alice", password: strings.Repeat("a", 8)},
		{name: "max password", username: "alice", password: strings.Repeat("a", 72)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctrl = gomock.NewController(t)
				repo = usersvc.NewMockRepository(ctrl)
			)

			repo.EXPECT().CreateUser(t.Context(), gomock.Any()).Return(nil)

			require.NoError(t, usersvc.New(repo).Create(t.Context(), tc.username, tc.password, time.Now()))
		})
	}
}

func TestCreateRepositoryError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repoErr error
	}{
		{name: "already exists", repoErr: perror.AlreadyExists("user already exists")},
		{name: "internal", repoErr: errors.New("connection refused")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctrl = gomock.NewController(t)
				repo = usersvc.NewMockRepository(ctrl)
			)

			repo.EXPECT().CreateUser(t.Context(), gomock.Any()).Return(tc.repoErr)

			err := usersvc.New(repo).Create(t.Context(), "alice", "secret123", time.Now())
			require.ErrorIs(t, err, tc.repoErr)
		})
	}
}

func TestUserByNameAndPwd(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	require.NoError(t, err)

	stored := user.User{ID: 1, Name: "alice", PasswordHash: hash}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl = gomock.NewController(t)
			repo = usersvc.NewMockRepository(ctrl)
		)

		repo.EXPECT().UserByName(t.Context(), "alice").Return(stored, nil)

		usr, err := usersvc.New(repo).UserByNameAndPwd(t.Context(), "alice", "secret123")
		require.NoError(t, err)
		assert.Equal(t, stored, usr)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl = gomock.NewController(t)
			repo = usersvc.NewMockRepository(ctrl)
		)

		repo.EXPECT().UserByName(t.Context(), "alice").Return(stored, nil)

		_, err := usersvc.New(repo).UserByNameAndPwd(t.Context(), "alice", "wrong")
		require.ErrorIs(t, err, bcrypt.ErrMismatchedHashAndPassword)
		assert.True(t, perror.IsType(err, perror.TypeUnauthorized), err)
	})

	t.Run("unknown user", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl = gomock.NewController(t)
			repo = usersvc.NewMockRepository(ctrl)
		)

		repo.EXPECT().UserByName(t.Context(), "bob").Return(user.User{}, perror.NotFound("user not found"))

		_, err := usersvc.New(repo).UserByNameAndPwd(t.Context(), "bob", "secret123")
		assert.True(t, perror.IsType(err, perror.TypeNotFound), err)
	})

	t.Run("repository error", func(t *testing.T) {
		t.Parallel()

		var (
			ctrl    = gomock.NewController(t)
			repo    = usersvc.NewMockRepository(ctrl)
			repoErr = errors.New("connection refused")
		)

		repo.EXPECT().UserByName(t.Context(), "alice").Return(user.User{}, repoErr)

		_, err := usersvc.New(repo).UserByNameAndPwd(t.Context(), "alice", "secret123")
		require.ErrorIs(t, err, repoErr)
		assert.False(t, perror.IsType(err, perror.TypeNotFound))
	})
}
