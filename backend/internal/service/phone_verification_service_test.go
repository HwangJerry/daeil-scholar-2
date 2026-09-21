// phone_verification_service_test.go — Guards the SMS verification contract: codes are
// never stored or sent in the clear beyond the SMS body, attempts are bounded, and a
// grant proves the exact phone number being registered.
package service_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/rs/zerolog"
)

type fakePhoneVerificationStore struct {
	records      map[string]*model.PhoneVerification
	grants       map[string]string
	consumed     map[string]bool
	recentCount  int
	insertedHash string
}

func newFakeStore() *fakePhoneVerificationStore {
	return &fakePhoneVerificationStore{
		records:  map[string]*model.PhoneVerification{},
		grants:   map[string]string{},
		consumed: map[string]bool{},
	}
}

func (f *fakePhoneVerificationStore) InsertVerification(id, phone, codeHash string, expiresAt time.Time) error {
	f.insertedHash = codeHash
	f.records[id] = &model.PhoneVerification{
		APVSeq: int64(len(f.records) + 1), APVID: id, Phone: phone,
		CodeHash: codeHash, ExpiresAt: expiresAt,
	}
	return nil
}

func (f *fakePhoneVerificationStore) CountRecentRequests(string, time.Time) (int, error) {
	return f.recentCount, nil
}

func (f *fakePhoneVerificationStore) FindPendingVerification(id string) (*model.PhoneVerification, error) {
	record, ok := f.records[id]
	if !ok || record.VerifiedYN == "Y" || record.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}
	return record, nil
}

func (f *fakePhoneVerificationStore) IncrementAttempts(seq int64) (int, error) {
	for _, record := range f.records {
		if record.APVSeq == seq {
			record.Attempts++
			return record.Attempts, nil
		}
	}
	return 0, errors.New("not found")
}

func (f *fakePhoneVerificationStore) ExpireVerification(seq int64) error {
	for _, record := range f.records {
		if record.APVSeq == seq {
			record.ExpiresAt = time.Now().Add(-time.Minute)
		}
	}
	return nil
}

func (f *fakePhoneVerificationStore) MarkVerified(seq int64, grantTokenHash string, _ time.Time) error {
	for _, record := range f.records {
		if record.APVSeq == seq {
			record.VerifiedYN = "Y"
			f.grants[grantTokenHash] = record.Phone
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakePhoneVerificationStore) FindUsableGrant(grantTokenHash string) (string, error) {
	if f.consumed[grantTokenHash] {
		return "", nil
	}
	return f.grants[grantTokenHash], nil
}

func (f *fakePhoneVerificationStore) ConsumeGrant(grantTokenHash string) (string, error) {
	if f.consumed[grantTokenHash] {
		return "", nil
	}
	phone, ok := f.grants[grantTokenHash]
	if !ok {
		return "", nil
	}
	f.consumed[grantTokenHash] = true
	return phone, nil
}

type capturingSMSSender struct {
	sent []model.SMSMessage
}

func (c *capturingSMSSender) Send(msg model.SMSMessage) error {
	c.sent = append(c.sent, msg)
	return nil
}

// sentCode pulls the 6-digit code out of the captured SMS body.
func (c *capturingSMSSender) sentCode(t *testing.T) string {
	t.Helper()
	if len(c.sent) == 0 {
		t.Fatal("no SMS was sent")
	}
	body := c.sent[len(c.sent)-1].Body
	var digits strings.Builder
	for _, character := range body {
		if character >= '0' && character <= '9' {
			digits.WriteRune(character)
		}
	}
	return digits.String()
}

func newServiceUnderTest() (*service.PhoneVerificationService, *fakePhoneVerificationStore, *capturingSMSSender) {
	store := newFakeStore()
	sender := &capturingSMSSender{}
	templates := service.NewTestNotificationTemplateService(nil)
	return service.NewPhoneVerificationService(store, sender, templates, zerolog.Nop()), store, sender
}

func TestRequestCodeStoresOnlyTheCodeHash(t *testing.T) {
	svc, store, sender := newServiceUnderTest()

	if _, err := svc.RequestCode("010-1234-5678"); err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}

	code := sender.sentCode(t)
	if code == "" {
		t.Fatal("SMS body carried no verification code")
	}
	if strings.Contains(store.insertedHash, code) {
		t.Error("verification code was persisted in recoverable form")
	}
	if len(store.insertedHash) != 64 {
		t.Errorf("stored code hash length = %d, want 64 (sha256 hex)", len(store.insertedHash))
	}
}

func TestRequestCodeRejectsUnusablePhoneNumber(t *testing.T) {
	svc, _, _ := newServiceUnderTest()

	if _, err := svc.RequestCode("12"); !errors.Is(err, service.ErrInvalidPhone) {
		t.Fatalf("RequestCode() error = %v, want ErrInvalidPhone", err)
	}
}

func TestRequestCodeThrottlesRepeatedRequests(t *testing.T) {
	svc, store, _ := newServiceUnderTest()
	store.recentCount = 5

	if _, err := svc.RequestCode("01012345678"); !errors.Is(err, service.ErrPhoneVerificationThrottled) {
		t.Fatalf("RequestCode() error = %v, want ErrPhoneVerificationThrottled", err)
	}
}

func TestConfirmCodeIssuesGrantForTheVerifiedNumber(t *testing.T) {
	svc, _, sender := newServiceUnderTest()
	requested, err := svc.RequestCode("01012345678")
	if err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}

	confirmed, err := svc.ConfirmCode(requested.VerificationID, sender.sentCode(t))
	if err != nil {
		t.Fatalf("ConfirmCode() error = %v", err)
	}
	if confirmed.VerificationToken == "" {
		t.Fatal("ConfirmCode() returned an empty grant token")
	}
	if err := svc.AssertPhoneVerified(confirmed.VerificationToken, "010-1234-5678"); err != nil {
		t.Errorf("AssertPhoneVerified() error = %v, want nil for the same number in a different format", err)
	}
}

func TestGrantDoesNotVerifyADifferentNumber(t *testing.T) {
	svc, _, sender := newServiceUnderTest()
	requested, _ := svc.RequestCode("01012345678")
	confirmed, err := svc.ConfirmCode(requested.VerificationID, sender.sentCode(t))
	if err != nil {
		t.Fatalf("ConfirmCode() error = %v", err)
	}

	if err := svc.AssertPhoneVerified(confirmed.VerificationToken, "01099998888"); !errors.Is(err, service.ErrPhoneNotVerified) {
		t.Fatalf("AssertPhoneVerified() error = %v, want ErrPhoneNotVerified", err)
	}
}

func TestGrantIsSpentExactlyOnce(t *testing.T) {
	svc, _, sender := newServiceUnderTest()
	requested, _ := svc.RequestCode("01012345678")
	confirmed, err := svc.ConfirmCode(requested.VerificationID, sender.sentCode(t))
	if err != nil {
		t.Fatalf("ConfirmCode() error = %v", err)
	}

	if err := svc.ConsumeGrantForPhone(confirmed.VerificationToken, "01012345678"); err != nil {
		t.Fatalf("first ConsumeGrantForPhone() error = %v", err)
	}
	if err := svc.ConsumeGrantForPhone(confirmed.VerificationToken, "01012345678"); !errors.Is(err, service.ErrPhoneNotVerified) {
		t.Fatalf("second ConsumeGrantForPhone() error = %v, want ErrPhoneNotVerified", err)
	}
}

func TestRegistrationWithoutAGrantIsRejected(t *testing.T) {
	svc, _, _ := newServiceUnderTest()

	if err := svc.AssertPhoneVerified("", "01012345678"); !errors.Is(err, service.ErrPhoneNotVerified) {
		t.Fatalf("AssertPhoneVerified() error = %v, want ErrPhoneNotVerified", err)
	}
	if err := svc.AssertPhoneVerified("forged-token", "01012345678"); !errors.Is(err, service.ErrPhoneNotVerified) {
		t.Fatalf("AssertPhoneVerified() error = %v, want ErrPhoneNotVerified", err)
	}
}

func TestConfirmCodeRetiresTheVerificationAfterRepeatedFailures(t *testing.T) {
	svc, _, sender := newServiceUnderTest()
	requested, _ := svc.RequestCode("01012345678")
	realCode := sender.sentCode(t)

	wrongCode := "000000"
	if wrongCode == realCode {
		wrongCode = "111111"
	}
	for attempt := 1; attempt < 5; attempt++ {
		if _, err := svc.ConfirmCode(requested.VerificationID, wrongCode); !errors.Is(err, service.ErrPhoneVerificationCodeMismatch) {
			t.Fatalf("attempt %d error = %v, want ErrPhoneVerificationCodeMismatch", attempt, err)
		}
	}
	if _, err := svc.ConfirmCode(requested.VerificationID, wrongCode); !errors.Is(err, service.ErrPhoneVerificationAttemptsExceeded) {
		t.Fatalf("final attempt error = %v, want ErrPhoneVerificationAttemptsExceeded", err)
	}
	// The correct code must no longer work once the verification is retired.
	if _, err := svc.ConfirmCode(requested.VerificationID, realCode); !errors.Is(err, service.ErrPhoneVerificationNotFound) {
		t.Fatalf("post-lockout error = %v, want ErrPhoneVerificationNotFound", err)
	}
}

// TestRequestCodeSendsTheDefaultTemplateBody pins the wording members receive when
// no administrator has edited the template.
func TestRequestCodeSendsTheDefaultTemplateBody(t *testing.T) {
	svc, _, sender := newServiceUnderTest()

	if _, err := svc.RequestCode("010-1234-5678"); err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}

	code := sender.sentCode(t)
	want := "[대일외고장학회] 인증번호 " + code + " 를 입력해 주세요."
	if got := sender.sent[0].Body; got != want {
		t.Fatalf("SMS body = %q, want %q", got, want)
	}
}

// TestRequestCodeSendsTheAdministratorEditedBody proves the edit reaches the
// gateway and that the code is still substituted.
func TestRequestCodeSendsTheAdministratorEditedBody(t *testing.T) {
	store := newFakeStore()
	sender := &capturingSMSSender{}
	templates := service.NewTestNotificationTemplateService(map[string]model.NotificationTemplate{
		model.NotificationTemplatePhoneVerificationSMS: {
			Key:     model.NotificationTemplatePhoneVerificationSMS,
			Channel: model.NotificationChannelSMS,
			Body:    "인증번호는 {code} 입니다.",
			Version: 4,
		},
	})
	svc := service.NewPhoneVerificationService(store, sender, templates, zerolog.Nop())

	if _, err := svc.RequestCode("010-1234-5678"); err != nil {
		t.Fatalf("RequestCode() error = %v", err)
	}

	want := "인증번호는 " + sender.sentCode(t) + " 입니다."
	if got := sender.sent[0].Body; got != want {
		t.Fatalf("SMS body = %q, want %q", got, want)
	}
}

// emptyTemplateRenderer stands in for a template that resolves to nothing at
// all, which the real service can produce only for an unknown key.
type emptyTemplateRenderer struct{}

func (emptyTemplateRenderer) Render(key string, _ map[string]string) service.RenderedTemplate {
	return service.RenderedTemplate{Key: key}
}

// TestRequestCodeRefusesToSendAnEmptyTemplate keeps a blank text off the
// gateway: a member receiving one has no code to enter and no way to continue.
func TestRequestCodeRefusesToSendAnEmptyTemplate(t *testing.T) {
	store := newFakeStore()
	sender := &capturingSMSSender{}
	svc := service.NewPhoneVerificationService(store, sender, emptyTemplateRenderer{}, zerolog.Nop())

	_, err := svc.RequestCode("010-1234-5678")

	if !errors.Is(err, service.ErrSMSTemplateEmpty) {
		t.Fatalf("RequestCode() error = %v, want ErrSMSTemplateEmpty", err)
	}
	if len(sender.sent) != 0 {
		t.Fatalf("an empty SMS was sent: %#v", sender.sent)
	}
}
