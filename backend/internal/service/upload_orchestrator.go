// UploadOrchestrator — coordinates file storage, image resize, and DB record insertion
package service

import "mime/multipart"

type UploadOrchestrator struct {
	storage *FileStorageService
	resizer *ImageResizeService
	record  *FileRecordService
}

func NewUploadOrchestrator(storage *FileStorageService, resizer *ImageResizeService, record *FileRecordService) *UploadOrchestrator {
	return &UploadOrchestrator{storage: storage, resizer: resizer, record: record}
}

// UploadResult is the API-facing upload response.
type UploadResult struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Upload saves the file, optionally resizes it, and records it in WEO_FILES.
func (o *UploadOrchestrator) Upload(file multipart.File, header *multipart.FileHeader, gate string) (*UploadResult, error) {
	stored, err := o.storage.Save(file, header, gate)
	if err != nil {
		return nil, err
	}

	var width, height int
	if dims, resizeErr := o.resizer.ResizeIfNeeded(stored.DiskPath); resizeErr == nil {
		width, height = dims.Width, dims.Height
	}

	_, _ = o.record.Record(stored, header.Filename, gate)

	return &UploadResult{URL: stored.URLPath, Width: width, Height: height}, nil
}
