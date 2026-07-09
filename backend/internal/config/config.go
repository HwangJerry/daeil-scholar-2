package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server         ServerConfig
	DB             DBConfig
	Kakao          KakaoConfig
	JWT            JWTConfig
	Push           PushConfig
	Upload         UploadConfig
	EasyPay        EasyPayConfig
	SMTP           SMTPConfig
	DebugAgent     DebugAgentConfig
	PGAuditLogPath string
	Environment    string // "dev" exposes manual subscription billing trigger; "prod" hides it
	VisitIPSalt    string
}

// DebugAgentConfig holds settings for the external Debug Agent error pipeline.
// When Endpoint is empty the reporter is disabled (no-op) — main.go skips hook
// installation entirely so dev environments do not leak secrets or noise.
type DebugAgentConfig struct {
	Endpoint    string
	Project     string
	Secret      string
	Environment string
}

// Enabled reports whether the debug agent reporter should be installed.
func (c DebugAgentConfig) Enabled() bool {
	return c.Endpoint != ""
}

// SMTPConfig holds SMTP server settings for transactional email delivery.
type SMTPConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

type ServerConfig struct {
	Port            string
	AllowedOrigin   string
	SiteBaseURL     string
	ShutdownTimeout time.Duration
}

// IsSecure returns true when the allowed origin uses HTTPS.
func (c ServerConfig) IsSecure() bool {
	return strings.HasPrefix(c.AllowedOrigin, "https://")
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DSN returns the MySQL/MariaDB data source name.
func (c DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Asia%%2FSeoul",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

type KakaoConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type JWTConfig struct {
	Secret string
	MaxAge time.Duration
}

type UploadConfig struct {
	BasePath      string
	LegacyPath    string
	MaxFileSizeMB int
}

type EasyPayConfig struct {
	ImmediatelyMallID string
	ProfileMallID     string
	GatewayURL        string
	GatewayPort       string
	BinBase           string
	ReturnBaseURL     string
	AutoTrCd          string // 자동결제 transaction code (v1 confirmed value: "00101000")
}

type PushConfig struct {
	APNsKeyID             string
	APNSTeamID            string
	APNsBundleID          string
	APNsKeyPath           string
	APNsKeyValue          string
	APNsEnvironment       string
	APNsUseSandbox        bool
	APNsRequestTimeout    time.Duration
	FCMProjectID          string
	FCMCredentialsFile    string
	FCMCredentialsJSON    string
	FCMRequestTimeout     time.Duration
	OutboxBatchSize       int
	OutboxPollInterval    time.Duration
	OutboxMaxAttempts     int
	OutboxBaseBackoff     time.Duration
	OutboxMaxBackoff      time.Duration
	OutboxRecoveryTimeout time.Duration
	OutboxRequestTimeout  time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			AllowedOrigin:   getEnv("ALLOWED_ORIGIN", "http://localhost:8000"),
			SiteBaseURL:     getEnv("SITE_BASE_URL", "http://localhost:8000"),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "127.0.0.1"),
			Port:            getEnv("DB_PORT", "3306"),
			User:            getEnv("DB_USER", "root"),
			Password:        getEnv("DB_PASSWORD", ""),
			Name:            getEnv("DB_NAME", "alumni"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME", 3*time.Minute),
		},
		Kakao: KakaoConfig{
			ClientID:     getEnv("KAKAO_CLIENT_ID", ""),
			ClientSecret: getEnv("KAKAO_CLIENT_SECRET", ""),
			RedirectURI:  getEnv("KAKAO_REDIRECT_URI", "http://localhost:8000/api/auth/kakao/callback"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "change-me-in-production"),
			MaxAge: getDurationEnv("JWT_MAX_AGE", 24*time.Hour),
		},
		Push: PushConfig{
			APNsKeyID:             getEnvWithFallback("APNS_KEY_ID", "PUSH_APNS_KEY_ID", ""),
			APNSTeamID:            getEnvWithFallback("APNS_TEAM_ID", "PUSH_APNS_TEAM_ID", ""),
			APNsBundleID:          getEnvWithFallback("APNS_BUNDLE_ID", "PUSH_APNS_BUNDLE_ID", ""),
			APNsKeyPath:           getEnvWithFallback("APNS_PRIVATE_KEY_PATH", "PUSH_APNS_KEY_PATH", ""),
			APNsKeyValue:          getEnvWithFallback("APNS_PRIVATE_KEY", "PUSH_APNS_KEY_VALUE", ""),
			APNsEnvironment:       normalizePushEnvironment(getEnv("APNS_ENVIRONMENT", "")),
			APNsUseSandbox:        strings.EqualFold(getEnv("PUSH_APNS_USE_SANDBOX", "false"), "true"),
			APNsRequestTimeout:    getDurationEnvWithFallback("APNS_REQUEST_TIMEOUT", "PUSH_APNS_REQUEST_TIMEOUT", 5*time.Second),
			FCMProjectID:          getEnvWithFallback("FCM_PROJECT_ID", "FIREBASE_PROJECT_ID", ""),
			FCMCredentialsFile:    getEnvWithFallback("FCM_CREDENTIALS_FILE", "FIREBASE_SERVICE_ACCOUNT_FILE", getEnv("GOOGLE_APPLICATION_CREDENTIALS", "")),
			FCMCredentialsJSON:    getEnvWithFallback("FCM_CREDENTIALS_JSON", "FIREBASE_SERVICE_ACCOUNT_JSON", ""),
			FCMRequestTimeout:     getDurationEnvWithFallback("FCM_REQUEST_TIMEOUT", "FIREBASE_REQUEST_TIMEOUT", 5*time.Second),
			OutboxBatchSize:       getIntEnv("PUSH_OUTBOX_BATCH_SIZE", 50),
			OutboxPollInterval:    getDurationEnv("PUSH_OUTBOX_POLL_INTERVAL", 5*time.Second),
			OutboxMaxAttempts:     getIntEnv("PUSH_OUTBOX_MAX_ATTEMPTS", 8),
			OutboxBaseBackoff:     getDurationEnv("PUSH_OUTBOX_BASE_BACKOFF", 30*time.Second),
			OutboxMaxBackoff:      getDurationEnv("PUSH_OUTBOX_MAX_BACKOFF", 15*time.Minute),
			OutboxRecoveryTimeout: getDurationEnv("PUSH_OUTBOX_RECOVERY_TIMEOUT", 5*time.Minute),
			OutboxRequestTimeout:  getDurationEnv("PUSH_OUTBOX_REQUEST_TIMEOUT", 10*time.Second),
		},
		Upload: UploadConfig{
			BasePath:      getEnv("UPLOAD_BASE_PATH", "/var/www/uploads"),
			LegacyPath:    getEnv("UPLOAD_LEGACY_PATH", "/var/www/legacy/files"),
			MaxFileSizeMB: getIntEnv("UPLOAD_MAX_FILE_SIZE_MB", 10),
		},
		EasyPay: EasyPayConfig{
			ImmediatelyMallID: getEnv("EASYPAY_IMMEDIATELY_MALL_ID", "05542574"),
			ProfileMallID:     getEnv("EASYPAY_PROFILE_MALL_ID", "05543499"),
			GatewayURL:        getEnv("EASYPAY_GW_URL", "testgw.easypay.co.kr"),
			GatewayPort:       getEnv("EASYPAY_GW_PORT", "80"),
			BinBase:           getEnv("EASYPAY_BIN_BASE", "/var/www/html/_sys/payment"),
			ReturnBaseURL:     getEnv("EASYPAY_RETURN_BASE_URL", "http://localhost:8080"),
			AutoTrCd:          getEnv("EASYPAY_AUTO_TR_CD", "00101000"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnv("SMTP_PORT", "587"),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@dflh.kr"),
		},
		DebugAgent: DebugAgentConfig{
			Endpoint:    getEnv("DEBUG_AGENT_ENDPOINT", ""),
			Project:     getEnv("DEBUG_AGENT_PROJECT", ""),
			Secret:      getEnv("DEBUG_AGENT_SECRET", ""),
			Environment: getEnv("DEBUG_AGENT_ENVIRONMENT", getEnv("ENV", "dev")),
		},
		PGAuditLogPath: getEnv("PG_AUDIT_LOG_PATH", "/var/logs/pg/pg-audit.log"),
		Environment:    getEnv("ENV", "prod"),
		VisitIPSalt:    getEnv("VISIT_IP_SALT", ""),
	}
}

// stripInlineComment removes a trailing " # ..." from env values.
func stripInlineComment(v string) string {
	if i := strings.Index(v, " #"); i >= 0 {
		return strings.TrimRight(v[:i], " ")
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return stripInlineComment(v)
	}
	return fallback
}

func getEnvWithFallback(primaryKey, fallbackKey, fallback string) string {
	if v := getEnv(primaryKey, ""); v != "" {
		return v
	}
	return getEnv(fallbackKey, fallback)
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		v = stripInlineComment(v)
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		v = stripInlineComment(v)
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getDurationEnvWithFallback(primaryKey, fallbackKey string, fallback time.Duration) time.Duration {
	if v := os.Getenv(primaryKey); v != "" {
		v = stripInlineComment(v)
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return getDurationEnv(fallbackKey, fallback)
}

func normalizePushEnvironment(env string) string {
	env = strings.ToLower(strings.TrimSpace(env))
	switch env {
	case "sandbox", "development", "debug", "dev":
		return "sandbox"
	case "production", "prod", "release", "testflight", "appstore":
		return "production"
	default:
		if strings.EqualFold(getEnv("PUSH_APNS_USE_SANDBOX", "false"), "true") {
			return "sandbox"
		}
		return "production"
	}
}
