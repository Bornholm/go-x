package form

import "mime/multipart"

type File struct {
	Header *multipart.FileHeader
	File   multipart.File
}

// HasFile checks if any of the dynamic fields are file inputs
func HasFile(fields []Field) bool {
	for _, field := range fields {
		if field.IsFile() {
			return true
		}
	}
	return false
}
