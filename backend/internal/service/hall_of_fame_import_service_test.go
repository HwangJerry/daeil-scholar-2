// hall_of_fame_import_service_test.go — Idempotency and manual-row protection using synthetic spreadsheets.
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

var hallOfFameTestHeader = []string{"성함", "연락처", "기수", "과", "신규 8명", "이동 10명"}

func hallOfFameWorkbook(t *testing.T, rows ...[]interface{}) *bytes.Buffer {
	t.Helper()
	workbook := excelize.NewFile()
	defer workbook.Close()
	sheet := workbook.GetSheetName(0)
	for column, label := range hallOfFameTestHeader {
		cell, _ := excelize.CoordinatesToCellName(column+1, 1)
		_ = workbook.SetCellValue(sheet, cell, label)
	}
	for rowOffset, row := range rows {
		for column, value := range row {
			cell, _ := excelize.CoordinatesToCellName(column+1, rowOffset+2)
			_ = workbook.SetCellValue(sheet, cell, value)
		}
	}
	buffer, err := workbook.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return buffer
}

// fakeHallOfFameRepo is an in-memory table that mirrors the repository's SQL rules.
type fakeHallOfFameRepo struct {
	entries []model.HallOfFameEntry
	batches []model.HallOfFameImportBatch
	nextID  int64
	writes  int
}

func (f *fakeHallOfFameRepo) copyEntries() []model.HallOfFameEntry {
	return append([]model.HallOfFameEntry{}, f.entries...)
}
func (f *fakeHallOfFameRepo) ListAllForImport() ([]model.HallOfFameEntry, error) {
	return f.copyEntries(), nil
}
func (f *fakeHallOfFameRepo) ListAllForImportTx(*sqlx.Tx) ([]model.HallOfFameEntry, error) {
	return f.copyEntries(), nil
}
func (f *fakeHallOfFameRepo) FileAlreadyCommitted(fileSHA256 string) (bool, error) {
	for _, batch := range f.batches {
		if batch.FileSHA256 == fileSHA256 {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeHallOfFameRepo) InsertTx(_ *sqlx.Tx, entry model.HallOfFameEntry) error {
	f.writes++
	f.nextID++
	entry.ID = f.nextID
	f.entries = append(f.entries, entry)
	return nil
}
func (f *fakeHallOfFameRepo) SaveTx(_ *sqlx.Tx, entry model.HallOfFameEntry) error {
	f.writes++
	for index := range f.entries {
		if f.entries[index].ID == entry.ID {
			entry.DeletedAt = nil
			f.entries[index] = entry
		}
	}
	return nil
}
func (f *fakeHallOfFameRepo) SoftDeleteImportedTx(_ *sqlx.Tx, ids []int64) error {
	now := time.Now()
	for _, id := range ids {
		f.writes++
		for index := range f.entries {
			if f.entries[index].ID == id && f.entries[index].Source == model.HallOfFameSourceExcel {
				f.entries[index].DeletedAt = &now
			}
		}
	}
	return nil
}
func (f *fakeHallOfFameRepo) InsertBatchTx(_ *sqlx.Tx, batch model.HallOfFameImportBatch) error {
	f.writes++
	f.batches = append(f.batches, batch)
	return nil
}
func (f *fakeHallOfFameRepo) RunInTransaction(operation func(*sqlx.Tx) error) error {
	return operation(nil)
}

func (f *fakeHallOfFameRepo) find(t *testing.T, name string) *model.HallOfFameEntry {
	t.Helper()
	for index := range f.entries {
		if f.entries[index].DisplayName == name {
			return &f.entries[index]
		}
	}
	t.Fatalf("entry %q not found", name)
	return nil
}

var hallOfFameTestActor = HallOfFameActor{UsrSeq: 7, IP: "127.0.0.1"}

func hallOfFameFixtureRows() [][]interface{} {
	return [][]interface{}{
		{"김하나", "010-1111-2222", 15, "불", "-", "백만원이상"},
		{"이두리(동행)", "010-3333-4444", 0, 0, "-", "삼백만원이상"},
		{"박세나(대일외고 교사)", "", "", "", "-", "오백만원이상"},
		{"최네오(교장)", "해외거주", "", "", "-", "천만원이상"},
		{"정다섯(비동문)", "010-5555-6666", "", "", "-", "백만원이상"},
		{"주식회사 대일상사(한여섯)", "010-7777-8888", "", "", "-", "천만원이상"},
		{"오일곱", "010-9999-0000", 20, "노", "-", "-"},
	}
}

func commitHallOfFameFile(t *testing.T, service *HallOfFameImportService, file *bytes.Buffer, removeIDs ...int64) *model.HallOfFameImportCommitResult {
	t.Helper()
	preview, err := service.Preview(bytes.NewReader(file.Bytes()), hallOfFameTestActor.UsrSeq)
	if err != nil {
		t.Fatal(err)
	}
	if preview.PreviewToken == "" {
		t.Fatalf("no token, errors: %+v", preview.Errors)
	}
	result, err := service.Commit(preview.PreviewToken, removeIDs, hallOfFameTestActor)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestHallOfFameSameFileCommittedTwiceChangesNothing(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	file := hallOfFameWorkbook(t, hallOfFameFixtureRows()...)

	first := commitHallOfFameFile(t, service, file)
	if first.AddedCount != 7 || first.BatchID == "" {
		t.Fatalf("first commit: %+v", first)
	}
	writesAfterFirst := len(repo.entries)
	writeCount := repo.writes
	snapshot, _ := json.Marshal(repo.entries)

	preview, err := service.Preview(bytes.NewReader(file.Bytes()), hallOfFameTestActor.UsrSeq)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Added) != 0 || len(preview.Updated) != 0 || len(preview.RemovalCandidates) != 0 || preview.UnchangedCount != 7 || !preview.AlreadyCommitted {
		t.Fatalf("second preview: %+v", preview)
	}
	second, err := service.Commit(preview.PreviewToken, nil, hallOfFameTestActor)
	if err != nil {
		t.Fatal(err)
	}
	if second.AddedCount != 0 || second.UpdatedCount != 0 || second.RemovedCount != 0 || second.UnchangedCount != 7 || second.BatchID != "" {
		t.Fatalf("second commit: %+v", second)
	}
	after, _ := json.Marshal(repo.entries)
	if repo.writes != writeCount || len(repo.entries) != writesAfterFirst || !bytes.Equal(snapshot, after) || len(repo.batches) != 1 {
		t.Fatalf("second commit wrote: writes %d->%d batches %d", writeCount, repo.writes, len(repo.batches))
	}
}

func TestHallOfFameImportAppliesClassification(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	commitHallOfFameFile(t, NewHallOfFameImportService(repo, "signing", "phone"), hallOfFameWorkbook(t, hallOfFameFixtureRows()...))

	alumni := repo.find(t, "김하나")
	if alumni.Cohort == nil || *alumni.Cohort != 15 || *alumni.Department != "불어과" || *alumni.TierManWon != 100 || alumni.PhoneHash == nil {
		t.Fatalf("alumni: %+v", alumni)
	}
	companion := repo.find(t, "이두리")
	if companion.Kind != model.HallOfFameKindCompanion || companion.Cohort != nil || companion.Department != nil {
		t.Fatalf("companion: %+v", companion)
	}
	principal := repo.find(t, "최네오")
	if principal.PhoneHash != nil || principal.Department != nil || hallOfFameLabel(principal.Kind, principal.Cohort, principal.Department, principal.RoleLabel) != "교장" {
		t.Fatalf("principal: %+v", principal)
	}
	org := repo.find(t, "주식회사 대일상사")
	if org.Kind != model.HallOfFameKindOrg || *org.ContactName != "한여섯" {
		t.Fatalf("org: %+v", org)
	}
	if repo.find(t, "오일곱").TierManWon != nil {
		t.Fatal("'-' must import with a NULL tier")
	}
}

func TestHallOfFameNoRawPhoneLeavesTheParser(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	preview, err := service.Preview(hallOfFameWorkbook(t, hallOfFameFixtureRows()...), 7)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(preview)
	payload, _ := service.verifyToken(preview.PreviewToken, 7)
	encodedToken, _ := json.Marshal(payload)
	for _, haystack := range []string{string(encoded), string(encodedToken)} {
		for _, phone := range []string{"1111-2222", "11112222", "01011112222"} {
			if strings.Contains(haystack, phone) {
				t.Fatalf("raw phone leaked: %s", phone)
			}
		}
	}
	if strings.Contains(string(encoded), "phoneHash") || strings.Contains(string(encoded), "naturalKey") {
		t.Fatal("preview must not carry identity hashes")
	}
}

func TestHallOfFameUpdatedRowsAndOperatorFieldsSurvive(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	commitHallOfFameFile(t, service, hallOfFameWorkbook(t, hallOfFameFixtureRows()...))
	repo.find(t, "김하나").Visible = false
	repo.find(t, "김하나").ConsentPublic = false

	rows := hallOfFameFixtureRows()
	rows[0][5] = "삼백만원이상"
	preview, err := service.Preview(hallOfFameWorkbook(t, rows...), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Updated) != 1 || len(preview.Updated[0].Changes) != 1 || preview.Updated[0].Changes[0].Field != "tierManWon" {
		t.Fatalf("updated: %+v", preview.Updated)
	}
	if _, err := service.Commit(preview.PreviewToken, nil, hallOfFameTestActor); err != nil {
		t.Fatal(err)
	}
	entry := repo.find(t, "김하나")
	if *entry.TierManWon != 300 || entry.Visible || entry.ConsentPublic {
		t.Fatalf("entry: %+v", entry)
	}
}

func TestHallOfFameManualRowsAreNeverTouched(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	commitHallOfFameFile(t, service, hallOfFameWorkbook(t, hallOfFameFixtureRows()...))

	// An operator edit flips the row to manual; a manually added row was never in the file.
	edited := repo.find(t, "김하나")
	edited.Source, edited.DisplayName = model.HallOfFameSourceManual, "김하나 동문"
	tier := 1000
	repo.nextID++
	repo.entries = append(repo.entries, model.HallOfFameEntry{
		ID: repo.nextID, NaturalKey: "manual-only", DisplayName: "수동행", Kind: model.HallOfFameKindAlumni,
		TierManWon: &tier, Source: model.HallOfFameSourceManual, Visible: true, ConsentPublic: true,
	})

	rows := hallOfFameFixtureRows()
	rows[0][5] = "천만원이상"
	preview, err := service.Preview(hallOfFameWorkbook(t, rows...), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Updated) != 0 || len(preview.RemovalCandidates) != 0 || preview.ManualSkipped != 1 {
		t.Fatalf("preview: %+v", preview)
	}
	// Manual rows are not candidates, so a crafted id is refused, not deleted.
	manualID := repo.find(t, "수동행").ID
	_, err = service.Commit(preview.PreviewToken, []int64{manualID}, hallOfFameTestActor)
	var rejection *HallOfFameImportRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("err = %v", err)
	}
	if _, err := service.Commit(preview.PreviewToken, nil, hallOfFameTestActor); err != nil {
		t.Fatal(err)
	}
	entry := repo.find(t, "김하나 동문")
	if *entry.TierManWon != 100 || entry.DeletedAt != nil {
		t.Fatalf("manual row overwritten: %+v", entry)
	}
}

func TestHallOfFameOperatorDeletedRowStaysDeleted(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	file := hallOfFameWorkbook(t, hallOfFameFixtureRows()...)
	commitHallOfFameFile(t, service, file)
	now := time.Now()
	deleted := repo.find(t, "김하나")
	deleted.Source, deleted.DeletedAt = model.HallOfFameSourceManual, &now

	preview, err := service.Preview(bytes.NewReader(file.Bytes()), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Added) != 0 || preview.ManualSkipped != 1 || len(preview.Warnings) == 0 {
		t.Fatalf("preview: %+v", preview)
	}
	found := false
	for _, warning := range preview.Warnings {
		found = found || warning.Code == "DELETED_BY_ADMIN"
	}
	if !found {
		t.Fatalf("warnings: %+v", preview.Warnings)
	}
}

func TestHallOfFameRestrictedPersonWithNewPhoneImportsHidden(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		restrict func(*model.HallOfFameEntry)
	}{
		{"consent withdrawn", func(entry *model.HallOfFameEntry) { entry.ConsentPublic = false }},
		{"hidden by operator", func(entry *model.HallOfFameEntry) { entry.Visible = false }},
		{"deleted by operator", func(entry *model.HallOfFameEntry) {
			entry.Source, entry.DeletedAt = model.HallOfFameSourceManual, &now
		}},
		{"consent withdrawn then removed by import", func(entry *model.HallOfFameEntry) {
			entry.ConsentPublic, entry.DeletedAt = false, &now
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeHallOfFameRepo{}
			service := NewHallOfFameImportService(repo, "signing", "phone")
			rows := hallOfFameFixtureRows()
			commitHallOfFameFile(t, service, hallOfFameWorkbook(t, rows...))
			test.restrict(repo.find(t, "김하나"))

			rows[0][1] = "010-1212-3434"
			preview, err := service.Preview(hallOfFameWorkbook(t, rows...), 7)
			if err != nil {
				t.Fatal(err)
			}
			warned := false
			for _, warning := range preview.Warnings {
				warned = warned || warning.Code == "RESTRICTED_MATCH"
			}
			if !warned {
				t.Fatalf("warnings: %+v", preview.Warnings)
			}
			if _, err := service.Commit(preview.PreviewToken, nil, hallOfFameTestActor); err != nil {
				t.Fatal(err)
			}
			inserted := repo.entries[len(repo.entries)-1]
			if inserted.DisplayName != "김하나" || inserted.Visible || inserted.ConsentPublic {
				t.Fatalf("restricted person went public: %+v", inserted)
			}
		})
	}
}

func TestHallOfFameImportFailsClosedWithoutPhoneSecret(t *testing.T) {
	service := NewHallOfFameImportService(&fakeHallOfFameRepo{}, "signing", "")
	if _, err := service.Preview(hallOfFameWorkbook(t, hallOfFameFixtureRows()...), 7); !errors.Is(err, ErrHallOfFamePhoneSecretMissing) {
		t.Fatalf("preview: %v", err)
	}
	if _, err := service.Commit("token", nil, hallOfFameTestActor); !errors.Is(err, ErrHallOfFamePhoneSecretMissing) {
		t.Fatalf("commit: %v", err)
	}
}

func TestHallOfFameRemovalOnlyForSelectedCandidatesAndRestorable(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	full := hallOfFameWorkbook(t, hallOfFameFixtureRows()...)
	commitHallOfFameFile(t, service, full)

	shorter := hallOfFameWorkbook(t, hallOfFameFixtureRows()[1:]...)
	preview, err := service.Preview(bytes.NewReader(shorter.Bytes()), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.RemovalCandidates) != 1 || preview.RemovalCandidates[0].DisplayName != "김하나" {
		t.Fatalf("candidates: %+v", preview.RemovalCandidates)
	}
	if result, err := service.Commit(preview.PreviewToken, nil, hallOfFameTestActor); err != nil || result.RemovedCount != 0 {
		t.Fatalf("unselected: %+v %v", result, err)
	}
	if repo.find(t, "김하나").DeletedAt != nil {
		t.Fatal("unselected candidate must survive")
	}
	id := preview.RemovalCandidates[0].ID
	if result, err := service.Commit(preview.PreviewToken, []int64{id, id}, hallOfFameTestActor); err != nil || result.RemovedCount != 1 {
		t.Fatalf("selected: %+v %v", result, err)
	}
	if repo.find(t, "김하나").DeletedAt == nil {
		t.Fatal("selected candidate must be soft deleted")
	}
	// The person returning on a later sheet restores the same row.
	if result := commitHallOfFameFile(t, service, full); result.AddedCount != 1 {
		t.Fatalf("restore: %+v", result)
	}
	if entry := repo.find(t, "김하나"); entry.DeletedAt != nil || len(repo.entries) != 7 {
		t.Fatalf("restored: %+v (%d rows)", entry, len(repo.entries))
	}
}

func TestHallOfFameDuplicatePersonMergesWithHigherTier(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	preview, err := service.Preview(hallOfFameWorkbook(t,
		[]interface{}{"전하나", "010-1234-5678", 7, "독", "-", "백만원이상"},
		[]interface{}{"전 하나", "01012345678", 7, "", "-", "삼백만원이상"},
	), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Added) != 1 || *preview.Added[0].TierManWon != 300 || *preview.Added[0].Department != "독일어과" {
		t.Fatalf("added: %+v", preview.Added)
	}
	if len(preview.Warnings) != 1 || preview.Warnings[0].Code != "DUPLICATE_MERGED" {
		t.Fatalf("warnings: %+v", preview.Warnings)
	}
}

func TestHallOfFamePreviewWarningsAndBlockingErrors(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	preview, err := service.Preview(hallOfFameWorkbook(t,
		[]interface{}{"김하나", "", 3, "노", "-", "백만원이상"},
		[]interface{}{"이두리", "", 4, "프", "-", "오천만원이상"},
	), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Warnings) != 1 || preview.Warnings[0].Code != "UNCONFIRMED_DEPARTMENT" || *preview.Added[0].Department != "노" {
		t.Fatalf("warnings: %+v", preview.Warnings)
	}
	if len(preview.Errors) != 1 || preview.Errors[0].Code != "INVALID_TIER" || preview.PreviewToken != "" {
		t.Fatalf("errors: %+v token %q", preview.Errors, preview.PreviewToken)
	}
}

func TestHallOfFameIssueMessagesNeverEchoPhones(t *testing.T) {
	service := NewHallOfFameImportService(&fakeHallOfFameRepo{}, "signing", "phone")
	preview, err := service.Preview(hallOfFameWorkbook(t,
		[]interface{}{"김하나", "010-1234-5678", 3, "독", "-", "백만원이상"},
		[]interface{}{"김하나", "010-1234-5678", 3, "독", "-", "백만원이상"},
		[]interface{}{"이두리", "", "010-2222-3333", "독", "-", "백만원이상"},
		[]interface{}{"박세나", "", 4, "독", "-", "010-4444-5555"},
		[]interface{}{"최네오", "", 5, "010-6666-7777", "-", "백만원이상"},
		[]interface{}{"정다섯 010 8888 9999", "", 6, "독", "-", "백만원이상"},
	), 7)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(preview)
	for _, fragment := range []string{"1234", "2222", "4444", "6666", "8888"} {
		if strings.Contains(string(encoded), fragment) {
			t.Fatalf("phone fragment %s leaked: %s", fragment, encoded)
		}
	}
	codes := map[string]int{}
	for _, issue := range preview.Errors {
		codes[issue.Code]++
	}
	if codes["INVALID_COHORT"] != 1 || codes["INVALID_TIER"] != 1 || codes["PHONE_IN_WRONG_COLUMN"] != 2 || preview.PreviewToken != "" {
		t.Fatalf("errors: %+v", preview.Errors)
	}
}

func TestHallOfFameHoldsPhoneSeesThroughSeparators(t *testing.T) {
	for _, value := range []string{
		"01012345678", "010-1234-5678", "010 1234 5678", "(010)1234-5678", "010–1234–5678", "010/1234/5678",
		"０１０－１２３４－５６７８", "김하나(010)1234-5678", "주식회사 대일(010.1234.5678)",
		"010ㅡ1234ㅡ5678", "010ㆍ1234ㆍ5678", "010一1234一5678", "٠١٠١٢٣٤٥٦٧٨",
	} {
		if !hallOfFameHoldsPhone(value) {
			t.Fatalf("%q must be detected", value)
		}
	}
	for _, value := range []string{"김하나", "주식회사 대일상사(김하나)", "2024년 3학년", "불어과"} {
		if hallOfFameHoldsPhone(value) {
			t.Fatalf("%q is not a phone", value)
		}
	}
}

func TestHallOfFameFileLevelIssueHasNullRowIndex(t *testing.T) {
	preview, err := NewHallOfFameImportService(&fakeHallOfFameRepo{}, "s", "p").Preview(hallOfFameWorkbook(t), 7)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(preview.Errors)
	if !strings.Contains(string(encoded), `"rowIndex":null`) {
		t.Fatalf("errors: %s", encoded)
	}
}

func TestHallOfFamePreviewRejectsWrongHeaders(t *testing.T) {
	workbook := excelize.NewFile()
	defer workbook.Close()
	_ = workbook.SetSheetRow(workbook.GetSheetName(0), "A1", &[]string{"이름", "금액"})
	buffer, _ := workbook.WriteToBuffer()
	_, err := NewHallOfFameImportService(&fakeHallOfFameRepo{}, "s", "p").Preview(buffer, 7)
	if !errors.Is(err, ErrInvalidHallOfFameFile) {
		t.Fatalf("err = %v", err)
	}
}

func TestHallOfFameRowLimit(t *testing.T) {
	rows := make([][]string, hallOfFameMaxImportRows+1)
	for index := range rows {
		rows[index] = []string{"김하나" + strings.Repeat("가", index%3), "", "", "", "", "-"}
	}
	result := parseHallOfFameRows(rows, newHallOfFameIdentity("p"))
	if len(result.Errors) != 1 || result.Errors[0].Code != "ROW_LIMIT_EXCEEDED" {
		t.Fatalf("errors: %+v", result.Errors)
	}
}

func TestHallOfFameCommitRejectsBadTokens(t *testing.T) {
	repo := &fakeHallOfFameRepo{}
	service := NewHallOfFameImportService(repo, "signing", "phone")
	preview, err := service.Preview(hallOfFameWorkbook(t, hallOfFameFixtureRows()...), 7)
	if err != nil {
		t.Fatal(err)
	}
	var rejection *HallOfFameImportRejection
	if _, err := service.Commit(preview.PreviewToken+"x", nil, hallOfFameTestActor); !errors.As(err, &rejection) {
		t.Fatalf("tampered: %v", err)
	}
	if _, err := service.Commit(preview.PreviewToken, nil, HallOfFameActor{UsrSeq: 8}); !errors.As(err, &rejection) {
		t.Fatalf("other admin: %v", err)
	}
	service.now = func() time.Time { return time.Now().Add(hallOfFamePreviewTokenTTL + time.Minute) }
	if _, err := service.Commit(preview.PreviewToken, nil, hallOfFameTestActor); !errors.As(err, &rejection) || !strings.Contains(rejection.Message, "만료") {
		t.Fatalf("expired: %v", err)
	}
	if len(repo.entries) != 0 {
		t.Fatal("rejected commits must not write")
	}
}
