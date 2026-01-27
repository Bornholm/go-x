package form

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHasFileFields(t *testing.T) {
	// Test with file field
	fieldsWithFile := []Field{
		NewField("name"),
		NewField("document", WithType("file")),
	}
	if !HasFile(fieldsWithFile) {
		t.Error("Expected HasFile to return true for fields with file type")
	}

	// Test without file field
	fieldsWithoutFile := []Field{
		NewField("name"),
		NewField("email", WithType("email")),
	}
	if HasFile(fieldsWithoutFile) {
		t.Error("Expected HasFile to return false for fields without file type")
	}
}

func TestNewFormWithFileField(t *testing.T) {
	// Create a multipart form with file upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add regular form fields
	writer.WriteField("name", "John Doe")

	// Add file field
	fileWriter, err := writer.CreateFormFile("document", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	fileWriter.Write([]byte("test file content"))

	writer.Close()

	// Create HTTP request
	req := httptest.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Define dynamic fields
	fields := []Field{
		NewField("name"),
		NewField("document", WithType("file")),
	}

	// Test dynamic form creation
	form := New(fields)

	if err := form.Handle(req); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(form.Values["name"]) == 0 {
		t.Errorf("Expected name []string{'John Doe'}, got '%s'", form.Values["name"])
	}

	// Verify form data was parsed correctly
	if len(form.Values["name"]) > 0 && form.Values["name"][0] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", form.Values["name"])
	}
	if _, exists := form.Files["document"]; !exists {
		t.Errorf("Expected document is missing")
	}
}

func TestNewFormWithoutFileField(t *testing.T) {
	// Create a regular form (application/x-www-form-urlencoded)
	formData := "name=Jane+Doe&email=jane%40example.com"
	req := httptest.NewRequest("POST", "/test", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Define dynamic fields
	fields := []Field{
		NewField("name"),
		NewField("email", WithType("email")),
	}

	// Test dynamic form creation
	form := New(fields)

	if err := form.Handle(req); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(form.Values["name"]) == 0 {
		t.Errorf("Expected name []string{'Jane Doe'}, got '%s'", form.Values["name"])
	}

	// Verify form data was parsed correctly
	if len(form.Values["name"]) > 0 && form.Values["name"][0] != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got '%s'", form.Values["name"])
	}

	if len(form.Values["email"]) == 0 {
		t.Errorf("Expected name []string{'jane@example.com'}, got '%s'", form.Values["name"])
	}

	if len(form.Values["email"]) > 0 && form.Values["email"][0] != "jane@example.com" {
		t.Errorf("Expected email 'jane@example.com', got '%s'", form.Values["email"])
	}
}
