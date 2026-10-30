package register

import (
	"context"
	"fmt"
	"time"
)

type UserService interface {
	Create(ctx context.Context, name string, password string, createdAt time.Time) error
}

type TokenService interface {
	IssueToken(subject string) (string, error)
}

type TimeProvider interface {
	Now() time.Time
}

type Usecase struct {
	userService  UserService
	tokenService TokenService
	timeProvider TimeProvider
}

func New(
	userService UserService,
	tokenService TokenService,
	timeProvider TimeProvider,
) *Usecase {
	return &Usecase{
		userService:  userService,
		tokenService: tokenService,
		timeProvider: timeProvider,
	}
}

func (u *Usecase) Register(
	ctx context.Context,
	name string,
	password string,
) (string, error) {
	if err := u.userService.Create(
		ctx,
		name,
		password,
		u.timeProvider.Now(),
	); err != nil {
		return "", fmt.Errorf("create new user: %w", err)
	}

	token, err := u.tokenService.IssueToken(name)
	if err != nil {
		return "", fmt.Errorf("issue token: %w", err)
	}

	return token, nil
}
