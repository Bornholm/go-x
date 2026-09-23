package tenant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
)

type testTenant struct {
	slug   string
	active bool
}

func (t testTenant) Slug() string { return t.slug }
func (t testTenant) Active() bool { return t.active }

func lookupFrom(tenants ...testTenant) Lookup[testTenant] {
	return func(ctx context.Context, slug string) (testTenant, error) {
		for _, t := range tenants {
			if t.slug == slug {
				return t, nil
			}
		}
		return testTenant{}, errors.WithStack(ErrNotFound)
	}
}

func TestResolveMultiTenant(t *testing.T) {
	resolver := NewResolver(lookupFrom(testTenant{"acme", true}, testTenant{"old", false}),
		Options{MultiTenant: true, HostPattern: "{tenant}.corpus.example"})

	ctx := context.Background()

	got, err := resolver.Resolve(ctx, "ACME.corpus.example:8443")
	if err != nil || got.Slug() != "acme" {
		t.Fatalf("expected acme, got %v / %+v", got, err)
	}

	for _, host := range []string{"unknown.corpus.example", "old.corpus.example", "corpus.example", "a.b.corpus.example", "acme.evil.example", "-x.corpus.example"} {
		if _, err := resolver.Resolve(ctx, host); !errors.Is(err, ErrNoTenant) {
			t.Errorf("%s: expected ErrNoTenant, got %+v", host, err)
		}
	}
}

func TestResolveSingleTenant(t *testing.T) {
	calls := 0
	lookup := func(ctx context.Context, slug string) (testTenant, error) {
		calls++
		return testTenant{slug, true}, nil
	}

	resolver := NewResolver(Lookup[testTenant](lookup), Options{DefaultSlug: "default", BaseURL: "https://Corpus.example/"})

	for range 3 {
		got, err := resolver.Resolve(context.Background(), "whatever.example")
		if err != nil || got.Slug() != "default" {
			t.Fatalf("expected the default tenant, got %v / %+v", got, err)
		}
	}
	if calls != 1 {
		t.Errorf("the default tenant must be memoized, looked up %d times", calls)
	}

	if host, ok := resolver.CanonicalHost("forged.example"); !ok || host != "corpus.example" {
		t.Errorf("expected the configured host, got %q", host)
	}
}

func TestMiddleware(t *testing.T) {
	resolver := NewResolver(lookupFrom(testTenant{"acme", true}), Options{MultiTenant: true, HostPattern: "{tenant}.corpus.example"})

	handler := Middleware(resolver, func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "not found", http.StatusNotFound)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant, ok := FromContext[testTenant](r.Context())
		if !ok {
			t.Fatal("no tenant in context")
		}
		_, _ = w.Write([]byte(tenant.Slug()))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://acme.corpus.example/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Body.String() != "acme" {
		t.Errorf("expected acme, got %q", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "http://nope.corpus.example/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
