// FileRecordService — WEO_FILES database persistence for uploaded files
package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type FileRecordService struct {
	repo *repository.FileRepository
}

func NewFileRecordService(repo *repository.FileRepository) *FileRecordService {
	return &FileRecordService{repo: repo}
}

// Record inserts a WEO_FILES row for an inline image upload (F_GATE derived from gate prefix).
func (s *FileRecordService) Record(stored *StoredFile, originalName string, gate string) (int, error) {
	return s.repo.InsertFile(&model.FileRecord{
		FGate:       strings.ToUpper(gate[:2]),
		FJoinSeq:    0,
		TypeName:    originalName,
		FileName:    stored.FileName,
		FileSize:    fmt.Sprintf("%d", stored.Size),
		FilePath:    fmt.Sprintf("/uploads/%s", gate),
		FileOrgName: originalName,
	})
}

// RecordAttachment inserts a WEO_FILES row as a notice attachment (F_GATE='BB', F_JOIN_SEQ=0).
func (s *FileRecordService) RecordAttachment(stored *StoredFile, originalName string, gate string) (int, error) {
	return s.repo.InsertFile(&model.FileRecord{
		FGate:       "BB",
		FJoinSeq:    0,
		TypeName:    originalName,
		FileName:    stored.FileName,
		FileSize:    fmt.Sprintf("%d", stored.Size),
		FilePath:    fmt.Sprintf("/uploads/%s", gate),
		FileOrgName: originalName,
	})
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
