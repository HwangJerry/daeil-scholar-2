package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func TestNormalizeMemberProfileValidatesFields(t *testing.T) {
	valid := model.AdminMemberProfileUpdate{USRName: " 홍길동 ", USRPhone: "010-3333-4444", USREmail: "m@example.com", USRFN: "30", USRDept: "영어"}
	tests := []struct {
		name string
		edit func(*model.AdminMemberProfileUpdate)
		want error
	}{
		{"blank name", func(r *model.AdminMemberProfileUpdate) { r.USRName = "  " }, ErrMemberNameRequired},
		{"long name", func(r *model.AdminMemberProfileUpdate) { r.USRName = strings.Repeat("가", 101) }, ErrMemberNameRequired},
		{"bad email", func(r *model.AdminMemberProfileUpdate) { r.USREmail = "not-an-email" }, ErrInvalidMemberEmail},
		{"email with display name", func(r *model.AdminMemberProfileUpdate) { r.USREmail = "Kim <k@example.com>" }, ErrInvalidMemberEmail},
		{"cohort zero", func(r *model.AdminMemberProfileUpdate) { r.USRFN = "0" }, ErrInvalidMemberCohort},
		{"cohort text", func(r *model.AdminMemberProfileUpdate) { r.USRFN = "30기" }, ErrInvalidMemberCohort},
		{"cohort too large", func(r *model.AdminMemberProfileUpdate) { r.USRFN = "100" }, ErrInvalidMemberCohort},
		{"unknown department", func(r *model.AdminMemberProfileUpdate) { r.USRDept = "경영학과" }, ErrInvalidDepartment},
		{"bad phone", func(r *model.AdminMemberProfileUpdate) { r.USRPhone = "12" }, ErrInvalidPhone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.edit(&req)
			if _, err := normalizeMemberProfile(req); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}

	got, err := normalizeMemberProfile(valid)
	if err != nil {
		t.Fatal(err)
	}
	if got.USRName != "홍길동" || got.USRPhone != "01033334444" {
		t.Fatalf("normalized = %+v", got)
	}
}

func TestNormalizeMemberProfileAllowsKeepingOptionalFields(t *testing.T) {
	got, err := normalizeMemberProfile(model.AdminMemberProfileUpdate{USRName: "홍길동"})
	if err != nil {
		t.Fatal(err)
	}
	if got.USRPhone != "" || got.USRFN != "" || got.USRDept != "" || got.USREmail != "" {
		t.Fatalf("optional fields should stay empty: %+v", got)
	}
}

func TestNormalizeMemberFilterRejectsInvalidValues(t *testing.T) {
	invalid := []model.AdminMemberFilter{
		{FN: "abc"},
		{Dept: "경영학과"},
		{RegFrom: "2026-13-01"},
		{RegFrom: "2026-02-01", RegTo: "2026-01-01"},
	}
	for _, filter := range invalid {
		if _, err := normalizeMemberFilter(filter); !errors.Is(err, ErrInvalidMemberFilter) {
			t.Fatalf("filter %+v: error = %v, want ErrInvalidMemberFilter", filter, err)
		}
	}
	got, err := normalizeMemberFilter(model.AdminMemberFilter{Query: " 홍 ", FN: "30", Dept: "영어", RegFrom: "2026-01-01", RegTo: "2026-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "홍" {
		t.Fatalf("query = %q", got.Query)
	}
}
