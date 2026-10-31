package usersvc

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
	"github.com/Mikhalevich/filestorage/internal/domain/model/user"
)

const (
	minPasswordLen = 1
	// bcrypt ignores bytes after the 72th one.
	maxPasswordLen = 72
)

// the name becomes the user's storage directory, so it is kept to a safe single path element.
var nameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

func (s *Service) Create(
	ctx context.Context,
	name string,
	password string,
	createdAt time.Time,
) error {
	if err := validateName(name); err != nil {
		return fmt.Errorf("validate name: %w", err)
	}

	if err := validatePassword(password); err != nil {
		return fmt.Errorf("validate password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.CreateUser(ctx, user.User{
		Name:         name,
		PasswordHash: hash,
		CreatedAt:    createdAt,
	}); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func validateName(name string) error {
	if !nameRegexp.MatchString(name) {
		return perror.InvalidArgument(
			"invalid username: 3-32 characters of lowercase letters, digits, '_' and '-', starting with a letter or digit",
		)
	}

	if name == resource.PublicUser {
		return perror.InvalidArgument("username is reserved")
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLen || len(password) > maxPasswordLen {
		return perror.InvalidArgument("invalid password: must be 1-72 bytes long")
	}

	return nil
}
