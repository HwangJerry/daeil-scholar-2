// auth_jwt.go — JWT token generation, validation, and claim parsing
package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

const (
	mobileTokenIssuer      = "dflh-saf-v2-backend"
	mobileTokenAudience    = "dflh-saf-v2-mobile"
	mobileTokenTypeAccess  = "access"
	mobileTokenTypeRefresh = "refresh"
	mobileTokenVersion     = 1
)

// GenerateJWT creates a signed JWT for the given authenticated user.
func (s *AuthService) GenerateJWT(user *model.AuthUser) (string, error) {
	claims := jwt.MapClaims{
		"sub":    strconv.Itoa(user.USRSeq),
		"name":   user.USRName,
		"status": user.USRStatus,
		"exp":    time.Now().Add(s.cfg.JWT.MaxAge).Unix(),
		"iat":    time.Now().Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWT.Secret))
}

// ValidateJWT parses and validates a JWT string, returning the authenticated user.
func (s *AuthService) ValidateJWT(tokenStr string) (*model.AuthUser, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	usrSeq, err := parseSubject(claims["sub"])
	if err != nil || usrSeq == 0 {
		return nil, errors.New("invalid subject")
	}
	return &model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   parseString(claims["name"]),
		USRStatus: parseString(claims["status"]),
	}, nil
}

func (s *AuthService) generateMobileAccessToken(user *model.AuthUser, sid string, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(s.cfg.JWT.AccessTokenTTL)
	claims := jwt.MapClaims{
		"iss":    mobileTokenIssuer,
		"aud":    mobileTokenAudience,
		"sub":    strconv.Itoa(user.USRSeq),
		"name":   user.USRName,
		"status": user.USRStatus,
		"exp":    expiresAt.Unix(),
		"iat":    now.Unix(),
		"nbf":    now.Unix(),
		"typ":    mobileTokenTypeAccess,
		"ver":    mobileTokenVersion,
		"sid":    sid,
		"jti":    s.GenerateSessionID(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWT.Secret))
	return token, expiresAt, err
}

func (s *AuthService) generateMobileRefreshToken(user *model.AuthUser, sid string, now time.Time) (string, string, time.Time, error) {
	expiresAt := now.Add(s.cfg.JWT.RefreshTokenTTL)
	jti := s.GenerateSessionID()
	claims := jwt.MapClaims{
		"iss":    mobileTokenIssuer,
		"aud":    mobileTokenAudience,
		"sub":    strconv.Itoa(user.USRSeq),
		"name":   user.USRName,
		"status": user.USRStatus,
		"exp":    expiresAt.Unix(),
		"iat":    now.Unix(),
		"nbf":    now.Unix(),
		"typ":    mobileTokenTypeRefresh,
		"ver":    mobileTokenVersion,
		"sid":    sid,
		"jti":    jti,
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWT.Secret))
	return token, jti, expiresAt, err
}

func (s *AuthService) ValidateMobileAccessToken(tokenStr string) (*model.AuthUser, error) {
	claims, err := s.validateMobileToken(tokenStr, mobileTokenTypeAccess)
	if err != nil {
		return nil, err
	}
	usrSeq, err := strconv.Atoi(claims.Subject)
	if err != nil || usrSeq <= 0 {
		return nil, errors.New("invalid subject")
	}
	return &model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   claims.Name,
		USRStatus: claims.Status,
		SessionID: claims.SessionID,
	}, nil
}

func (s *AuthService) ValidateMobileRefreshToken(tokenStr string) (*model.AuthUser, string, error) {
	claims, err := s.validateMobileToken(tokenStr, mobileTokenTypeRefresh)
	if err != nil {
		return nil, "", err
	}
	usrSeq, err := strconv.Atoi(claims.Subject)
	if err != nil || usrSeq <= 0 || claims.ID == "" || claims.SessionID == "" {
		return nil, "", errors.New("invalid refresh claims")
	}
	return &model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   claims.Name,
		USRStatus: claims.Status,
		SessionID: claims.SessionID,
	}, claims.ID, nil
}

func (s *AuthService) validateMobileToken(tokenStr string, expectedType string) (*mobileClaims, error) {
	claims := &mobileClaims{}
	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(s.cfg.JWT.Secret), nil
		},
		jwt.WithAudience(mobileTokenAudience),
		jwt.WithIssuer(mobileTokenIssuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid mobile token")
	}
	if claims.Type != expectedType || claims.Version != mobileTokenVersion {
		return nil, errors.New("invalid mobile token metadata")
	}
	return claims, nil
}

type mobileClaims struct {
	jwt.RegisteredClaims
	Name      string `json:"name"`
	Status    string `json:"status"`
	Type      string `json:"typ"`
	Version   int    `json:"ver"`
	SessionID string `json:"sid"`
}

// parseSubject extracts an integer user ID from a JWT claim value.
func parseSubject(value interface{}) (int, error) {
	switch v := value.(type) {
	case string:
		return strconv.Atoi(v)
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	default:
		return 0, errors.New("invalid subject")
	}
}

// parseString safely extracts a string from a JWT claim value.
func parseString(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}
