// notification_template_validation_test.go — Rules an administrator's notification text must satisfy.
package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/service"
)

func definitionFor(t *testing.T, key string) model.NotificationTemplateDefinition {
	t.Helper()
	definition, found := model.NotificationTemplateDefinitionFor(key)
	if !found {
		t.Fatalf("catalog is missing %q", key)
	}
	return definition
}

// fieldErrors returns the rejected fields, failing the test when err is not a
// validation error.
func fieldErrors(t *testing.T, err error) []service.NotificationTemplateFieldError {
	t.Helper()
	var validation *service.NotificationTemplateValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want *NotificationTemplateValidationError", err)
	}
	return validation.Fields
}

func hasFieldError(fields []service.NotificationTemplateFieldError, field string) bool {
	for _, candidate := range fields {
		if candidate.Field == field {
			return true
		}
	}
	return false
}

func TestCatalogDefaultsAreValid(t *testing.T) {
	for _, definition := range model.NotificationTemplateCatalog() {
		if err := service.ValidateNotificationTemplate(definition, definition.DefaultTitle, definition.DefaultBody); err != nil {
			t.Errorf("default for %s is invalid: %v", definition.Key, err)
		}
	}
}

func TestValidateRejectsMisspelledPlaceholder(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplatePhoneVerificationSMS)

	err := service.ValidateNotificationTemplate(definition, "", "인증번호 {cod} 를 입력해 주세요.")

	fields := fieldErrors(t, err)
	if !hasFieldError(fields, "body") {
		t.Fatalf("fields = %#v, want a body error for the unknown token", fields)
	}
}

func TestValidateRejectsMissingRequiredPlaceholder(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplateNoticeNew)

	err := service.ValidateNotificationTemplate(definition, "새 소식", "새 공지가 등록되었습니다.")

	if !hasFieldError(fieldErrors(t, err), "body") {
		t.Fatalf("a body missing the required {subject} must be rejected")
	}
}

func TestValidateAcceptsPlaceholderOnlyBody(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplateMessagePreviewOn)

	if err := service.ValidateNotificationTemplate(definition, "{senderName}", "{content}"); err != nil {
		t.Fatalf("placeholder-only body rejected: %v", err)
	}
}

func TestValidateRejectsEmptyBody(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplateVerificationApproved)

	if err := service.ValidateNotificationTemplate(definition, "동문 인증 결과", "   "); err == nil {
		t.Fatal("an empty body must be rejected")
	}
}

func TestValidateTitleRulesFollowTheChannel(t *testing.T) {
	sms := definitionFor(t, model.NotificationTemplatePhoneVerificationSMS)
	if err := service.ValidateNotificationTemplate(sms, "인증", "인증번호 {code}"); err == nil {
		t.Fatal("an SMS template with a title must be rejected")
	}

	push := definitionFor(t, model.NotificationTemplateVerificationApproved)
	if err := service.ValidateNotificationTemplate(push, "", "승인되었습니다."); err == nil {
		t.Fatal("a push template without a title must be rejected")
	}
}

// TestValidateSMSLengthBoundary pins the limit to the message the gateway
// receives: the sample code is substituted before the bytes are counted.
func TestValidateSMSLengthBoundary(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplatePhoneVerificationSMS)
	tests := []struct {
		name       string
		body       string
		wantReject bool
	}{
		// "{code}" renders as 6 ASCII bytes, so the filler carries the rest.
		{name: "ascii exactly 80 bytes", body: "{code}" + strings.Repeat("A", 74), wantReject: false},
		{name: "ascii 81 bytes", body: "{code}" + strings.Repeat("A", 75), wantReject: true},
		// Hangul costs two EUC-KR bytes per syllable.
		{name: "hangul exactly 80 bytes", body: "{code}" + strings.Repeat("가", 37), wantReject: false},
		{name: "hangul 82 bytes", body: "{code}" + strings.Repeat("가", 38), wantReject: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := service.ValidateNotificationTemplate(definition, "", test.body)
			if test.wantReject && err == nil {
				t.Fatal("over-long SMS body accepted")
			}
			if !test.wantReject && err != nil {
				t.Fatalf("SMS body at the limit rejected: %v", err)
			}
		})
	}
}

func TestValidateRejectsRunesEUCKRCannotEncode(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplatePhoneVerificationSMS)

	err := service.ValidateNotificationTemplate(definition, "", "인증번호 {code} 입니다 🎉")

	if !hasFieldError(fieldErrors(t, err), "body") {
		t.Fatal("an emoji must be rejected: the SMS gateway encodes EUC-KR")
	}
}

func TestValidatePushLengthCaps(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplateVerificationApproved)

	longTitle := service.ValidateNotificationTemplate(
		definition, strings.Repeat("가", model.PushTemplateMaxTitleRunes+1), "본문",
	)
	if !hasFieldError(fieldErrors(t, longTitle), "title") {
		t.Fatal("an over-long push title must be rejected")
	}

	longBody := service.ValidateNotificationTemplate(
		definition, "제목", strings.Repeat("가", model.PushTemplateMaxBodyRunes+1),
	)
	if !hasFieldError(fieldErrors(t, longBody), "body") {
		t.Fatal("an over-long push body must be rejected")
	}

	atCap := service.ValidateNotificationTemplate(
		definition,
		strings.Repeat("가", model.PushTemplateMaxTitleRunes),
		strings.Repeat("가", model.PushTemplateMaxBodyRunes),
	)
	if atCap != nil {
		t.Fatalf("push text exactly at the caps rejected: %v", atCap)
	}
}

// TestValidatePushAllowsEmoji guards against the SMS encoding rule leaking into
// the push channel, where emoji are ordinary content.
func TestValidatePushAllowsEmoji(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplateVerificationApproved)

	if err := service.ValidateNotificationTemplate(definition, "동문 인증 결과", "축하합니다 🎉"); err != nil {
		t.Fatalf("emoji rejected in a push template: %v", err)
	}
}

// TestValidateRejectsUnbalancedBraces covers text that passes the token rules
// yet still renders literal braces to the member.
func TestValidateRejectsUnbalancedBraces(t *testing.T) {
	sms := definitionFor(t, model.NotificationTemplatePhoneVerificationSMS)
	tests := []struct {
		name string
		body string
	}{
		{name: "doubled braces", body: "인증번호 {{code}} 입니다"},
		{name: "stray closing brace", body: "인증번호 {code} 입니다}"},
		{name: "stray opening brace", body: "인증번호 {code} 입니다{"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := service.ValidateNotificationTemplate(sms, "", test.body)
			if !hasFieldError(fieldErrors(t, err), "body") {
				t.Fatalf("%q was accepted; it renders braces to the member", test.body)
			}
		})
	}
}

// TestValidateReportsTitleBeforeBody pins the field order the admin UI uses to
// focus the first offending input.
func TestValidateReportsTitleBeforeBody(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplateNoticeNew)

	err := service.ValidateNotificationTemplate(definition, "{nope}", "{alsoNope}")

	fields := fieldErrors(t, err)
	if len(fields) < 2 || fields[0].Field != "title" {
		t.Fatalf("fields = %#v, want the title error first", fields)
	}
}

func TestValidateNotificationTemplateVersionRequiresAVersion(t *testing.T) {
	if err := service.ValidateNotificationTemplateVersion(0); err == nil {
		t.Fatal("a submission without the edited version must be rejected")
	}
	if err := service.ValidateNotificationTemplateVersion(-1); err == nil {
		t.Fatal("a negative version must be rejected")
	}
	if err := service.ValidateNotificationTemplateVersion(1); err != nil {
		t.Fatalf("version 1 rejected: %v", err)
	}
}

// TestDefaultSMSBodyFitsTheStricterVendorCap guards the shipped default against
// the 80-byte NCP SENS ceiling.
func TestDefaultSMSBodyFitsTheStricterVendorCap(t *testing.T) {
	definition := definitionFor(t, model.NotificationTemplatePhoneVerificationSMS)

	if err := service.ValidateNotificationTemplate(definition, "", definition.DefaultBody); err != nil {
		t.Fatalf("the default verification SMS no longer fits the cap: %v", err)
	}
}
