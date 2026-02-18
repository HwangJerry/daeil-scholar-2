// FileStorageService — disk I/O for uploaded files
package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type FileStorageService struct {
	basePath string
}

func NewFileStorageService(basePath string) *FileStorageService {
	return &FileStorageService{basePath: basePath}
}

// StoredFile holds the result of saving a file to disk.
type StoredFile struct {
	DiskPath  string // absolute path on disk
	URLPath   string // web-accessible URL path
	FileName  string // generated unique file name
	Extension string // lowercase extension including dot
	Size      int64  // original size in bytes
}

// Save writes the uploaded file to disk under basePath/gate/ with a unique name.
func (s *FileStorageService) Save(file multipart.File, header *multipart.FileHeader, gate string) (*StoredFile, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !IsAllowedImageExt(ext) {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Path traversal protection: clean the gate and verify it stays within basePath
	dir := filepath.Join(s.basePath, filepath.Clean(gate))
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid upload path")
	}
	absBase, err := filepath.Abs(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid base path")
	}
	if !strings.HasPrefix(absDir, absBase) {
		return nil, fmt.Errorf("invalid upload directory")
	}

	uniqueName := generateFileName() + ext
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, err
	}

	destPath := filepath.Join(absDir, uniqueName)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, err
	}

	// Validate actual content type via magic bytes
	if err := ValidateImageContentType(destPath); err != nil {
		os.Remove(destPath)
		return nil, err
	}

	return &StoredFile{
		DiskPath:  destPath,
		URLPath:   fmt.Sprintf("/uploads/%s/%s", gate, uniqueName),
		FileName:  uniqueName,
		Extension: ext,
		Size:      header.Size,
	}, nil
}

func generateFileName() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
