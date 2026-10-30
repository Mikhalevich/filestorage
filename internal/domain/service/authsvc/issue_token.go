package authsvc

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

// IssueToken creates a signed token for subject valid for the service ttl.
func (s *Service) IssueToken(subject string) (string, error) {
	if s.privateKey == nil {
		return "", errors.New("private key is not configured")
	}

	if subject == "" {
		return "", fmt.Errorf("validate subject: %w", perror.InvalidArgument("empty subject"))
	}

	now := s.now()

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	})

	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
