package form

// FormOptions holds configuration for form behavior and rendering
type FormOptions struct {
	// FieldRenderers maps field names or types to custom renderers
	FieldRenderers map[string]FieldRenderer
	// DefaultRenderer is used when no specific renderer is found
	DefaultRenderer FieldRenderer
	// MaxMemory is the max memory allocated to the parsing of a multipart form
	MaxMemory int64
}

type FormOptionFunc func(opts *FormOptions)

func NewFormOptions(funcs ...FormOptionFunc) *FormOptions {
	opts := &FormOptions{
		FieldRenderers:  make(map[string]FieldRenderer),
		DefaultRenderer: NewDefaultRenderer(),
		MaxMemory:       32 << 20,
	}

	for _, fn := range funcs {
		fn(opts)
	}

	return opts
}

func WithFieldRenderer(fieldType string, r FieldRenderer) FormOptionFunc {
	return func(opts *FormOptions) {
		opts.FieldRenderers[fieldType] = r
	}
}

func WithDefaultRenderer(r FieldRenderer) FormOptionFunc {
	return func(opts *FormOptions) {
		opts.DefaultRenderer = r
	}
}

func WithMaxMemory(max int64) FormOptionFunc {
	return func(opts *FormOptions) {
		opts.MaxMemory = max
	}
}
