//go:build !linux && !darwin

// erasure_unlink_other.go — Fail closed on platforms without verified unlink support.
package service

import "github.com/dflh-saf/backend/internal/model"

func unlinkErasureFile(string) error {
	return &model.ErasureBlocked{Code: "FILE_DELETE_RETRY_REQUIRED"}
}
