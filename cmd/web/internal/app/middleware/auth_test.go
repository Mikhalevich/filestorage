package middleware_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mikhalevich/filestorage/cmd/web/internal/app/middleware"
	"github.com/Mikhalevich/filestorage/internal/domain/model/auth"
	"github.com/Mikhalevich/filestorage/internal/domain/service/authsvc"
	"github.com/Mikhalevich/filestorage/internal/infra/logger"
)

func TestAuth(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc := authsvc.New(pub, authsvc.WithPrivateKey(priv))

	token, err := svc.IssueToken("alice")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	tests := []struct {
		name     string
		header   string
		wantCode int
	}{
		{name: "valid", header: "Bearer " + token, wantCode: http.StatusOK},
		{name: "valid lowercase scheme", header: "bearer " + token, wantCode: http.StatusOK},
		{name: "missing header", header: "", wantCode: http.StatusUnauthorized},
		{name: "basic scheme", header: "Basic YWxpY2U6c2VjcmV0", wantCode: http.StatusUnauthorized},
		{name: "invalid token", header: "Bearer " + token + "x", wantCode: http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				gotSubject string
				log        = newFakeLogger()
				rec        = httptest.NewRecorder()
				req        = httptest.NewRequestWithContext(
					logger.WithLogger(t.Context(), log),
					http.MethodGet,
					"/files/",
					nil,
				)
			)

			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				claims, ok := auth.ClaimsFromContext(r.Context())
				if !ok {
					t.Error("claims missing in context")
				}

				gotSubject = claims.Subject

				logger.FromContext(r.Context()).Info("inside handler")
			})

			middleware.Auth(svc)(handler).ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("response code = %d, want %d", rec.Code, tc.wantCode)
			}

			if tc.wantCode != http.StatusOK {
				if rec.Header().Get("WWW-Authenticate") != "Bearer" {
					t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
				}

				return
			}

			if gotSubject != "alice" {
				t.Errorf("subject = %q, want alice", gotSubject)
			}

			if len(*log.entries) != 1 || (*log.entries)[0].fields["subject"] != "alice" {
				t.Errorf("handler logger missing subject field: %v", *log.entries)
			}
		})
	}
}
