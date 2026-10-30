package authsvc

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Mikhalevich/filestorage/internal/domain/model/auth"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
)

// VerifyToken checks token signature, expiration and issuer and returns its claims.
// Any invalid token is reported as perror.TypeUnauthorized.
func (s *Service) VerifyToken(token string) (auth.Claims, error) {
	var claims jwt.RegisteredClaims

	if _, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(*jwt.Token) (any, error) {
			return s.publicKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(s.issuer),
		jwt.WithTimeFunc(s.now),
	); err != nil {
		return auth.Claims{}, fmt.Errorf("parse token: %w: %w", perror.Unauthorized("invalid token"), err)
	}

	if claims.Subject == "" {
		return auth.Claims{}, fmt.Errorf("validate subject: %w", perror.Unauthorized("invalid token"))
	}

	return auth.Claims{
		Subject:   claims.Subject,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
