package golden

import (
	"bytes"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		volatile []string
		want     string
	}{
		{
			name: "nested objects and array order",
			body: `{"z":[{"b":2,"a":1},3,1],"a":{"z":false,"a":null}}`,
			want: "{\n  \"a\": {\n    \"a\": null,\n    \"z\": false\n  },\n  \"z\": [\n    {\n      \"a\": 1,\n      \"b\": 2\n    },\n    3,\n    1\n  ]\n}\n",
		},
		{
			name: "timestamp formats",
			body: `["2026-09-27T10:11:12Z","2026-09-27T10:11:12.123456789+09:00","2026-09-27 10:11:12"]`,
			want: "[\n  \"<timestamp>\",\n  \"<timestamp>\",\n  \"<timestamp>\"\n]\n",
		},
		{
			name: "JWT and opaque tokens",
			body: `{"accessToken":"synthetic-access","refresh_token":"synthetic-refresh","device-token":"synthetic-device","jwt":"synthetic-jwt","tokens":["one","two"],"value":"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiI0MiJ9.c2lnbmF0dXJl","authorization":"Bearer synthetic-opaque"}`,
			want: "{\n  \"accessToken\": \"<token>\",\n  \"authorization\": \"Bearer <token>\",\n  \"device-token\": \"<token>\",\n  \"jwt\": \"<token>\",\n  \"refresh_token\": \"<token>\",\n  \"tokens\": [\n    \"<token>\",\n    \"<token>\"\n  ],\n  \"value\": \"<token>\"\n}\n",
		},
		{
			name:     "volatile keys at any depth",
			body:     `{"requestId":"random","nested":[{"requestId":123,"elapsed":0.125},{"requestId":null}],"metadata":{"anything":true}}`,
			volatile: []string{"requestId", "elapsed", "metadata"},
			want:     "{\n  \"metadata\": \"<volatile>\",\n  \"nested\": [\n    {\n      \"elapsed\": \"<volatile>\",\n      \"requestId\": \"<volatile>\"\n    },\n    {\n      \"requestId\": null\n    }\n  ],\n  \"requestId\": \"<volatile>\"\n}\n",
		},
		{
			name: "ordinary strings and exact numbers",
			body: `{"id":9007199254740993,"amount":1.2300,"text":"동문 <hello>","date":"2026-09-27","invalid":"2026-99-99T10:00:00Z","version":"one.two.three","token_type":"Bearer","expires_in":3600}`,
			want: "{\n  \"amount\": 1.2300,\n  \"date\": \"2026-09-27\",\n  \"expires_in\": 3600,\n  \"id\": 9007199254740993,\n  \"invalid\": \"2026-99-99T10:00:00Z\",\n  \"text\": \"동문 <hello>\",\n  \"token_type\": \"Bearer\",\n  \"version\": \"one.two.three\"\n}\n",
		},
		{
			name: "empty and null tokens preserve contract",
			body: `{"token":"","refreshToken":null,"tokens":[],"nested":{"token":"opaque"}}`,
			want: "{\n  \"nested\": {\n    \"token\": \"<token>\"\n  },\n  \"refreshToken\": null,\n  \"token\": \"\",\n  \"tokens\": []\n}\n",
		},
		{name: "root null", body: `null`, want: "null\n"},
		{name: "root timestamp", body: `"2026-09-27T10:00:00Z"`, want: "\"<timestamp>\"\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Normalize([]byte(test.body), test.volatile...)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("normalized JSON:\n%s\nwant:\n%s", got, test.want)
			}
			again, err := Normalize(got, test.volatile...)
			if err != nil || !bytes.Equal(got, again) {
				t.Fatalf("normalization is not idempotent: %s (%v)", again, err)
			}
		})
	}
}

func TestNormalizeRejectsInvalidJSON(t *testing.T) {
	for _, body := range []string{"", "{", "{\"a\":}", "{} {}", "null true", "{} trailing"} {
		t.Run(body, func(t *testing.T) {
			if _, err := Normalize([]byte(body)); err == nil {
				t.Fatal("accepted invalid or multiple JSON values")
			}
		})
	}
}
