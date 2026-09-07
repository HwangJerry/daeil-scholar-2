// pg_audit_sanitize.go — Prepare minimized historical PG records without losing evidence.
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"io"
	"regexp"
	"time"
)

var auditRecordToken = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)
var auditEventToken = regexp.MustCompile(`^[a-z_]{1,64}$`)

// Unknown fields/types require review instead of silently dropping new financial
// evidence. The output retains transaction identifiers and is NOT anonymous.
func SanitizePGAuditRecord(line []byte) ([]byte, error) {
	invalid := errors.New("PG_AUDIT_RECORD_REVIEW_REQUIRED")
	var source struct {
		Timestamp string          `json:"ts"`
		OrderNo   string          `json:"order_no"`
		Event     string          `json:"event"`
		Data      json.RawMessage `json:"data"`
		Error     string          `json:"error"`
	}
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&source) != nil || decoder.Decode(new(interface{})) != io.EOF {
		return nil, invalid
	}
	if _, err := time.Parse(time.RFC3339, source.Timestamp); err != nil {
		return nil, invalid
	}
	if !auditRecordToken.MatchString(source.OrderNo) || !auditEventToken.MatchString(source.Event) {
		return nil, invalid
	}
	var fields map[string]json.RawMessage
	if len(source.Data) > 0 && json.Unmarshal(source.Data, &fields) != nil {
		return nil, invalid
	}
	var value interface{}
	for key := range fields {
		switch key {
		case "ResCode", "ResMsg", "CNO", "Amount", "AuthNo", "TranDate", "CardNo", "PayType", "IssuerName", "AcquirerName":
		case "reason", "failCount":
		default:
			return nil, invalid
		}
	}
	if fields["reason"] != nil || fields["failCount"] != nil {
		if len(fields) != 1 {
			return nil, invalid
		}
		if raw := fields["failCount"]; raw != nil {
			var count int
			if json.Unmarshal(raw, &count) != nil || count < 0 {
				return nil, invalid
			}
			value = map[string]int{"failCount": count}
		}
	} else if len(fields) > 0 {
		var approval model.ApproveResult
		if json.Unmarshal(source.Data, &approval) != nil {
			return nil, invalid
		}
		value = approval
	}
	output := PGAuditEntry{Timestamp: source.Timestamp, OrderNo: source.OrderNo, Event: source.Event, Data: minimizedPGAuditData(value)}
	if source.Error != "" {
		output.Error = "operation_failed"
	}
	return json.Marshal(output)
}
