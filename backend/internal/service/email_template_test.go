// email_template_test.go — Unit tests for transactional email template rendering
package service

import (
	"strings"
	"testing"
)

func TestRenderPasswordResetEmail(t *testing.T) {
	data := PasswordResetEmailData{
		Name:     "홍길동",
		ResetURL: "https://example.com/reset?token=abc123",
	}

	html, err := RenderPasswordResetEmail(data)
	if err != nil {
		t.Fatalf("RenderPasswordResetEmail returned error: %v", err)
	}

	checks := []struct {
		desc     string
		contains string
	}{
		{"recipient name", "홍길동"},
		{"reset URL in link", "https://example.com/reset?token=abc123"},
		{"HTML doctype", "<!DOCTYPE html>"},
		{"heading text", "비밀번호 재설정"},
		{"brand color", "#4F46E5"},
	}
	for _, c := range checks {
		t.Run(c.desc, func(t *testing.T) {
			if !strings.Contains(html, c.contains) {
				t.Errorf("output should contain %q", c.contains)
			}
		})
	}
}

func TestRenderNewMessageEmail(t *testing.T) {
	data := NewMessageEmailData{
		RecipientName: "김철수",
		SenderName:    "이영희",
		Preview:       "안녕하세요, 오랜만입니다.",
		InboxURL:      "https://example.com/messages",
	}

	html, err := RenderNewMessageEmail(data)
	if err != nil {
		t.Fatalf("RenderNewMessageEmail returned error: %v", err)
	}

	checks := []struct {
		desc     string
		contains string
	}{
		{"recipient name", "김철수"},
		{"sender name", "이영희"},
		{"message preview", "안녕하세요, 오랜만입니다."},
		{"inbox URL", "https://example.com/messages"},
		{"heading text", "새 쪽지가 도착했습니다"},
	}
	for _, c := range checks {
		t.Run(c.desc, func(t *testing.T) {
			if !strings.Contains(html, c.contains) {
				t.Errorf("output should contain %q", c.contains)
			}
		})
	}
}

func TestRenderApprovalEmail(t *testing.T) {
	data := ApprovalEmailData{
		Name:     "박민수",
		LoginURL: "https://example.com/login",
	}

	html, err := RenderApprovalEmail(data)
	if err != nil {
		t.Fatalf("RenderApprovalEmail returned error: %v", err)
	}

	checks := []struct {
		desc     string
		contains string
	}{
		{"recipient name", "박민수"},
		{"login URL", "https://example.com/login"},
		{"heading text", "가입 승인 완료"},
		{"brand color", "#4F46E5"},
	}
	for _, c := range checks {
		t.Run(c.desc, func(t *testing.T) {
			if !strings.Contains(html, c.contains) {
				t.Errorf("output should contain %q", c.contains)
			}
		})
	}
}

func TestRenderPasswordResetEmailEscapesHTML(t *testing.T) {
	data := PasswordResetEmailData{
		Name:     `<script>alert("xss")</script>`,
		ResetURL: "https://example.com/reset?token=safe",
	}

	html, err := RenderPasswordResetEmail(data)
	if err != nil {
		t.Fatalf("RenderPasswordResetEmail returned error: %v", err)
	}

	if strings.Contains(html, "<script>") {
		t.Error("template should escape HTML in Name field")
	}
}
