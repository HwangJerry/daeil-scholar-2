// email_template.go — HTML email templates for transactional emails (Korean)
package service

import (
	"bytes"
	"html/template"
)

// passwordResetTmpl is the HTML template for password reset emails.
var passwordResetTmpl = template.Must(template.New("passwordReset").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:'Apple SD Gothic Neo','Malgun Gothic',sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:40px 0;">
    <tr><td align="center">
      <table width="480" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;padding:40px;">
        <tr><td style="text-align:center;padding-bottom:24px;">
          <h2 style="color:#4F46E5;margin:0;font-size:20px;">비밀번호 재설정</h2>
        </td></tr>
        <tr><td style="color:#374151;font-size:14px;line-height:1.6;padding-bottom:24px;">
          <p style="margin:0 0 12px;">안녕하세요, <strong>{{.Name}}</strong>님.</p>
          <p style="margin:0 0 12px;">비밀번호 재설정이 요청되었습니다. 아래 버튼을 클릭하여 새 비밀번호를 설정해 주세요.</p>
          <p style="margin:0;color:#6b7280;font-size:12px;">이 링크는 15분 동안 유효합니다.</p>
        </td></tr>
        <tr><td align="center" style="padding-bottom:24px;">
          <a href="{{.ResetURL}}" style="display:inline-block;background:#4F46E5;color:#ffffff;text-decoration:none;padding:12px 32px;border-radius:6px;font-size:14px;font-weight:600;">비밀번호 재설정</a>
        </td></tr>
        <tr><td style="color:#9ca3af;font-size:12px;line-height:1.5;border-top:1px solid #e5e7eb;padding-top:16px;">
          <p style="margin:0 0 4px;">버튼이 작동하지 않으면 아래 링크를 브라우저에 복사하세요:</p>
          <p style="margin:0;word-break:break-all;">{{.ResetURL}}</p>
          <p style="margin:12px 0 0;color:#d1d5db;">본인이 요청하지 않았다면 이 이메일을 무시하세요.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`))

// PasswordResetEmailData holds template data for password reset emails.
type PasswordResetEmailData struct {
	Name     string
	ResetURL string
}

// RenderPasswordResetEmail renders the password reset HTML email body.
func RenderPasswordResetEmail(data PasswordResetEmailData) (string, error) {
	var buf bytes.Buffer
	if err := passwordResetTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// newMessageTmpl is the HTML template for new message notification emails.
var newMessageTmpl = template.Must(template.New("newMessage").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:'Apple SD Gothic Neo','Malgun Gothic',sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:40px 0;">
    <tr><td align="center">
      <table width="480" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;padding:40px;">
        <tr><td style="text-align:center;padding-bottom:24px;">
          <h2 style="color:#4F46E5;margin:0;font-size:20px;">새 쪽지가 도착했습니다</h2>
        </td></tr>
        <tr><td style="color:#374151;font-size:14px;line-height:1.6;padding-bottom:24px;">
          <p style="margin:0 0 12px;">안녕하세요, <strong>{{.RecipientName}}</strong>님.</p>
          <p style="margin:0 0 12px;"><strong>{{.SenderName}}</strong>님으로부터 새 쪽지가 도착했습니다.</p>
          <div style="background:#f9fafb;border-radius:6px;padding:16px;margin:12px 0;color:#374151;font-size:13px;">
            {{.Preview}}
          </div>
        </td></tr>
        <tr><td align="center" style="padding-bottom:24px;">
          <a href="{{.InboxURL}}" style="display:inline-block;background:#4F46E5;color:#ffffff;text-decoration:none;padding:12px 32px;border-radius:6px;font-size:14px;font-weight:600;">쪽지함 확인</a>
        </td></tr>
        <tr><td style="color:#9ca3af;font-size:12px;border-top:1px solid #e5e7eb;padding-top:16px;">
          <p style="margin:0;">동문 커뮤니티에서 보낸 알림 메일입니다.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`))

// NewMessageEmailData holds template data for new message notification emails.
type NewMessageEmailData struct {
	RecipientName string
	SenderName    string
	Preview       string
	InboxURL      string
}

// RenderNewMessageEmail renders the new message notification HTML email body.
func RenderNewMessageEmail(data NewMessageEmailData) (string, error) {
	var buf bytes.Buffer
	if err := newMessageTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// approvalTmpl is the HTML template for member approval notification emails.
var approvalTmpl = template.Must(template.New("approval").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:'Apple SD Gothic Neo','Malgun Gothic',sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:40px 0;">
    <tr><td align="center">
      <table width="480" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;padding:40px;">
        <tr><td style="text-align:center;padding-bottom:24px;">
          <h2 style="color:#4F46E5;margin:0;font-size:20px;">가입 승인 완료</h2>
        </td></tr>
        <tr><td style="color:#374151;font-size:14px;line-height:1.6;padding-bottom:24px;">
          <p style="margin:0 0 12px;">안녕하세요, <strong>{{.Name}}</strong>님.</p>
          <p style="margin:0 0 12px;">동문 커뮤니티 가입이 승인되었습니다. 지금 바로 로그인하여 서비스를 이용해 보세요.</p>
        </td></tr>
        <tr><td align="center" style="padding-bottom:24px;">
          <a href="{{.LoginURL}}" style="display:inline-block;background:#4F46E5;color:#ffffff;text-decoration:none;padding:12px 32px;border-radius:6px;font-size:14px;font-weight:600;">로그인하기</a>
        </td></tr>
        <tr><td style="color:#9ca3af;font-size:12px;border-top:1px solid #e5e7eb;padding-top:16px;">
          <p style="margin:0;">동문 커뮤니티에서 보낸 알림 메일입니다.</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`))

// ApprovalEmailData holds template data for member approval notification emails.
type ApprovalEmailData struct {
	Name     string
	LoginURL string
}

// RenderApprovalEmail renders the member approval notification HTML email body.
func RenderApprovalEmail(data ApprovalEmailData) (string, error) {
	var buf bytes.Buffer
	if err := approvalTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
