// hall_of_fame_diff.go — Pure diff between spreadsheet rows and stored entries.
package service

import (
	"fmt"

	"github.com/dflh-saf/backend/internal/model"
)

// hallOfFamePlan is what a commit would change. It is rebuilt from current rows
// inside the commit transaction, so a stale preview can never overwrite newer data.
type hallOfFamePlan struct {
	Inserts       []model.HallOfFameImportRow
	HiddenInserts map[string]bool
	Restores      []hallOfFameRowUpdate
	Updates       []hallOfFameRowUpdate
	Unchanged     int
	ManualSkipped int
	Removals      []model.HallOfFameEntry
	Warnings      []model.HallOfFameImportIssue
}

type hallOfFameRowUpdate struct {
	Entry   model.HallOfFameEntry
	Row     model.HallOfFameImportRow
	Changes []model.HallOfFameFieldChange
}

func (p *hallOfFamePlan) changeCount(removals int) int {
	return len(p.Inserts) + len(p.Restores) + len(p.Updates) + removals
}

func diffHallOfFame(rows []model.HallOfFameImportRow, existing []model.HallOfFameEntry) *hallOfFamePlan {
	plan := &hallOfFamePlan{HiddenInserts: make(map[string]bool)}
	byKey := make(map[string]model.HallOfFameEntry, len(existing))
	liveManualByName := make(map[string]bool)
	restrictedByName := make(map[string]bool)
	for _, entry := range existing {
		byKey[entry.NaturalKey] = entry
		nameKey := normalizeHallOfFameName(entry.DisplayName) + "|" + hallOfFameCohortKey(entry.Cohort)
		if entry.Source == model.HallOfFameSourceManual && entry.DeletedAt == nil {
			liveManualByName[nameKey] = true
		}
		if isHallOfFameRestricted(entry) {
			restrictedByName[nameKey] = true
		}
	}
	inFile := make(map[string]bool, len(rows))
	for _, row := range rows {
		inFile[row.NaturalKey] = true
		plan.classifyRow(row, byKey, liveManualByName, restrictedByName)
	}
	for _, entry := range existing {
		if entry.Source == model.HallOfFameSourceExcel && entry.DeletedAt == nil && !inFile[entry.NaturalKey] {
			plan.Removals = append(plan.Removals, entry)
		}
	}
	return plan
}

// isHallOfFameRestricted marks a person an operator took down or who has no public
// consent. A changed phone gives such a person a new natural_key, so a key lookup
// alone would publish them again under a fresh row.
func isHallOfFameRestricted(entry model.HallOfFameEntry) bool {
	takenDown := entry.DeletedAt != nil && entry.Source == model.HallOfFameSourceManual
	return takenDown || !entry.ConsentPublic || !entry.Visible
}

func (p *hallOfFamePlan) classifyRow(row model.HallOfFameImportRow, byKey map[string]model.HallOfFameEntry, liveManualByName, restrictedByName map[string]bool) {
	entry, exists := byKey[row.NaturalKey]
	switch {
	case !exists:
		p.Inserts = append(p.Inserts, row)
		nameKey := normalizeHallOfFameName(row.DisplayName) + "|" + hallOfFameCohortKey(row.Cohort)
		if restrictedByName[nameKey] {
			p.HiddenInserts[row.NaturalKey] = true
			p.Warnings = append(p.Warnings, hallOfFameIssue(row.RowIndex, "displayName", "RESTRICTED_MATCH",
				"같은 이름·기수의 비공개 또는 삭제된 행이 있어 비공개로 추가합니다. 동명이인이면 공개로 바꿔 주세요."))
		}
		if liveManualByName[nameKey] {
			p.Warnings = append(p.Warnings, hallOfFameIssue(row.RowIndex, "displayName", "MANUAL_OVERLAP",
				"같은 이름·기수의 수동 등록 행이 있습니다. 중복이면 하나를 삭제해 주세요."))
		}
	case entry.Source == model.HallOfFameSourceManual:
		p.ManualSkipped++
		if entry.DeletedAt != nil {
			p.Warnings = append(p.Warnings, hallOfFameIssue(row.RowIndex, "displayName", "DELETED_BY_ADMIN",
				"관리자가 삭제한 행이라 다시 가져오지 않습니다."))
		}
	case entry.DeletedAt != nil:
		p.Restores = append(p.Restores, hallOfFameRowUpdate{Entry: entry, Row: row})
	default:
		changes := hallOfFameRowChanges(entry, row)
		if len(changes) == 0 {
			p.Unchanged++
			return
		}
		p.Updates = append(p.Updates, hallOfFameRowUpdate{Entry: entry, Row: row, Changes: changes})
	}
}

// hallOfFameRowChanges compares only imported fields; visible and consent_public
// belong to operators and an import never reads or writes them.
func hallOfFameRowChanges(entry model.HallOfFameEntry, row model.HallOfFameImportRow) []model.HallOfFameFieldChange {
	changes := make([]model.HallOfFameFieldChange, 0)
	add := func(field string, before, after interface{}) {
		changes = append(changes, model.HallOfFameFieldChange{Field: field, Before: before, After: after})
	}
	if entry.DisplayName != row.DisplayName {
		add("displayName", entry.DisplayName, row.DisplayName)
	}
	if entry.Kind != row.Kind {
		add("kind", entry.Kind, row.Kind)
	}
	if !equalHallOfFameValue(entry.Cohort, row.Cohort) {
		add("cohort", entry.Cohort, row.Cohort)
	}
	if !equalHallOfFameValue(entry.Department, row.Department) {
		add("department", entry.Department, row.Department)
	}
	if !equalHallOfFameValue(entry.TierManWon, row.TierManWon) {
		add("tierManWon", entry.TierManWon, row.TierManWon)
	}
	if !equalHallOfFameValue(entry.ContactName, row.ContactName) {
		add("contactName", entry.ContactName, row.ContactName)
	}
	if !equalHallOfFameValue(entry.RoleLabel, row.RoleLabel) {
		add("roleLabel", entry.RoleLabel, row.RoleLabel)
	}
	return changes
}

func equalHallOfFameValue[T comparable](left, right *T) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// applyHallOfFameRow copies imported fields onto a stored entry.
func applyHallOfFameRow(entry model.HallOfFameEntry, row model.HallOfFameImportRow, batchID string) model.HallOfFameEntry {
	entry.DisplayName, entry.Kind, entry.Cohort = row.DisplayName, row.Kind, row.Cohort
	entry.Department, entry.TierManWon, entry.ContactName = row.Department, row.TierManWon, row.ContactName
	entry.RoleLabel, entry.PhoneHash = row.RoleLabel, row.PhoneHash
	entry.SourceBatchID = &batchID
	return entry
}

func hallOfFameRemovalSummary(entries []model.HallOfFameEntry) []model.HallOfFameRemovalCandidate {
	candidates := make([]model.HallOfFameRemovalCandidate, 0, len(entries))
	for _, entry := range entries {
		candidates = append(candidates, model.HallOfFameRemovalCandidate{
			ID: entry.ID, DisplayName: entry.DisplayName, Kind: entry.Kind,
			Cohort: entry.Cohort, Department: entry.Department, TierManWon: entry.TierManWon,
		})
	}
	return candidates
}

func hallOfFameSummaryJSON(result model.HallOfFameImportCommitResult) string {
	return fmt.Sprintf(`{"added":%d,"updated":%d,"removed":%d,"unchanged":%d,"manualSkipped":%d}`,
		result.AddedCount, result.UpdatedCount, result.RemovedCount, result.UnchangedCount, result.ManualSkipped)
}
