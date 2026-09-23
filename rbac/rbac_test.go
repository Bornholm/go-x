package rbac

import (
	"testing"

	"github.com/pkg/errors"
)

var catalog = &Catalog{
	Groups: []Group{
		{Section: "members", Permissions: []Definition{{Code: "members:read"}, {Code: "members:write"}}},
		{Section: "jobs", Permissions: []Definition{{Code: "jobs:read"}, {Code: "jobs:write"}, {Code: "jobs:run"}}},
	},
	Implies: map[Permission][]Permission{
		"members:write": {"members:read"},
		"jobs:write":    {"jobs:read"},
		"jobs:run":      {"jobs:read"},
	},
}

func TestSetExpandsImplications(t *testing.T) {
	s := catalog.NewSet("members:write", "jobs:run")

	if !s.HasAll("members:write", "members:read", "jobs:run", "jobs:read") {
		t.Errorf("implications not expanded: %v", s.Permissions())
	}
	if s.Has("jobs:write") {
		t.Error("an implication must not grant more than declared")
	}
	if !s.HasAny("jobs:write", "jobs:read") || s.HasAny("jobs:write") {
		t.Error("HasAny mismatch")
	}
}

func TestUnion(t *testing.T) {
	u := catalog.NewSet("members:read").Union(catalog.NewSet("jobs:write"))
	if !u.HasAll("members:read", "jobs:write", "jobs:read") {
		t.Errorf("unexpected union %v", u.Permissions())
	}
}

func TestValidate(t *testing.T) {
	if err := catalog.Validate("members:read", "jobs:run"); err != nil {
		t.Errorf("unexpected error %+v", err)
	}
	if err := catalog.Validate("members:admin"); !errors.Is(err, ErrUnknownPermission) {
		t.Errorf("expected ErrUnknownPermission, got %+v", err)
	}
}
