// hall_of_fame_import_service.go — Idempotent spreadsheet import: preview diff, signed token, commit.
package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

const (
	hallOfFamePreviewTokenTTL = time.Hour
	hallOfFameBatchIDBytes    = 16
)

// ErrInvalidHallOfFameFile means the upload is not a readable template spreadsheet.
var ErrInvalidHallOfFameFile = errors.New("invalid hall of fame import file")

// ErrHallOfFamePhoneSecretMissing means import is disabled: hashing phones with a
// missing or borrowed key would give every row an identity that cannot be kept stable.
var ErrHallOfFamePhoneSecretMissing = errors.New("hall of fame phone hash secret is not configured")

// HallOfFameImportRejection is a commit refused for an operator-fixable reason.
type HallOfFameImportRejection struct{ Message string }

func (e *HallOfFameImportRejection) Error() string { return e.Message }

type hallOfFameImportRepository interface {
	ListAllForImport() ([]model.HallOfFameEntry, error)
	ListAllForImportTx(tx *sqlx.Tx) ([]model.HallOfFameEntry, error)
	FileAlreadyCommitted(fileSHA256 string) (bool, error)
	InsertTx(tx *sqlx.Tx, entry model.HallOfFameEntry) error
	SaveTx(tx *sqlx.Tx, entry model.HallOfFameEntry) error
	SoftDeleteImportedTx(tx *sqlx.Tx, ids []int64) error
	InsertBatchTx(tx *sqlx.Tx, batch model.HallOfFameImportBatch) error
	RunInTransaction(operation func(*sqlx.Tx) error) error
}

type HallOfFameImportService struct {
	repo       hallOfFameImportRepository
	configured bool
	identity   hallOfFameIdentity
	signingKey []byte
	now        func() time.Time
}

func NewHallOfFameImportService(repo hallOfFameImportRepository, signingKey, phoneHashSecret string) *HallOfFameImportService {
	return &HallOfFameImportService{
		repo:       repo,
		configured: phoneHashSecret != "",
		identity:   newHallOfFameIdentity(phoneHashSecret),
		signingKey: []byte("hall-of-fame-import-preview:v1:" + signingKey),
		now:        time.Now,
	}
}

// Preview parses the file and diffs it against the table without writing anything.
func (s *HallOfFameImportService) Preview(file io.Reader, adminSeq int) (*model.HallOfFameImportPreview, error) {
	if !s.configured {
		return nil, ErrHallOfFamePhoneSecretMissing
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHallOfFameFile, err)
	}
	parsed, err := parseHallOfFameWorkbook(bytes.NewReader(content), s.identity)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHallOfFameFile, err)
	}
	existing, err := s.repo.ListAllForImport()
	if err != nil {
		return nil, fmt.Errorf("명예의 전당 조회: %w", err)
	}
	fileHash := sha256.Sum256(content)
	fileSHA256 := hex.EncodeToString(fileHash[:])
	committed, err := s.repo.FileAlreadyCommitted(fileSHA256)
	if err != nil {
		return nil, fmt.Errorf("가져오기 이력 조회: %w", err)
	}

	plan := diffHallOfFame(parsed.Rows, existing)
	preview := &model.HallOfFameImportPreview{
		Added:             make([]model.HallOfFameImportRow, 0, len(plan.Inserts)+len(plan.Restores)),
		Updated:           make([]model.HallOfFameUpdatedRow, 0, len(plan.Updates)),
		UnchangedCount:    plan.Unchanged,
		ManualSkipped:     plan.ManualSkipped,
		RemovalCandidates: hallOfFameRemovalSummary(plan.Removals),
		Warnings:          append(append(make([]model.HallOfFameImportIssue, 0), parsed.Warnings...), plan.Warnings...),
		Errors:            append(make([]model.HallOfFameImportIssue, 0), parsed.Errors...),
		AlreadyCommitted:  committed,
	}
	preview.Added = append(preview.Added, plan.Inserts...)
	for _, restore := range plan.Restores {
		preview.Added = append(preview.Added, restore.Row)
	}
	for _, update := range plan.Updates {
		preview.Updated = append(preview.Updated, model.HallOfFameUpdatedRow{
			ID: update.Entry.ID, RowIndex: update.Row.RowIndex, DisplayName: update.Row.DisplayName, Changes: update.Changes,
		})
	}
	if len(preview.Errors) == 0 {
		preview.PreviewToken, err = s.signToken(hallOfFamePreviewToken{
			ExpiresAt: s.now().Add(hallOfFamePreviewTokenTTL).Unix(), AdminSeq: adminSeq, FileSHA256: fileSHA256,
			Rows: hallOfFameTokenRows(parsed.Rows),
		})
		if err != nil {
			return nil, fmt.Errorf("미리보기 토큰 생성: %w", err)
		}
	}
	return preview, nil
}

// Commit recomputes the diff under row locks and applies it in one transaction.
// A commit that would change nothing writes nothing, including no batch row.
func (s *HallOfFameImportService) Commit(previewToken string, removeIDs []int64, actor HallOfFameActor) (*model.HallOfFameImportCommitResult, error) {
	if !s.configured {
		return nil, ErrHallOfFamePhoneSecretMissing
	}
	token, err := s.verifyToken(previewToken, actor.UsrSeq)
	if err != nil {
		return nil, err
	}
	rows := token.importRows()
	batchID, err := newHallOfFameBatchID()
	if err != nil {
		return nil, err
	}
	result := &model.HallOfFameImportCommitResult{}
	err = s.repo.RunInTransaction(func(tx *sqlx.Tx) error {
		existing, listErr := s.repo.ListAllForImportTx(tx)
		if listErr != nil {
			return listErr
		}
		plan := diffHallOfFame(rows, existing)
		removals, removalErr := selectHallOfFameRemovals(plan.Removals, removeIDs)
		if removalErr != nil {
			return removalErr
		}
		*result = model.HallOfFameImportCommitResult{
			AddedCount: len(plan.Inserts) + len(plan.Restores), UpdatedCount: len(plan.Updates), RemovedCount: len(removals),
			UnchangedCount: plan.Unchanged, ManualSkipped: plan.ManualSkipped,
		}
		if plan.changeCount(len(removals)) == 0 {
			return nil
		}
		result.BatchID = batchID
		return s.applyPlan(tx, plan, removals, batchID, token.FileSHA256, actor.UsrSeq, *result)
	})
	if err != nil {
		return nil, err
	}
	logHallOfFameAudit(actor, "import_commit").Str("batchId", result.BatchID).
		Int("added", result.AddedCount).Int("updated", result.UpdatedCount).Int("removed", result.RemovedCount).
		Msg("admin hall of fame change")
	return result, nil
}

func (s *HallOfFameImportService) applyPlan(tx *sqlx.Tx, plan *hallOfFamePlan, removals []int64, batchID, fileSHA256 string, actorSeq int, summary model.HallOfFameImportCommitResult) error {
	batch := model.HallOfFameImportBatch{ID: batchID, FileSHA256: fileSHA256, UploadedBy: actorSeq, SummaryJSON: hallOfFameSummaryJSON(summary)}
	if err := s.repo.InsertBatchTx(tx, batch); err != nil {
		return err
	}
	for _, row := range plan.Inserts {
		public := !plan.HiddenInserts[row.NaturalKey]
		entry := applyHallOfFameRow(model.HallOfFameEntry{NaturalKey: row.NaturalKey, Source: model.HallOfFameSourceExcel, ConsentPublic: public, Visible: public}, row, batchID)
		if err := s.repo.InsertTx(tx, entry); err != nil {
			return err
		}
	}
	for _, update := range append(append([]hallOfFameRowUpdate{}, plan.Restores...), plan.Updates...) {
		if err := s.repo.SaveTx(tx, applyHallOfFameRow(update.Entry, update.Row, batchID)); err != nil {
			return err
		}
	}
	return s.repo.SoftDeleteImportedTx(tx, removals)
}

// selectHallOfFameRemovals only honours ids that are current removal candidates,
// which keeps manual rows out of reach of a crafted request.
func selectHallOfFameRemovals(candidates []model.HallOfFameEntry, removeIDs []int64) ([]int64, error) {
	allowed := make(map[int64]bool, len(candidates))
	for _, candidate := range candidates {
		allowed[candidate.ID] = true
	}
	selected := make([]int64, 0, len(removeIDs))
	seen := make(map[int64]bool, len(removeIDs))
	for _, id := range removeIDs {
		if !allowed[id] {
			return nil, &HallOfFameImportRejection{Message: "삭제 후보가 아닌 행이 포함되어 있습니다. 다시 미리보기해 주세요."}
		}
		if !seen[id] {
			seen[id] = true
			selected = append(selected, id)
		}
	}
	return selected, nil
}

func newHallOfFameBatchID() (string, error) {
	buffer := make([]byte, hallOfFameBatchIDBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// hallOfFamePreviewToken carries the parsed rows because the commit request has
// only the token and removeIds. It holds hashes, never a raw phone number.
type hallOfFamePreviewToken struct {
	ExpiresAt  int64                `json:"exp"`
	AdminSeq   int                  `json:"adminSeq"`
	FileSHA256 string               `json:"fileSha256"`
	Rows       []hallOfFameTokenRow `json:"rows"`
}

type hallOfFameTokenRow struct {
	model.HallOfFameImportRow
	Key   string  `json:"key"`
	Phone *string `json:"phoneHash"`
}

func hallOfFameTokenRows(rows []model.HallOfFameImportRow) []hallOfFameTokenRow {
	tokenRows := make([]hallOfFameTokenRow, 0, len(rows))
	for _, row := range rows {
		tokenRows = append(tokenRows, hallOfFameTokenRow{HallOfFameImportRow: row, Key: row.NaturalKey, Phone: row.PhoneHash})
	}
	return tokenRows
}

func (t hallOfFamePreviewToken) importRows() []model.HallOfFameImportRow {
	rows := make([]model.HallOfFameImportRow, 0, len(t.Rows))
	for _, tokenRow := range t.Rows {
		row := tokenRow.HallOfFameImportRow
		row.NaturalKey, row.PhoneHash = tokenRow.Key, tokenRow.Phone
		rows = append(rows, row)
	}
	return rows
}

func (s *HallOfFameImportService) tokenSignature(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, s.signingKey)
	_, _ = mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}

func (s *HallOfFameImportService) signToken(token hallOfFamePreviewToken) (string, error) {
	payload, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(s.tokenSignature(encoded)), nil
}

func (s *HallOfFameImportService) verifyToken(value string, adminSeq int) (*hallOfFamePreviewToken, error) {
	invalid := &HallOfFameImportRejection{Message: "미리보기 토큰이 올바르지 않습니다. 다시 미리보기해 주세요."}
	parts := strings.Split(value, ".")
	if len(parts) != 2 || parts[0] == "" {
		return nil, invalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, s.tokenSignature(parts[0])) {
		return nil, invalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, invalid
	}
	var token hallOfFamePreviewToken
	if err := json.Unmarshal(payload, &token); err != nil || token.AdminSeq != adminSeq {
		return nil, invalid
	}
	if token.ExpiresAt <= s.now().Unix() {
		return nil, &HallOfFameImportRejection{Message: "미리보기 토큰이 만료되었습니다. 다시 미리보기해 주세요."}
	}
	return &token, nil
}
