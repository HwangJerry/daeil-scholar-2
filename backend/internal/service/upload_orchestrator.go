// UploadOrchestrator — coordinates file storage, image resize, and DB record insertion
package service

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/observability"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/rs/zerolog"
	"mime/multipart"
	"sync"
	"time"
)

type UploadOrchestrator struct {
	storage           *FileStorageService
	resizer           *ImageResizeService
	record            *FileRecordService
	siteOrigin        string
	signupCleanupMu   sync.Mutex
	signupRetirements map[signupRetirementKey]*signupRetirementRetry
	logger            zerolog.Logger
}

func NewUploadOrchestrator(storage *FileStorageService, resizer *ImageResizeService, record *FileRecordService) *UploadOrchestrator {
	return &UploadOrchestrator{storage: storage, resizer: resizer, record: record}
}

// UploadResult is the API-facing upload response.
type UploadResult struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	FSeq   int    `json:"fSeq"`
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

	fSeq, err := o.record.Record(stored, header.Filename, gate)
	if err != nil {
		_ = o.storage.DeleteUploadedURL(stored.URLPath, gate)
		return nil, err
	}

	return &UploadResult{URL: stored.URLPath, Width: width, Height: height, FSeq: fSeq}, nil
}

func (o *UploadOrchestrator) Discard(result *UploadResult, gate string) error {
	if err := o.storage.DeleteUploadedURL(result.URL, gate); err != nil {
		return err
	}
	return o.record.repo.DeleteUnassignedUpload(result.FSeq)
}

const signupPhotoReceiptTTL = 5 * time.Minute
const signupPhotoMaxUnlinkAttempts = 4

type signupRetirementKey struct {
	FileID int
	URL    string
}
type signupRetirementRetry struct {
	receipt   *repository.SignupPhotoRetirement
	attempts  int
	expiresAt time.Time
	timer     *time.Timer
}

// DiscardUnclaimedProfile accepts actual signup UploadResults, never arbitrary
// provider/form URLs. Only the exact committed receipt survives a disk failure;
// missing metadata never authorizes unlink. Retries are bounded by count and TTL.
func (o *UploadOrchestrator) DiscardUnclaimedProfile(result *UploadResult) error {
	o.signupCleanupMu.Lock()
	defer o.signupCleanupMu.Unlock()
	key := signupRetirementKey{result.FSeq, result.URL}
	pending := o.signupRetirements[key]
	if pending != nil && !pending.expiresAt.After(time.Now()) {
		delete(o.signupRetirements, key)
		if pending.timer != nil {
			pending.timer.Stop()
		}
		pending = nil
	}
	if pending == nil {
		receipt, err := o.record.repo.RetireSignupProfileUpload(result.FSeq, result.URL, o.siteOrigin)
		var blocked *model.ErasureBlocked
		if errors.As(err, &blocked) {
			observability.RecordSignupPhotoCleanupBlocked(blocked.Code)
			o.logger.Warn().Str("reason", blocked.Code).Int("fSeq", result.FSeq).Msg("social signup photo cleanup blocked")
			return nil
		}
		if err != nil || receipt == nil {
			return err
		}
		pending = &signupRetirementRetry{receipt: receipt, expiresAt: time.Now().Add(signupPhotoReceiptTTL)}
		if o.signupRetirements == nil {
			o.signupRetirements = make(map[signupRetirementKey]*signupRetirementRetry)
		}
		o.signupRetirements[key] = pending
		// Do not retain an abandoned in-memory unlink capability indefinitely.
		pending.timer = time.AfterFunc(signupPhotoReceiptTTL, func() {
			o.signupCleanupMu.Lock()
			defer o.signupCleanupMu.Unlock()
			if o.signupRetirements[key] == pending {
				delete(o.signupRetirements, key)
			}
		})
	}
	pending.attempts++
	err := pending.receipt.Erase(func(local string) error { return o.storage.DeleteUploadedURL(local, "profile") })
	if err == nil || pending.attempts >= signupPhotoMaxUnlinkAttempts {
		delete(o.signupRetirements, key)
		if pending.timer != nil {
			pending.timer.Stop()
		}
	}
	return err
}

func (o *UploadOrchestrator) SetLogger(logger zerolog.Logger) { o.logger = logger }

func (o *UploadOrchestrator) SetSiteOrigin(origin string) { o.siteOrigin = origin }
