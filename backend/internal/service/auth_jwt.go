// auth_jwt.go — JWT token generation, validation, and claim parsing
package service

import (
	"errors"
	"strconv"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/golang-jwt/jwt/v5"
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
