package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
)

type identifierStub struct {
	matches []model.AccountDeletionFootprint
	seen    []string
}

func (s *identifierStub) IdentifierMatches(values []string) ([]model.AccountDeletionFootprint, error) {
	s.seen = values
	return s.matches, nil
}

type sentryStub struct {
	count int
	err   error
}

func (s *sentryStub) CountUserEvents(context.Context, string) (int, error) { return s.count, s.err }

var verifierNow = time.Date(2026, 9, 14, 3, 0, 0, 0, time.UTC)

func writeRotationStatus(t *testing.T, checkedAgo time.Duration, newestAgo time.Duration, oldestDays float64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "status.json")
	body := `{"checkedAt":"` + verifierNow.Add(-checkedAgo).Format(time.RFC3339) + `","retentionDays":28,"newestBackupAt":"` +
		verifierNow.Add(-newestAgo).Format(time.RFC3339) + `","oldestBackupAgeDays":` + formatFloat(oldestDays) + `,"journalRetentionDays":28,"httpLogMaxAgeDays":28}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func formatFloat(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

func subjectForVerifier() model.ErasureExternalSubject {
	return model.ErasureExternalSubject{RequestID: 4, UserSeq: 42, Login: "fake42", Email: "fake42@example.org", Phone: "010-0000-0042", ProviderSubjects: []string{"KT:synthetic-kakao"}}
}

func TestInternalVerifierClosesAllTargetsWithEvidence(t *testing.T) {
	ids := &identifierStub{}
	v := &InternalErasureVerifier{Identifiers: ids, Sentry: &sentryStub{}, StatusPath: writeRotationStatus(t, time.Hour, 48*time.Hour, 20), Now: func() time.Time { return verifierNow }}
	targets, err := v.EraseTargets(context.Background(), subjectForVerifier())
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 4 {
		t.Fatalf("targets = %+v", targets)
	}
	for _, target := range targets {
		if !target.Verified() || len(target.Evidence) > 200 {
			t.Fatalf("target not verified with short evidence: %+v", target)
		}
	}
	want := map[string]bool{"fake42": true, "fake42@example.org": true, "01000000042": true, "010-0000-0042": true, "synthetic-kakao": true}
	for _, value := range ids.seen {
		delete(want, value)
	}
	if len(want) != 0 {
		t.Fatalf("identifiers not searched: %v (searched %v)", want, ids.seen)
	}
}

func TestInternalVerifierLeavesUnprovenTargetsPending(t *testing.T) {
	cases := map[string]struct {
		verifier *InternalErasureVerifier
		subject  func(*model.ErasureExternalSubject)
		pending  []string
	}{
		"stale status":     {&InternalErasureVerifier{Identifiers: &identifierStub{}, Sentry: &sentryStub{}}, nil, []string{"backups", "external_data"}},
		"old backup kept":  {&InternalErasureVerifier{Identifiers: &identifierStub{}, Sentry: &sentryStub{}}, nil, []string{"backups"}},
		"weekly overdue":   {&InternalErasureVerifier{Identifiers: &identifierStub{}, Sentry: &sentryStub{}}, nil, []string{"backups"}},
		"identifier found": {&InternalErasureVerifier{Identifiers: &identifierStub{matches: []model.AccountDeletionFootprint{{Table: "WEO_MEMBER_SOCIAL", Column: "NMS_EMAIL", Count: 1}}}, Sentry: &sentryStub{}}, nil, []string{"other_identifiers"}},
		"sentry events":    {&InternalErasureVerifier{Identifiers: &identifierStub{}, Sentry: &sentryStub{count: 2}}, nil, []string{"external_data"}},
		"sentry error":     {&InternalErasureVerifier{Identifiers: &identifierStub{}, Sentry: &sentryStub{err: errors.New("down")}}, nil, []string{"external_data"}},
		"external file":    {&InternalErasureVerifier{Identifiers: &identifierStub{}, Sentry: &sentryStub{}}, func(s *model.ErasureExternalSubject) { s.ExternalFileURLs = []string{"https://cdn.example/a.jpg"} }, []string{"historical_files"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tc.verifier.Now = func() time.Time { return verifierNow }
			switch name {
			case "stale status":
				tc.verifier.StatusPath = writeRotationStatus(t, 48*time.Hour, 48*time.Hour, 20)
			case "old backup kept":
				tc.verifier.StatusPath = writeRotationStatus(t, time.Hour, 48*time.Hour, 40)
			case "weekly overdue":
				tc.verifier.StatusPath = writeRotationStatus(t, time.Hour, 10*24*time.Hour, 20)
			default:
				tc.verifier.StatusPath = writeRotationStatus(t, time.Hour, 48*time.Hour, 20)
			}
			subject := subjectForVerifier()
			if tc.subject != nil {
				tc.subject(&subject)
			}
			targets, err := tc.verifier.EraseTargets(context.Background(), subject)
			if err != nil {
				t.Fatal(err)
			}
			pending := map[string]bool{}
			for _, target := range targets {
				if !target.Verified() {
					pending[target.Name] = true
					if target.Evidence != "" {
						t.Fatalf("pending target carries evidence: %+v", target)
					}
				}
			}
			if len(pending) != len(tc.pending) {
				t.Fatalf("pending = %v, want %v", pending, tc.pending)
			}
			for _, want := range tc.pending {
				if !pending[want] {
					t.Fatalf("pending = %v, want %v", pending, tc.pending)
				}
			}
		})
	}
}
