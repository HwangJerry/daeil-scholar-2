// password_service.go — MySQL native password hashing for legacy ID/PW authentication.
package service

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

// MysqlNativePassword computes MySQL's native password hash: "*" + upper(hex(sha1(sha1(password)))).
// This matches the format stored in WEO_MEMBER.USR_PWD by the legacy PHP system.
func MysqlNativePassword(password string) string {
	first := sha1.Sum([]byte(password))
	second := sha1.Sum(first[:])
	return "*" + strings.ToUpper(fmt.Sprintf("%x", second))
}
