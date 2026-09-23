// Package rbac is a small permission engine: a catalog of permission codes,
// implications between them (write implies read), and permission sets built
// from roles.
package rbac

import (
	"slices"

	"github.com/pkg/errors"
)

// Permission is a permission code, e.g. "members:write".
type Permission string

// Definition describes a permission for display.
type Definition struct {
	Code  Permission
	Label string
}

// Group gathers the permissions of one section of a catalog.
type Group struct {
	Section     string
	Label       string
	Permissions []Definition
}

// Catalog lists the known permissions and their implications.
type Catalog struct {
	Groups []Group
	// Implies maps a permission to the ones it grants as well, e.g.
	// "members:write" → "members:read". Implications are transitive.
	Implies map[Permission][]Permission
}

// ErrUnknownPermission is returned for a code absent from the catalog.
var ErrUnknownPermission = errors.New("unknown permission")

// Known reports whether the catalog defines the permission.
func (c *Catalog) Known(p Permission) bool {
	for _, g := range c.Groups {
		for _, d := range g.Permissions {
			if d.Code == p {
				return true
			}
		}
	}
	return false
}

// Validate checks that every code is defined by the catalog.
func (c *Catalog) Validate(codes ...Permission) error {
	for _, code := range codes {
		if !c.Known(code) {
			return errors.Wrapf(ErrUnknownPermission, "%q", code)
		}
	}
	return nil
}

// All returns every permission of the catalog.
func (c *Catalog) All() []Permission {
	var all []Permission
	for _, g := range c.Groups {
		for _, d := range g.Permissions {
			all = append(all, d.Code)
		}
	}
	return all
}

// Set is a set of granted permissions, implications included.
type Set struct {
	granted map[Permission]struct{}
}

// NewSet builds the set granted by the given codes, expanding the catalog's
// implications.
func (c *Catalog) NewSet(codes ...Permission) Set {
	s := Set{granted: map[Permission]struct{}{}}

	pending := slices.Clone(codes)
	for len(pending) > 0 {
		p := pending[len(pending)-1]
		pending = pending[:len(pending)-1]

		if _, done := s.granted[p]; done {
			continue
		}
		s.granted[p] = struct{}{}

		pending = append(pending, c.Implies[p]...)
	}

	return s
}

// Has reports whether the set grants the permission.
func (s Set) Has(p Permission) bool {
	_, ok := s.granted[p]
	return ok
}

// HasAll reports whether the set grants every permission.
func (s Set) HasAll(permissions ...Permission) bool {
	for _, p := range permissions {
		if !s.Has(p) {
			return false
		}
	}
	return true
}

// HasAny reports whether the set grants at least one permission.
func (s Set) HasAny(permissions ...Permission) bool {
	return slices.ContainsFunc(permissions, s.Has)
}

// Union returns the permissions granted by either set.
func (s Set) Union(other Set) Set {
	u := Set{granted: make(map[Permission]struct{}, len(s.granted)+len(other.granted))}
	for p := range s.granted {
		u.granted[p] = struct{}{}
	}
	for p := range other.granted {
		u.granted[p] = struct{}{}
	}
	return u
}

// Permissions returns the granted permissions, sorted.
func (s Set) Permissions() []Permission {
	out := make([]Permission, 0, len(s.granted))
	for p := range s.granted {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}
