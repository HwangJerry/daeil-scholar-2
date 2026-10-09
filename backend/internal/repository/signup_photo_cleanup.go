package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
)

// DiscardSignupProfileUpload accepts only a tracked UploadResult. The row and
// exact tracked upload stays locked through filesystem unlink and commit.
// Official claim writers lock this PK before member/owner writes. Reference
// scans are nonlocking current reads, preserving aliases without stopping unrelated
// writes. Arbitrary concurrent copied URLs in legacy content are outside that
// claim protocol; existing references still block deletion.
func (r *FileRepository) DiscardSignupProfileUpload(id int, url, origin string, erase func(string) error) error {
	tx, err := r.DB.BeginTxx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var file struct {
		Gate string `db:"F_GATE"`
		Join int    `db:"F_JOIN_SEQ"`
		URL  string `db:"URL_PATH"`
	}
	err = tx.Get(&file, `SELECT COALESCE(F_GATE,'') AS F_GATE,COALESCE(F_JOIN_SEQ,0) AS F_JOIN_SEQ,CONCAT(FILE_PATH,'/',FILE_NAME) AS URL_PATH FROM WEO_FILES WHERE F_SEQ=? FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if file.Gate != "PR" || file.Join != 0 || file.URL != url {
		return &model.ErasureBlocked{Code: "UPLOAD_OWNERSHIP_CHANGED"}
	}
	local, external, err := model.ErasureFilePath(url, origin)
	if err != nil {
		return err
	}
	if external {
		return nil
	}
	schema, err := readErasureSchema(tx)
	if err != nil {
		return err
	}
	if schema["ALUMNI_UPLOAD_OWNER"] != nil {
		var owners int
		if err = tx.Get(&owners, `SELECT COUNT(*) FROM ALUMNI_UPLOAD_OWNER WHERE F_SEQ=?`, id); err != nil {
			return err
		}
		if owners > 0 {
			return &model.ErasureBlocked{Code: "FILE_STILL_REFERENCED"}
		}
	}
	if err = scanSurvivingFileReferences(tx, schema, local, origin, nil, id, false, signupPhotoReferenceCandidate(local)); err != nil {
		return err
	}
	if err = erase(local); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM WEO_FILES WHERE F_SEQ=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
