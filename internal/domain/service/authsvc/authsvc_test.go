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

func generateKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	return pub, priv
}

func fixedTime(tm time.Time) func() time.Time {
	return func() time.Time { return tm }
}

func TestIssueAndVerify(t *testing.T) {
	t.Parallel()

	pub, priv := generateKey(t)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	svc := authsvc.New(
		pub,
		authsvc.WithPrivateKey(priv),
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

	pub, priv := generateKey(t)

	_, err := authsvc.New(pub).IssueToken("alice")
	require.Error(t, err, "verify-only service must not issue tokens")

	_, err = authsvc.New(pub, authsvc.WithPrivateKey(priv)).IssueToken("")
	require.Error(t, err)
	assert.True(t, perror.IsType(err, perror.TypeInvalidArgument), "%v", err)
}

func TestVerifyTokenInvalid(t *testing.T) {
	t.Parallel()

	pub, priv := generateKey(t)
	_, otherPriv := generateKey(t)
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
	}).SignedString([]byte(pub))
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
	}).SignedString(priv)
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
	}{
		{name: "garbage", token: "not a token"},
		{name: "empty", token: ""},
		{name: "expired", token: issue(t, priv, testIssuer, -time.Second, "alice")},
		{name: "wrong key", token: issue(t, otherPriv, testIssuer, time.Hour, "alice")},
		{name: "wrong issuer", token: issue(t, priv, "other", time.Hour, "alice")},
		{name: "empty subject", token: issue(t, priv, testIssuer, time.Hour, "")},
		{name: "no expiration", token: noExpToken},
		{name: "hs256 with public key as secret", token: hsToken},
		{name: "alg none", token: noneToken},
	}

	svc := authsvc.New(pub, authsvc.WithIssuer(testIssuer), authsvc.WithTimeFunc(fixedTime(now)))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.VerifyToken(tc.token)
			require.Error(t, err)
			assert.True(t, perror.IsType(err, perror.TypeUnauthorized), "%v", err)
		})
	}
}

func TestParseKeysPEM(t *testing.T) {
	t.Parallel()

	pub, priv := generateKey(t)

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	parsedPub, err := authsvc.ParsePublicKeyPEM(pubPEM)
	require.NoError(t, err)
	assert.True(t, pub.Equal(parsedPub))

	parsedPriv, err := authsvc.ParsePrivateKeyPEM(privPEM)
	require.NoError(t, err)
	assert.True(t, priv.Equal(parsedPriv))

	_, err = authsvc.ParsePublicKeyPEM(privPEM)
	require.Error(t, err)

	_, err = authsvc.ParsePrivateKeyPEM([]byte("garbage"))
	require.Error(t, err)
}
