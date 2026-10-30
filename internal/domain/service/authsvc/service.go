package authsvc

import (
	"crypto/ed25519"
	"fmt"
	"time"
)

const (
	defaultTTL = time.Hour
)

type Service struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	issuer     string
	ttl        time.Duration
	now        func() time.Time
}

type Option func(s *Service)

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
func New(publicKey, privateKey []byte, opts ...Option) (*Service, error) {
	var (
		edPublicKey  ed25519.PublicKey
		edPrivateKey ed25519.PrivateKey
		err          error
	)

	edPublicKey, err = parsePublicKeyPEM(publicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	edPrivateKey, err = parsePrivateKeyPEM(privateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	svc := &Service{
		publicKey:  edPublicKey,
		privateKey: edPrivateKey,
		ttl:        defaultTTL,
		now:        time.Now,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}
