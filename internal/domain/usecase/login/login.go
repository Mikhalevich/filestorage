package login

import (
	"context"
	"fmt"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
)

type UserService interface {
	UserByNameAndPwd(
		ctx context.Context,
		name string,
		password string,
	) (user.User, error)
}

type TokenService interface {
	IssueToken(subject string) (string, error)
}

type Usecase struct {
	userService  UserService
	tokenService TokenService
}

func New(
	userService UserService,
	tokenService TokenService,
) *Usecase {
	return &Usecase{
		userService:  userService,
		tokenService: tokenService,
	}
}

func (u *Usecase) Login(
	ctx context.Context,
	name string,
	password string,
) (string, error) {
	usr, err := u.userService.UserByNameAndPwd(ctx, name, password)
	if err != nil {
		if perror.IsType(err, perror.TypeNotFound) {
			return "", perror.Unauthorized("invalid credentials")
		}

		return "", fmt.Errorf("user by credentials: %w", err)
	}

	token, err := u.tokenService.IssueToken(usr.Name)
	if err != nil {
		return "", fmt.Errorf("issue token: %w", err)
	}

	return token, nil
}
