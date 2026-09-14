// account_erasure_preview.go — Read-only operator preview of the records an erasure would change.
package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// PreviewErasure never writes. Its transaction is always rolled back, which
// also releases the locking reads shared with the worker's safety checks.
func (r *AccountDeletionRequestRepository) PreviewErasure(id int64) (model.ErasurePreview, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return model.ErasurePreview{}, err
	}
	defer tx.Rollback()
	var user int
	if err = tx.Get(&user, `SELECT USR_SEQ FROM ALUMNI_ACCOUNT_DELETION_REQUEST WHERE REQUEST_ID=? AND STATUS IN ('pending','processing') AND USR_SEQ IS NOT NULL`, id); err != nil {
		return model.ErasurePreview{}, err
	}
	return r.buildErasurePreview(tx, id, user)
}

func (r *AccountDeletionRequestRepository) buildErasurePreview(tx *sqlx.Tx, id int64, user int) (model.ErasurePreview, error) {
	preview, _, err := r.collectErasurePreview(tx, id, user)
	return preview, err
}

// collectErasurePreview also returns every affected row's hash, per table.
func (r *AccountDeletionRequestRepository) collectErasurePreview(tx *sqlx.Tx, id int64, user int) (model.ErasurePreview, []previewTable, error) {
	preview := model.ErasurePreview{RequestID: id, GeneratedAt: time.Now().UTC(), Blockers: []string{},
		Tables: []model.ErasurePreviewTable{}, Files: []string{}, Social: []model.ErasureSocialUnlink{}, Unhandled: []model.AccountDeletionFootprint{}}
	s, err := readErasureSchema(tx)
	if err != nil {
		return preview, nil, err
	}
	email := ""
	if s.has("WEO_MEMBER", "USR_EMAIL") {
		if err = tx.Get(&email, `SELECT COALESCE(USR_EMAIL,'') FROM WEO_MEMBER WHERE USR_SEQ=?`, user); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return preview, nil, err
		}
	}
	if preview.Blockers, err = previewBlockers(tx, s, id, user); err != nil {
		return preview, nil, err
	}
	social, socialBlockers, err := previewSocialUnlinks(tx, s, user)
	if err != nil {
		return preview, nil, err
	}
	preview.Social = social
	preview.Blockers = append(preview.Blockers, socialBlockers...)
	plan, err := planErasureFiles(tx, s, model.ErasureWork{RequestID: id, UserSeq: user}, r.SiteOrigin)
	var blocked *model.ErasureBlocked
	if errors.As(err, &blocked) {
		preview.Blockers = append(preview.Blockers, blocked.Code)
		plan = erasureFilePlan{locals: []string{}, fileIDs: []int{}}
	} else if err != nil {
		return preview, nil, err
	}
	preview.Files = plan.locals
	queries := erasurePreviewQueries(s, user, email, plan)
	covered := map[string]bool{}
	affected := []previewTable{}
	for _, q := range queries {
		covered[q.table] = true
		engineBlocker, err := previewEngineBlocker(tx, q.table)
		if err != nil {
			return preview, nil, err
		}
		if engineBlocker != "" {
			preview.Blockers = append(preview.Blockers, engineBlocker)
		}
		table, err := readPreviewTable(tx, q)
		if err != nil {
			return preview, nil, err
		}
		if table.Count == 0 {
			continue
		}
		if table.Table == "WEO_ORDER" {
			if err = annotateDonationRows(tx, s, &table); err != nil {
				return preview, nil, err
			}
		}
		affected = append(affected, table)
		preview.Tables = append(preview.Tables, table.ErasurePreviewTable)
	}
	footprint, err := deletionFootprint(tx, user)
	if err != nil {
		return preview, nil, err
	}
	for _, item := range footprint {
		// Pending provider revocations are delivered by the worker before deletion.
		if !covered[item.Table] && item.Table != "ALUMNI_SOCIAL_REVOCATION_OUTBOX" {
			preview.Unhandled = append(preview.Unhandled, item)
		}
	}
	if len(preview.Unhandled) > 0 {
		preview.Blockers = append(preview.Blockers, "UNHANDLED_ACCOUNT_REFERENCE")
	}
	preview.Blockers = uniqueStrings(preview.Blockers)
	preview.PlanDigest = previewDigest(affected, preview.Files)
	return preview, affected, nil
}

// erasurePreviewQueries mirrors EraseDatabase's order: files, donations, posts,
// account references, then the member row. Repeated tables merge by OR so a
// row matched by several deletion steps is shown and counted once.
func erasurePreviewQueries(s erasureSchema, user int, email string, plan erasureFilePlan) []previewQuery {
	queries := []previewQuery{}
	index := map[string]int{}
	add := func(q previewQuery) {
		if s[q.table] == nil {
			return
		}
		if i, ok := index[q.table]; ok && queries[i].action == q.action {
			queries[i].where = "(" + queries[i].where + ") OR (" + q.where + ")"
			queries[i].args = append(queries[i].args, q.args...)
			return
		}
		index[q.table] = len(queries)
		queries = append(queries, q)
	}
	if len(plan.fileIDs) > 0 {
		args := make([]interface{}, len(plan.fileIDs))
		for i, id := range plan.fileIDs {
			args[i] = id
		}
		add(previewQuery{table: "WEO_FILES", action: model.ErasureActionDelete, where: "F_SEQ IN (?" + strings.Repeat(",?", len(args)-1) + ")", args: args})
	}
	if s["WEO_ORDER"] != nil {
		orders := "O_SEQ IN (SELECT O_SEQ FROM WEO_ORDER WHERE USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?)"
		add(previewQuery{table: "WEO_PG_DATA", action: model.ErasureActionDelete, where: orders, args: []interface{}{user, user}})
		add(previewQuery{table: "WEO_ORDER", action: model.ErasureActionDelete, where: "USR_SEQ=? OR O_ACCOUNT_USR_SEQ=?", args: []interface{}{user, user}})
		add(previewQuery{table: "ALUMNI_DONATION_RETENTION", action: model.ErasureActionDelete, where: orders, args: []interface{}{user, user}})
	}
	add(previewQuery{table: "WEO_BOARDBBS", action: model.ErasureActionAnonymize, where: "USR_SEQ=?", args: []interface{}{user}, after: boardAnonymizedColumns(s)})
	for _, step := range accountReferenceSteps(s, user, email) {
		add(previewQuery{table: step.table, action: model.ErasureActionDelete, where: step.where, args: step.args})
	}
	add(previewQuery{table: "WEO_MEMBER", action: model.ErasureActionDelete, where: "USR_SEQ=?", args: []interface{}{user}})
	return queries
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
