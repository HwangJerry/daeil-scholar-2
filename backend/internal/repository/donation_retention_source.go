// donation_retention_source.go — Bind reviewed evidence to a locked donation snapshot.
package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
	"time"
)

// Include every persisted field, including legacy values. Unknown schema changes
// invalidate old evidence safely instead of silently omitting new financial data.
func donationSourceFingerprint(tx *sqlx.Tx, id int) (string, error) {
	row := map[string]interface{}{}
	if err := tx.QueryRowx(`SELECT * FROM WEO_ORDER WHERE O_SEQ=? FOR UPDATE`, id).MapScan(row); err != nil {
		return "", err
	}
	for key, value := range row {
		if raw, ok := value.([]byte); ok {
			row[key] = string(raw)
		}
	}
	data, err := json.Marshal(row)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// DonationRetentionSource returns only a version token, never donor identifiers.
func (r *AccountDeletionRequestRepository) DonationRetentionSource(id int) (string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	return donationSourceFingerprint(tx, id)
}

func verifyDonationRetentionSource(d model.DonationRetentionDecision, current string) error {
	if d.SourceFingerprint == "" || d.SourceFingerprint != current {
		return &model.ErasureBlocked{Code: "DONATION_RETENTION_REVIEW_REQUIRED"}
	}
	return nil
}

func contextIncludesRetention(snapshot []model.DonationRetentionDecision, current model.DonationRetentionDecision) bool {
	date := func(v *time.Time) string {
		if v == nil {
			return ""
		}
		return v.Format("2006-01-02")
	}
	for _, old := range snapshot {
		if old.OrderID == current.OrderID {
			return old.SourceFingerprint == current.SourceFingerprint && old.Basis == current.Basis && old.Evidence == current.Evidence && date(old.BasisDate) == date(current.BasisDate) && date(old.Until) == date(current.Until)
		}
	}
	return false
}
