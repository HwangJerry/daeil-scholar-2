package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
)

// DiscardSignupProfileUpload accepts only a tracked UploadResult. The row and
// surviving reference ranges stay locked through filesystem unlink and commit.
// Reuse erasure's canonical alias/reference handling; never discard owned files.
func (r *FileRepository) DiscardSignupProfileUpload(id int, url, origin string, erase func(string) error) error {
	tx, err := r.DB.Beginx()
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
		return nil
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
	if err = rejectOtherErasureFileReferencesExceptUpload(tx, schema, local, origin, nil, id); err != nil {
		var blocked *model.ErasureBlocked
		if errors.As(err, &blocked) {
			return nil
		}
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
