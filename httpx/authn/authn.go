// Package authn chains authenticators in front of an HTTP handler and exposes
// the resulting identity through the request context.
package authn

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"
)

// ErrSkipRequest is returned by an authenticator that has already answered
// the request itself (a redirection to a login page, typically): the chain
// stops without calling the next handler.
var ErrSkipRequest = errors.New("skip request")

// User is an authenticated identity.
type User struct {
	Email       string
	Provider    string
	Subject     string
	DisplayName string
	// TokenID identifies the credential the identity was authenticated with,
	// when it is a token.
	TokenID string
	// TenantID is the tenant the identity was authenticated on. The
	// middleware rejects an identity stamped with another tenant than the
	// request's: a session cookie set on a parent domain, or a token issued
	// elsewhere, must never authenticate on this tenant.
	TenantID string
	// Scopes restricts what a token-based identity may do; empty means no
	// restriction beyond the user's own rights.
	Scopes []string
}

// Authenticator identifies the user of a request. It returns a nil user, and
// no error, when the request carries no credential it handles.
type Authenticator interface {
	Authenticate(w http.ResponseWriter, r *http.Request) (*User, error)
}

// AuthenticatorFunc adapts a function to Authenticator.
type AuthenticatorFunc func(w http.ResponseWriter, r *http.Request) (*User, error)

// Authenticate implements Authenticator.
func (fn AuthenticatorFunc) Authenticate(w http.ResponseWriter, r *http.Request) (*User, error) {
	return fn(w, r)
}

// Options configures the middleware.
type Options struct {
	// OnUnauthorized answers a request no authenticator identified.
	OnUnauthorized func(w http.ResponseWriter, r *http.Request)
	// OnError answers a request whose authentication failed. Defaults to a 500.
	OnError func(w http.ResponseWriter, r *http.Request, err error)
	// RequestTenantID returns the tenant of the request (empty when unknown),
	// against which the TenantID of an identity is checked.
	RequestTenantID func(ctx context.Context) string
}

// Middleware tries each authenticator in order and serves the request with
// the first identity found.
func Middleware(opts Options, authenticators ...Authenticator) func(http.Handler) http.Handler {
	onError := opts.OnError
	if onError == nil {
		onError = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}

	onUnauthorized := opts.OnUnauthorized
	if onUnauthorized == nil {
		onUnauthorized = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			for _, authenticator := range authenticators {
				user, err := authenticator.Authenticate(w, r)
				if err != nil {
					if errors.Is(err, ErrSkipRequest) {
						return
					}

					slog.ErrorContext(ctx, "could not authenticate user",
						slog.String("authenticator", fmt.Sprintf("%T", authenticator)),
						slog.Any("error", errors.WithStack(err)))
					onError(w, r, err)
					return
				}

				if user == nil {
					continue
				}

				// Checked once here rather than in every authenticator, so a
				// new one can not forget it.
				if opts.RequestTenantID != nil {
					if tenantID := opts.RequestTenantID(ctx); tenantID != "" && user.TenantID != "" && user.TenantID != tenantID {
						slog.WarnContext(ctx, "rejecting identity authenticated on another tenant",
							slog.String("identityTenantID", user.TenantID),
							slog.String("requestTenantID", tenantID))
						continue
					}
				}

				next.ServeHTTP(w, r.WithContext(WithUser(ctx, user)))
				return
			}

			onUnauthorized(w, r)
		})
	}
}

type contextKey struct{}

// WithUser returns a context carrying the identity.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// UserFromContext returns the identity carried by the context, if any.
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(contextKey{}).(*User)
	return user, ok && user != nil
}
