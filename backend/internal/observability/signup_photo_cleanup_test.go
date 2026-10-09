package observability

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSignupPhotoCleanupMetricsAreBoundedReasons(t *testing.T) {
	before := signupPhotoCleanupBlockCounts()
	RecordSignupPhotoCleanupBlocked("FILE_STILL_REFERENCED")
	RecordSignupPhotoCleanupBlocked("FILE_PATH_REVIEW_REQUIRED")
	RecordSignupPhotoCleanupBlocked("UPLOAD_OWNERSHIP_CHANGED")
	RecordSignupPhotoCleanupBlocked("synthetic-untrusted-value")
	after := signupPhotoCleanupBlockCounts()
	for _, reason := range []string{"referenced", "review_required", "ownership_changed", "other"} {
		if after[reason] != before[reason]+1 {
			t.Errorf("counter %s did not increment", reason)
		}
	}
	encoded, err := json.Marshal(testTelemetry(t).metrics())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "daeil_social_signup_photo_cleanup_blocked") || strings.Contains(string(encoded), "synthetic-untrusted-value") {
		t.Fatal("metric missing or exposes arbitrary reason payload")
	}
}
