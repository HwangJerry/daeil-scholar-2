package service

import "testing"

func TestMessageContentFilter(t *testing.T) {
	filter := NewMessageContentFilter([]string{"금지광고", " "})
	for _, text := range []string{"씨 발", "개\u200b새끼", "ＫＩＬＬ yourself", "금지.광고"} {
		if filter.Allows(text) {
			t.Errorf("obfuscated blocked phrase was accepted: %q", text)
		}
	}
	for _, text := range []string{"안녕하세요! 동문 모임에서 뵙겠습니다.", "오늘 날씨가 좋네요.", "Thank you for your help."} {
		if !filter.Allows(text) {
			t.Errorf("ordinary message was rejected: %q", text)
		}
	}
}
