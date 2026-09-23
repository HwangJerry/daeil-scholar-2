package config

import "testing"

func TestReviewTestConfiguredRequiresNumbersAndSixDigitCode(t *testing.T) {
	cases := []struct {
		name   string
		config SMSConfig
		want   bool
	}{
		{"both set", SMSConfig{ReviewTestPhones: []string{"01099990001"}, ReviewTestCode: "246810"}, true},
		{"no numbers", SMSConfig{ReviewTestCode: "246810"}, false},
		{"no code", SMSConfig{ReviewTestPhones: []string{"01099990001"}}, false},
		{"short code", SMSConfig{ReviewTestPhones: []string{"01099990001"}, ReviewTestCode: "1234"}, false},
		{"non-numeric code", SMSConfig{ReviewTestPhones: []string{"01099990001"}, ReviewTestCode: "24681a"}, false},
	}
	for _, tc := range cases {
		if got := tc.config.ReviewTestConfigured(); got != tc.want {
			t.Errorf("%s: ReviewTestConfigured() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
