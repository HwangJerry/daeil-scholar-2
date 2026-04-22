// departments.go — Daeil Foreign Language High School department enum values
package model

// ValidDepartments is the canonical list of 대일외고 departments (학과).
// Keep this in sync with frontend/src/constants/departments.ts.
var ValidDepartments = []string{
	"영어과",
	"독일어과",
	"일본어과",
	"중국어과",
	"프랑스어과",
	"스페인어과",
}

func IsValidDepartment(value string) bool {
	for _, d := range ValidDepartments {
		if d == value {
			return true
		}
	}
	return false
}
