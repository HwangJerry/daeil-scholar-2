// File repository — WEO_FILES insert and notice attachment management
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type FileRepository struct {
	DB *sqlx.DB
}

func NewFileRepository(db *sqlx.DB) *FileRepository {
	return &FileRepository{DB: db}
}

func (r *FileRepository) InsertFile(f *model.FileRecord) (int, error) {
	res, err := r.DB.Exec(`
		INSERT INTO WEO_FILES (F_GATE, F_JOIN_SEQ, TYPE_NAME, FILE_NAME, FILE_SIZE, FILE_PATH, FILE_ORG_NAME, OPEN_YN, REG_DATE)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'Y', NOW())
	`, f.FGate, f.FJoinSeq, f.TypeName, f.FileName, f.FileSize, f.FilePath, f.FileOrgName)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// AttachFilesToNotice links orphaned attachment rows (F_JOIN_SEQ=0, F_GATE='BB')
// to a notice. Empty fSeqs is a no-op.
func (r *FileRepository) AttachFilesToNotice(noticeSeq int, fSeqs []int) error {
	if len(fSeqs) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`
		UPDATE WEO_FILES SET F_JOIN_SEQ = ?
		WHERE F_SEQ IN (?) AND F_GATE = 'BB' AND F_JOIN_SEQ = 0 AND OPEN_YN = 'Y'
	`, noticeSeq, fSeqs)
	if err != nil {
		return err
	}
	query = r.DB.Rebind(query)
	_, err = r.DB.Exec(query, args...)
	return err
}

// ReconcileAttachments soft-deletes existing attachments not in keepFSeqs and
// links any new orphaned fSeqs in keepFSeqs to the notice. Atomic per-statement.
func (r *FileRepository) ReconcileAttachments(noticeSeq int, keepFSeqs []int) error {
	if len(keepFSeqs) == 0 {
		_, err := r.DB.Exec(`
			UPDATE WEO_FILES SET OPEN_YN = 'N'
			WHERE F_JOIN_SEQ = ? AND F_GATE = 'BB' AND OPEN_YN = 'Y'
		`, noticeSeq)
		return err
	}

	query, args, err := sqlx.In(`
		UPDATE WEO_FILES SET OPEN_YN = 'N'
		WHERE F_JOIN_SEQ = ? AND F_GATE = 'BB' AND OPEN_YN = 'Y' AND F_SEQ NOT IN (?)
	`, noticeSeq, keepFSeqs)
	if err != nil {
		return err
	}
	query = r.DB.Rebind(query)
	if _, err = r.DB.Exec(query, args...); err != nil {
		return err
	}

	return r.AttachFilesToNotice(noticeSeq, keepFSeqs)
}

// SoftDeleteFilesByJoin marks all attachments of a notice as deleted.
func (r *FileRepository) SoftDeleteFilesByJoin(noticeSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_FILES SET OPEN_YN = 'N'
		WHERE F_JOIN_SEQ = ? AND F_GATE = 'BB'
	`, noticeSeq)
	return err
}

// GetAttachmentsByNotice returns active attachments for a notice (admin edit view).
func (r *FileRepository) GetAttachmentsByNotice(noticeSeq int) ([]model.FileRecord, error) {
	files := make([]model.FileRecord, 0)
	err := r.DB.Select(&files, `
		SELECT F_SEQ, IFNULL(F_GATE,'') AS F_GATE, F_JOIN_SEQ,
		       IFNULL(TYPE_NAME,'') AS TYPE_NAME, IFNULL(FILE_NAME,'') AS FILE_NAME,
		       IFNULL(FILE_SIZE,'0') AS FILE_SIZE, IFNULL(FILE_PATH,'') AS FILE_PATH,
		       IFNULL(FILE_ORG_NAME,'') AS FILE_ORG_NAME, IFNULL(OPEN_YN,'N') AS OPEN_YN
		FROM WEO_FILES
		WHERE F_JOIN_SEQ = ? AND F_GATE = 'BB' AND OPEN_YN = 'Y'
	`, noticeSeq)
	if err != nil {
		return nil, err
	}
	return files, nil
}
