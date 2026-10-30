package authsvc_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mikhalevich/filestorage/internal/domain/model/perror"
	"github.com/Mikhalevich/filestorage/internal/domain/service/authsvc"
)

const (
	testIssuer = "filestorage"
)

type testKey struct {
	pub     ed25519.PublicKey
	priv    ed25519.PrivateKey
	pubPEM  []byte
	privPEM []byte
}

func generateKey(t *testing.T) testKey {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)

	return testKey{
		pub:     pub,
		priv:    priv,
		pubPEM:  pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}),
		privPEM: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}),
	}
}

func newService(t *testing.T, publicKey, privateKey []byte, opts ...authsvc.Option) *authsvc.Service {
	t.Helper()

	svc, err := authsvc.New(publicKey, privateKey, opts...)
	require.NoError(t, err)

	return svc
}

func fixedTime(tm time.Time) func() time.Time {
	return func() time.Time { return tm }
}

func TestIssueAndVerify(t *testing.T) {
	t.Parallel()

	key := generateKey(t)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	svc := newService(
		t,
		key.pubPEM,
		key.privPEM,
		authsvc.WithIssuer(testIssuer),
		authsvc.WithTTL(time.Minute),
		authsvc.WithTimeFunc(fixedTime(now)),
	)

	token, err := svc.IssueToken("alice")
	require.NoError(t, err)

	claims, err := svc.VerifyToken(token)
	require.NoError(t, err)
	assert.Equal(t, "alice", claims.Subject)
	assert.True(t, now.Add(time.Minute).Equal(claims.ExpiresAt), "expires at %v", claims.ExpiresAt)
}

func TestIssueTokenErrors(t *testing.T) {
	t.Parallel()

	key := generateKey(t)

	_, err := newService(t, key.pubPEM, key.privPEM).IssueToken("")
	require.Error(t, err)
	assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "%v", err)
}

func TestVerifyTokenInvalid(t *testing.T) {
	t.Parallel()

	key := generateKey(t)
	otherKey := generateKey(t)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	issue := func(t *testing.T, priv ed25519.PrivateKey, issuer string, ttl time.Duration, subject string) string {
		t.Helper()

		token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		}).SignedString(priv)
		require.NoError(t, err)

		return token
	}

	hsToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Subject:   "alice",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString([]byte(key.pub))
	require.NoError(t, err)

	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Issuer:    testIssuer,
		Subject:   "alice",
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	noExpToken, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.RegisteredClaims{
		Issuer:  testIssuer,
		Subject: "alice",
	}).SignedString(key.priv)
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
	}{
		{name: "garbage", token: "not a token"},
		{name: "empty", token: ""},
		{name: "expired", token: issue(t, key.priv, testIssuer, -time.Second, "alice")},
		{name: "wrong key", token: issue(t, otherKey.priv, testIssuer, time.Hour, "alice")},
		{name: "wrong issuer", token: issue(t, key.priv, "other", time.Hour, "alice")},
		{name: "empty subject", token: issue(t, key.priv, testIssuer, time.Hour, "")},
		{name: "no expiration", token: noExpToken},
		{name: "hs256 with public key as secret", token: hsToken},
		{name: "alg none", token: noneToken},
	}

	svc := newService(t, key.pubPEM, key.privPEM, authsvc.WithIssuer(testIssuer), authsvc.WithTimeFunc(fixedTime(now)))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.VerifyToken(tc.token)
			require.Error(t, err)
			assert.True(t, perror.IsType(err, perror.TypeUnauthorized), "%v", err)
		})
	}
}

func TestNewInvalidKeys(t *testing.T) {
	t.Parallel()

	key := generateKey(t)

	tests := []struct {
		name       string
		publicKey  []byte
		privateKey []byte
	}{
		{name: "garbage public key", publicKey: []byte("garbage")},
		{name: "empty public key", publicKey: nil},
		{name: "private key as public key", publicKey: key.privPEM},
		{name: "empty private key", publicKey: key.pubPEM, privateKey: nil},
		{name: "garbage private key", publicKey: key.pubPEM, privateKey: []byte("garbage")},
		{name: "public key as private key", publicKey: key.pubPEM, privateKey: key.pubPEM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := authsvc.New(tc.publicKey, tc.privateKey)
			require.Error(t, err)
		})
	}
}
