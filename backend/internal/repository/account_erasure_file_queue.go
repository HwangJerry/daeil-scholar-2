// account_erasure_file_queue.go — Verify ownership before queuing local file erasure.
package repository

import (
	"crypto/sha256"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func erasureFileCandidates(tx *sqlx.Tx, s erasureSchema, w model.ErasureWork) ([]string, []string, error) {
	urls, err := postErasureURLs(tx, s, w.UserSeq)
	if err != nil {
		return nil, nil, err
	}
	ownedPaths := []string{}
	for _, col := range []string{"USR_PHOTO", "USR_BIZ_CARD", "USR_THUMNAIL"} {
		if s.has("WEO_MEMBER", col) {
			var url string
			if err := tx.Get(&url, "SELECT COALESCE(`"+col+"`,'') FROM WEO_MEMBER WHERE USR_SEQ=?", w.UserSeq); err != nil {
				return nil, nil, err
			}
			if url != "" {
				urls = append(urls, url)
				ownedPaths = append(ownedPaths, url)
			}
		}
	}
	var owned []string
	if err := tx.Select(&owned, `SELECT URL_PATH FROM ALUMNI_UPLOAD_OWNER WHERE USR_SEQ=?`, w.UserSeq); err != nil {
		return nil, nil, err
	}
	urls = append(urls, owned...)
	ownedPaths = append(ownedPaths, owned...)
	if s["ALUMNI_PROFILE_FILE_HISTORY"] != nil {
		var previous []string
		if err := tx.Select(&previous, `SELECT URL_PATH FROM ALUMNI_PROFILE_FILE_HISTORY WHERE USR_SEQ=?`, w.UserSeq); err != nil {
			return nil, nil, err
		}
		urls = append(urls, previous...)
		ownedPaths = append(ownedPaths, previous...)
	}
	if s["WEO_FILES"] != nil && s["WEO_BOARDBBS"] != nil {
		var attachments []string
		if err := tx.Select(&attachments, `SELECT CONCAT(FILE_PATH,'/',FILE_NAME) FROM WEO_FILES WHERE F_GATE='BB' AND F_JOIN_SEQ IN (SELECT SEQ FROM WEO_BOARDBBS WHERE USR_SEQ=?)`, w.UserSeq); err != nil {
			return nil, nil, err
		}
		urls = append(urls, attachments...)
		ownedPaths = append(ownedPaths, attachments...)
	}
	return urls, ownedPaths, nil
}

// erasureFilePlan is the verified set of local files to unlink and the
// WEO_FILES metadata rows that point at them.
type erasureFilePlan struct {
	locals  []string
	fileIDs []int
}

// planErasureFiles performs every ownership and shared-reference check without
// writing. A blocked plan returns *model.ErasureBlocked.
func planErasureFiles(tx *sqlx.Tx, s erasureSchema, w model.ErasureWork, origin string) (erasureFilePlan, error) {
	plan := erasureFilePlan{locals: []string{}, fileIDs: []int{}}
	urls, owned, err := erasureFileCandidates(tx, s, w)
	if err != nil {
		return plan, err
	}
	ownership := map[string]bool{}
	for _, raw := range owned {
		local, external, err := model.ErasureFilePath(raw, origin)
		if err != nil {
			return plan, err
		}
		if !external {
			ownership[local] = true
		}
	}
	seen := map[string]bool{}
	for _, raw := range urls {
		local, external, err := model.ErasureFilePath(raw, origin)
		if err != nil {
			return plan, err
		}
		if external || seen[local] {
			continue
		}
		seen[local] = true
		if !ownership[local] {
			return plan, &model.ErasureBlocked{Code: "FILE_OWNERSHIP_REVIEW_REQUIRED"}
		}
		if err := rejectOtherErasureFileReferences(tx, s, local, origin, &w.UserSeq); err != nil {
			return plan, err
		}
		plan.locals = append(plan.locals, local)
		if s["WEO_FILES"] != nil {
			var rows []struct {
				ID  int    `db:"F_SEQ"`
				URL string `db:"URL_PATH"`
			}
			if err := tx.Select(&rows, `SELECT F_SEQ,CONCAT(FILE_PATH,'/',FILE_NAME) AS URL_PATH FROM WEO_FILES FOR UPDATE`); err != nil {
				return plan, err
			}
			for _, row := range rows {
				candidate, ext, e := model.ErasureFilePath(row.URL, origin)
				if e == nil && !ext && candidate == local {
					plan.fileIDs = append(plan.fileIDs, row.ID)
				}
			}
		}
	}
	return plan, nil
}

func queueErasureFiles(tx *sqlx.Tx, s erasureSchema, w model.ErasureWork, origin string) error {
	plan, err := planErasureFiles(tx, s, w, origin)
	if err != nil {
		return err
	}
	for _, local := range plan.locals {
		sum := sha256.Sum256([]byte(local))
		if _, err := tx.Exec(`INSERT IGNORE INTO ALUMNI_ERASURE_FILE (REQUEST_ID,URL_PATH,URL_HASH) VALUES (?,?,?)`, w.RequestID, local, fmt.Sprintf("%x", sum)); err != nil {
			return err
		}
	}
	for _, id := range plan.fileIDs {
		if err := s.erase(tx, "WEO_FILES", "F_SEQ=?", id); err != nil {
			return err
		}
	}
	return nil
}
