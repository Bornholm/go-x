package form

import (
	"context"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ValidationRule represents a validation rule that can be applied at runtime
type ValidationRule interface {
	Validate(ctx context.Context, form *Form, field Field) error
}

// RequiredRule validates that a field is not empty
type RequiredRule struct{}

var _ ValidationRule = &RequiredRule{}

func (r RequiredRule) Validate(ctx context.Context, f *Form, field Field) error {
	found := true
	if field.IsFile() {
		if _, exists := f.Files[field.Name]; !exists {
			found = false
		}
	} else {
		value, exists := f.Values[field.Name]
		if !exists || strings.TrimSpace(value) == "" {
			found = false
		}
	}

	if !found {
		return errors.New("this field is required")
	}

	return nil
}

// MinLengthRule validates minimum string length
type MinLengthRule struct {
	MinLength int
}

var _ ValidationRule = &MinLengthRule{}

func (r MinLengthRule) Validate(ctx context.Context, f *Form, field Field) error {
	value := f.Values[field.Name]
	if len(value) < r.MinLength {
		return errors.Errorf("minimum length is %d characters", r.MinLength)
	}
	return nil
}

// MaxLengthRule validates maximum string length
type MaxLengthRule struct {
	MaxLength int
}

func (r MaxLengthRule) Validate(f *Form, field *Field) error {
	value := f.Values[field.Name]
	if len(value) > r.MaxLength {
		return errors.Errorf("maximum length is %d characters", r.MaxLength)
	}
	return nil
}

// NumberRangeRule validates number ranges
type NumberRangeRule struct {
	Min *int
	Max *int
}

var _ ValidationRule = &NumberRangeRule{}

func (r NumberRangeRule) Validate(ctx context.Context, f *Form, field Field) error {
	value := f.Values[field.Name]
	if value == "" {
		return nil // Let required rule handle empty values
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		return errors.New("must be a valid number")
	}

	if r.Min != nil && num < *r.Min {
		return errors.Errorf("must be at least %d", *r.Min)
	}

	if r.Max != nil && num > *r.Max {
		return errors.Errorf("must be at most %d", *r.Max)
	}

	return nil
}
