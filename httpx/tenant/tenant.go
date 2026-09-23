// Package tenant resolves the tenant a request is addressed to, from its host,
// and injects it in the request context.
//
// It is meant to be the outermost middleware of an HTTP chain: authentication
// resolves a user within a tenant, so the tenant must be known first.
package tenant

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

// Tenant is what the resolver needs to know about a tenant.
type Tenant interface {
	Slug() string
	Active() bool
}

// Lookup returns the tenant with the given slug, or an error matching
// ErrNotFound when there is none.
type Lookup[T Tenant] func(ctx context.Context, slug string) (T, error)

var (
	// ErrNotFound is returned by a Lookup for an unknown slug.
	ErrNotFound = errors.New("tenant not found")

	// ErrNoTenant reports a request no tenant can be resolved for. It should be
	// answered with a 404: an unknown host must not reveal whether an instance
	// exists at all.
	ErrNoTenant = errors.New("no tenant matches this request")
)

// Placeholder marks the tenant slug in a host pattern.
const Placeholder = "{tenant}"

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// IsValidSlug reports whether slug is usable as a tenant slug: a lowercase DNS
// label, since it becomes a subdomain.
func IsValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

// Options configures a Resolver.
type Options struct {
	// MultiTenant enables resolution from the host. When false, every request
	// lands on the tenant named DefaultSlug.
	MultiTenant bool
	// HostPattern frames the slug in the host, e.g. "{tenant}.example.net".
	HostPattern string
	// DefaultSlug is the tenant of a single-tenant deployment.
	DefaultSlug string
	// BaseURL is the public URL of a single-tenant deployment, whose host
	// CanonicalHost returns.
	BaseURL string
}

// Resolver turns a request host into a tenant.
type Resolver[T Tenant] struct {
	lookup Lookup[T]
	opts   Options

	// hostPrefix and hostSuffix frame the slug inside the host, derived once
	// from the pattern: matching is a prefix/suffix test.
	hostPrefix string
	hostSuffix string

	singleTenantHost string

	// The single-tenant resolution never varies: only a success is memoized,
	// so a transient lookup failure does not disable the instance.
	defaultMutex  sync.RWMutex
	defaultTenant T
	defaultCached bool
}

// NewResolver builds a resolver.
func NewResolver[T Tenant](lookup Lookup[T], opts Options) *Resolver[T] {
	prefix, suffix, _ := strings.Cut(stripPort(opts.HostPattern), Placeholder)

	return &Resolver[T]{
		lookup:           lookup,
		opts:             opts,
		hostPrefix:       strings.ToLower(prefix),
		hostSuffix:       strings.ToLower(suffix),
		singleTenantHost: canonicalHostFromBaseURL(opts.BaseURL),
	}
}

// Resolve returns the tenant addressed by the host. An unknown or inactive
// tenant yields ErrNoTenant.
func (r *Resolver[T]) Resolve(ctx context.Context, host string) (T, error) {
	var zero T

	if !r.opts.MultiTenant {
		return r.resolveDefault(ctx)
	}

	slug, ok := r.SlugFromHost(host)
	if !ok {
		return zero, errors.WithStack(ErrNoTenant)
	}

	tenant, err := r.lookup(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return zero, errors.WithStack(ErrNoTenant)
		}
		return zero, errors.WithStack(err)
	}

	// A deactivated tenant is indistinguishable from an unknown one.
	if !tenant.Active() {
		return zero, errors.WithStack(ErrNoTenant)
	}

	return tenant, nil
}

// SlugFromHost extracts the tenant slug framed by the host pattern.
func (r *Resolver[T]) SlugFromHost(host string) (string, bool) {
	host = strings.ToLower(stripPort(host))

	if len(host) <= len(r.hostPrefix)+len(r.hostSuffix) {
		return "", false
	}
	if !strings.HasPrefix(host, r.hostPrefix) || !strings.HasSuffix(host, r.hostSuffix) {
		return "", false
	}

	slug := host[len(r.hostPrefix) : len(host)-len(r.hostSuffix)]
	if !IsValidSlug(slug) {
		return "", false
	}

	return slug, true
}

// HostFor returns the host of the tenant with the given slug.
func (r *Resolver[T]) HostFor(slug string) string {
	if !r.opts.MultiTenant {
		return r.singleTenantHost
	}
	return r.hostPrefix + slug + r.hostSuffix
}

// CanonicalHost returns the normalized host designated by host without
// querying the tenants: whether this host could name a tenant, not whether
// that tenant exists. Neither a forged host, its casing, nor a client-supplied
// port can leak into a URL built from the result.
func (r *Resolver[T]) CanonicalHost(host string) (string, bool) {
	if r.opts.MultiTenant {
		slug, ok := r.SlugFromHost(host)
		if !ok {
			return "", false
		}
		return r.hostPrefix + slug + r.hostSuffix, true
	}

	if r.singleTenantHost == "" {
		return "", false
	}

	return r.singleTenantHost, true
}

func (r *Resolver[T]) resolveDefault(ctx context.Context) (T, error) {
	r.defaultMutex.RLock()
	cached, ok := r.defaultTenant, r.defaultCached
	r.defaultMutex.RUnlock()

	if ok {
		return cached, nil
	}

	tenant, err := r.lookup(ctx, r.opts.DefaultSlug)
	if err != nil {
		var zero T
		return zero, errors.WithStack(err)
	}

	r.defaultMutex.Lock()
	r.defaultTenant, r.defaultCached = tenant, true
	r.defaultMutex.Unlock()

	return tenant, nil
}

type contextKey struct{}

// WithTenant returns a context carrying the tenant.
func WithTenant[T Tenant](ctx context.Context, tenant T) context.Context {
	return context.WithValue(ctx, contextKey{}, tenant)
}

// FromContext returns the tenant carried by the context.
func FromContext[T Tenant](ctx context.Context) (T, bool) {
	tenant, ok := ctx.Value(contextKey{}).(T)
	return tenant, ok
}

// Middleware resolves the tenant of every request and injects it in the
// context. onError answers the requests whose tenant cannot be resolved; it
// receives ErrNoTenant (to answer with a 404) or a lookup failure.
func Middleware[T Tenant](resolver *Resolver[T], onError func(w http.ResponseWriter, r *http.Request, err error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, err := resolver.Resolve(r.Context(), r.Host)
			if err != nil {
				onError(w, r, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithTenant(r.Context(), tenant)))
		})
	}
}

func stripPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func canonicalHostFromBaseURL(baseURL string) string {
	if baseURL == "" {
		return ""
	}

	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return ""
	}

	return strings.ToLower(u.Host)
}
