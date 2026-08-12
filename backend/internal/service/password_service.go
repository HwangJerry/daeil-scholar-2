// password_service.go — Canonical Argon2id and legacy MySQL password verification.
package service

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dflh-saf/backend/internal/model"
	"golang.org/x/crypto/argon2"
)

const (
	argon2idVersion                    = argon2.Version
	argon2idSaltLength                 = 16
	argon2idKeyLength           uint32 = 32
	maxArgon2idMemoryKiB        uint32 = 32 * 1024
	maxArgon2idIterations       uint32 = 10
	maxArgon2idParallelism      uint8  = 16
	maxConcurrentPasswordHashes        = 1
)

var (
	ErrInvalidPasswordHash    = errors.New("invalid password hash")
	passwordHashSemaphore     = make(chan struct{}, maxConcurrentPasswordHashes)
	defaultArgon2idParameters = Argon2idParameters{
		MemoryKiB:   19 * 1024,
		Iterations:  2,
		Parallelism: 1,
		KeyLength:   argon2idKeyLength,
	}
)

// Argon2idParameters controls canonical password hashing cost and output size.
type Argon2idParameters struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
}

// PasswordVerification describes both validity and any required hash transition.
type PasswordVerification struct {
	Valid       bool
	NeedsRehash bool
	UsedLegacy  bool
}

// PasswordHasher hashes new passwords and verifies canonical or legacy hashes.
type PasswordHasher struct {
	parameters Argon2idParameters
	random     io.Reader
}

func NewPasswordHasher() *PasswordHasher {
	return newPasswordHasher(defaultArgon2idParameters, rand.Reader)
}

func newPasswordHasher(parameters Argon2idParameters, random io.Reader) *PasswordHasher {
	return &PasswordHasher{parameters: parameters, random: random}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	if err := validateArgon2idParameters(h.parameters); err != nil {
		return "", err
	}
	salt := make([]byte, argon2idSaltLength)
	if _, err := io.ReadFull(h.random, salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}

	key := deriveArgon2idKey(password, salt, h.parameters)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion,
		h.parameters.MemoryKiB,
		h.parameters.Iterations,
		h.parameters.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (h *PasswordHasher) NewCredential(identityID int64, provider model.IdentityProvider, password string) (model.PasswordCredential, error) {
	hash, err := h.Hash(password)
	if err != nil {
		return model.PasswordCredential{}, err
	}
	parametersText := fmt.Sprintf(
		"m=%d,t=%d,p=%d",
		h.parameters.MemoryKiB,
		h.parameters.Iterations,
		h.parameters.Parallelism,
	)
	return model.PasswordCredential{
		IdentityID:     identityID,
		Provider:       provider,
		Algorithm:      model.PasswordAlgorithmArgon2id,
		ParametersText: &parametersText,
		PasswordHash:   hash,
		Status:         model.PasswordCredentialStatusActive,
	}, nil
}

func (h *PasswordHasher) Verify(password, encodedHash string) (PasswordVerification, error) {
	if strings.HasPrefix(encodedHash, "*") {
		return verifyMysqlNativePassword(password, encodedHash)
	}

	parameters, salt, expectedKey, err := parseArgon2idHash(encodedHash)
	if err != nil {
		return PasswordVerification{}, err
	}
	actualKey := deriveArgon2idKey(password, salt, parameters)
	valid := subtle.ConstantTimeCompare(actualKey, expectedKey) == 1
	return PasswordVerification{
		Valid:       valid,
		NeedsRehash: valid && (parameters != h.parameters || len(salt) != argon2idSaltLength),
	}, nil
}

func (h *PasswordHasher) VerifyCredential(algorithm model.PasswordAlgorithm, password, encodedHash string) (PasswordVerification, error) {
	switch algorithm {
	case model.PasswordAlgorithmArgon2id:
		if !strings.HasPrefix(encodedHash, "$argon2id$") {
			return PasswordVerification{}, ErrInvalidPasswordHash
		}
		return h.Verify(password, encodedHash)
	case model.PasswordAlgorithmMysqlNativePassword:
		if !strings.HasPrefix(encodedHash, "*") {
			return PasswordVerification{}, ErrInvalidPasswordHash
		}
		return verifyMysqlNativePassword(password, encodedHash)
	default:
		return PasswordVerification{}, ErrInvalidPasswordHash
	}
}

func deriveArgon2idKey(password string, salt []byte, parameters Argon2idParameters) []byte {
	passwordHashSemaphore <- struct{}{}
	defer func() { <-passwordHashSemaphore }()
	return argon2.IDKey(
		[]byte(password),
		salt,
		parameters.Iterations,
		parameters.MemoryKiB,
		parameters.Parallelism,
		parameters.KeyLength,
	)
}

func parseArgon2idHash(encodedHash string) (Argon2idParameters, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != fmt.Sprintf("v=%d", argon2idVersion) {
		return Argon2idParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	parameters := Argon2idParameters{}
	parsed, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &parameters.MemoryKiB, &parameters.Iterations, &parameters.Parallelism)
	if err != nil || parsed != 3 || parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", parameters.MemoryKiB, parameters.Iterations, parameters.Parallelism) {
		return Argon2idParameters{}, nil, nil, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return Argon2idParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	expectedKey, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(expectedKey) < 16 || len(expectedKey) > 64 {
		return Argon2idParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	parameters.KeyLength = uint32(len(expectedKey))
	if err := validateArgon2idParameters(parameters); err != nil {
		return Argon2idParameters{}, nil, nil, ErrInvalidPasswordHash
	}
	return parameters, salt, expectedKey, nil
}

func validateArgon2idParameters(parameters Argon2idParameters) error {
	invalidMemory := parameters.MemoryKiB < 8*uint32(parameters.Parallelism) || parameters.MemoryKiB > maxArgon2idMemoryKiB
	invalidIterations := parameters.Iterations == 0 || parameters.Iterations > maxArgon2idIterations
	invalidParallelism := parameters.Parallelism == 0 || parameters.Parallelism > maxArgon2idParallelism
	invalidKeyLength := parameters.KeyLength < 16 || parameters.KeyLength > 64
	if invalidMemory || invalidIterations || invalidParallelism || invalidKeyLength {
		return ErrInvalidPasswordHash
	}
	return nil
}

func verifyMysqlNativePassword(password, encodedHash string) (PasswordVerification, error) {
	if len(encodedHash) != 41 {
		return PasswordVerification{}, ErrInvalidPasswordHash
	}
	if _, err := hex.DecodeString(encodedHash[1:]); err != nil {
		return PasswordVerification{}, ErrInvalidPasswordHash
	}
	expectedHash := MysqlNativePassword(password)
	valid := subtle.ConstantTimeCompare([]byte(expectedHash), []byte(strings.ToUpper(encodedHash))) == 1
	return PasswordVerification{Valid: valid, NeedsRehash: valid, UsedLegacy: true}, nil
}

// MysqlNativePassword computes MySQL's native password hash: "*" + upper(hex(sha1(sha1(password)))).
// This matches the format stored in WEO_MEMBER.USR_PWD by the legacy PHP system.
func MysqlNativePassword(password string) string {
	first := sha1.Sum([]byte(password))
	second := sha1.Sum(first[:])
	return "*" + strings.ToUpper(fmt.Sprintf("%x", second))
}
