// password_service_test.go — Unit tests for MySQL native password hashing
package service

import (
	"strings"
	"testing"
)

func TestMysqlNativePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{
			name:     "known hash for test",
			password: "test",
			want:     "*94BDCEBE19083CE2A1F959FD02F964C7AF4CFC29",
		},
		{
			name:     "known hash for empty string",
			password: "",
			want:     "*BE1BDEC0AA74B4DCB079943E70528096CCA985F8",
		},
		{
			name:     "known hash for password",
			password: "password",
			want:     "*2470C0C06DEE42FD1618BB99005ADCA2EC9D1E19",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MysqlNativePassword(tt.password)
			if got != tt.want {
				t.Errorf("MysqlNativePassword(%q) = %q, want %q", tt.password, got, tt.want)
			}
		})
	}
}

func TestMysqlNativePasswordFormat(t *testing.T) {
	hash := MysqlNativePassword("anything")

	if !strings.HasPrefix(hash, "*") {
		t.Errorf("hash should start with *, got %q", hash)
	}
	if len(hash) != 41 {
		t.Errorf("hash length should be 41 (1 asterisk + 40 hex), got %d", len(hash))
	}
	if hash != strings.ToUpper(hash) {
		t.Errorf("hash should be uppercase, got %q", hash)
	}
}
