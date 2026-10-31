package model

import (
	"time"

	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
)

type User struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	PasswordHash []byte    `db:"password_hash"`
	CreatedAt    time.Time `db:"created_at"`
}

func (u User) ToDom() user.User {
	return user.User{
		ID:           u.ID,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
	}
}

func ToDBUser(u user.User) User {
	return User{
		ID:           u.ID,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
	}
}
