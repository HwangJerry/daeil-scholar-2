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
	Upload         UploadConfig
	EasyPay        EasyPayConfig
	SMTP           SMTPConfig
	PGAuditLogPath string
	Environment    string // "dev" exposes manual subscription billing trigger; "prod" hides it
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
		PGAuditLogPath: getEnv("PG_AUDIT_LOG_PATH", "/var/logs/pg/pg-audit.log"),
		Environment:    getEnv("ENV", "prod"),
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
