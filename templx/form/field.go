package form

import "github.com/a-h/templ"

// Field represents a form field defined at runtime
type Field struct {
	Name string

	FieldOptions
}

func (f Field) IsFile() bool {
	return f.Type == "file"
}

type FieldOptions struct {
	Label       string
	Type        string
	Required    bool
	Placeholder string
	Description string
	Attributes  map[string]any
	Options     map[string]any
	Validation  []ValidationRule
}

type FieldOptionFunc func(opts *FieldOptions)

func WithLabel(label string) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Label = label
	}
}

func WithType(fieldType string) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Type = fieldType
	}
}

func WithRequired(required bool) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Required = required
	}
}

func WithPlaceholder(placeholder string) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Placeholder = placeholder
	}
}

func WithDescription(description string) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Description = description
	}
}

func WithAttributes(attributes map[string]any) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Attributes = attributes
	}
}

func WithAttribute(name string, value any) FieldOptionFunc {
	return func(opts *FieldOptions) {
		if opts.Attributes == nil {
			opts.Attributes = map[string]any{}
		}

		opts.Attributes[name] = value
	}
}

func WithValidation(rules ...ValidationRule) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Validation = rules
	}
}

func WithOptions(options map[string]any) FieldOptionFunc {
	return func(opts *FieldOptions) {
		opts.Options = options
	}
}

func NewFieldOptions(funcs ...FieldOptionFunc) *FieldOptions {
	opts := &FieldOptions{
		Label:       "",
		Type:        "text",
		Required:    false,
		Placeholder: "",
		Description: "",
		Attributes:  map[string]any{},
		Options:     map[string]any{},
		Validation:  make([]ValidationRule, 0),
	}
	for _, fn := range funcs {
		fn(opts)
	}
	return opts
}

func NewField(name string, funcs ...FieldOptionFunc) Field {
	opts := NewFieldOptions(funcs...)
	return Field{
		Name:         name,
		FieldOptions: *opts,
	}
}

// FieldContext contains all information needed to render a form field
type FieldContext struct {
	Field
	Value string
	Error string
	Class string
}

// FieldRenderer describes a component that can render a single field
type FieldRenderer interface {
	RenderField(fieldCtx FieldContext) templ.Component
}

func GetFieldOption[T any](fieldCtx FieldContext, name string, defaultValue T) T {
	rawAttr, exists := fieldCtx.Attributes[name]
	if !exists {
		return defaultValue
	}

	attr, ok := rawAttr.(T)
	if !ok {
		return defaultValue
	}

	return attr
}

const (
	OptSelectOptions string = "_select_options"
)

func GetSelectOptions(fieldCtx FieldContext, defaultValue []SelectOption) []SelectOption {
	return GetFieldOption(fieldCtx, OptSelectOptions, defaultValue)
}

type SelectOption struct {
	Label string
	Value string
}

func WithSelectOptions(options ...SelectOption) FieldOptionFunc {
	return func(opts *FieldOptions) {
		if opts.Options == nil {
			opts.Options = map[string]any{}
		}

		opts.Options[OptSelectOptions] = options
	}
}
