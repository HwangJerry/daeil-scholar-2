// hall_of_fame_classifier.go — Turns spreadsheet name/cohort/department/tier cells into wall fields.
package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"golang.org/x/text/unicode/norm"
)

const (
	hallOfFamePrincipalDepartment = "교장"
	hallOfFameMaxCohort           = 32767
	hallOfFameOrgParenInner       = "주"
)

// hallOfFameDepartmentAlias is the single source of department abbreviations.
// 노 and 프 stay out on purpose: they have no confirmed mapping (see SPEC).
var hallOfFameDepartmentAlias = map[string]string{
	"독": "독일어과",
	"중": "중국어과",
	"불": "불어과",
	"일": "일본어과",
	"서": "서반아어과",
	"영": "영어과",
	"러": "러시아어과",
	"스": "스페인어과",
}

// hallOfFameTierWords maps the Korean tier cells used by the source spreadsheet.
var hallOfFameTierWords = map[string]int{
	"백만원이상":  100,
	"삼백만원이상": 300,
	"오백만원이상": 500,
	"천만원이상":  1000,
}

var (
	hallOfFameParenPattern  = regexp.MustCompile(`\(([^()]*)\)`)
	hallOfFameParenReplacer = strings.NewReplacer("（", "(", "）", ")")
)

type hallOfFameClassification struct {
	DisplayName string
	Kind        model.HallOfFameKind
	ContactName *string
	IsPrincipal bool
	UnknownMark bool
}

func normalizeHallOfFameText(value string) string {
	return strings.Join(strings.Fields(norm.NFC.String(value)), " ")
}

// classifyHallOfFameName applies the SPEC rules, most specific kind first.
func classifyHallOfFameName(raw string) hallOfFameClassification {
	name := hallOfFameParenReplacer.Replace(normalizeHallOfFameText(raw))
	if strings.Contains(name, "주식회사") || strings.Contains(name, "㈜") || strings.Contains(name, "(주)") {
		return classifyHallOfFameOrg(name)
	}
	marks := hallOfFameParenPattern.FindAllStringSubmatch(name, -1)
	bare := normalizeHallOfFameText(hallOfFameParenPattern.ReplaceAllString(name, " "))
	markHas := func(keyword string) bool {
		for _, mark := range marks {
			if strings.Contains(mark[1], keyword) {
				return true
			}
		}
		return false
	}
	switch {
	case markHas("교장"):
		return hallOfFameClassification{DisplayName: bare, Kind: model.HallOfFameKindTeacher, IsPrincipal: true}
	case markHas("교사"):
		return hallOfFameClassification{DisplayName: bare, Kind: model.HallOfFameKindTeacher}
	case strings.Contains(name, "지인"), strings.Contains(name, "비동문"):
		stripped := strings.NewReplacer("비동문", " ", "지인", " ").Replace(bare)
		return hallOfFameClassification{DisplayName: normalizeHallOfFameText(stripped), Kind: model.HallOfFameKindGuest}
	case markHas("동행"):
		return hallOfFameClassification{DisplayName: bare, Kind: model.HallOfFameKindCompanion}
	}
	return hallOfFameClassification{DisplayName: name, Kind: model.HallOfFameKindAlumni, UnknownMark: len(marks) > 0}
}

// classifyHallOfFameOrg keeps the legal name public and moves the parenthesized
// person to contact_name. "(주)" itself is part of the legal name.
func classifyHallOfFameOrg(name string) hallOfFameClassification {
	contacts := make([]string, 0, 1)
	display := hallOfFameParenPattern.ReplaceAllStringFunc(name, func(group string) string {
		inner := strings.TrimSpace(group[1 : len(group)-1])
		if inner == hallOfFameOrgParenInner {
			return group
		}
		if inner != "" {
			contacts = append(contacts, inner)
		}
		return " "
	})
	result := hallOfFameClassification{DisplayName: normalizeHallOfFameText(display), Kind: model.HallOfFameKindOrg}
	if len(contacts) > 0 {
		contact := strings.Join(contacts, ", ")
		result.ContactName = &contact
	}
	return result
}

// parseHallOfFameCohort maps 0/blank to NULL and accepts "15" or "15기".
func parseHallOfFameCohort(raw string) (*int, error) {
	value := strings.TrimSuffix(strings.TrimSpace(raw), "기")
	if value == "" || value == "0" {
		return nil, nil
	}
	cohort, err := strconv.Atoi(value)
	if err != nil || cohort < 0 || cohort > hallOfFameMaxCohort {
		return nil, fmt.Errorf("기수 값 %q을(를) 해석할 수 없습니다", raw)
	}
	return &cohort, nil
}

// expandHallOfFameDepartment returns the stored value and whether it is unconfirmed.
func expandHallOfFameDepartment(raw string) (*string, bool) {
	value := normalizeHallOfFameText(raw)
	if value == "" || value == "0" {
		return nil, false
	}
	if expanded, ok := hallOfFameDepartmentAlias[value]; ok {
		return &expanded, false
	}
	for _, known := range hallOfFameDepartmentAlias {
		if known == value {
			return &value, false
		}
	}
	return &value, true
}

// parseHallOfFameTier reads column F. "-"/blank means not on the wall (nil, nil).
func parseHallOfFameTier(raw string) (*int, error) {
	value := strings.NewReplacer(" ", "", ",", "").Replace(strings.TrimSpace(raw))
	if value == "" || value == "-" {
		return nil, nil
	}
	if tier, ok := hallOfFameTierWords[value]; ok {
		return &tier, nil
	}
	number := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(value, "이상"), "원"), "만")
	if tier, err := strconv.Atoi(number); err == nil && model.IsHallOfFameTier(tier) {
		return &tier, nil
	}
	return nil, fmt.Errorf("등급 값 %q은(는) 허용되지 않습니다 (100/300/500/1000만원)", raw)
}

// hallOfFameLabel builds the public subtitle so clients never see the internal kind.
func hallOfFameLabel(kind model.HallOfFameKind, cohort *int, department *string) string {
	switch kind {
	case model.HallOfFameKindTeacher:
		if department != nil && *department == hallOfFamePrincipalDepartment {
			return "교장"
		}
		return "교사"
	case model.HallOfFameKindGuest, model.HallOfFameKindOrg:
		return ""
	}
	parts := make([]string, 0, 2)
	if cohort != nil {
		parts = append(parts, fmt.Sprintf("%d기", *cohort))
	}
	if department != nil && *department != "" {
		parts = append(parts, *department)
	}
	return strings.Join(parts, " · ")
}
