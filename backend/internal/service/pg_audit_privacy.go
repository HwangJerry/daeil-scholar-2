// pg_audit_privacy.go — Preserve reconciliation fields without raw card or error data.
package service

import "github.com/dflh-saf/backend/internal/model"

// This is still restricted transaction evidence, not anonymous telemetry. Its
// retention follows the reviewed transaction decision, never a new blanket term.
type pgAuditApproval struct {
	ResCode  string `json:"ResCode"`
	CNO      string `json:"CNO"`
	Amount   string `json:"Amount"`
	AuthNo   string `json:"AuthNo"`
	TranDate string `json:"TranDate"`
	PayType  string `json:"PayType"`
}

func minimizedPGAuditData(data interface{}) interface{} {
	switch value := data.(type) {
	case *model.ApproveResult:
		if value == nil {
			return nil
		}
		return minimizedPGAuditData(*value)
	case model.ApproveResult:
		return pgAuditApproval{ResCode: value.ResCode, CNO: value.CNO, Amount: value.Amount, AuthNo: value.AuthNo, TranDate: value.TranDate, PayType: value.PayType}
	case map[string]int:
		if count, ok := value["failCount"]; ok && count >= 0 {
			return map[string]int{"failCount": count}
		}
	}
	// Raw gateway messages and arbitrary caller maps can contain personal data.
	return nil
}
