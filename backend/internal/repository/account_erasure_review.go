// account_erasure_review.go — Operator approval bound to the exact reviewed erasure plan.
package repository

import (
	"crypto/subtle"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrErasurePlanChanged = errors.New("account erasure plan changed after review")

// ExpediteReviewed starts processing immediately only while the records the
// erasure would change still match the plan the operator reviewed. The check
// runs under the worker lock and the locked request row, like every control.
func (r *AccountDeletionRequestRepository) ExpediteReviewed(id int64, operator int, digest string) error {
	return r.controlSchedule(id, operator, "expedite", func(tx *sqlx.Tx, user int) error {
		preview, err := r.buildErasurePreview(tx, id, user)
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(preview.PlanDigest), []byte(digest)) != 1 {
			return ErrErasurePlanChanged
		}
		return nil
	})
}
