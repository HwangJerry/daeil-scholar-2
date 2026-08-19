package repository

import (
	"database/sql"
	"errors"
	"sort"
	"sync/atomic"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	ErrRefreshTokenInvalid  = errors.New("refresh token is invalid")
	ErrRefreshTokenReplay   = errors.New("refresh token replay detected")
	ErrChallengeInvalid     = errors.New("apple challenge is invalid")
	ErrAuthorizationReplay  = errors.New("authorization code replay detected")
	ErrSocialIdentityOwner  = errors.New("social identity belongs to another member")
	ErrPhoneAlreadyClaimed  = errors.New("canonical phone already claimed")
	ErrInvalidPhone         = errors.New("canonical phone is invalid")
	ErrPhoneClaimsMigrating = errors.New("phone claim migration is in progress")
	ErrLastLoginMethod      = errors.New("cannot disconnect the last login method")
)

// MariaDB 10.1 has no REGEXP_REPLACE. This compatibility expression is kept in
// one place for legacy rows that still contain hyphens or spaces. Exact
// canonical matches remain first in each predicate so migrated/current data can
// use the USR_PHONE index.
const legacyCanonicalPhoneSQL = "REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')"

type AuthRepository struct {
	DB                      *sqlx.DB
	phoneClaimsReady        atomic.Bool
	phoneClaimAutoDetection atomic.Bool
}

type SocialAccountFields struct {
	Provider            string
	SocialID            string
	SocialEmail         string
	USRSeq              int
	USRID               string
	Name                string
	Phone               string
	FN                  string
	Email               string
	FmDept              string
	JobCat              *int
	BizName             string
	BizDesc             string
	BizAddr             string
	Position            string
	USRPhonePublic      string
	USREmailPublic      string
	ProfileImageURL     string
	EncryptedCredential string
}

type SocialDisconnectPhase int

const (
	SocialDisconnectNotConnected SocialDisconnectPhase = iota
	SocialDisconnectRevokeFresh
	SocialDisconnectRevokeRetry
	SocialDisconnectFinalizePending
)

func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{DB: db}
}

func (r *AuthRepository) EnablePhoneClaims() {
	r.phoneClaimsReady.Store(true)
}

func (r *AuthRepository) EnablePhoneClaimAutoDetection() {
	r.phoneClaimAutoDetection.Store(true)
}

func (r *AuthRepository) phoneClaimsEnabledTx(tx *sqlx.Tx) (bool, error) {
	if r.phoneClaimsReady.Load() {
		return true, nil
	}
	if !r.phoneClaimAutoDetection.Load() {
		return false, nil
	}
	return detectPhoneClaimsInWriteTransaction(tx)
}

func detectPhoneClaimsInWriteTransaction(tx *sqlx.Tx) (bool, error) {
	var state string
	err := tx.Get(&state, `
		SELECT state FROM _migration_journal
		WHERE filename = '044_enforce_account_lifecycle_invariants.sql'
		FOR UPDATE
	`)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if state == "STARTED" {
		return false, ErrPhoneClaimsMigrating
	}
	ready := state == "APPLIED"
	return ready, nil
}

func (r *AuthRepository) LookupLegacySession(sessionID string) (*model.AuthUser, error) {
	var user model.AuthUser
	err := r.DB.Get(&user, `
		SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS
		FROM WEO_MEMBER_LOG l
		JOIN WEO_MEMBER m ON l.USR_SEQ = m.USR_SEQ
		WHERE l.SESSIONID = ?
		  AND l.LOG_DATE > DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY l.LOG_DATE DESC
		LIMIT 1
	`, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberBySocialID(gate string, socialID string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS, m.USR_PHONE, m.USR_FN, m.USR_EMAIL, m.USR_NICK, m.USR_PHOTO
		FROM WEO_MEMBER_SOCIAL s
		JOIN WEO_MEMBER m ON s.USR_SEQ = m.USR_SEQ
		WHERE s.NMS_GATE = ? AND s.NMS_ID = ?
		  AND s.NMS_STATUS = 'ACTIVE'
		LIMIT 1
	`, gate, socialID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByKakaoID(kakaoID string) (*model.User, error) {
	return r.FindMemberBySocialID("KT", kakaoID)
}

func (r *AuthRepository) FindMemberByNamePhone(name string, phone string) (*model.User, error) {
	canonicalPhone := model.NormalizePhoneNumber(phone).String()
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_NAME = ?
		  AND (USR_PHONE = ? OR `+legacyCanonicalPhoneSQL+` = ?)
		  AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, name, canonicalPhone, canonicalPhone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByFNName(fn string, name string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_FN = ? AND USR_NAME = ? AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, fn, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) InsertSocialLink(usrSeq int, gate string, socialID string, email string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER_SOCIAL (USR_SEQ, NMS_GATE, NMS_ID, NMS_EMAIL, REG_DATE)
		VALUES (?, ?, ?, ?, NOW())
	`, usrSeq, gate, socialID, email)
	return err
}

func (r *AuthRepository) CreateSocialAccount(fields SocialAccountFields) (*model.User, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	phoneClaimsEnabled, err := r.phoneClaimsEnabledTx(tx)
	if err != nil {
		return nil, err
	}
	if !phoneClaimsEnabled && len(model.NormalizePhoneNumber(fields.Phone)) > model.LegacyPhoneDigitsLimit {
		return nil, ErrInvalidPhone
	}

	phonePublic, emailPublic := socialPrivacyDefaults(fields.USRPhonePublic, fields.USREmailPublic)
	var photo sql.NullString
	if fields.ProfileImageURL != "" {
		photo = sql.NullString{String: fields.ProfileImageURL, Valid: true}
	}
	result, err := tx.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT,
			USR_PHOTO, USR_DEPT, USR_JOB_CAT, USR_BIZ_NAME, USR_BIZ_DESC, USR_BIZ_ADDR,
			USR_POSITION, USR_PHONE_PUBLIC, USR_EMAIL_PUBLIC)
		VALUES (?, ?, ?, ?, ?, 'BBB', '', NOW(), 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, fields.USRID, fields.Name, fields.Phone, fields.FN, fields.Email, photo, fields.FmDept,
		fields.JobCat, fields.BizName, fields.BizDesc, fields.BizAddr, fields.Position, phonePublic, emailPublic)
	if err != nil {
		return nil, err
	}
	usrSeq, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	fields.USRSeq = int(usrSeq)
	if phoneClaimsEnabled {
		if err := claimPhoneTx(tx, fields.Phone, fields.USRSeq); err != nil {
			return nil, err
		}
	}
	if err := insertAlumniVerificationCompanionTx(tx, fields.USRSeq); err != nil {
		return nil, err
	}
	if err := insertSocialConnectionTx(tx, fields); err != nil {
		return nil, err
	}

	user, err := getMemberBySequenceTx(tx, fields.USRSeq)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) MergeSocialAccount(fields SocialAccountFields) (*model.User, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var socialOwner int
	err = tx.Get(&socialOwner, `
		SELECT USR_SEQ
		FROM WEO_MEMBER_SOCIAL
		WHERE NMS_GATE = ? AND NMS_ID = ?
		LIMIT 1
		FOR UPDATE
	`, fields.Provider, fields.SocialID)
	switch {
	case err == nil && socialOwner != fields.USRSeq:
		return nil, ErrSocialIdentityOwner
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	phonePublic, emailPublic := socialPrivacyDefaults(fields.USRPhonePublic, fields.USREmailPublic)
	if _, err := tx.Exec(`
		UPDATE WEO_MEMBER
		SET USR_NAME = ?, USR_EMAIL = ?, USR_JOB_CAT = ?, USR_BIZ_NAME = ?, USR_BIZ_DESC = ?,
		    USR_BIZ_ADDR = ?, USR_POSITION = ?, USR_PHONE_PUBLIC = ?, USR_EMAIL_PUBLIC = ?
		WHERE USR_SEQ = ?
	`, fields.Name, fields.Email, fields.JobCat, fields.BizName,
		fields.BizDesc, fields.BizAddr, fields.Position, phonePublic, emailPublic, fields.USRSeq); err != nil {
		return nil, err
	}
	if err := insertSocialConnectionTx(tx, fields); err != nil {
		return nil, err
	}
	if fields.ProfileImageURL != "" {
		if _, err := tx.Exec(`
			UPDATE WEO_MEMBER
			SET USR_PHOTO = ?
			WHERE USR_SEQ = ? AND (USR_PHOTO IS NULL OR USR_PHOTO = '')
		`, fields.ProfileImageURL, fields.USRSeq); err != nil {
			return nil, err
		}
	}

	user, err := getMemberBySequenceTx(tx, fields.USRSeq)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) AttachSocialAccount(fields SocialAccountFields) (*model.User, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var socialOwner int
	err = tx.Get(&socialOwner, `
		SELECT USR_SEQ
		FROM WEO_MEMBER_SOCIAL
		WHERE NMS_GATE = ? AND NMS_ID = ?
		LIMIT 1
		FOR UPDATE
	`, fields.Provider, fields.SocialID)
	switch {
	case err == nil && socialOwner != fields.USRSeq:
		return nil, ErrSocialIdentityOwner
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}

	if err := insertSocialConnectionTx(tx, fields); err != nil {
		return nil, err
	}
	user, err := getMemberBySequenceTx(tx, fields.USRSeq)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

func insertAlumniVerificationCompanionTx(tx *sqlx.Tx, usrSeq int) error {
	_, err := tx.Exec(`
		INSERT INTO ALUMNI_VERIFICATION (USR_SEQ, STATUS, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, NOW(), NOW())
	`, usrSeq, model.VerificationUnsubmitted)
	return err
}

func insertSocialConnectionTx(tx *sqlx.Tx, fields SocialAccountFields) error {
	if _, err := tx.Exec(`
		INSERT INTO WEO_MEMBER_SOCIAL (USR_SEQ, NMS_GATE, NMS_ID, NMS_EMAIL, REG_DATE)
		VALUES (?, ?, ?, ?, NOW())
	`, fields.USRSeq, fields.Provider, fields.SocialID, fields.SocialEmail); err != nil {
		return err
	}
	if fields.EncryptedCredential == "" {
		return nil
	}
	_, err := tx.Exec(`
		INSERT INTO ALUMNI_SOCIAL_CREDENTIAL
			(USR_SEQ, PROVIDER, ENCRYPTED_CREDENTIAL, UPDATED_AT)
		VALUES (?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			ENCRYPTED_CREDENTIAL = VALUES(ENCRYPTED_CREDENTIAL),
			UPDATED_AT = NOW()
	`, fields.USRSeq, fields.Provider, fields.EncryptedCredential)
	return err
}

func getMemberBySequenceTx(tx *sqlx.Tx, usrSeq int) (*model.User, error) {
	var user model.User
	if err := tx.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		LIMIT 1
	`, usrSeq); err != nil {
		return nil, err
	}
	return &user, nil
}

func socialPrivacyDefaults(phonePublic string, emailPublic string) (string, string) {
	if phonePublic == "" {
		phonePublic = "N"
	}
	if emailPublic == "" {
		emailPublic = "N"
	}
	return phonePublic, emailPublic
}

func (r *AuthRepository) DeleteSocialLink(usrSeq int, gate string) error {
	_, err := r.DB.Exec(`DELETE FROM WEO_MEMBER_SOCIAL WHERE USR_SEQ = ? AND NMS_GATE = ?`, usrSeq, gate)
	return err
}

func (r *AuthRepository) ListSocialProviders(usrSeq int) ([]string, error) {
	var providers []string
	err := r.DB.Select(&providers, `
		SELECT NMS_GATE
		FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_STATUS = 'ACTIVE'
	`, usrSeq)
	return providers, err
}

func (r *AuthRepository) GetAccountConnections(usrSeq int) (model.AccountConnections, error) {
	return getAccountConnections(r.DB, usrSeq)
}

type connectionReader interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

func getAccountConnections(reader connectionReader, usrSeq int) (model.AccountConnections, error) {
	var hasPassword bool
	if err := reader.Get(&hasPassword, `
		SELECT EXISTS(
			SELECT 1
			FROM AUTH_IDENTITY i
			JOIN AUTH_PASSWORD_CREDENTIAL c ON c.IDENTITY_ID = i.IDENTITY_ID
			JOIN AUTH_ACCOUNT_STATE account_state ON account_state.ACCOUNT_ID = i.ACCOUNT_ID
			WHERE i.ACCOUNT_ID = ?
			  AND i.PROVIDER IN ('EMAIL', 'LOCAL_USERNAME')
			  AND i.STATUS = 'ACTIVE'
			  AND c.STATUS = 'ACTIVE'
			  AND account_state.STATUS = 'ACTIVE'
		)
	`, usrSeq); err != nil {
		return model.AccountConnections{}, err
	}
	var rawProviders []string
	err := reader.Select(&rawProviders, `
		SELECT NMS_GATE
		FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_STATUS = 'ACTIVE'
	`, usrSeq)
	if err != nil {
		return model.AccountConnections{}, err
	}
	providers := make([]model.SocialProvider, 0, len(rawProviders))
	for _, rawProvider := range rawProviders {
		provider := model.SocialProvider(rawProvider)
		if provider.Valid() {
			providers = append(providers, provider)
		}
	}
	sort.Slice(providers, func(left int, right int) bool {
		return providers[left] < providers[right]
	})
	return model.AccountConnections{
		Providers:   providers,
		HasPassword: hasPassword,
	}, nil
}

// ReserveSocialDisconnect serializes the last-login-method check on the member
// row, making the invariant effective across backend replicas.
func (r *AuthRepository) ReserveSocialDisconnect(usrSeq int, provider model.SocialProvider) (model.AccountConnections, SocialDisconnectPhase, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	defer tx.Rollback()

	var lockedAccount int
	if err := tx.Get(&lockedAccount, `
		SELECT USR_SEQ
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		FOR UPDATE
	`, usrSeq); err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	var providerStatus string
	err = tx.Get(&providerStatus, `
		SELECT NMS_STATUS
		FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_GATE = ?
		LIMIT 1
	`, usrSeq, string(provider))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	connections, err := getAccountConnections(tx, usrSeq)
	if err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	if providerStatus == "FINALIZE_PENDING" {
		if err := tx.Commit(); err != nil {
			return model.AccountConnections{}, SocialDisconnectNotConnected, err
		}
		return connections, SocialDisconnectFinalizePending, nil
	}
	if providerStatus == "DISCONNECTING" {
		if err := tx.Commit(); err != nil {
			return model.AccountConnections{}, SocialDisconnectNotConnected, err
		}
		return connections, SocialDisconnectRevokeRetry, nil
	}
	if providerStatus != "ACTIVE" {
		if err := tx.Commit(); err != nil {
			return model.AccountConnections{}, SocialDisconnectNotConnected, err
		}
		return connections, SocialDisconnectNotConnected, nil
	}
	if !connections.HasAlternativeTo(provider) {
		return model.AccountConnections{}, SocialDisconnectRevokeFresh, ErrLastLoginMethod
	}
	result, err := tx.Exec(`
		UPDATE WEO_MEMBER_SOCIAL
		SET NMS_STATUS = 'DISCONNECTING'
		WHERE USR_SEQ = ? AND NMS_GATE = ? AND NMS_STATUS = 'ACTIVE'
	`, usrSeq, string(provider))
	if err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	if affected != 1 {
		return model.AccountConnections{}, SocialDisconnectNotConnected, errors.New("social disconnect reservation was not acquired")
	}
	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX
			(USR_SEQ, PROVIDER, ACTION, STATUS, ATTEMPT_COUNT, NEXT_ATTEMPT_AT, CREATED_AT, UPDATED_AT)
		SELECT ?, ?, 'DISCONNECT', 'PENDING', 0, NOW(), NOW(), NOW()
		FROM DUAL
		WHERE NOT EXISTS (
			SELECT 1 FROM ALUMNI_SOCIAL_REVOCATION_OUTBOX
			WHERE USR_SEQ = ? AND PROVIDER = ? AND ACTION = 'DISCONNECT'
			  AND STATUS IN ('PENDING','PROCESSING')
		)
	`, usrSeq, string(provider), usrSeq, string(provider)); err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	if err := tx.Commit(); err != nil {
		return model.AccountConnections{}, SocialDisconnectNotConnected, err
	}
	return connections, SocialDisconnectRevokeFresh, nil
}

func (r *AuthRepository) MarkSocialDisconnectRevoked(usrSeq int, provider string) error {
	result, err := r.DB.Exec(`
		UPDATE WEO_MEMBER_SOCIAL
		SET NMS_STATUS = 'FINALIZE_PENDING'
		WHERE USR_SEQ = ? AND NMS_GATE = ? AND NMS_STATUS = 'DISCONNECTING'
	`, usrSeq, provider)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("social disconnect revocation state is missing")
	}
	return nil
}

func (r *AuthRepository) DeleteSocialConnection(usrSeq int, provider string) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		DELETE FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_GATE = ?
	`, usrSeq, provider); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		DELETE FROM ALUMNI_SOCIAL_CREDENTIAL
		WHERE USR_SEQ = ? AND PROVIDER = ?
	`, usrSeq, provider); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'DELIVERED', UPDATED_AT = NOW(), LAST_ERROR = NULL
		WHERE USR_SEQ = ? AND PROVIDER = ? AND ACTION = 'DISCONNECT'
		  AND STATUS IN ('PENDING','PROCESSING','REVOKED')
	`, usrSeq, provider); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AuthRepository) UpdateSocialProviderEmailEnabled(usrSeq int, gate string, emailEnabled bool) error {
	emailEnabledValue := "N"
	if emailEnabled {
		emailEnabledValue = "Y"
	}
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER_SOCIAL
		SET NMS_EMAIL_ENABLED = ?
		WHERE USR_SEQ = ? AND NMS_GATE = ? AND NMS_STATUS = 'ACTIVE'
	`, emailEnabledValue, usrSeq, gate)
	return err
}

func (r *AuthRepository) UpdateMemberStatus(usrSeq int, status string) error {
	_, err := r.DB.Exec(`UPDATE WEO_MEMBER SET USR_STATUS = ? WHERE USR_SEQ = ?`, status, usrSeq)
	return err
}

func (r *AuthRepository) UpsertSocialCredential(usrSeq int, provider string, encryptedCredential string) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_SOCIAL_CREDENTIAL
			(USR_SEQ, PROVIDER, ENCRYPTED_CREDENTIAL, UPDATED_AT)
		VALUES (?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE
			ENCRYPTED_CREDENTIAL = VALUES(ENCRYPTED_CREDENTIAL),
			UPDATED_AT = NOW()
	`, usrSeq, provider, encryptedCredential)
	return err
}

func (r *AuthRepository) GetSocialCredential(usrSeq int, provider string) (string, error) {
	var encrypted string
	err := r.DB.Get(&encrypted, `
		SELECT ENCRYPTED_CREDENTIAL
		FROM ALUMNI_SOCIAL_CREDENTIAL
		WHERE USR_SEQ = ? AND PROVIDER = ?
	`, usrSeq, provider)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return encrypted, err
}

func (r *AuthRepository) DeleteSocialCredential(usrSeq int, provider string) error {
	_, err := r.DB.Exec(`
		DELETE FROM ALUMNI_SOCIAL_CREDENTIAL
		WHERE USR_SEQ = ? AND PROVIDER = ?
	`, usrSeq, provider)
	return err
}

func (r *AuthRepository) EnqueueSocialRevocation(usrSeq int, provider string, action string, failure error) error {
	lastError := ""
	if failure != nil {
		lastError = failure.Error()
		if len(lastError) > 500 {
			lastError = lastError[:500]
		}
	}
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_SOCIAL_REVOCATION_OUTBOX
			(USR_SEQ, PROVIDER, ACTION, STATUS, ATTEMPT_COUNT, NEXT_ATTEMPT_AT, LAST_ERROR, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, 'PENDING', 0, NOW(), ?, NOW(), NOW())
	`, usrSeq, provider, action, lastError)
	return err
}

func (r *AuthRepository) RecordSocialDisconnectFailure(usrSeq int, provider string, failure error, restoreConnection bool) error {
	lastError := "provider revocation failed"
	if failure != nil && failure.Error() != "" {
		lastError = failure.Error()
		if len(lastError) > 500 {
			lastError = lastError[:500]
		}
	}
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var lockedAccount int
	if err := tx.Get(&lockedAccount, `
		SELECT USR_SEQ FROM WEO_MEMBER WHERE USR_SEQ = ? FOR UPDATE
	`, usrSeq); err != nil {
		return err
	}
	result, err := tx.Exec(`
		UPDATE ALUMNI_SOCIAL_REVOCATION_OUTBOX
		SET STATUS = 'PENDING', ATTEMPT_COUNT = ATTEMPT_COUNT + 1,
		    NEXT_ATTEMPT_AT = DATE_ADD(NOW(), INTERVAL 5 MINUTE), LAST_ERROR = ?, UPDATED_AT = NOW()
		WHERE USR_SEQ = ? AND PROVIDER = ? AND ACTION = 'DISCONNECT'
		  AND STATUS IN ('PENDING','PROCESSING')
	`, lastError, usrSeq, provider)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("social disconnect outbox reservation is missing")
	}
	if !restoreConnection {
		return tx.Commit()
	}
	result, err = tx.Exec(`
		UPDATE WEO_MEMBER_SOCIAL
		SET NMS_STATUS = 'ACTIVE'
		WHERE USR_SEQ = ? AND NMS_GATE = ? AND NMS_STATUS = 'DISCONNECTING'
	`, usrSeq, provider)
	if err != nil {
		return err
	}
	affected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("social disconnect reservation is missing")
	}
	return tx.Commit()
}

func (r *AuthRepository) InsertMobileRefreshToken(usrSeq int, sid string, jti string, expiresAt time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN
			(MRT_JTI, USR_SEQ, MRT_SID, EXPIRES_AT, CREATED_AT)
		VALUES (?, ?, ?, ?, NOW())
	`, jti, usrSeq, sid, expiresAt)
	return err
}

func (r *AuthRepository) RotateMobileRefreshToken(usrSeq int, sid string, oldJTI string, newJTI string, newExpiresAt time.Time) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var row struct {
		SessionID  string       `db:"MRT_SID"`
		ExpiresAt  time.Time    `db:"EXPIRES_AT"`
		ConsumedAt sql.NullTime `db:"CONSUMED_AT"`
		RevokedAt  sql.NullTime `db:"REVOKED_AT"`
	}
	err = tx.Get(&row, `
		SELECT MRT_SID, EXPIRES_AT, CONSUMED_AT, REVOKED_AT
		FROM ALUMNI_MOBILE_REFRESH_TOKEN
		WHERE MRT_JTI = ? AND USR_SEQ = ?
		FOR UPDATE
	`, oldJTI, usrSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRefreshTokenInvalid
	}
	if err != nil {
		return err
	}
	if row.SessionID != sid {
		return ErrRefreshTokenInvalid
	}

	isReplay := row.ConsumedAt.Valid || row.RevokedAt.Valid
	isExpired := !row.ExpiresAt.After(time.Now())
	if isReplay || isExpired {
		if _, err := tx.Exec(`
			UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
			SET REVOKED_AT = COALESCE(REVOKED_AT, NOW())
			WHERE USR_SEQ = ? AND MRT_SID = ? AND REVOKED_AT IS NULL
		`, usrSeq, sid); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		if isReplay {
			return ErrRefreshTokenReplay
		}
		return ErrRefreshTokenInvalid
	}

	result, err := tx.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET CONSUMED_AT = NOW(), ROTATED_TO_JTI = ?
		WHERE MRT_JTI = ? AND USR_SEQ = ? AND CONSUMED_AT IS NULL AND REVOKED_AT IS NULL
	`, newJTI, oldJTI, usrSeq)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrRefreshTokenReplay
	}
	if _, err := tx.Exec(`
		INSERT INTO ALUMNI_MOBILE_REFRESH_TOKEN
			(MRT_JTI, USR_SEQ, MRT_SID, EXPIRES_AT, CREATED_AT)
		VALUES (?, ?, ?, ?, NOW())
	`, newJTI, usrSeq, sid, newExpiresAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *AuthRepository) RevokeMobileSession(usrSeq int, sid string) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET REVOKED_AT = COALESCE(REVOKED_AT, NOW())
		WHERE USR_SEQ = ? AND MRT_SID = ?
	`, usrSeq, sid)
	return err
}

func (r *AuthRepository) RevokeAllMobileSessions(usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE ALUMNI_MOBILE_REFRESH_TOKEN
		SET REVOKED_AT = COALESCE(REVOKED_AT, NOW())
		WHERE USR_SEQ = ?
	`, usrSeq)
	return err
}

// DeleteExpiredMobileRefreshTokens removes refresh token rows that are no
// longer useful to retain: naturally expired (EXPIRES_AT in the past), or
// revoked long enough ago (REVOKED_AT older than revokedBefore) that replay
// detection no longer needs them. Rows that are merely CONSUMED (rotated to a
// successor token) but not yet expired or revoked are never deleted - they
// remain needed for replay detection until they naturally expire.
func (r *AuthRepository) DeleteExpiredMobileRefreshTokens(revokedBefore time.Time) (int64, error) {
	result, err := r.DB.Exec(`
		DELETE FROM ALUMNI_MOBILE_REFRESH_TOKEN
		WHERE EXPIRES_AT < NOW()
		   OR (REVOKED_AT IS NOT NULL AND REVOKED_AT < ?)
	`, revokedBefore)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *AuthRepository) DeletePushDevicesByUser(usrSeq int) error {
	_, err := r.DB.Exec(`DELETE FROM ALUMNI_PUSH_DEVICE WHERE USR_SEQ = ?`, usrSeq)
	return err
}

func (r *AuthRepository) IsMobileSessionActive(usrSeq int, sid string) (bool, error) {
	if usrSeq <= 0 || sid == "" {
		return false, nil
	}
	var count int
	err := r.DB.Get(&count, `
		SELECT COUNT(*)
		FROM ALUMNI_MOBILE_REFRESH_TOKEN
		WHERE USR_SEQ = ?
		  AND MRT_SID = ?
		  AND REVOKED_AT IS NULL
		  AND EXPIRES_AT > NOW()
	`, usrSeq, sid)
	return count > 0, err
}

func (r *AuthRepository) InsertAppleChallenge(challengeID string, nonceHash string, expiresAt time.Time) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_APPLE_NONCE_CHALLENGE
			(CHALLENGE_ID, NONCE_HASH, EXPIRES_AT, CREATED_AT)
		VALUES (?, ?, ?, NOW())
	`, challengeID, nonceHash, expiresAt)
	return err
}

func (r *AuthRepository) ConsumeAppleChallenge(challengeID string) (string, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var row struct {
		NonceHash  string       `db:"NONCE_HASH"`
		ExpiresAt  time.Time    `db:"EXPIRES_AT"`
		ConsumedAt sql.NullTime `db:"CONSUMED_AT"`
	}
	err = tx.Get(&row, `
		SELECT NONCE_HASH, EXPIRES_AT, CONSUMED_AT
		FROM ALUMNI_APPLE_NONCE_CHALLENGE
		WHERE CHALLENGE_ID = ?
		FOR UPDATE
	`, challengeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrChallengeInvalid
	}
	if err != nil {
		return "", err
	}
	if row.ConsumedAt.Valid || !row.ExpiresAt.After(time.Now()) {
		return "", ErrChallengeInvalid
	}
	if _, err := tx.Exec(`
		UPDATE ALUMNI_APPLE_NONCE_CHALLENGE
		SET CONSUMED_AT = NOW()
		WHERE CHALLENGE_ID = ? AND CONSUMED_AT IS NULL
	`, challengeID); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return row.NonceHash, nil
}

func (r *AuthRepository) ConsumeAppleAuthorizationCode(codeHash string) error {
	_, err := r.DB.Exec(`
		INSERT INTO ALUMNI_APPLE_CODE_REPLAY (CODE_HASH, CREATED_AT)
		VALUES (?, NOW())
	`, codeHash)
	if err == nil {
		return nil
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrAuthorizationReplay
	}
	return err
}

// UpdateProfilePhotoIfEmpty sets USR_PHOTO only when the column is currently NULL or empty.
// Used after an explicitly authenticated social-link flow so a provider avatar
// does not overwrite an existing member photo the user deliberately uploaded.
func (r *AuthRepository) UpdateProfilePhotoIfEmpty(usrSeq int, url string) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER
		SET USR_PHOTO = ?
		WHERE USR_SEQ = ? AND (USR_PHOTO IS NULL OR USR_PHOTO = '')
	`, url, usrSeq)
	return err
}

func (r *AuthRepository) DeleteLegacySessionsByUser(usrSeq int) error {
	_, err := r.DB.Exec(`DELETE FROM WEO_MEMBER_LOG WHERE USR_SEQ = ?`, usrSeq)
	return err
}

func (r *AuthRepository) DeleteLegacySession(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	_, err := r.DB.Exec(`DELETE FROM WEO_MEMBER_LOG WHERE SESSIONID = ?`, sessionID)
	return err
}

func (r *AuthRepository) InsertLoginLog(usrSeq int, sessionID string, ipAddr string, userAgent string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_MEMBER_LOG
			(USR_SEQ, LOG_DATE, REG_DATE, REG_IPADDR, SESSIONID, REG_AGENT)
		VALUES (?, NOW(), NOW(), ?, ?, ?)
	`, usrSeq, ipAddr, sessionID, userAgent)
	return err
}

func (r *AuthRepository) UpdateLastLogin(usrSeq int) error {
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER
		SET TOTAL_LOG_CNT = TOTAL_LOG_CNT + 1, LAST_LOG_DATE = NOW()
		WHERE USR_SEQ = ?
	`, usrSeq)
	return err
}

func (r *AuthRepository) FindMemberByLogin(usrID string, hashedPwd string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_ID = ? AND USR_PWD = ? AND USR_STATUS IN ('CCC', 'ZZZ')
		LIMIT 1
	`, usrID, hashedPwd)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByPhone(phone string) (*model.User, error) {
	canonicalPhone := model.NormalizePhoneNumber(phone).String()
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE (USR_PHONE = ? OR `+legacyCanonicalPhoneSQL+` = ?)
		  AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, canonicalPhone, canonicalPhone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindMemberByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_EMAIL = ? AND USR_STATUS IN ('BBB','CCC','ZZZ')
		LIMIT 1
	`, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) InsertMember(usrID, name, phone, fn, email, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic, profileImageURL string) (int, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	phoneClaimsEnabled, err := r.phoneClaimsEnabledTx(tx)
	if err != nil {
		return 0, err
	}
	if !phoneClaimsEnabled && len(model.NormalizePhoneNumber(phone)) > model.LegacyPhoneDigitsLimit {
		return 0, ErrInvalidPhone
	}

	phonePublic := usrPhonePublic
	if phonePublic == "" {
		phonePublic = "N"
	}
	emailPublic := usrEmailPublic
	if emailPublic == "" {
		emailPublic = "N"
	}
	var photo sql.NullString
	if profileImageURL != "" {
		photo = sql.NullString{String: profileImageURL, Valid: true}
	}
	result, err := tx.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT,
			USR_PHOTO, USR_DEPT, USR_JOB_CAT, USR_BIZ_NAME, USR_BIZ_DESC, USR_BIZ_ADDR,
			USR_POSITION, USR_PHONE_PUBLIC, USR_EMAIL_PUBLIC)
		VALUES (?, ?, ?, ?, ?, 'BBB', '', NOW(), 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, usrID, name, phone, fn, email, photo, fmDept, jobCat, bizName, bizDesc, bizAddr, position, phonePublic, emailPublic)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	usrSeq := int(id)
	if phoneClaimsEnabled {
		if err := claimPhoneTx(tx, phone, usrSeq); err != nil {
			return 0, err
		}
	}
	if err := insertAlumniVerificationCompanionTx(tx, usrSeq); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return usrSeq, nil
}

// UpdateMemberMergeFields updates fields editable from the merge signup form.
// USR_PHONE and USR_STATUS are intentionally untouched.
func (r *AuthRepository) UpdateMemberMergeFields(usrSeq int, name, email, fn, fmDept string, jobCat *int, bizName, bizDesc, bizAddr, position, usrPhonePublic, usrEmailPublic string) error {
	phonePublic := usrPhonePublic
	if phonePublic == "" {
		phonePublic = "N"
	}
	emailPublic := usrEmailPublic
	if emailPublic == "" {
		emailPublic = "N"
	}
	_, err := r.DB.Exec(`
		UPDATE WEO_MEMBER
		SET USR_NAME = ?, USR_EMAIL = ?, USR_JOB_CAT = ?, USR_BIZ_NAME = ?, USR_BIZ_DESC = ?,
		    USR_BIZ_ADDR = ?, USR_POSITION = ?, USR_PHONE_PUBLIC = ?, USR_EMAIL_PUBLIC = ?
		WHERE USR_SEQ = ?
	`, name, email, jobCat, bizName, bizDesc, bizAddr, position, phonePublic, emailPublic, usrSeq)
	return err
}

func (r *AuthRepository) GetMemberBySeq(usrSeq int) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		LIMIT 1
	`, usrSeq)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) CheckIDExists(usrID string) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_ID = ?`, usrID)
	return count > 0, err
}

func (r *AuthRepository) CheckPhoneExists(phone string) (bool, error) {
	canonicalPhone := model.NormalizePhoneNumber(phone).String()
	var count int
	err := r.DB.Get(
		&count,
		`SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_PHONE = ? OR `+legacyCanonicalPhoneSQL+` = ?`,
		canonicalPhone,
		canonicalPhone,
	)
	return count > 0, err
}

func (r *AuthRepository) CheckEmailExists(email string) (bool, error) {
	var count int
	err := r.DB.Get(&count, `SELECT COUNT(*) FROM WEO_MEMBER WHERE USR_EMAIL = ?`, email)
	return count > 0, err
}

func (r *AuthRepository) InsertMemberWithPwd(req model.RegisterRequest, hashedPwd string) (int, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	phoneClaimsEnabled, err := r.phoneClaimsEnabledTx(tx)
	if err != nil {
		return 0, err
	}
	if !phoneClaimsEnabled && len(model.NormalizePhoneNumber(req.Phone)) > model.LegacyPhoneDigitsLimit {
		return 0, ErrInvalidPhone
	}

	phonePublic := req.USRPhonePublic
	if phonePublic == "" {
		phonePublic = "N"
	}
	emailPublic := req.USREmailPublic
	if emailPublic == "" {
		emailPublic = "N"
	}
	result, err := tx.Exec(`
		INSERT INTO WEO_MEMBER (USR_ID, USR_NAME, USR_PHONE, USR_FN, USR_EMAIL, USR_STATUS, USR_PWD, REG_DATE, TOTAL_LOG_CNT,
			USR_NICK, USR_DEPT, USR_JOB_CAT, USR_BIZ_NAME, USR_BIZ_DESC, USR_BIZ_ADDR,
			USR_POSITION, USR_PHONE_PUBLIC, USR_EMAIL_PUBLIC)
		VALUES (?, ?, ?, ?, ?, 'BBB', ?, NOW(), 0, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.UsrID, req.Name, req.Phone, req.FN, req.Email, hashedPwd,
		req.Nick, req.FmDept, req.JobCat, req.BizName, req.BizDesc, req.BizAddr,
		req.Position, phonePublic, emailPublic)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	usrSeq := int(id)
	if phoneClaimsEnabled {
		if err := claimPhoneTx(tx, req.Phone, usrSeq); err != nil {
			return 0, err
		}
	}
	if err := insertAlumniVerificationCompanionTx(tx, usrSeq); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return usrSeq, nil
}

func claimPhoneTx(tx *sqlx.Tx, phone string, accountSeq int) error {
	canonicalPhone := model.NormalizePhoneNumber(phone)
	if !canonicalPhone.Valid() {
		return ErrInvalidPhone
	}
	_, err := tx.Exec(`
		INSERT INTO AUTH_PHONE_CLAIM (CANONICAL_PHONE, ACCOUNT_ID, CREATED_AT)
		VALUES (?, ?, NOW())
	`, canonicalPhone.String(), accountSeq)
	if err == nil {
		return nil
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrPhoneAlreadyClaimed
	}
	return err
}

func (r *AuthRepository) FindMemberByIDAndPwdAny(usrID, hashedPwd string) (*model.User, error) {
	var user model.User
	err := r.DB.Get(&user, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE USR_ID = ? AND USR_PWD = ?
		LIMIT 1
	`, usrID, hashedPwd)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
