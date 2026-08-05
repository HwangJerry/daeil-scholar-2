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
	minSupportedTokenVer   = 1
	maxSupportedTokenVer   = 1
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

// GenerateMobileJWT issues a strict mobile access token for iOS apps.
func (s *AuthService) GenerateMobileJWT(user *model.AuthUser, sid string) (string, error) {
	issuedAt := time.Now()
	expAt := issuedAt.Add(s.cfg.JWT.MaxAge)
	claims := jwt.MapClaims{
		"iss":    mobileTokenIssuer,
		"aud":    mobileTokenAudience,
		"sub":    strconv.Itoa(user.USRSeq),
		"name":   user.USRName,
		"status": user.USRStatus,
		"exp":    expAt.Unix(),
		"iat":    issuedAt.Unix(),
		"nbf":    issuedAt.Unix(),
		"typ":    mobileTokenTypeAccess,
		"ver":    minSupportedTokenVer,
		"sid":    sid,
		"jti":    s.GenerateSessionID(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWT.Secret))
}

// GenerateMobileRefreshJWT creates a refresh token pair member with same claims plus refresh type.
func (s *AuthService) GenerateMobileRefreshJWT(user *model.AuthUser, sid string) (string, string, time.Time, error) {
	issuedAt := time.Now()
	expAt := issuedAt.Add(s.cfg.JWT.MaxAge)
	jti := s.GenerateSessionID()
	claims := jwt.MapClaims{
		"iss":    mobileTokenIssuer,
		"aud":    mobileTokenAudience,
		"sub":    strconv.Itoa(user.USRSeq),
		"name":   user.USRName,
		"status": user.USRStatus,
		"exp":    expAt.Unix(),
		"iat":    issuedAt.Unix(),
		"nbf":    issuedAt.Unix(),
		"typ":    mobileTokenTypeRefresh,
		"ver":    minSupportedTokenVer,
		"sid":    sid,
		"jti":    jti,
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", "", time.Time{}, err
	}
	return refreshToken, jti, expAt, nil
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

// ValidateMobileAccessToken parses and validates a mobile access token from Authorization header.
// It enforces stricter claims than the legacy cookie-based ValidateJWT path.
func (s *AuthService) ValidateMobileAccessToken(tokenStr string) (*model.AuthUser, error) {
	claims := &mobileAccessClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}

	if claims.Issuer != mobileTokenIssuer {
		return nil, errors.New("invalid issuer")
	}
	if claims.Type != mobileTokenTypeAccess {
		return nil, errors.New("invalid token type")
	}
	if claims.Ver < minSupportedTokenVer || claims.Ver > maxSupportedTokenVer {
		return nil, errors.New("unsupported token version")
	}
	if !hasAudience(claims.Audience, mobileTokenAudience) {
		return nil, errors.New("invalid audience")
	}

	usrSeq, err := strconv.Atoi(claims.Subject)
	if err != nil || usrSeq == 0 {
		return nil, errors.New("invalid subject")
	}

	return &model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   claims.Name,
		USRStatus: claims.Status,
		SessionID: claims.SID,
	}, nil
}

// ValidateMobileRefreshToken parses and validates a mobile refresh token.
// It returns the user and session ID embedded in the claim.
func (s *AuthService) ValidateMobileRefreshToken(tokenStr string) (*model.AuthUser, string, string, error) {
	claims := &mobileRefreshClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, "", "", errors.New("invalid token")
	}

	if claims.Issuer != mobileTokenIssuer {
		return nil, "", "", errors.New("invalid issuer")
	}
	if claims.Type != mobileTokenTypeRefresh {
		return nil, "", "", errors.New("invalid token type")
	}
	if claims.Ver < minSupportedTokenVer || claims.Ver > maxSupportedTokenVer {
		return nil, "", "", errors.New("unsupported token version")
	}
	if !hasAudience(claims.Audience, mobileTokenAudience) {
		return nil, "", "", errors.New("invalid audience")
	}

	usrSeq, err := strconv.Atoi(claims.Subject)
	if err != nil || usrSeq == 0 {
		return nil, "", "", errors.New("invalid subject")
	}

	if claims.ID == "" {
		return nil, "", "", errors.New("missing token identifier")
	}

	return &model.AuthUser{
		USRSeq:    usrSeq,
		USRName:   claims.Name,
		USRStatus: claims.Status,
	}, claims.SessionID, claims.ID, nil
}

// ClassifyMobileToken inspects JWT metadata to determine if a bearer token
// belongs to mobile token family (based on issuer and audience) and returns
// the declared token type (`access`, `refresh`, ...). Time-based validations are
// intentionally skipped so expired/nbf-invalid tokens are still classified.
func (s *AuthService) ClassifyMobileToken(tokenStr string) (string, bool, error) {
	claims := &mobileAccessClaims{}
	parser := jwt.NewParser(
		jwt.WithoutClaimsValidation(),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	parsed, err := parser.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil || !parsed.Valid {
		return "", false, err
	}

	if claims.Issuer != mobileTokenIssuer {
		return "", false, nil
	}
	if !hasAudience(claims.Audience, mobileTokenAudience) {
		return "", false, nil
	}

	return claims.Type, true, nil
}

type mobileAccessClaims struct {
	jwt.RegisteredClaims
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"typ"`
	Ver    int    `json:"ver"`
	SID    string `json:"sid"`
}

type mobileRefreshClaims struct {
	jwt.RegisteredClaims
	Name      string `json:"name"`
	Status    string `json:"status"`
	Type      string `json:"typ"`
	Ver       int    `json:"ver"`
	SessionID string `json:"sid"`
}

func hasAudience(audiences jwt.ClaimStrings, expected string) bool {
	for _, audience := range audiences {
		if audience == expected {
			return true
		}
	}
	return false
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
