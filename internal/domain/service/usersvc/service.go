package usersvc

import (
	"context"

	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
)

type Repository interface {
	CreateUser(ctx context.Context, usr user.User) error
	UserByName(ctx context.Context, name string) (user.User, error)
}

type Service struct {
	repo Repository
}

func New(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}
