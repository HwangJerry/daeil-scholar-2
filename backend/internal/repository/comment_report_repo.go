package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

var ErrCommentReportResolved = errors.New("comment report already resolved")

type CommentReportRepository struct{ DB *sqlx.DB }

func (r *CommentReportRepository) Create(reporter int, request model.CommentReportRequest) (int64, error) {
	// A retry stays successful even after moderation or author deletion.
	var existing int64
	err := r.DB.Get(&existing, `SELECT REPORT_ID FROM ALUMNI_COMMENT_REPORT WHERE REPORTER_SEQ=? AND COMMENT_SEQ=? AND POST_SEQ=?`, reporter, request.CommentID, request.PostID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	// Keep an in-flight request from recreating personal data after account erasure.
	var activeReporter int
	if err = tx.Get(&activeReporter, `SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_SEQ=? AND USR_STATUS IN ('BBB','CCC','ZZZ') LOCK IN SHARE MODE`, reporter); err != nil {
		return 0, err
	}
	var evidence struct {
		Author  int    `db:"USR_SEQ"`
		Content string `db:"CONTENTS"`
	}
	// Lock the parent and comment before capturing evidence; moderation uses the same order.
	var parent int64
	if err = tx.Get(&parent, `SELECT SEQ FROM WEO_BOARDBBS WHERE SEQ=? AND GATE='NOTICE' AND OPEN_YN='Y' LOCK IN SHARE MODE`, request.PostID); err != nil {
		return 0, err
	}
	if err = tx.Get(&evidence, `SELECT USR_SEQ,IFNULL(CONTENTS,'') AS CONTENTS FROM WEO_BOARDCOMAND WHERE SEQ=? AND JOIN_SEQ=? AND BC_TYPE='B' AND OPEN_YN='Y' AND USR_SEQ<>? FOR UPDATE`, request.CommentID, request.PostID, reporter); err != nil {
		return 0, err
	}
	result, err := tx.Exec(`INSERT INTO ALUMNI_COMMENT_REPORT (POST_SEQ,COMMENT_SEQ,REPORTER_SEQ,REPORTED_SEQ,REASON,DETAILS,CONTENT_SNAPSHOT,CREATED_AT) VALUES (?,?,?,?,?,?,?,UTC_TIMESTAMP()) ON DUPLICATE KEY UPDATE REPORT_ID=LAST_INSERT_ID(REPORT_ID)`, request.PostID, request.CommentID, reporter, evidence.Author, request.Reason, request.Details, evidence.Content)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
func (r *CommentReportRepository) List(status string, before int64) ([]model.CommentReport, error) {
	items := []model.CommentReport{}
	err := r.DB.Select(&items, `SELECT r.REPORT_ID,r.POST_SEQ,r.COMMENT_SEQ,r.REPORTER_SEQ,r.REPORTED_SEQ,r.REASON,r.DETAILS,r.CONTENT_SNAPSHOT,r.STATUS,IFNULL(r.MODERATOR_NOTE,'') AS MODERATOR_NOTE,DATE_FORMAT(r.CREATED_AT,'%Y-%m-%dT%H:%i:%sZ') AS CREATED_AT,IFNULL(c.OPEN_YN='Y' AND c.BC_TYPE='B' AND b.OPEN_YN='Y' AND b.GATE='NOTICE',0) AS VISIBLE FROM ALUMNI_COMMENT_REPORT r LEFT JOIN WEO_BOARDCOMAND c ON c.SEQ=r.COMMENT_SEQ AND c.JOIN_SEQ=r.POST_SEQ LEFT JOIN WEO_BOARDBBS b ON b.SEQ=r.POST_SEQ WHERE r.STATUS=? AND (?=0 OR r.REPORT_ID<?) ORDER BY r.REPORT_ID DESC LIMIT 50`, status, before, before)
	return items, err
}
func (r *CommentReportRepository) Resolve(id int64, moderator int, resolution model.CommentReportResolution) error {
	// Find the target first, then serialize all decisions for that comment.
	var target struct {
		Comment int64 `db:"COMMENT_SEQ"`
		Post    int64 `db:"POST_SEQ"`
	}
	if err := r.DB.Get(&target, `SELECT COMMENT_SEQ,POST_SEQ FROM ALUMNI_COMMENT_REPORT WHERE REPORT_ID=?`, id); err != nil {
		return err
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var locked int64
	err = tx.Get(&locked, `SELECT SEQ FROM WEO_BOARDBBS WHERE SEQ=? LOCK IN SHARE MODE`, target.Post)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	err = tx.Get(&locked, `SELECT SEQ FROM WEO_BOARDCOMAND WHERE SEQ=? AND JOIN_SEQ=? AND BC_TYPE='B' FOR UPDATE`, target.Comment, target.Post)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var previous struct {
		Status string `db:"STATUS"`
		Note   string `db:"NOTE"`
	}
	if err = tx.Get(&previous, `SELECT STATUS,IFNULL(MODERATOR_NOTE,'') AS NOTE FROM ALUMNI_COMMENT_REPORT WHERE REPORT_ID=? FOR UPDATE`, id); err != nil {
		return err
	}
	if previous.Status != "open" {
		if previous.Status == resolution.Status && previous.Note == resolution.Note {
			return tx.Commit()
		}
		return ErrCommentReportResolved
	}
	if resolution.Status == "removed" {
		if _, err = tx.Exec(`UPDATE WEO_BOARDCOMAND SET OPEN_YN='N' WHERE SEQ=? AND JOIN_SEQ=? AND BC_TYPE='B'`, target.Comment, target.Post); err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE ALUMNI_COMMENT_REPORT SET STATUS='removed',MODERATOR_SEQ=?,MODERATOR_NOTE=?,RESOLVED_AT=UTC_TIMESTAMP() WHERE COMMENT_SEQ=? AND STATUS='open'`, moderator, resolution.Note, target.Comment)
	} else {
		_, err = tx.Exec(`UPDATE ALUMNI_COMMENT_REPORT SET STATUS='dismissed',MODERATOR_SEQ=?,MODERATOR_NOTE=?,RESOLVED_AT=UTC_TIMESTAMP() WHERE REPORT_ID=?`, moderator, resolution.Note, id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
