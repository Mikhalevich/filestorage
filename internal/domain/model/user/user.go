package user

import (
	"time"
)

// User is a registered account, its name is the owner of the user's storage directory.
type User struct {
	ID           int64
	Name         string
	PasswordHash []byte
	CreatedAt    time.Time
}
