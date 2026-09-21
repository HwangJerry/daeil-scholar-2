// notification_template_validation.go — Field validation for admin-submitted notification texts.
package service

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"golang.org/x/text/encoding/korean"
)

// NotificationTemplateFieldError names one rejected field so the admin UI can
// point at the input that needs fixing.
type NotificationTemplateFieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// NotificationTemplateValidationError carries every field rejected by one
// submission.
type NotificationTemplateValidationError struct {
	Fields []NotificationTemplateFieldError
}

func (e *NotificationTemplateValidationError) Error() string {
	reasons := make([]string, 0, len(e.Fields))
	for _, field := range e.Fields {
		reasons = append(reasons, fmt.Sprintf("%s: %s", field.Field, field.Reason))
	}
	return "invalid notification template (" + strings.Join(reasons, "; ") + ")"
}

// ValidateNotificationTemplateVersion rejects a submission that omits the
// version the administrator was editing. Without it the store would skip its
// concurrency check and one administrator could silently overwrite another.
func ValidateNotificationTemplateVersion(expectedVersion int) error {
	if expectedVersion > 0 {
		return nil
	}
	return &NotificationTemplateValidationError{Fields: []NotificationTemplateFieldError{{
		Field:  "expectedVersion",
		Reason: "편집 중이던 버전 정보가 필요합니다. 화면을 새로 고친 뒤 다시 저장해 주세요",
	}}}
}

// ValidateNotificationTemplate checks one title/body pair against its catalog
// entry. It is used both for admin submissions and for stored rows before they
// are rendered, so a row that became invalid falls back to the default instead
// of sending broken text.
func ValidateNotificationTemplate(definition model.NotificationTemplateDefinition, title, body string) error {
	fields := make([]NotificationTemplateFieldError, 0)
	add := func(field, reason string) {
		fields = append(fields, NotificationTemplateFieldError{Field: field, Reason: reason})
	}

	if strings.TrimSpace(body) == "" {
		add("body", "본문을 입력해 주세요")
	}
	validateTemplateTitlePresence(definition, title, add)
	validateTemplatePlaceholders(definition, title, body, add)

	if definition.Channel == model.NotificationChannelSMS {
		validateSMSTemplateLength(definition, body, add)
	} else {
		validatePushTemplateLength(title, body, add)
	}

	if len(fields) == 0 {
		return nil
	}
	return &NotificationTemplateValidationError{Fields: fields}
}

// validateTemplateTitlePresence enforces that only push notifications carry a
// title: an SMS has no title field to send one in, so a value entered there
// would be silently dropped.
func validateTemplateTitlePresence(
	definition model.NotificationTemplateDefinition,
	title string,
	add func(field, reason string),
) {
	if definition.Channel == model.NotificationChannelSMS {
		if strings.TrimSpace(title) != "" {
			add("title", "문자 템플릿에는 제목을 사용하지 않습니다")
		}
		return
	}
	if strings.TrimSpace(title) == "" {
		add("title", "제목을 입력해 주세요")
	}
}

// validateTemplatePlaceholders rejects tokens this key does not support and
// requires every mandatory token to appear in the body. A typo such as `{cod}`
// would otherwise reach a member verbatim in place of their code.
//
// Fields are walked in a fixed order so one submission always reports its
// problems in the same sequence, which the admin UI relies on to highlight the
// first offending input.
func validateTemplatePlaceholders(
	definition model.NotificationTemplateDefinition,
	title, body string,
	add func(field, reason string),
) {
	texts := []struct {
		field string
		text  string
	}{
		{field: "title", text: title},
		{field: "body", text: body},
	}
	for _, entry := range texts {
		for _, token := range templateBraceTokens(entry.text) {
			name := templateTokenName(token)
			if !templatePlaceholderNamePattern.MatchString(name) || !definition.AllowsPlaceholder(name) {
				add(entry.field, fmt.Sprintf("%s 은(는) 사용할 수 없는 치환 항목입니다", token))
			}
		}
		// Any brace outside a well-formed token is a mistake that renders
		// literally: `{{code}}` would reach the member as `{000000}`, and a lone
		// `}` as itself.
		if strings.ContainsAny(templateBraceTokenPattern.ReplaceAllString(entry.text, ""), "{}") {
			add(entry.field, "중괄호 { } 는 치환 항목에만 사용할 수 있습니다")
		}
	}
	for _, placeholder := range definition.Placeholders {
		if placeholder.Required && !templateContainsPlaceholder(body, placeholder.Name) {
			add("body", fmt.Sprintf("본문에 {%s} 을(를) 반드시 포함해야 합니다", placeholder.Name))
		}
	}
}

// validateSMSTemplateLength measures the message the gateway will actually
// receive: placeholders replaced by their worst-case samples, counted in EUC-KR
// bytes, which is how the SMS vendor bills and truncates.
func validateSMSTemplateLength(
	definition model.NotificationTemplateDefinition,
	body string,
	add func(field, reason string),
) {
	rendered := renderTemplateWithSamples(body, definition)
	byteLength, unencodable, err := eucKRByteLength(rendered)
	if err != nil {
		add("body", fmt.Sprintf("문자로 보낼 수 없는 문자가 있습니다: %q", unencodable))
		return
	}
	if byteLength > model.SMSTemplateMaxEUCKRBytes {
		add("body", fmt.Sprintf(
			"치환 항목을 채운 길이가 %d바이트로 최대 %d바이트를 넘습니다",
			byteLength, model.SMSTemplateMaxEUCKRBytes,
		))
	}
}

func validatePushTemplateLength(title, body string, add func(field, reason string)) {
	if utf8.RuneCountInString(title) > model.PushTemplateMaxTitleRunes {
		add("title", fmt.Sprintf("제목은 %d자 이하여야 합니다", model.PushTemplateMaxTitleRunes))
	}
	if utf8.RuneCountInString(body) > model.PushTemplateMaxBodyRunes {
		add("body", fmt.Sprintf("본문은 %d자 이하여야 합니다", model.PushTemplateMaxBodyRunes))
	}
}

// eucKRByteLength returns the encoded length of the text in EUC-KR. When a rune
// has no EUC-KR representation — an emoji, for example — it is returned so the
// administrator can see which character to remove.
func eucKRByteLength(text string) (int, string, error) {
	encoder := korean.EUCKR.NewEncoder()
	encoded, err := encoder.String(text)
	if err == nil {
		return len(encoded), "", nil
	}
	for _, character := range text {
		if _, runeErr := korean.EUCKR.NewEncoder().String(string(character)); runeErr != nil {
			return 0, string(character), runeErr
		}
	}
	return 0, "", err
}
