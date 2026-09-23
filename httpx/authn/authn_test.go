package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
)

func serve(t *testing.T, opts Options, authenticators ...Authenticator) *httptest.ResponseRecorder {
	t.Helper()

	handler := Middleware(opts, authenticators...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Fatal("no user in context")
		}
		_, _ = w.Write([]byte(user.Subject))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec
}

func identity(subject, tenantID string) Authenticator {
	return AuthenticatorFunc(func(w http.ResponseWriter, r *http.Request) (*User, error) {
		return &User{Subject: subject, TenantID: tenantID}, nil
	})
}

var nobody = AuthenticatorFunc(func(w http.ResponseWriter, r *http.Request) (*User, error) { return nil, nil })

func TestMiddlewareFirstIdentityWins(t *testing.T) {
	rec := serve(t, Options{}, nobody, identity("alice", ""), identity("bob", ""))
	if rec.Body.String() != "alice" {
		t.Errorf("expected alice, got %q", rec.Body.String())
	}
}

func TestMiddlewareUnauthorized(t *testing.T) {
	rec := serve(t, Options{}, nobody)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddlewareRejectsForeignTenant(t *testing.T) {
	opts := Options{RequestTenantID: func(ctx context.Context) string { return "t1" }}

	rec := serve(t, opts, identity("mallory", "t2"), identity("alice", "t1"))
	if rec.Body.String() != "alice" {
		t.Errorf("an identity of another tenant must be skipped, got %q", rec.Body.String())
	}

	rec = serve(t, opts, identity("mallory", "t2"))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddlewareErrors(t *testing.T) {
	failing := AuthenticatorFunc(func(w http.ResponseWriter, r *http.Request) (*User, error) {
		return nil, errors.New("boom")
	})
	if rec := serve(t, Options{}, failing, identity("alice", "")); rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	skipping := AuthenticatorFunc(func(w http.ResponseWriter, r *http.Request) (*User, error) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil, errors.WithStack(ErrSkipRequest)
	})
	if rec := serve(t, Options{}, skipping); rec.Code != http.StatusFound {
		t.Errorf("expected the authenticator's own answer, got %d", rec.Code)
	}
}
