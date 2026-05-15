// AttachmentUploadOrchestrator — coordinates non-image attachment storage and DB record insertion
package service

import (
	"fmt"
	"mime/multipart"
)

type AttachmentUploadOrchestrator struct {
	storage *AttachmentStorageService
	record  *FileRecordService
}

func NewAttachmentUploadOrchestrator(storage *AttachmentStorageService, record *FileRecordService) *AttachmentUploadOrchestrator {
	return &AttachmentUploadOrchestrator{storage: storage, record: record}
}

// AttachmentUploadResult is the API-facing response for attachment uploads.
type AttachmentUploadResult struct {
	FSeq        int    `json:"fSeq"`
	URL         string `json:"url"`
	FileName    string `json:"fileName"`
	FilePath    string `json:"filePath"`
	FileOrgName string `json:"fileOrgName"`
	FileSize    string `json:"fileSize"`
}

// Upload saves the attachment, records it as F_GATE='BB' with F_JOIN_SEQ=0, and returns the new fSeq.
func (o *AttachmentUploadOrchestrator) Upload(file multipart.File, header *multipart.FileHeader, gate string) (*AttachmentUploadResult, error) {
	stored, err := o.storage.Save(file, header, gate)
	if err != nil {
		return nil, err
	}

	fSeq, err := o.record.RecordAttachment(stored, header.Filename, gate)
	if err != nil {
		return nil, err
	}

	return &AttachmentUploadResult{
		FSeq:        fSeq,
		URL:         stored.URLPath,
		FileName:    stored.FileName,
		FilePath:    fmt.Sprintf("/uploads/%s", gate),
		FileOrgName: header.Filename,
		FileSize:    formatInt64(stored.Size),
	}, nil
}
