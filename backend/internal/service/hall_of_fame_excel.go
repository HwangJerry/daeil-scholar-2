// hall_of_fame_excel.go — Parses the donor spreadsheet into normalized rows, warnings and errors.
package service

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/xuri/excelize/v2"
)

const (
	hallOfFameHeaderRowIndex   = 0
	hallOfFameNameColumn       = 0
	hallOfFamePhoneColumn      = 1
	hallOfFameCohortColumn     = 2
	hallOfFameDepartmentColumn = 3
	hallOfFameFinalTierColumn  = 5
	hallOfFameMaxImportRows    = 1000
	hallOfFameMaxNameRunes     = 100
	hallOfFameMaxDeptRunes     = 30
)

// Nine digits, optionally split by separators, is longer than any name or department.
var hallOfFamePhoneRunPattern = regexp.MustCompile(`[0-9](?:[ .\-]?[0-9]){8,}`)

type hallOfFameHeaderRule struct {
	columnIndex int
	columnName  string
	keywords    []string
}

// Column order is fixed; labels match by keyword because E/F carry changing counts.
var hallOfFameHeaderRules = []hallOfFameHeaderRule{
	{0, "A(성함)", []string{"성함", "이름", "성명"}},
	{1, "B(연락처)", []string{"연락처", "전화"}},
	{2, "C(기수)", []string{"기수"}},
	{3, "D(과)", []string{"과"}},
	{4, "E(신규)", []string{"신규"}},
	{5, "F(이동)", []string{"이동"}},
}

type hallOfFameParseResult struct {
	Rows     []model.HallOfFameImportRow
	Warnings []model.HallOfFameImportIssue
	Errors   []model.HallOfFameImportIssue
}

func parseHallOfFameWorkbook(reader io.Reader, identity hallOfFameIdentity) (*hallOfFameParseResult, error) {
	workbook, err := excelize.OpenReader(reader, excelize.Options{UnzipSizeLimit: donationImportUnzipSizeLimit})
	if err != nil {
		return nil, fmt.Errorf("엑셀 파일을 열 수 없습니다: %w", err)
	}
	defer workbook.Close()
	sheets := workbook.GetSheetList()
	if len(sheets) != 1 {
		return nil, errors.New("엑셀 파일에는 업로드 시트가 하나만 있어야 합니다")
	}
	worksheetRows, err := workbook.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("엑셀 행을 읽을 수 없습니다: %w", err)
	}
	if err := validateHallOfFameHeaders(worksheetRows); err != nil {
		return nil, err
	}
	return parseHallOfFameRows(worksheetRows[hallOfFameHeaderRowIndex+1:], identity), nil
}

func validateHallOfFameHeaders(worksheetRows [][]string) error {
	if len(worksheetRows) == 0 {
		return errors.New("1행 헤더가 없습니다")
	}
	header := worksheetRows[hallOfFameHeaderRowIndex]
	for _, rule := range hallOfFameHeaderRules {
		cell := cellAt(header, rule.columnIndex)
		matched := false
		for _, keyword := range rule.keywords {
			matched = matched || strings.Contains(cell, keyword)
		}
		if !matched {
			return fmt.Errorf("1행 %s 열의 헤더가 올바르지 않습니다", rule.columnName)
		}
	}
	return nil
}

func parseHallOfFameRows(dataRows [][]string, identity hallOfFameIdentity) *hallOfFameParseResult {
	result := &hallOfFameParseResult{Rows: make([]model.HallOfFameImportRow, 0, len(dataRows))}
	byKey := make(map[string]int, len(dataRows))
	dataRowCount := 0
	for index, worksheetRow := range dataRows {
		if isBlankExcelRow(worksheetRow) {
			continue
		}
		rowIndex := index + hallOfFameHeaderRowIndex + 2
		dataRowCount++
		if dataRowCount > hallOfFameMaxImportRows {
			result.Errors = append(result.Errors, hallOfFameIssue(rowIndex, "rows", "ROW_LIMIT_EXCEEDED", "한 번에 최대 1,000행까지 가져올 수 있습니다."))
			break
		}
		row, ok := parseHallOfFameRow(rowIndex, worksheetRow, identity, result)
		if !ok {
			continue
		}
		if first, duplicate := byKey[row.NaturalKey]; duplicate {
			mergeHallOfFameRow(&result.Rows[first], row)
			result.Warnings = append(result.Warnings, hallOfFameIssue(rowIndex, "row", "DUPLICATE_MERGED",
				fmt.Sprintf("%d행과 같은 사람으로 판단해 한 행으로 합쳤습니다.", result.Rows[first].RowIndex)))
			continue
		}
		byKey[row.NaturalKey] = len(result.Rows)
		result.Rows = append(result.Rows, row)
	}
	if dataRowCount == 0 {
		result.Errors = append(result.Errors, hallOfFameIssue(0, "rows", "NO_ROWS", "가져올 데이터 행이 없습니다."))
	}
	for index := range result.Rows {
		row := &result.Rows[index]
		row.Label = hallOfFameLabel(row.Kind, row.Cohort, row.Department, row.RoleLabel)
	}
	result.Warnings = append(result.Warnings, hallOfFameNameCollisionWarnings(result.Rows)...)
	return result
}

func parseHallOfFameRow(rowIndex int, cells []string, identity hallOfFameIdentity, result *hallOfFameParseResult) (model.HallOfFameImportRow, bool) {
	failed := false
	fail := func(field, code, message string) {
		failed = true
		result.Errors = append(result.Errors, hallOfFameIssue(rowIndex, field, code, message))
	}
	rawName := cellAt(cells, hallOfFameNameColumn)
	rawDepartment := cellAt(cells, hallOfFameDepartmentColumn)
	classified := classifyHallOfFameName(rawName)
	if hallOfFameHoldsPhone(rawName) || hallOfFameHoldsPhone(rawDepartment) {
		fail("row", "PHONE_IN_WRONG_COLUMN", "성함 또는 과 열에 전화번호로 보이는 값이 있습니다. 연락처 열로 옮겨 주세요.")
	}
	if classified.DisplayName == "" {
		fail("displayName", "REQUIRED", "성함 값이 필요합니다.")
	} else if utf8.RuneCountInString(classified.DisplayName) > hallOfFameMaxNameRunes {
		fail("displayName", "TOO_LONG", "성함은 100자 이하여야 합니다.")
	}
	cohort, err := parseHallOfFameCohort(cellAt(cells, hallOfFameCohortColumn))
	if err != nil {
		fail("cohort", "INVALID_COHORT", err.Error())
	}
	tier, err := parseHallOfFameTier(cellAt(cells, hallOfFameFinalTierColumn))
	if err != nil {
		fail("tierManWon", "INVALID_TIER", err.Error())
	}
	department, unconfirmed := expandHallOfFameDepartment(rawDepartment)
	if department != nil && utf8.RuneCountInString(*department) > hallOfFameMaxDeptRunes {
		fail("department", "TOO_LONG", "과는 30자 이하여야 합니다.")
	}
	if failed {
		return model.HallOfFameImportRow{}, false
	}
	if unconfirmed {
		result.Warnings = append(result.Warnings, hallOfFameIssue(rowIndex, "department", "UNCONFIRMED_DEPARTMENT",
			"과 표기의 확정 매핑이 없어 원문을 유지합니다. 관리자가 수정해 주세요."))
	}
	if classified.UnknownMark {
		result.Warnings = append(result.Warnings, hallOfFameIssue(rowIndex, "displayName", "UNKNOWN_NAME_MARK",
			"성함의 괄호 표기를 인식하지 못해 원문을 유지합니다. 공개 전 확인해 주세요."))
	}
	phoneHash := identity.phoneHash(cellAt(cells, hallOfFamePhoneColumn))
	return model.HallOfFameImportRow{
		RowIndex:    rowIndex,
		NaturalKey:  hallOfFameNaturalKey(classified.DisplayName, cohort, classified.Kind, phoneHash),
		PhoneHash:   phoneHash,
		DisplayName: classified.DisplayName,
		Kind:        classified.Kind,
		Cohort:      cohort,
		Department:  department,
		TierManWon:  tier,
		ContactName: classified.ContactName,
		RoleLabel:   classified.RoleLabel,
	}, true
}

// mergeHallOfFameRow keeps the first row and the higher tier of a duplicated person.
func mergeHallOfFameRow(kept *model.HallOfFameImportRow, duplicate model.HallOfFameImportRow) {
	if duplicate.TierManWon != nil && (kept.TierManWon == nil || *duplicate.TierManWon > *kept.TierManWon) {
		kept.TierManWon = duplicate.TierManWon
	}
	if kept.Department == nil {
		kept.Department = duplicate.Department
	}
	if kept.ContactName == nil {
		kept.ContactName = duplicate.ContactName
	}
	if kept.RoleLabel == nil {
		kept.RoleLabel = duplicate.RoleLabel
	}
}

// hallOfFameNameCollisionWarnings flags same name+cohort rows with different keys:
// they stay separate (possibly namesakes) but an operator should look at them.
func hallOfFameNameCollisionWarnings(rows []model.HallOfFameImportRow) []model.HallOfFameImportIssue {
	warnings := make([]model.HallOfFameImportIssue, 0)
	seen := make(map[string]int, len(rows))
	for _, row := range rows {
		key := normalizeHallOfFameName(row.DisplayName) + "|" + hallOfFameCohortKey(row.Cohort)
		if first, duplicate := seen[key]; duplicate {
			warnings = append(warnings, hallOfFameIssue(row.RowIndex, "displayName", "POSSIBLE_DUPLICATE",
				fmt.Sprintf("%d행과 이름·기수가 같습니다. 동명이인이 아니면 수정해 주세요.", first)))
			continue
		}
		seen[key] = row.RowIndex
	}
	return warnings
}

// hallOfFameIssue treats rowIndex 0 as a file-level issue (JSON null).
func hallOfFameIssue(rowIndex int, field, code, message string) model.HallOfFameImportIssue {
	issue := model.HallOfFameImportIssue{Field: field, Code: code, Message: message}
	if rowIndex > 0 {
		issue.RowIndex = &rowIndex
	}
	return issue
}

// hallOfFameHoldsPhone catches a phone number typed into a column that gets stored.
func hallOfFameHoldsPhone(value string) bool {
	return hallOfFamePhoneRunPattern.MatchString(value)
}
