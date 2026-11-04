// hall_of_fame_classifier_test.go — Table tests for kind, cohort, department, tier, phone and label rules.
package service

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestClassifyHallOfFameName(t *testing.T) {
	tests := []struct {
		raw       string
		name      string
		kind      model.HallOfFameKind
		contact   string
		principal bool
		unknown   bool
	}{
		{raw: "김하나", name: "김하나", kind: model.HallOfFameKindAlumni},
		{raw: "  김  하나 ", name: "김 하나", kind: model.HallOfFameKindAlumni},
		{raw: "김하나(동행)", name: "김하나", kind: model.HallOfFameKindCompanion},
		{raw: "김하나 (동행)", name: "김하나", kind: model.HallOfFameKindCompanion},
		{raw: "김하나（동행）", name: "김하나", kind: model.HallOfFameKindCompanion},
		{raw: "김하나(대일외고 교사)", name: "김하나", kind: model.HallOfFameKindTeacher},
		{raw: "김하나(교장)", name: "김하나", kind: model.HallOfFameKindTeacher, principal: true},
		{raw: "김하나(지인)", name: "김하나", kind: model.HallOfFameKindGuest},
		{raw: "김하나(박둘 지인)", name: "김하나", kind: model.HallOfFameKindGuest},
		{raw: "김하나(비동문)", name: "김하나", kind: model.HallOfFameKindGuest},
		{raw: "김하나 지인", name: "김하나", kind: model.HallOfFameKindGuest},
		{raw: "주식회사 대일상사(김하나)", name: "주식회사 대일상사", kind: model.HallOfFameKindOrg, contact: "김하나"},
		{raw: "㈜대일상사(김하나 대표)", name: "㈜대일상사", kind: model.HallOfFameKindOrg, contact: "김하나 대표"},
		{raw: "(주)대일상사", name: "(주)대일상사", kind: model.HallOfFameKindOrg},
		{raw: "(주)대일상사(김하나)", name: "(주)대일상사", kind: model.HallOfFameKindOrg, contact: "김하나"},
		{raw: "주식회사 대일상사", name: "주식회사 대일상사", kind: model.HallOfFameKindOrg},
		{raw: "김하나(미국)", name: "김하나(미국)", kind: model.HallOfFameKindAlumni, unknown: true},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got := classifyHallOfFameName(test.raw)
			if got.DisplayName != test.name || got.Kind != test.kind || (got.RoleLabel != nil) != test.principal || got.UnknownMark != test.unknown {
				t.Fatalf("got %+v", got)
			}
			contact := ""
			if got.ContactName != nil {
				contact = *got.ContactName
			}
			if contact != test.contact {
				t.Fatalf("contact = %q, want %q", contact, test.contact)
			}
		})
	}
}

func TestClassifyHallOfFameNameComposesDecomposedHangul(t *testing.T) {
	if got := classifyHallOfFameName("김").DisplayName; got != "김" {
		t.Fatalf("got %q", got)
	}
}

func TestParseHallOfFameCohort(t *testing.T) {
	for raw, want := range map[string]int{"15": 15, " 15 ": 15, "15기": 15} {
		got, err := parseHallOfFameCohort(raw)
		if err != nil || got == nil || *got != want {
			t.Fatalf("%q: got %v, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"", "0", " 0 ", "0기"} {
		if got, err := parseHallOfFameCohort(raw); err != nil || got != nil {
			t.Fatalf("%q must be NULL: %v, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"열다섯", "-3", "99999"} {
		if _, err := parseHallOfFameCohort(raw); err == nil {
			t.Fatalf("%q must fail", raw)
		}
	}
}

func TestExpandHallOfFameDepartment(t *testing.T) {
	tests := []struct {
		raw, want   string
		unconfirmed bool
	}{
		{raw: "독", want: "독일어과"},
		{raw: "중", want: "중국어과"},
		{raw: "불", want: "불어과"},
		{raw: "일", want: "일본어과"},
		{raw: "서", want: "서반아어과"},
		{raw: "영", want: "영어과"},
		{raw: "러", want: "러시아어과"},
		{raw: "스", want: "스페인어과"},
		{raw: "불어과", want: "불어과"},
		{raw: "노", want: "노", unconfirmed: true},
		{raw: "프", want: "프", unconfirmed: true},
	}
	for _, test := range tests {
		got, unconfirmed := expandHallOfFameDepartment(test.raw)
		if got == nil || *got != test.want || unconfirmed != test.unconfirmed {
			t.Fatalf("%q: got %v, %v", test.raw, got, unconfirmed)
		}
	}
	for _, raw := range []string{"", "0", " "} {
		if got, unconfirmed := expandHallOfFameDepartment(raw); got != nil || unconfirmed {
			t.Fatalf("%q must be NULL", raw)
		}
	}
}

func TestParseHallOfFameTier(t *testing.T) {
	for raw, want := range map[string]int{"백만원이상": 100, "삼백만원이상": 300, "오백만원이상": 500, "천만원이상": 1000, "1,000": 1000, "300만원": 300} {
		got, err := parseHallOfFameTier(raw)
		if err != nil || got == nil || *got != want {
			t.Fatalf("%q: got %v, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"", "-", " - "} {
		if got, err := parseHallOfFameTier(raw); err != nil || got != nil {
			t.Fatalf("%q must be NULL: %v %v", raw, got, err)
		}
	}
	for _, raw := range []string{"오천만원이상", "5000", "0", "많이"} {
		if _, err := parseHallOfFameTier(raw); err == nil {
			t.Fatalf("%q must be rejected", raw)
		}
	}
}

func TestCanonicalHallOfFamePhone(t *testing.T) {
	tests := map[string]string{
		"010-1234-5678":    "01012345678",
		"01012345678":      "01012345678",
		"1012345678":       "01012345678",
		"+82 10-1234-5678": "01012345678",
		"해외거주":             "",
		"":                 "",
		"미국 거주 1234":       "",
		"6141234567":       "",
		"+1 415 555 0100":  "",
		"4512345678":       "",
	}
	for raw, want := range tests {
		if got := canonicalHallOfFamePhone(raw); got != want {
			t.Fatalf("%q: got %q, want %q", raw, got, want)
		}
	}
}

func TestHallOfFameIdentityHashesAreKeyedAndStable(t *testing.T) {
	first := newHallOfFameIdentity("a").phoneHash("010-1234-5678")
	again := newHallOfFameIdentity("a").phoneHash("01012345678")
	other := newHallOfFameIdentity("b").phoneHash("01012345678")
	if first == nil || *first != *again || *first == *other || len(*first) != 64 {
		t.Fatalf("hashes: %v %v %v", first, again, other)
	}
	if newHallOfFameIdentity("a").phoneHash("해외거주") != nil {
		t.Fatal("text phones get no hash")
	}
}

func TestHallOfFameNaturalKey(t *testing.T) {
	cohort, hash := 15, "abc"
	base := hallOfFameNaturalKey("김 하나", &cohort, model.HallOfFameKindAlumni, nil)
	if base != hallOfFameNaturalKey(" 김하나", &cohort, model.HallOfFameKindAlumni, nil) {
		t.Fatal("whitespace must not change the key")
	}
	if base == hallOfFameNaturalKey("김하나", &cohort, model.HallOfFameKindGuest, nil) {
		t.Fatal("phoneless keys must include kind")
	}
	if hallOfFameNaturalKey("김하나", &cohort, model.HallOfFameKindAlumni, &hash) != hallOfFameNaturalKey("김하나", &cohort, model.HallOfFameKindCompanion, &hash) {
		t.Fatal("phone keys must ignore kind")
	}
	if len(base) != 64 {
		t.Fatalf("key length %d", len(base))
	}
}

func TestHallOfFameLabel(t *testing.T) {
	cohort, department, principal := 15, "불어과", "교장"
	tests := []struct {
		kind       model.HallOfFameKind
		cohort     *int
		department *string
		role       *string
		want       string
	}{
		{model.HallOfFameKindAlumni, &cohort, &department, nil, "15기 · 불어과"},
		{model.HallOfFameKindAlumni, &cohort, nil, nil, "15기"},
		{model.HallOfFameKindAlumni, nil, nil, nil, ""},
		{model.HallOfFameKindCompanion, &cohort, &department, nil, "15기 · 불어과"},
		{model.HallOfFameKindTeacher, &cohort, &department, nil, "교사"},
		{model.HallOfFameKindTeacher, nil, nil, &principal, "교장"},
		{model.HallOfFameKindGuest, &cohort, &department, nil, ""},
		{model.HallOfFameKindOrg, nil, nil, nil, ""},
	}
	for _, test := range tests {
		if got := hallOfFameLabel(test.kind, test.cohort, test.department, test.role); got != test.want {
			t.Fatalf("%s: got %q, want %q", test.kind, got, test.want)
		}
	}
}
