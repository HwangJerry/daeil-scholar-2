// File validation — extension and content-type checks for uploaded files
package service

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// IsAllowedImageExt checks if the extension is an allowed image type.
func IsAllowedImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

// ValidateImageContentType reads the first 512 bytes and verifies the file is an actual image.
func ValidateImageContentType(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return err
	}
	contentType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("file content is not a valid image: %s", contentType)
	}
	return nil
}
