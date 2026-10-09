package repository

import (
	"path"
	"strconv"
	"strings"
)

// Only candidate filtering, never an authorized deletion identity. Decode valid
// percent pairs independently so an unrelated malformed escape cannot hide this
// upload's filename. Exact comparison still uses ErasureFileReferencePath.
func signupPhotoReferenceCandidate(local string) func(string) bool {
	filename := path.Base(local)
	return func(raw string) bool {
		for round := 0; round < 3; round++ {
			if strings.Contains(raw, filename) {
				return true
			}
			var decoded strings.Builder
			changed := false
			for i := 0; i < len(raw); i++ {
				if raw[i] == '%' && i+2 < len(raw) {
					if value, err := strconv.ParseUint(raw[i+1:i+3], 16, 8); err == nil {
						decoded.WriteByte(byte(value))
						i += 2
						changed = true
						continue
					}
				}
				decoded.WriteByte(raw[i])
			}
			if !changed {
				return false
			}
			raw = decoded.String()
		}
		return strings.Contains(raw, filename)
	}
}
