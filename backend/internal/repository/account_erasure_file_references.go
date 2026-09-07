// account_erasure_file_references.go — Lock and reject surviving references before unlink.
package repository

import (
	"database/sql"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// A nil excluded user checks every surviving reference, including public authors.
// Locking reads cover the scanned ranges through unlink and queue acknowledgement.
func rejectOtherErasureFileReferences(tx *sqlx.Tx, s erasureSchema, local, origin string, excludedUser *int) error {
	type referenceQuery struct {
		sql     string
		args    []interface{}
		content bool
	}
	queries := []referenceQuery{}
	predicate := ""
	var userArgs []interface{}
	if excludedUser != nil {
		predicate = " WHERE (USR_SEQ<>? OR USR_SEQ IS NULL)"
		userArgs = []interface{}{*excludedUser}
	}
	for _, table := range []string{"ALUMNI_UPLOAD_OWNER", "ALUMNI_PROFILE_FILE_HISTORY"} {
		if s[table] != nil {
			queries = append(queries, referenceQuery{"SELECT URL_PATH FROM " + table + predicate + " FOR UPDATE", userArgs, false})
		}
	}
	for _, column := range []string{"USR_PHOTO", "USR_BIZ_CARD", "USR_THUMNAIL"} {
		if s.has("WEO_MEMBER", column) {
			queries = append(queries, referenceQuery{"SELECT COALESCE(" + column + ",'') FROM WEO_MEMBER" + predicate + " FOR UPDATE", userArgs, false})
		}
	}
	for _, column := range []string{"CONTENTS", "CONTENTS_MD", "THUMBNAIL_URL", "FILES", "RE_FILES"} {
		if s.has("WEO_BOARDBBS", column) {
			queries = append(queries, referenceQuery{"SELECT COALESCE(" + column + ",'') FROM WEO_BOARDBBS" + predicate + " FOR UPDATE", userArgs, true})
		}
	}
	if s["WEO_FILES"] != nil {
		predicate := "1=1"
		if excludedUser != nil {
			predicate = "(F_JOIN_SEQ<>0 OR F_JOIN_SEQ IS NULL)"
		}
		args := []interface{}{}
		if excludedUser != nil && s["WEO_BOARDBBS"] != nil {
			predicate += " AND NOT (COALESCE(F_GATE,'')='BB' AND COALESCE(F_JOIN_SEQ,0) IN (SELECT SEQ FROM WEO_BOARDBBS WHERE USR_SEQ=?))"
			args = append(args, *excludedUser)
		}
		if excludedUser != nil && s["ALUMNI_UPLOAD_OWNER"] != nil {
			predicate += " AND F_SEQ NOT IN (SELECT F_SEQ FROM ALUMNI_UPLOAD_OWNER WHERE USR_SEQ=? AND F_SEQ IS NOT NULL)"
			args = append(args, *excludedUser)
		}
		queries = append(queries, referenceQuery{"SELECT CONCAT(FILE_PATH,'/',FILE_NAME) FROM WEO_FILES WHERE " + predicate + " FOR UPDATE", args, false})
	}
	for _, query := range queries {
		var values []string
		if err := tx.Select(&values, query.sql, query.args...); err != nil {
			return err
		}
		for _, value := range values {
			urls := []string{value}
			if query.content {
				urls = survivingContentURLs(value)
			}
			for _, raw := range urls {
				candidate, external, err := model.ErasureFileReferencePath(raw, origin)
				if err != nil {
					return err
				}
				if !external && candidate == local {
					return &model.ErasureBlocked{Code: "FILE_STILL_REFERENCED"}
				}
			}
		}
	}
	return nil
}

// EraseFileIfUnreferenced repeats the check after DB erasure, including files
// enqueued by the historical review CLI. Filesystem success is retry-safe if
// committing the queue acknowledgement subsequently fails.
func (r *AccountDeletionRequestRepository) EraseFileIfUnreferenced(w model.ErasureWork, file model.ErasureFile, erase func(string) error) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active int
	if err = tx.Get(&active, `SELECT REQUEST_ID FROM ALUMNI_ACCOUNT_ERASURE WHERE REQUEST_ID=? AND MODE='automatic' AND STAGE='database_erased' FOR UPDATE`, w.RequestID); err != nil {
		return err
	}
	var raw string
	if err = tx.Get(&raw, `SELECT URL_PATH FROM ALUMNI_ERASURE_FILE WHERE ID=? AND REQUEST_ID=? FOR UPDATE`, file.ID, w.RequestID); err != nil {
		return err
	}
	if raw != file.URL {
		return sql.ErrNoRows
	}
	local, external, err := model.ErasureFilePath(raw, r.SiteOrigin)
	if err != nil {
		return err
	}
	if external {
		return &model.ErasureBlocked{Code: "EXTERNAL_FILE_HANDOFF_REVIEW_REQUIRED"}
	}
	s, err := readErasureSchema(tx)
	if err != nil {
		return err
	}
	if err = rejectOtherErasureFileReferences(tx, s, local, r.SiteOrigin, nil); err != nil {
		return err
	}
	if err = erase(local); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM ALUMNI_ERASURE_FILE WHERE ID=? AND REQUEST_ID=?`, file.ID, w.RequestID); err != nil {
		return err
	}
	return tx.Commit()
}
