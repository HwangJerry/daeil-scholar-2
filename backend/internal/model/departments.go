// departments.go — Daeil Foreign Language High School department enum values
package model

// ValidDepartments is the canonical list of 대일외고 departments (학과).
// Keep this in sync with frontend/src/constants/departments.ts.
var ValidDepartments = []string{
	"프랑스어",
	"독일어",
	"일본어",
	"중국어",
	"스페인어",
	"러시아어",
	"영어",
}

func IsValidDepartment(value string) bool {
	for _, d := range ValidDepartments {
		if d == value {
			return true
		}
	}
	return false
}
