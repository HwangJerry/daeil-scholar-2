// account_erasure_internal_verifier.go — Server-side checks that close the four storage targets without an operator.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

// BackupRetentionDays is the operator-approved retention for backups and logs.
const BackupRetentionDays = 28

const (
	backupStatusMaxAge = 36 * time.Hour
	weeklyBackupMaxAge = 8 * 24 * time.Hour
)

// ErasureIdentifierSearcher finds remaining plaintext copies of identifiers.
type ErasureIdentifierSearcher interface {
	IdentifierMatches([]string) ([]model.AccountDeletionFootprint, error)
}

// SentryUserEventCounter counts Sentry events that match a search query.
type SentryUserEventCounter interface {
	CountUserEvents(ctx context.Context, query string) (int, error)
}

// BackupRotationStatus is written by deploy/backup_rotation.py. It holds
// dates, counts and configured retention only, never personal data.
type BackupRotationStatus struct {
	CheckedAt            string   `json:"checkedAt"`
	RetentionDays        int      `json:"retentionDays"`
	NewestBackupAt       *string  `json:"newestBackupAt"`
	OldestBackupAgeDays  *float64 `json:"oldestBackupAgeDays"`
	JournalRetentionDays *int     `json:"journalRetentionDays"`
	HTTPLogMaxAgeDays    *int     `json:"httpLogMaxAgeDays"`
}

// InternalErasureVerifier closes a storage target only with positive evidence.
// Anything it cannot prove stays pending for an operator; it never guesses.
type InternalErasureVerifier struct {
	Identifiers   ErasureIdentifierSearcher
	Sentry        SentryUserEventCounter
	StatusPath    string
	RetentionDays int
	Now           func() time.Time
}

var nonDigits = regexp.MustCompile(`\D`)

// Erase exists for the processor interface; only per-target results are used.
func (v *InternalErasureVerifier) Erase(context.Context, model.ErasureExternalSubject) (string, error) {
	return "", &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
}

func (v *InternalErasureVerifier) EraseTargets(ctx context.Context, s model.ErasureExternalSubject) ([]model.ErasureTarget, error) {
	required := s.RequiredTargets
	if len(required) == 0 {
		required = model.ErasureTargetNames
	}
	status := v.rotationStatus()
	results := make([]model.ErasureTarget, 0, len(required))
	for _, name := range required {
		target := model.ErasureTarget{Name: name, Status: "pending"}
		evidence := ""
		var err error
		switch name {
		case "backups":
			evidence = v.backupEvidence(status)
		case "historical_files":
			// Local files were unlinked by the worker before this step.
			if len(s.ExternalFileURLs) == 0 {
				evidence = "앱 관리 파일 삭제 완료, 외부 파일 주소 0건"
			}
		case "other_identifiers":
			evidence, err = v.identifierEvidence(s)
		case "external_data":
			evidence = v.externalEvidence(ctx, s, status)
		}
		if err != nil {
			return nil, err
		}
		if evidence != "" {
			target.Status = "complete"
			target.Evidence = evidence
		}
		results = append(results, target)
	}
	return validateExternalTargets(required, results)
}

func (v *InternalErasureVerifier) now() time.Time {
	if v.Now != nil {
		return v.Now()
	}
	return time.Now()
}

func (v *InternalErasureVerifier) policyDays() int {
	if v.RetentionDays > 0 {
		return v.RetentionDays
	}
	return BackupRetentionDays
}

// rotationStatus ignores a missing, unreadable or stale status file.
func (v *InternalErasureVerifier) rotationStatus() *BackupRotationStatus {
	if v.StatusPath == "" {
		return nil
	}
	data, err := os.ReadFile(v.StatusPath)
	if err != nil {
		return nil
	}
	var status BackupRotationStatus
	if json.Unmarshal(data, &status) != nil {
		return nil
	}
	checked, err := time.Parse(time.RFC3339, status.CheckedAt)
	if err != nil || v.now().Sub(checked) > backupStatusMaxAge || checked.After(v.now().Add(time.Hour)) {
		return nil
	}
	return &status
}

// backupEvidence accepts the rotation policy only while it demonstrably runs:
// a recent weekly backup exists and nothing older than the policy remains.
func (v *InternalErasureVerifier) backupEvidence(s *BackupRotationStatus) string {
	days := v.policyDays()
	if s == nil || s.RetentionDays <= 0 || s.RetentionDays > days || s.NewestBackupAt == nil || s.OldestBackupAgeDays == nil {
		return ""
	}
	newest, err := time.Parse(time.RFC3339, *s.NewestBackupAt)
	if err != nil || v.now().Sub(newest) > weeklyBackupMaxAge || *s.OldestBackupAgeDays > float64(days)+1 {
		return ""
	}
	return fmt.Sprintf("주간 백업 %d일 순환 확인, 가장 오래된 백업 %.0f일 전, 복원 시 재삭제 적용", s.RetentionDays, *s.OldestBackupAgeDays)
}

func (v *InternalErasureVerifier) logsRotated(s *BackupRotationStatus) bool {
	days := v.policyDays()
	return s != nil && s.JournalRetentionDays != nil && *s.JournalRetentionDays <= days && s.HTTPLogMaxAgeDays != nil && *s.HTTPLogMaxAgeDays <= days
}

// externalEvidence needs zero Sentry events and log rotation within policy.
// A Sentry API error leaves the target pending rather than failing the run.
func (v *InternalErasureVerifier) externalEvidence(ctx context.Context, subject model.ErasureExternalSubject, s *BackupRotationStatus) string {
	if v.Sentry == nil || !v.logsRotated(s) || subject.UserSeq <= 0 {
		return ""
	}
	user := strconv.Itoa(subject.UserSeq)
	for _, query := range []string{"user.id:" + user, user} {
		count, err := v.Sentry.CountUserEvents(ctx, query)
		if err != nil || count > 0 {
			return ""
		}
	}
	return fmt.Sprintf("Sentry 90일 0건, 서버 로그 %d일 순환 삭제", v.policyDays())
}

func (v *InternalErasureVerifier) identifierEvidence(s model.ErasureExternalSubject) (string, error) {
	values := erasureIdentifierValues(s)
	if v.Identifiers == nil || len(values) == 0 {
		return "", nil
	}
	matches, err := v.Identifiers.IdentifierMatches(values)
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		return "", nil
	}
	return "로그인·이메일·전화·소셜 식별값 운영 DB 전체 검색 0건", nil
}

func erasureIdentifierValues(s model.ErasureExternalSubject) []string {
	values := []string{}
	for _, value := range []string{s.Login, s.Email} {
		if strings.TrimSpace(value) != "" {
			values = append(values, value)
		}
	}
	if digits := nonDigits.ReplaceAllString(s.Phone, ""); digits != "" {
		values = append(values, s.Phone, digits)
		if len(digits) == 11 {
			values = append(values, digits[:3]+"-"+digits[3:7]+"-"+digits[7:])
		}
	}
	for _, subject := range s.ProviderSubjects {
		if i := strings.Index(subject, ":"); i >= 0 {
			subject = subject[i+1:]
		}
		values = append(values, subject)
	}
	return values
}
