package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	ErrRefreshTokenInvalid         = errors.New("refresh token is invalid")
	ErrRefreshTokenReplay          = errors.New("refresh token replay detected")
	ErrChallengeInvalid            = errors.New("apple challenge is invalid")
	ErrAuthorizationReplay         = errors.New("authorization code replay detected")
	ErrPhoneAlreadyClaimed         = errors.New("canonical phone already claimed")
	ErrSocialIdentityAlreadyLinked = errors.New("social identity already linked")
	ErrLastLoginMethod             = errors.New("cannot disconnect the last login method")
	ErrInvalidPhone                = errors.New("canonical phone is invalid")
	ErrPhoneClaimsMigrating        = errors.New("phone claim migration is in progress")
)

// MariaDB 10.1 has no REGEXP_REPLACE. This compatibility expression is kept in
// one place for legacy rows that still contain hyphens or spaces. Exact
// canonical matches remain first in each predicate so migrated/current data can
// use the USR_PHONE index.
const legacyCanonicalPhoneSQL = "REPLACE(REPLACE(TRIM(USR_PHONE), '-', ''), ' ', '')"

type AuthRepository struct {
	DB                      *sqlx.DB
	canonicalIdentityReady  atomic.Bool
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

func NewAuthRepository(db *sqlx.DB) *AuthRepository {
	return &AuthRepository{DB: db}
}

func (r *AuthRepository) EnableCanonicalIdentityWrites() {
	r.canonicalIdentityReady.Store(true)
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
	canonicalIdentityEnabled := r.canonicalIdentityReady.Load()
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
	if canonicalIdentityEnabled {
		if err := insertActiveAccountStateTx(tx, fields.USRSeq); err != nil {
			return nil, err
		}
	}
	if err := insertAlumniVerificationCompanionTx(tx, fields.USRSeq); err != nil {
		return nil, err
	}
	if err := insertSocialConnectionTx(tx, fields); err != nil {
		return nil, err
	}
	if canonicalIdentityEnabled {
		if err := insertSocialIdentityTx(tx, fields); err != nil {
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
		return classifySocialIdentityInsertError(err)
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

func insertSocialIdentityTx(tx *sqlx.Tx, fields SocialAccountFields) error {
	provider, err := canonicalSocialIdentityProvider(fields.Provider)
	if err != nil {
		return err
	}

	// NORMALIZED_EMAIL is an EMAIL login authority, not provider metadata.
	// Social emails remain in WEO_MEMBER_SOCIAL.NMS_EMAIL so multiple login
	// methods can use the same email without claiming another identity's key.
	_, err = tx.Exec(`
		INSERT INTO AUTH_IDENTITY
			(ACCOUNT_ID, PROVIDER, SUBJECT_KEY, NORMALIZED_EMAIL, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, ?, ?, 'ACTIVE', NOW(), NOW(), NOW())
	`, fields.USRSeq, string(provider), fields.SocialID, nil)
	return classifySocialIdentityInsertError(err)
}

func classifySocialIdentityInsertError(err error) error {
	if err == nil {
		return nil
	}
	const duplicateEntryCode = 1062
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) || mysqlErr.Number != duplicateEntryCode {
		return err
	}
	// Duplicate email, account/provider and primary-key errors do not prove
	// that this provider subject belongs to another member.
	const keyMarker = " for key "
	keyOffset := strings.LastIndex(mysqlErr.Message, keyMarker)
	if keyOffset < 0 {
		return err
	}
	key := strings.Trim(mysqlErr.Message[keyOffset+len(keyMarker):], "'`")
	if qualifier := strings.LastIndex(key, "."); qualifier >= 0 {
		key = key[qualifier+1:]
	}
	switch key {
	case "UK_PROVIDER_SUBJECT", "UQ_AUTH_IDENTITY_PROVIDER_SUBJECT":
		return fmt.Errorf("%w: %w", ErrSocialIdentityAlreadyLinked, err)
	}
	return err
}

func canonicalSocialIdentityProvider(provider string) (model.IdentityProvider, error) {
	switch model.SocialProvider(provider) {
	case model.SocialProviderKakao:
		return model.IdentityProviderKakao, nil
	case model.SocialProviderApple:
		return model.IdentityProviderApple, nil
	default:
		return "", model.ErrUnsupportedIdentityProvider
	}
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

func (r *AuthRepository) DeleteSocialConnection(usrSeq int, provider string) error {
	return r.deleteSocialConnection(usrSeq, provider, true)
}

func (r *AuthRepository) ForceDeleteSocialConnection(usrSeq int, provider string) error {
	return r.deleteSocialConnection(usrSeq, provider, false)
}

func (r *AuthRepository) deleteSocialConnection(usrSeq int, provider string, enforceLastLoginMethod bool) error {
	tx, err := r.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var activeProviders []string
	if err := tx.Select(&activeProviders, `
		SELECT NMS_GATE
		FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_STATUS IN ('Y', 'ACTIVE')
		FOR UPDATE
	`, usrSeq); err != nil {
		return err
	}

	var hasPassword bool
	if err := tx.Get(&hasPassword, `
		SELECT CASE
			WHEN COALESCE(TRIM(USR_PWD), '') <> '' THEN 1
			ELSE 0
		END
		FROM WEO_MEMBER
		WHERE USR_SEQ = ?
		FOR UPDATE
	`, usrSeq); err != nil {
		return err
	}

	targetIsActive := false
	for _, activeProvider := range activeProviders {
		if activeProvider == provider {
			targetIsActive = true
			break
		}
	}
	if enforceLastLoginMethod && targetIsActive && !hasPassword && len(activeProviders) == 1 {
		return ErrLastLoginMethod
	}

	if _, err := tx.Exec(`
		DELETE FROM WEO_MEMBER_SOCIAL
		WHERE USR_SEQ = ? AND NMS_GATE = ?
	`, usrSeq, provider); err != nil {
		return err
	}
	if r.canonicalIdentityReady.Load() {
		canonicalProvider, err := canonicalSocialIdentityProvider(provider)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
			UPDATE AUTH_IDENTITY
			SET STATUS = 'REVOKED', REVOKED_AT = NOW(), UPDATED_AT = NOW()
			WHERE ACCOUNT_ID = ? AND PROVIDER = ? AND STATUS = 'ACTIVE'
		`, usrSeq, string(canonicalProvider)); err != nil {
			return err
		}
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
	_, err := updateMemberStatusAndAdminRole(r.DB, usrSeq, status)
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
	_, err := r.DB.Exec(`DELETE FROM ALUMNI_MOBILE_DEVICE_TOKEN WHERE USR_SEQ = ?`, usrSeq)
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
