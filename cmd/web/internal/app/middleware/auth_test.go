package middleware_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/internal/domain/model/resource"
	"github.com/Mikhalevich/filestorage/internal/domain/service/authsvc"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}

	svc, err := authsvc.New(
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}),
	)
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	token, err := svc.IssueToken("alice")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	tests := []struct {
		name     string
		user     string
		header   string
		wantCode int
	}{
		{name: "valid", user: "alice", header: "Bearer " + token, wantCode: http.StatusOK},
		{name: "valid lowercase scheme", user: "alice", header: "bearer " + token, wantCode: http.StatusOK},
		{name: "public without token", user: "public", header: "", wantCode: http.StatusOK},
		{name: "missing header", user: "alice", header: "", wantCode: http.StatusUnauthorized},
		{name: "basic scheme", user: "alice", header: "Basic YWxpY2U6c2VjcmV0", wantCode: http.StatusUnauthorized},
		{name: "invalid token", user: "alice", header: "Bearer " + token + "x", wantCode: http.StatusUnauthorized},
		{name: "other user", user: "bob", header: "Bearer " + token, wantCode: http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				called bool
				log    = newFakeLogger()
				rec    = httptest.NewRecorder()
				ctx    = resource.WithContext(
					logger.WithLogger(t.Context(), log),
					resource.Resource{User: tc.user},
				)
				req = httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/"+tc.user+"/", nil)
			)

			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				called = true

				logger.FromContext(r.Context()).Info("inside handler")
			})

			middleware.Auth(svc)(handler).ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("response code = %d, want %d", rec.Code, tc.wantCode)
			}

			if called != (tc.wantCode == http.StatusOK) {
				t.Fatalf("handler called = %t, want %t", called, tc.wantCode == http.StatusOK)
			}

			switch {
			case tc.wantCode == http.StatusUnauthorized:
				if rec.Header().Get("WWW-Authenticate") != "Bearer" {
					t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
				}
			case tc.wantCode == http.StatusOK && tc.header != "":
				if len(*log.entries) != 1 || (*log.entries)[0].fields["subject"] != "alice" {
					t.Errorf("handler logger missing subject field: %v", *log.entries)
				}
			}
		})
	}
}

func TestAuthMissingResource(t *testing.T) {
	t.Parallel()

	var (
		called  bool
		rec     = httptest.NewRecorder()
		req     = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/files/", nil)
		handler = http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			called = true
		})
	)

	middleware.Auth(nil)(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("response code = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	if called {
		t.Error("handler called without resource")
	}
}
