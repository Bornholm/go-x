package form

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pkg/errors"
)

// Form represents a form with fields defined at runtime
type Form struct {
	Fields  []Field
	Values  map[string]string
	Files   map[string][]*multipart.FileHeader
	Errors  map[string]string
	options *FormOptions
}

// New creates a form from field definitions
func New(fields []Field, funcs ...FormOptionFunc) *Form {
	options := NewFormOptions(funcs...)

	form := &Form{
		Fields:  fields,
		Values:  make(map[string]string),
		Errors:  make(map[string]string),
		Files:   make(map[string][]*multipart.FileHeader),
		options: options,
	}

	return form
}

func (f *Form) Handle(r *http.Request) error {
	// Check if we have file fields and use appropriate parsing method
	if HasFile(f.Fields) {
		// Use ParseMultipartForm for file uploads
		if err := r.ParseMultipartForm(f.options.MaxMemory); err != nil {
			return errors.Wrap(err, "failed to parse multipart form")
		}
	} else {
		// Use regular ParseForm for non-file forms
		if err := r.ParseForm(); err != nil {
			return errors.Wrap(err, "failed to parse form")
		}
	}

	for _, field := range f.Fields {
		if field.IsFile() {
			fileHeaders, exists := r.MultipartForm.File[field.Name]
			if !exists {
				continue
			}

			f.Files[field.Name] = fileHeaders

		} else {
			f.Values[field.Name] = r.FormValue(field.Name)
		}
	}

	return nil
}

// IsValid validates all fields in the dynamic form
func (f *Form) IsValid(ctx context.Context) bool {
	f.Errors = make(map[string]string)

	for _, field := range f.Fields {
		// Apply validation rules
		for _, rule := range field.Validation {
			if err := rule.Validate(ctx, f, field); err != nil {
				f.Errors[field.Name] = err.Error()
				break // Stop at first error
			}
		}
	}

	return len(f.Errors) == 0
}

// ValidateField validates a specific field
func (f *Form) ValidateField(ctx context.Context, fieldName string) bool {
	// Clear existing error for this field
	delete(f.Errors, fieldName)

	// Find the field
	var field *Field
	for i := range f.Fields {
		if f.Fields[i].Name == fieldName {
			field = &f.Fields[i]
			break
		}
	}

	if field == nil {
		return false
	}

	// Apply validation rules
	for _, rule := range field.Validation {
		if err := rule.Validate(ctx, f, *field); err != nil {
			f.Errors[fieldName] = err.Error()
			return false
		}
	}

	return true
}

// GetFieldContext returns the rendering context for a specific field
func (f *Form) GetFieldContext(fieldName string) (FieldContext, error) {
	// Find the field
	var field *Field
	for i := range f.Fields {
		if f.Fields[i].Name == fieldName {
			field = &f.Fields[i]
			break
		}
	}

	if field == nil {
		return FieldContext{}, errors.Errorf("field %s not found", fieldName)
	}

	ctx := FieldContext{
		Field: *field,
		Error: f.Errors[field.Name],
	}

	return ctx, nil
}

// RenderField renders a specific field using the configured renderer
func (f *Form) RenderField(fieldName string) (templ.Component, error) {
	ctx, err := f.GetFieldContext(fieldName)
	if err != nil {
		return nil, err
	}

	// Find appropriate renderer
	renderer := f.findRenderer(fieldName, ctx.Type)
	return renderer.RenderField(ctx), nil
}

// GetFieldNames returns all field names
func (f *Form) GetFieldNames() []string {
	names := make([]string, len(f.Fields))
	for i, field := range f.Fields {
		names[i] = field.Name
	}
	return names
}

// findRenderer finds the appropriate renderer for a field
func (f *Form) findRenderer(fieldName, fieldType string) FieldRenderer {
	// Check for field-specific renderer
	if renderer, exists := f.options.FieldRenderers[fieldName]; exists {
		return renderer
	}

	// Check for type-specific renderer
	if renderer, exists := f.options.FieldRenderers[fieldType]; exists {
		return renderer
	}

	// Fall back to default renderer
	return f.options.DefaultRenderer
}
