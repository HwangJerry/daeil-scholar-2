package observability

import "sync/atomic"

var signupPhotoReferenced, signupPhotoReview, signupPhotoOwnership, signupPhotoOther atomic.Uint64

// Bounded reason counters contain no upload URLs, provider/session data or PII.
func RecordSignupPhotoCleanupBlocked(code string) {
	switch code {
	case "FILE_STILL_REFERENCED":
		signupPhotoReferenced.Add(1)
	case "FILE_PATH_REVIEW_REQUIRED":
		signupPhotoReview.Add(1)
	case "UPLOAD_OWNERSHIP_CHANGED":
		signupPhotoOwnership.Add(1)
	default:
		signupPhotoOther.Add(1)
	}
}
func signupPhotoCleanupBlockCounts() map[string]uint64 {
	return map[string]uint64{"referenced": signupPhotoReferenced.Load(), "review_required": signupPhotoReview.Load(), "ownership_changed": signupPhotoOwnership.Load(), "other": signupPhotoOther.Load()}
}
