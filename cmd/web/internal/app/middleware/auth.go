package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Mikhalevich/filestorage/internal/domain/model/auth"
	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

const (
	bearerPrefix = "Bearer "
)

type TokenVerifier interface {
	VerifyToken(token string) (auth.Claims, error)
}

// Auth rejects requests without a valid "Authorization: Bearer <token>" header with 401.
// Claims of a valid token are put into the request context, subject is added to the request logger.
func Auth(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rspWriter http.ResponseWriter, req *http.Request) {
			claims, err := verifyRequest(verifier, req)
			if err != nil {
				msg := "unauthorized"
				if perr, ok := perror.ParseError(err); ok {
					msg = perr.Message
				}

				rspWriter.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(rspWriter, msg, http.StatusUnauthorized)

				return
			}

			ctx := auth.WithClaims(req.Context(), claims)
			ctx = logger.WithLogger(ctx, logger.FromContext(ctx).WithField("subject", claims.Subject))

			next.ServeHTTP(rspWriter, req.WithContext(ctx))
		})
	}
}

func verifyRequest(verifier TokenVerifier, req *http.Request) (auth.Claims, error) {
	header := req.Header.Get("Authorization")
	if header == "" {
		return auth.Claims{}, errors.New("missing authorization header")
	}

	if len(header) < len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return auth.Claims{}, errors.New("authorization header is not a bearer token")
	}

	claims, err := verifier.VerifyToken(strings.TrimSpace(header[len(bearerPrefix):]))
	if err != nil {
		return auth.Claims{}, fmt.Errorf("verify token: %w", err)
	}

	return claims, nil
}
