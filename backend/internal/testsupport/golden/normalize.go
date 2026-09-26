// Package golden normalizes synthetic JSON responses and compares golden files.
package golden

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Normalize preserves array order, nulls and number precision, sorts object keys,
// and emits indented JSON with a final newline. Volatile keys match exactly at
// any depth. Timestamps must occupy a whole string. Opaque tokens are recognized
// by token field names; unusual names should be listed as volatile keys.
func Normalize(body []byte, volatileKeys ...string) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode golden JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("expected one JSON value")
	}
	volatile := make(map[string]bool, len(volatileKeys))
	for _, key := range volatileKeys {
		volatile[key] = true
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(normalizeValue(value, "", volatile)); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func normalizeValue(value any, key string, volatile map[string]bool) any {
	if value == nil {
		return nil
	}
	if volatile[key] {
		return "<volatile>"
	}
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			value[key] = normalizeValue(child, key, volatile)
		}
	case []any:
		for i, child := range value {
			value[i] = normalizeValue(child, key, volatile)
		}
	case string:
		if value == "" {
			return value
		}
		if isTokenKey(key) || isJWT(value) {
			return "<token>"
		}
		if strings.HasPrefix(strings.ToLower(value), "bearer ") {
			return "Bearer <token>"
		}
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
			if _, err := time.Parse(layout, value); err == nil {
				return "<timestamp>"
			}
		}
	}
	return value
}

func isTokenKey(key string) bool {
	key = strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
	return strings.HasSuffix(key, "token") || strings.HasSuffix(key, "tokens") || key == "jwt"
}

func isJWT(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	var fields struct {
		Algorithm string `json:"alg"`
	}
	if json.Unmarshal(header, &fields) != nil || fields.Algorithm == "" {
		return false
	}
	for _, part := range parts[1:] {
		if _, err := base64.RawURLEncoding.DecodeString(part); err != nil {
			return false
		}
	}
	return true
}
