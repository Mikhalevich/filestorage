package authsvc

import (
	"crypto/ed25519"
	"time"
)

const (
	defaultTTL = time.Hour
)

type Service struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	issuer     string
	ttl        time.Duration
	now        func() time.Time
}

type Option func(s *Service)

// WithPrivateKey enables token issuing. Without it the service can only verify tokens.
func WithPrivateKey(key ed25519.PrivateKey) Option {
	return func(s *Service) {
		s.privateKey = key
	}
}

// WithIssuer sets the issuer written into issued tokens and required in verified ones.
func WithIssuer(issuer string) Option {
	return func(s *Service) {
		s.issuer = issuer
	}
}

// WithTTL sets the lifetime of issued tokens, non-positive values are ignored.
func WithTTL(ttl time.Duration) Option {
	return func(s *Service) {
		if ttl > 0 {
			s.ttl = ttl
		}
	}
}

// WithTimeFunc overrides the clock used to issue and validate tokens.
func WithTimeFunc(now func() time.Time) Option {
	return func(s *Service) {
		s.now = now
	}
}

// New creates auth service issuing and verifying EdDSA (Ed25519) signed JWTs.
// Tokens are verified with publicKey.
func New(publicKey ed25519.PublicKey, opts ...Option) *Service {
	svc := &Service{
		publicKey: publicKey,
		ttl:       defaultTTL,
		now:       time.Now,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}
