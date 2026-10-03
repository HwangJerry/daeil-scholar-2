package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

const (
	goldenPhone          = "01000000001"
	goldenCode           = "123456"
	goldenPassword       = "Synthetic-password-07!"
	goldenConsentVersion = "2026-09-27-test"
	goldenMemberID       = 100
)

func TestMain(m *testing.M) {
	os.Exit(mariadb.Run(m))
}

type goldenServer struct {
	db     *sqlx.DB
	server *httptest.Server
	client *http.Client
	deps   *deps
}

func newGoldenServer(t *testing.T) *goldenServer {
	t.Helper()
	db := mariadb.Start(t).NewDatabase(t, append(mariadb.ProdBaseline(t), postBaselineMigrations()...)...).DB
	seedGoldenIdentityReadiness(t, db)
	cfg := goldenConfig(t)
	if err := validateErasureRuntime(cfg); err != nil {
		t.Fatalf("erasure readiness: %v", err)
	}
	logger := zerolog.Nop()
	d, err := wireDeps(db, cfg, logger)
	if err != nil {
		t.Fatalf("wire real dependencies: %v", err)
	}
	t.Cleanup(func() {
		if err := d.pgAuditLog.Close(); err != nil {
			t.Errorf("close audit log: %v", err)
		}
	})
	router := registerRoutes(d.handlers, d.authService, d.cacheStore, []string{cfg.Server.AllowedOrigin}, cfg, logger, appVersionGate{
		policies: d.appUpdatePolicyService,
		observer: d.appClientBuildService,
	})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()
	client.Timeout = 10 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &goldenServer{db: db, server: server, client: client, deps: d}
}

// postBaselineMigrations are the migrations newer than the production baseline
// (077). They are applied in order on top of it so the harness serves the schema
// this branch deploys; drop an entry once a refreshed baseline includes it.
func postBaselineMigrations() []mariadb.SQL {
	return []mariadb.SQL{
		mariadb.File(filepath.Join("..", "..", "migrations", "078_create_notification_inbox_state.sql")),
		mariadb.File(filepath.Join("..", "..", "migrations", "079_create_feed_categories.sql")),
		mariadb.File(filepath.Join("..", "..", "migrations", "080_add_board_official_profile.sql")),
	}
}

func goldenConfig(t *testing.T) *config.Config {
	t.Helper()
	// Clear provider credentials and optional overrides before config.Load; never load env files.
	for _, key := range strings.Fields(`
		KAKAO_ADMIN_KEY KAKAO_CLIENT_ID KAKAO_CLIENT_SECRET KAKAO_ALLOWED_REDIRECT_URIS
		APPLE_TEAM_ID APPLE_KEY_ID APPLE_CLIENT_ID APPLE_BUNDLE_ID APPLE_PRIVATE_KEY APPLE_PRIVATE_KEY_PATH
		APPLE_ALLOWED_AUDIENCES APPLE_CHALLENGE_TTL APPLE_JWKS_CACHE_TTL SOCIAL_CREDENTIAL_ENCRYPTION_KEY
		FCM_PROJECT_ID FCM_CREDENTIALS_FILE APNS_TEAM_ID APNS_KEY_ID APNS_PRIVATE_KEY_FILE
		SMS_PROVIDER SMS_SENDER SMS_API_KEY SMS_USER_ID SMS_NCP_ACCESS_KEY SMS_NCP_SECRET_KEY SMS_NCP_SERVICE_ID
		SMTP_HOST SMTP_USER SMTP_PASSWORD SENTRY_AUTH_TOKEN SENTRY_ORG SENTRY_IOS_PROJECT SENTRY_ANDROID_PROJECT
		ACCOUNT_ERASURE_EXTERNAL_URL ACCOUNT_ERASURE_EXTERNAL_TOKEN DONATION_LEDGER_RETENTION_EVIDENCE
		MESSAGE_BLOCKED_PHRASES
	`) {
		t.Setenv(key, "")
	}
	// Resolve macOS /var -> /private/var so the real storage-readiness check passes.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{
		"ENV": "test", "SERVER_PORT": "0", "SHUTDOWN_TIMEOUT": "1s",
		"ALLOWED_ORIGIN": "https://alumni.example.test", "ADMIN_ORIGIN": "https://admin.example.test",
		"SITE_BASE_URL": "https://alumni.example.test",
		"DB_HOST":       "127.0.0.1", "DB_PORT": "1", "DB_USER": "synthetic", "DB_PASSWORD": "synthetic", "DB_NAME": "unused",
		"DB_MAX_OPEN_CONNS": "2", "DB_MAX_IDLE_CONNS": "2", "DB_CONN_MAX_LIFETIME": "1m", "DB_CONN_MAX_IDLE_TIME": "1m",
		"JWT_SECRET": "ts07-synthetic-jwt-secret-never-used-outside-tests", "JWT_MAX_AGE": "24h",
		"ACCESS_TOKEN_TTL": "15m", "REFRESH_TOKEN_TTL": "720h", "VISIT_IP_SALT": "synthetic-visit-salt",
		"SMS_REVIEW_TEST_PHONES": goldenPhone, "SMS_REVIEW_TEST_CODE": goldenCode,
		"PRIVACY_CONSENT_VERSION": goldenConsentVersion, "PRIVACY_CONSENT_ENFORCE": "true", "PUSH_ENABLED": "false",
		"PG_AUDIT_LOG_PATH": filepath.Join(root, "pg-audit.log"),
		"UPLOAD_BASE_PATH":  root, "UPLOAD_LEGACY_PATH": root, "UPLOAD_MAX_FILE_SIZE_MB": "1",
		"ACCOUNT_ERASURE_REQUESTS_ENABLED": "true", "ACCOUNT_ERASURE_WORKER_ENABLED": "false",
		"PRIVACY_RETENTION_ENABLED": "false", "ACCOUNT_ERASURE_TEST_USER_SEQ": "0",
		"ACCOUNT_ERASURE_WAIT_HOURS": "24", "ACCOUNT_ERASURE_EXTERNAL_MODE": "manual",
		"ACCOUNT_ERASURE_LEGACY_ROOT": root, "ACCOUNT_ERASURE_BACKUP_STATUS_PATH": filepath.Join(root, "backup-status.json"),
		"ACCOUNT_ERASURE_CONTEXT_KEY": strings.Repeat("11", 32), "DONATION_ARCHIVE_KEY": strings.Repeat("22", 32),
		"DONATION_LEDGER_RETENTION_CONFIRMED": "false", "DONATION_RECEIPT_ORIGINALS_SEPARATE": "false", "DONATION_LEDGER_YEAR_END_MONTH": "0",
		"EASYPAY_IMMEDIATELY_MALL_ID": "synthetic", "EASYPAY_PROFILE_MALL_ID": "synthetic",
		"EASYPAY_GW_URL": "127.0.0.1", "EASYPAY_GW_PORT": "1", "EASYPAY_BIN_BASE": root,
		"EASYPAY_RETURN_BASE_URL": "http://127.0.0.1:1", "EASYPAY_AUTO_TR_CD": "00101000",
		"KAKAO_REDIRECT_URI": "http://127.0.0.1:1/callback", "APPLE_JWKS_URL": "http://127.0.0.1:1/keys",
		"APPLE_TOKEN_URL": "http://127.0.0.1:1/token", "APPLE_REVOKE_URL": "http://127.0.0.1:1/revoke",
		"SMTP_PORT": "1", "SMTP_FROM": "synthetic@example.test",
	} {
		t.Setenv(key, value)
	}
	return config.Load()
}

func seedGoldenIdentityReadiness(t *testing.T, db *sqlx.DB) {
	t.Helper()
	// The baseline contains schema, not backfill data. Synthetic completion records
	// activate the same canonical password wiring without changing schema or services.
	goldenExec(t, db, `INSERT INTO AUTH_IDENTITY_MIGRATION_RUN
		(RUN_ID, STATUS, SOURCE_FINGERPRINT, CONFLICT_COUNT, STARTED_AT, COMPLETED_AT, UPDATED_AT)
		VALUES ('ts07-synthetic-run', 'APPLIED', REPEAT('a', 64), 0, NOW(), NOW(), NOW())`)
	goldenExec(t, db, `INSERT INTO AUTH_IDENTITY_MIGRATION_JOURNAL
		(RUN_ID, STEP_KEY, STATUS, STARTED_AT, APPLIED_AT, UPDATED_AT)
		VALUES ('ts07-synthetic-run', 'synthetic-empty-backfill', 'APPLIED', NOW(), NOW(), NOW())`)
	ready, err := repository.CanonicalPasswordWriteReady(db)
	if err != nil || !ready {
		t.Fatalf("canonical password readiness = %v, %v", ready, err)
	}
}

// goldenMemberSeed describes one synthetic approved member and its companion rows.
type goldenMemberSeed struct {
	seq                       int
	usrID, name, phone, email string
}

var defaultGoldenMember = goldenMemberSeed{
	seq: goldenMemberID, usrID: "golden_member", name: "합성 동문", phone: "01000000002", email: "member@example.test",
}

func seedGoldenMember(t *testing.T, db *sqlx.DB) {
	t.Helper()
	seedGoldenMemberAs(t, db, defaultGoldenMember)
}

func seedGoldenMemberAs(t *testing.T, db *sqlx.DB, member goldenMemberSeed) {
	t.Helper()
	credential, err := service.NewPasswordHasher().NewCredential(int64(member.seq), model.IdentityProviderLocalUsername, goldenPassword)
	if err != nil {
		t.Fatal(err)
	}
	goldenExec(t, db, `INSERT INTO WEO_MEMBER
		(USR_SEQ, USR_ID, USR_NAME, USR_PWD, USR_STATUS, USR_PHONE, USR_EMAIL, USR_FN, USR_DEPT, REG_DATE)
		VALUES (?, ?, ?, ?, 'CCC', ?, ?, 20, '영어', NOW())`,
		member.seq, member.usrID, member.name, service.MysqlNativePassword(goldenPassword), member.phone, member.email)
	goldenExec(t, db, `INSERT INTO AUTH_ACCOUNT_STATE (ACCOUNT_ID, STATUS, CREATED_AT, UPDATED_AT) VALUES (?, 'ACTIVE', NOW(), NOW())`, member.seq)
	goldenExec(t, db, `INSERT INTO AUTH_PHONE_CLAIM (CANONICAL_PHONE, ACCOUNT_ID, CREATED_AT) VALUES (?, ?, NOW())`, member.phone, member.seq)
	goldenExec(t, db, `INSERT INTO ALUMNI_VERIFICATION
		(USR_SEQ, STATUS, GRADUATION_YEAR, COHORT, DEPARTMENT, SUBMITTED_AT, REVIEWED_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, 'approved', 2007, '20', '영어', NOW(), NOW(), NOW(), NOW())`, member.seq)
	goldenExec(t, db, `INSERT INTO AUTH_IDENTITY
		(IDENTITY_ID, ACCOUNT_ID, PROVIDER, SUBJECT_KEY, STATUS, VERIFIED_AT, CREATED_AT, UPDATED_AT)
		VALUES (?, ?, 'LOCAL_USERNAME', ?, 'ACTIVE', NOW(), NOW(), NOW())`, member.seq, member.seq, member.usrID)
	goldenExec(t, db, `INSERT INTO AUTH_PASSWORD_CREDENTIAL
		(IDENTITY_ID, PROVIDER, ALGORITHM, PARAMETERS_TEXT, PASSWORD_HASH, STATUS, CREATED_AT, UPDATED_AT)
		VALUES (?, 'LOCAL_USERNAME', ?, ?, ?, 'ACTIVE', NOW(), NOW())`, member.seq, credential.Algorithm, credential.ParametersText, credential.PasswordHash)
}

func goldenExec(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func goldenCount(t *testing.T, db *sqlx.DB, want int, query string, args ...any) {
	t.Helper()
	var got int
	if err := db.Get(&got, query, args...); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("database count = %d, want %d (%s)", got, want, query)
	}
}

func (s *goldenServer) request(t *testing.T, method, path string, body any, accessToken, platform, build string, wantStatus int) []byte {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, s.server.URL+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	// Native clients send JSON and bearer auth, without browser Origin or cookies.
	req.Header.Set("Content-Type", "application/json")
	setGoldenClientHeaders(req, accessToken, platform, build)
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s: status %d, want %d: %s", method, path, resp.StatusCode, wantStatus, data)
	}
	if wantStatus != http.StatusNoContent && resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}
	return data
}

func setGoldenClientHeaders(req *http.Request, accessToken, platform, build string) {
	req.Header.Set("X-App-Platform", platform)
	req.Header.Set("X-App-Build", build)
	req.Header.Set("X-App-Version", "1.0.0")
	osVersion := "18.1"
	if platform == "android" {
		osVersion = "34"
	}
	req.Header.Set("X-App-OS-Version", osVersion)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
}

func decodeGolden[T any](t *testing.T, data []byte) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func (s *goldenServer) login(t *testing.T, platform string) ([]byte, model.MobileSession) {
	t.Helper()
	return s.loginAs(t, defaultGoldenMember, platform)
}

func (s *goldenServer) loginAs(t *testing.T, member goldenMemberSeed, platform string) ([]byte, model.MobileSession) {
	t.Helper()
	body := s.request(t, http.MethodPost, "/api/auth/mobile/login", map[string]string{
		"usrId": member.usrID, "password": goldenPassword,
	}, "", platform, "100", http.StatusOK)
	result := decodeGolden[model.SocialAuthResult](t, body)
	if result.Status != model.SocialAuthAuthenticated || result.Session == nil {
		t.Fatalf("login did not authenticate: %s", body)
	}
	assertGoldenSessionFor(t, *result.Session, member.seq)
	return body, *result.Session
}

func assertGoldenSession(t *testing.T, session model.MobileSession) {
	t.Helper()
	assertGoldenSessionFor(t, session, goldenMemberID)
}

func assertGoldenSessionFor(t *testing.T, session model.MobileSession, memberSeq int) {
	t.Helper()
	if session.User.USRSeq != memberSeq || session.User.Verification.Status != model.VerificationApproved ||
		session.AccessToken == "" || session.RefreshToken == "" || session.SID == "" || session.JTI == "" ||
		session.AccessIssuedAt <= 0 || session.AccessExpiresAt-session.AccessIssuedAt != 15*60 ||
		session.RefreshExpiresAt-session.AccessIssuedAt != 30*24*60*60 {
		t.Fatal("session must contain the approved member, tokens, identifiers and configured lifetimes")
	}
}
