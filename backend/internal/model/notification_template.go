// notification_template.go — Catalog and domain model for admin-editable notification texts.
package model

import "time"

// Notification channels. The channel decides which length and encoding rules a
// template body must satisfy, so it lives in the catalog rather than the row.
const (
	NotificationChannelSMS  = "sms"
	NotificationChannelPush = "push"
)

// Template keys. Call sites reference these constants, never a literal string.
const (
	NotificationTemplatePhoneVerificationSMS = "sms.phone_verification"
	NotificationTemplateVerificationApproved = "push.verification.approved"
	NotificationTemplateVerificationRejected = "push.verification.rejected"
	NotificationTemplateMessagePreviewOff    = "push.message.new_preview_off"
	NotificationTemplateMessagePreviewOn     = "push.message.new_preview_on"
	NotificationTemplateNoticeNew            = "push.notice.new"
)

// Channel limits. SMS is bounded in EUC-KR bytes because the SENS gateway counts
// the message that way; push is bounded in runes as a sanity cap, since the
// operating systems truncate visually long text on their own.
const (
	// SMSTemplateMaxEUCKRBytes is the short-message ceiling of the strictest
	// vendor we can be configured against. NCP SENS treats Type="SMS" as 80
	// EUC-KR bytes and silently promotes anything longer to a billed LMS;
	// Aligo and KT allow 90. Taking 80 keeps a template deliverable whichever
	// provider SMS_PROVIDER names.
	SMSTemplateMaxEUCKRBytes  = 80
	PushTemplateMaxTitleRunes = 50
	PushTemplateMaxBodyRunes  = 200
)

// NotificationTemplatePlaceholder describes one `{name}` token a template may
// use. Sample is the worst-case value used when estimating SMS length.
type NotificationTemplatePlaceholder struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Sample      string `json:"sample"`
}

// NotificationTemplateDefinition is one catalog entry. The catalog is defined in
// code: administrators edit the title and body of existing keys, and can neither
// add nor remove keys, because every key is referenced by a call site.
type NotificationTemplateDefinition struct {
	Key          string
	Channel      string
	DisplayName  string
	Description  string
	DefaultTitle string
	DefaultBody  string
	Placeholders []NotificationTemplatePlaceholder
}

// NotificationTemplate is one stored override row in notification_templates.
type NotificationTemplate struct {
	Key       string    `db:"NT_KEY" json:"key"`
	Channel   string    `db:"NT_CHANNEL" json:"channel"`
	Title     string    `db:"NT_TITLE" json:"title"`
	Body      string    `db:"NT_BODY" json:"body"`
	Version   int       `db:"NT_VERSION" json:"version"`
	UpdatedAt time.Time `db:"UPDATED_AT" json:"updatedAt"`
	UpdatedBy *int      `db:"UPDATED_BY" json:"updatedBy"`
}

// NotificationChannelLimits are the caps the admin UI shows as live counters.
// Only the fields that apply to the channel are populated.
type NotificationChannelLimits struct {
	MaxBodyEUCKRBytes int `json:"maxBodyEucKrBytes,omitempty"`
	MaxTitleRunes     int `json:"maxTitleRunes,omitempty"`
	MaxBodyRunes      int `json:"maxBodyRunes,omitempty"`
}

// notificationTemplateCatalog is the single source of truth for which templates
// exist and what they fall back to. Defaults here must stay identical to the
// text the call sites used before they read from this store.
var notificationTemplateCatalog = []NotificationTemplateDefinition{
	{
		Key:          NotificationTemplatePhoneVerificationSMS,
		Channel:      NotificationChannelSMS,
		DisplayName:  "휴대폰 인증번호 문자",
		Description:  "회원가입 시 휴대폰 본인확인을 위해 발송되는 인증번호 문자입니다.",
		DefaultTitle: "",
		DefaultBody:  "[대일외고장학회] 인증번호 {code} 를 입력해 주세요.",
		Placeholders: []NotificationTemplatePlaceholder{
			{Name: "code", Required: true, Description: "6자리 인증번호", Sample: "000000"},
		},
	},
	{
		Key:          NotificationTemplateVerificationApproved,
		Channel:      NotificationChannelPush,
		DisplayName:  "동문 인증 승인 푸시",
		Description:  "관리자가 동문 인증 신청을 승인했을 때 발송되는 푸시 알림입니다.",
		DefaultTitle: "동문 인증 결과",
		DefaultBody:  "동문 인증이 승인되었습니다. 이제 동문 커뮤니티를 이용할 수 있어요.",
		Placeholders: nil,
	},
	{
		Key:          NotificationTemplateVerificationRejected,
		Channel:      NotificationChannelPush,
		DisplayName:  "동문 인증 반려 푸시",
		Description:  "관리자가 동문 인증 신청을 반려했을 때 발송되는 푸시 알림입니다.",
		DefaultTitle: "동문 인증 결과",
		DefaultBody:  "동문 인증 신청이 반려되었습니다. 앱에서 사유를 확인하고 다시 신청해 주세요.",
		Placeholders: nil,
	},
	{
		Key:          NotificationTemplateMessagePreviewOff,
		Channel:      NotificationChannelPush,
		DisplayName:  "새 쪽지 푸시 (미리보기 꺼짐)",
		Description:  "쪽지 미리보기를 끈 회원에게 발송되는 새 쪽지 알림입니다. 본문에 쪽지 내용을 넣지 마세요.",
		DefaultTitle: "{senderName}",
		DefaultBody:  "새 메시지가 도착했습니다.",
		Placeholders: []NotificationTemplatePlaceholder{
			{Name: "senderName", Required: false, Description: "보낸 사람 이름", Sample: "홍길동"},
		},
	},
	{
		Key:          NotificationTemplateMessagePreviewOn,
		Channel:      NotificationChannelPush,
		DisplayName:  "새 쪽지 푸시 (미리보기 켜짐)",
		Description:  "쪽지 미리보기를 켠 회원에게 발송되는 새 쪽지 알림입니다.",
		DefaultTitle: "{senderName}",
		DefaultBody:  "{content}",
		Placeholders: []NotificationTemplatePlaceholder{
			{Name: "senderName", Required: false, Description: "보낸 사람 이름", Sample: "홍길동"},
			{Name: "content", Required: true, Description: "쪽지 내용 미리보기", Sample: "쪽지 내용"},
		},
	},
	{
		Key:          NotificationTemplateNoticeNew,
		Channel:      NotificationChannelPush,
		DisplayName:  "새 공지 푸시",
		Description:  "새 공지사항이 등록되었을 때 발송되는 푸시 알림입니다.",
		DefaultTitle: "새 소식",
		DefaultBody:  "{subject}",
		Placeholders: []NotificationTemplatePlaceholder{
			{Name: "subject", Required: true, Description: "공지 제목", Sample: "공지 제목"},
		},
	},
}

// NotificationTemplateCatalog returns a copy of the catalog in display order.
// Callers receive their own slices so no consumer can mutate the defaults the
// renderer falls back to.
func NotificationTemplateCatalog() []NotificationTemplateDefinition {
	catalog := make([]NotificationTemplateDefinition, 0, len(notificationTemplateCatalog))
	for _, definition := range notificationTemplateCatalog {
		catalog = append(catalog, cloneTemplateDefinition(definition))
	}
	return catalog
}

// NotificationTemplateDefinitionFor looks up one catalog entry. The second
// result reports whether the key exists at all; an unknown key is a programming
// error at a call site or an unknown key submitted by the admin UI.
func NotificationTemplateDefinitionFor(key string) (NotificationTemplateDefinition, bool) {
	for _, definition := range notificationTemplateCatalog {
		if definition.Key == key {
			return cloneTemplateDefinition(definition), true
		}
	}
	return NotificationTemplateDefinition{}, false
}

// AllowsPlaceholder reports whether the named token may appear in this
// template's title or body.
func (d NotificationTemplateDefinition) AllowsPlaceholder(name string) bool {
	for _, placeholder := range d.Placeholders {
		if placeholder.Name == name {
			return true
		}
	}
	return false
}

// Limits returns the caps that apply to this template's channel.
func (d NotificationTemplateDefinition) Limits() NotificationChannelLimits {
	if d.Channel == NotificationChannelSMS {
		return NotificationChannelLimits{MaxBodyEUCKRBytes: SMSTemplateMaxEUCKRBytes}
	}
	return NotificationChannelLimits{
		MaxTitleRunes: PushTemplateMaxTitleRunes,
		MaxBodyRunes:  PushTemplateMaxBodyRunes,
	}
}

func cloneTemplateDefinition(definition NotificationTemplateDefinition) NotificationTemplateDefinition {
	if definition.Placeholders != nil {
		placeholders := make([]NotificationTemplatePlaceholder, len(definition.Placeholders))
		copy(placeholders, definition.Placeholders)
		definition.Placeholders = placeholders
	}
	return definition
}
