package bulma

import (
	"github.com/a-h/templ"
	"github.com/bornholm/go-x/templx/form"
)

// FieldRenderer provides basic HTML field rendering
type FieldRenderer struct{}

// RenderField renders a field using basic HTML
func (r *FieldRenderer) RenderField(ctx form.FieldContext) templ.Component {
	switch ctx.Type {
	case "textarea":
		return Textarea(ctx)
	case "checkbox":
		return Checkbox(ctx)
	case "file":
		return FileInput(ctx)
	case "select":
		return Input(ctx)
	default:
		return Input(ctx)
	}
}

func NewFieldRenderer() *FieldRenderer {
	return &FieldRenderer{}
}

// TextareaRenderer renders textarea fields
type TextareaRenderer struct{}

func (r *TextareaRenderer) RenderField(ctx form.FieldContext) templ.Component {
	return Textarea(ctx)
}

// CheckboxRenderer renders checkbox fields
type CheckboxRenderer struct{}

func (r *CheckboxRenderer) RenderField(ctx form.FieldContext) templ.Component {
	return Checkbox(ctx)
}

// FileRenderer renders file input fields
type FileRenderer struct{}

func (r *FileRenderer) RenderField(ctx form.FieldContext) templ.Component {
	return FileInput(ctx)
}

// SelectRenderer renders select dropdown fields
type SelectRenderer struct {
	FormOptions []form.SelectOption
}

func (r *SelectRenderer) RenderField(ctx form.FieldContext) templ.Component {
	return Select(ctx)
}

// NewSelectRenderer creates a new select renderer with FormOptions
func NewSelectRenderer(FormOptions []form.SelectOption) *SelectRenderer {
	return &SelectRenderer{FormOptions: FormOptions}
}

var _ form.FieldRenderer = &FieldRenderer{}
