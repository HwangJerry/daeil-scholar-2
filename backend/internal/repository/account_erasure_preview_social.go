// account_erasure_preview_social.go — Provider unlink steps shown before an operator approves erasure.
package repository

import (
	"database/sql"
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// previewSocialUnlinks lists each linked Apple/Kakao account and whether its
// unlink can run. Kakao account deletion uses the admin key; Apple needs the
// stored credential, so an Apple link without one holds the worker.
func previewSocialUnlinks(tx *sqlx.Tx, s erasureSchema, user int) ([]model.ErasureSocialUnlink, []string, error) {
	unlinks := []model.ErasureSocialUnlink{}
	blockers := []string{}
	if s["WEO_MEMBER_SOCIAL"] == nil {
		return unlinks, blockers, nil
	}
	var gates []string
	if err := tx.Select(&gates, `SELECT DISTINCT NMS_GATE FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ=? AND NMS_GATE IN ('AP','KT') ORDER BY NMS_GATE`, user); err != nil {
		return nil, nil, err
	}
	for _, gate := range gates {
		status := "pending"
		if s["ALUMNI_SOCIAL_REVOCATION_OUTBOX"] != nil {
			var outbox string
			err := tx.Get(&outbox, `SELECT STATUS FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX WHERE USR_SEQ=? AND PROVIDER=? AND ACTION='ACCOUNT_DELETE' ORDER BY CREATED_AT DESC LIMIT 1`, user, gate)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, nil, err
			}
			switch outbox {
			case "DELIVERED":
				status = "delivered"
			case "FAILED", "FINALIZE_FAILED":
				status = "failed"
			}
		}
		if gate == "AP" && status == "pending" && s["ALUMNI_SOCIAL_CREDENTIAL"] != nil {
			var stored int
			if err := tx.Get(&stored, `SELECT COUNT(*) FROM ALUMNI_SOCIAL_CREDENTIAL WHERE USR_SEQ=? AND PROVIDER='AP'`, user); err != nil {
				return nil, nil, err
			}
			if stored == 0 {
				status = "missing_credential"
				blockers = append(blockers, "PROVIDER_CREDENTIAL_MISSING")
			}
		}
		unlinks = append(unlinks, model.ErasureSocialUnlink{Provider: gate, Status: status})
	}
	return unlinks, blockers, nil
}
