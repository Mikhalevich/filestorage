package auth

import (
	"context"
	"time"
)

// Claims describes an authorized token holder.
type Claims struct {
	Subject   string
	ExpiresAt time.Time
}

type contextKey string

const (
	contextClaims = contextKey("contextClaims")
)

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, contextClaims, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(contextClaims).(Claims)

	return claims, ok
}
