// phone_verification_service.go — Issues and checks one-time SMS codes proving a
// signup applicant controls the phone number they entered.
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

const (
	phoneCodeDigits        = 6
	phoneCodeExpiry        = 5 * time.Minute
	phoneGrantExpiry       = 30 * time.Minute
	phoneCodeMaxAttempts   = 5
	phoneRequestWindow     = time.Hour
	phoneRequestsPerWindow = 5
	phoneVerificationIDLen = 16 // hex-encoded to the CHAR(32) column
	phoneGrantTokenBytes   = 32
)

var (
	// ErrPhoneVerificationNotFound covers an unknown, expired, or already-verified handle.
	ErrPhoneVerificationNotFound = errors.New("phone verification not found or expired")
	// ErrPhoneVerificationCodeMismatch is returned while attempts remain.
	ErrPhoneVerificationCodeMismatch = errors.New("phone verification code mismatch")
	// ErrPhoneVerificationAttemptsExceeded retires the verification; the user must request a new code.
	ErrPhoneVerificationAttemptsExceeded = errors.New("phone verification attempts exceeded")
	// ErrPhoneVerificationThrottled means too many codes were requested for this number.
	ErrPhoneVerificationThrottled = errors.New("phone verification requests throttled")
	// ErrPhoneNotVerified is returned when registration presents no usable grant token.
	ErrPhoneNotVerified = errors.New("phone number is not verified")
)

type phoneVerificationStore interface {
	InsertVerification(id, phone, codeHash string, expiresAt time.Time) error
	CountRecentRequests(phone string, since time.Time) (int, error)
	FindPendingVerification(id string) (*model.PhoneVerification, error)
	IncrementAttempts(seq int64) (int, error)
	ExpireVerification(seq int64) error
	MarkVerified(seq int64, grantTokenHash string, grantExpiresAt time.Time) error
	FindUsableGrant(grantTokenHash string) (string, error)
	ConsumeGrant(grantTokenHash string) (string, error)
}

// PhoneVerificationService owns the SMS code lifecycle: issue, confirm, and the
// one-shot grant that registration consumes.
type PhoneVerificationService struct {
	store  phoneVerificationStore
	sender SMSSender
	logger zerolog.Logger
}

// NewPhoneVerificationService creates a PhoneVerificationService.
func NewPhoneVerificationService(store phoneVerificationStore, sender SMSSender, logger zerolog.Logger) *PhoneVerificationService {
	return &PhoneVerificationService{store: store, sender: sender, logger: logger}
}

// RequestCode validates the number, throttles resends, and dispatches a fresh code.
func (s *PhoneVerificationService) RequestCode(phone string) (*model.PhoneVerificationRequestResult, error) {
	canonicalPhone := model.NormalizePhoneNumber(phone)
	if !canonicalPhone.Valid() {
		return nil, ErrInvalidPhone
	}

	recent, err := s.store.CountRecentRequests(canonicalPhone.String(), time.Now().Add(-phoneRequestWindow))
	if err != nil {
		return nil, err
	}
	if recent >= phoneRequestsPerWindow {
		return nil, ErrPhoneVerificationThrottled
	}

	verificationID, err := randomHex(phoneVerificationIDLen)
	if err != nil {
		return nil, err
	}
	code, err := randomNumericCode(phoneCodeDigits)
	if err != nil {
		return nil, err
	}

	if err := s.store.InsertVerification(verificationID, canonicalPhone.String(), hashSecret(code), time.Now().Add(phoneCodeExpiry)); err != nil {
		return nil, err
	}
	if err := s.sender.Send(model.SMSMessage{
		To:   canonicalPhone.String(),
		Body: fmt.Sprintf("[대일외고장학회] 인증번호 %s 를 입력해 주세요.", code),
	}); err != nil {
		// The record stays; the user can retry within the throttle budget.
		return nil, err
	}

	return &model.PhoneVerificationRequestResult{
		VerificationID: verificationID,
		ExpiresInSec:   int(phoneCodeExpiry.Seconds()),
	}, nil
}

// ConfirmCode checks a submitted code and, on success, issues the grant token that
// registration must present.
func (s *PhoneVerificationService) ConfirmCode(verificationID, code string) (*model.PhoneVerificationConfirmResult, error) {
	record, err := s.store.FindPendingVerification(verificationID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrPhoneVerificationNotFound
	}
	if record.Attempts >= phoneCodeMaxAttempts {
		return nil, ErrPhoneVerificationAttemptsExceeded
	}

	if hashSecret(code) != record.CodeHash {
		attempts, incrementErr := s.store.IncrementAttempts(record.APVSeq)
		if incrementErr != nil {
			return nil, incrementErr
		}
		if attempts >= phoneCodeMaxAttempts {
			if expireErr := s.store.ExpireVerification(record.APVSeq); expireErr != nil {
				return nil, expireErr
			}
			return nil, ErrPhoneVerificationAttemptsExceeded
		}
		return nil, ErrPhoneVerificationCodeMismatch
	}

	grantToken, err := randomHex(phoneGrantTokenBytes)
	if err != nil {
		return nil, err
	}
	if err := s.store.MarkVerified(record.APVSeq, hashSecret(grantToken), time.Now().Add(phoneGrantExpiry)); err != nil {
		return nil, err
	}

	return &model.PhoneVerificationConfirmResult{
		VerificationToken: grantToken,
		ExpiresInSec:      int(phoneGrantExpiry.Seconds()),
	}, nil
}

// AssertPhoneVerified checks that a usable grant exists for the phone number without
// spending it. Callers confirm eligibility with this before creating the account, then
// spend the grant with ConsumeGrantForPhone once the account exists.
func (s *PhoneVerificationService) AssertPhoneVerified(grantToken, phone string) error {
	_, err := s.grantSubject(grantToken, phone, s.store.FindUsableGrant)
	return err
}

// ConsumeGrantForPhone spends a grant token and confirms it was issued for the phone
// number being registered. It succeeds at most once per token.
func (s *PhoneVerificationService) ConsumeGrantForPhone(grantToken, phone string) error {
	_, err := s.grantSubject(grantToken, phone, s.store.ConsumeGrant)
	return err
}

// grantSubject resolves a grant token through the given lookup and confirms it was
// issued for the supplied phone number.
func (s *PhoneVerificationService) grantSubject(grantToken, phone string, lookup func(string) (string, error)) (string, error) {
	if grantToken == "" {
		return "", ErrPhoneNotVerified
	}
	canonicalPhone := model.NormalizePhoneNumber(phone)
	if !canonicalPhone.Valid() {
		return "", ErrInvalidPhone
	}
	grantedPhone, err := lookup(hashSecret(grantToken))
	if err != nil {
		return "", err
	}
	if grantedPhone == "" || grantedPhone != canonicalPhone.String() {
		return "", ErrPhoneNotVerified
	}
	return grantedPhone, nil
}

func hashSecret(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func randomHex(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func randomNumericCode(digits int) (string, error) {
	code := make([]byte, digits)
	for index := range code {
		digit, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[index] = byte('0' + digit.Int64())
	}
	return string(code), nil
}
