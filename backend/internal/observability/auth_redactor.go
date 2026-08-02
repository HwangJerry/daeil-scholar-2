package observability

import (
	"encoding/json"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveAuthKeys = map[string]struct{}{
	"accesstoken":       {},
	"refreshtoken":      {},
	"idtoken":           {},
	"identitytoken":     {},
	"authorizationcode": {},
	"clientsecret":      {},
	"nonce":             {},
}

func RedactAuthJSON(raw []byte) []byte {
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return []byte(redactedValue)
	}
	redactAuthValue(value)
	redacted, err := json.Marshal(value)
	if err != nil {
		return []byte(redactedValue)
	}
	return redacted
}

func redactAuthValue(value interface{}) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, child := range current {
			normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
			if _, sensitive := sensitiveAuthKeys[normalized]; sensitive {
				current[key] = redactedValue
				continue
			}
			redactAuthValue(child)
		}
	case []interface{}:
		for _, child := range current {
			redactAuthValue(child)
		}
	}
}
