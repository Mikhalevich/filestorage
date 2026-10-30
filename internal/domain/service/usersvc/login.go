package usersvc

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
)

// dummyHash is compared against for unknown users, so they take as long as a wrong password.
//
//nolint:gochecknoglobals
var dummyHash = mustHash("dummy password")

func (s *Service) UserByNameAndPwd(
	ctx context.Context,
	name string,
	password string,
) (user.User, error) {
	usr, err := s.repo.UserByName(ctx, name)
	if err != nil {
		if !perror.IsType(err, perror.TypeNotFound) {
			return user.User{}, fmt.Errorf("get user: %w", err)
		}

		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))

		return user.User{}, fmt.Errorf("get user: %w", perror.NotFound("user not found"))
	}

	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(password)); err != nil {
		return user.User{}, fmt.Errorf("compare password: %w: %w", perror.Unauthorized("invalid credentials"), err)
	}

	return usr, nil
}

func mustHash(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("hash password: %v", err))
	}

	return hash
}
