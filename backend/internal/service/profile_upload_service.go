// profile_upload_service.go — Profile image upload operations (photo and biz card)
package service

import (
	"mime/multipart"

	"github.com/dflh-saf/backend/internal/repository"
)

type ProfileUploadService struct {
	repo         *repository.ProfileRepository
	orchestrator *UploadOrchestrator
}

func NewProfileUploadService(repo *repository.ProfileRepository, orch *UploadOrchestrator) *ProfileUploadService {
	return &ProfileUploadService{repo: repo, orchestrator: orch}
}

// UploadProfilePhoto saves the image file and updates USR_PHOTO in DB.
func (s *ProfileUploadService) UploadProfilePhoto(usrSeq int, file multipart.File, header *multipart.FileHeader) (string, error) {
	result, err := s.orchestrator.Upload(file, header, "profile")
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateProfilePhoto(usrSeq, result.URL); err != nil {
		return "", err
	}
	return result.URL, nil
}

// UploadBizCard saves the image file and updates USR_BIZ_CARD in DB.
func (s *ProfileUploadService) UploadBizCard(usrSeq int, file multipart.File, header *multipart.FileHeader) (string, error) {
	result, err := s.orchestrator.Upload(file, header, "bizcard")
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateBizCard(usrSeq, result.URL); err != nil {
		return "", err
	}
	return result.URL, nil
}
