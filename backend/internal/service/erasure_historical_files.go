// erasure_historical_files.go — Validate a reviewed file inventory before queuing it.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const maxHistoricalErasureFiles = 500

var historicalEvidenceReference = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$`)

func ValidateHistoricalErasurePlan(plan model.ErasureHistoricalFilePlan) error {
	blocked := &model.ErasureBlocked{Code: "HISTORICAL_FILE_PLAN_REVIEW_REQUIRED"}
	if plan.RequestID <= 0 || plan.UserSeq <= 0 || !plan.OwnershipVerified || !plan.RetentionRespected || !historicalEvidenceReference.MatchString(plan.Evidence) || len(plan.Files) == 0 || len(plan.Files) > maxHistoricalErasureFiles {
		return blocked
	}
	seen := map[string]bool{}
	for _, raw := range plan.Files {
		u, err := url.Parse(raw)
		if err != nil || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" || u.Path != raw || path.Clean(raw) != raw || strings.ContainsAny(raw, "\\\x00\r\n") || len(raw) > 2048 || seen[raw] {
			return blocked
		}
		if !strings.HasPrefix(raw, "/uploads/") && !strings.HasPrefix(raw, "/files/") {
			return blocked
		}
		seen[raw] = true
	}
	return nil
}

type HistoricalErasureQueue interface {
	QueueHistoricalErasureFiles(model.ErasureHistoricalFilePlan, bool) error
}

func QueueReviewedHistoricalFiles(store HistoricalErasureQueue, files *AccountErasureFiles, plan model.ErasureHistoricalFilePlan, apply bool) error {
	if err := ValidateHistoricalErasurePlan(plan); err != nil {
		return err
	}
	if files == nil {
		return &model.ErasureBlocked{Code: "FILE_STORAGE_REQUIRED"}
	}
	for _, raw := range plan.Files {
		if err := files.InspectURL(raw); err != nil {
			return err
		}
	}
	return store.QueueHistoricalErasureFiles(plan, apply)
}
