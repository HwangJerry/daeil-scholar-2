package repository

import (
	"errors"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

// SignupRepository owns the transaction that creates a password account and
// all of its required canonical companions.
type SignupRepository struct {
	db *sqlx.DB
}

func NewSignupRepository(db *sqlx.DB) *SignupRepository {
	return &SignupRepository{db: db}
}

func (r *SignupRepository) CreatePasswordAccount(request model.RegisterRequest, legacyHash string, credential model.PasswordCredential) (int, error) {
	invalidCredential := credential.Provider != model.IdentityProviderLocalUsername || credential.Algorithm != model.PasswordAlgorithmArgon2id || credential.PasswordHash == ""
	if request.UsrID == "" || legacyHash == "" || invalidCredential {
		return 0, errors.New("invalid password signup transaction input")
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	accountSeq, err := insertPasswordMemberTx(tx, request, legacyHash)
	if err != nil {
		return 0, err
	}
	if err := insertActiveAccountStateTx(tx, accountSeq); err != nil {
		return 0, err
	}
	if err := insertAlumniVerificationCompanionTx(tx, accountSeq); err != nil {
		return 0, err
	}
	identityID, err := insertLocalPasswordIdentityTx(tx, accountSeq, request.UsrID)
	if err != nil {
		return 0, err
	}
	if err := insertPasswordCredentialTx(tx, identityID, credential); err != nil {
		return 0, err
	}
	if err := insertSignupTagsTx(tx, accountSeq, request.Tags); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return accountSeq, nil
}

func insertActiveAccountStateTx(tx *sqlx.Tx, accountSeq int) error {
	_, err := tx.Exec(`
		INSERT INTO AUTH_ACCOUNT_STATE (ACCOUNT_ID, STATUS, CREATED_AT, UPDATED_AT)
		VALUES (?, 'ACTIVE', NOW(), NOW())
	`, accountSeq)
	return err
}

func insertPasswordMemberTx(tx *sqlx.Tx, request model.RegisterRequest, legacyHash string) (int, error) {
	phonePublic := request.USRPhonePublic
	if phonePublic == "" {
		phonePublic = "N"
	}
	emailPublic := request.USREmailPublic
	if emailPublic == "" {
		emailPublic = "N"
	}
	result, err := tx.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT,
			USR_NICK, USR_DEPT, USR_JOB_CAT, USR_BIZ_NAME, USR_BIZ_DESC, USR_BIZ_ADDR,
			USR_POSITION, USR_PHONE_PUBLIC, USR_EMAIL_PUBLIC)
		VALUES (?, ?, ?, ?, ?, 'BBB', ?, NOW(), 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, request.UsrID, request.Name, request.Phone, request.FN, request.Email, legacyHash,
		request.Nick, request.FmDept, request.JobCat, request.BizName, request.BizDesc, request.BizAddr,
		request.Position, phonePublic, emailPublic)
	if err != nil {
		return 0, err
	}
	accountID, err := result.LastInsertId()
	return int(accountID), err
}

func insertLocalPasswordIdentityTx(tx *sqlx.Tx, accountSeq int, subject string) (int64, error) {
	result, err := tx.Exec(`
		INSERT INTO AUTH_IDENTITY
			(ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, NULL, 'ACTIVE', NOW(), NOW(), NOW())
	`, accountSeq, string(model.IdentityProviderLocalUsername), subject)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func insertPasswordCredentialTx(tx *sqlx.Tx, identityID int64, credential model.PasswordCredential) error {
	_, err := tx.Exec(`
		INSERT INTO AUTH_PASSWORD_CREDENTIAL
			(IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', NOW(), NOW())
	`, identityID, string(model.IdentityProviderLocalUsername), credential.Algorithm, credential.ParametersText, credential.PasswordHash)
	return err
}

func insertSignupTagsTx(tx *sqlx.Tx, accountSeq int, tags []string) error {
	for index, tag := range tags {
		if tag == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO ALUMNI_USER_TAG (USR_SEQ, AUT_TAG, AUT_INDX, REG_DATE)
			VALUES (?, ?, ?, NOW())
		`, accountSeq, tag, index); err != nil {
			return err
		}
	}
	return nil
}
