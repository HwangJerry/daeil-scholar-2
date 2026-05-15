// AttachmentStorageService — disk I/O for non-image notice attachment uploads
package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

type AttachmentStorageService struct {
	basePath string
}

func NewAttachmentStorageService(basePath string) *AttachmentStorageService {
	return &AttachmentStorageService{basePath: basePath}
}

// Save writes an attachment to <basePath>/<gate>/<unique><ext> after extension policy checks.
func (s *AttachmentStorageService) Save(file multipart.File, header *multipart.FileHeader, gate string) (*StoredFile, error) {
	ext := ExtensionOf(header.Filename)
	if IsBlockedAttachmentExt(ext) {
		return nil, fmt.Errorf("blocked file type: %s", ext)
	}
	if !IsAllowedAttachmentExt(ext) {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

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

	return &StoredFile{
		DiskPath:  destPath,
		URLPath:   fmt.Sprintf("/uploads/%s/%s", gate, uniqueName),
		FileName:  uniqueName,
		Extension: ext,
		Size:      header.Size,
	}, nil
}
