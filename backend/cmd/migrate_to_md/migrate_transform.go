// migrate_transform — per-post transformation pipeline: legacy HTML → markdown → encoded HTML
package main

import (
	"github.com/dflh-saf/backend/internal/service"
)

const systemPrompt = `당신은 한국어 게시글을 마크다운으로 변환하는 전문가입니다.
원문의 의미와 구조를 최대한 보존하면서 적절한 마크다운 문법으로 재구성하세요.

규칙:
- 제목/소제목은 ## ### 으로 표현
- 목록은 - 또는 1. 으로 표현
- 중요 내용은 **굵게** 표현
- 코드는 백틱으로 표현
- 원시 HTML 태그는 제거하고 의미만 보존
- 불필요한 연속 빈 줄 정리 (최대 1개)
- 마크다운 외의 설명 텍스트 금지 — 변환된 마크다운만 출력`

func convertToMarkdown(post legacyPost) (string, error) {
	userContent := post.Subject + "\n\n" + post.Contents
	return callClaude(userContent)
}

// processPost runs the full transformation pipeline for a single legacy post:
// claude CLI markdown conversion → goldmark/bluemonday encoding → summary/thumbnail extraction.
func processPost(post legacyPost) (postUpdate, error) {
	md, err := convertToMarkdown(post)
	if err != nil {
		return postUpdate{}, err
	}

	encoded, summary, thumbnail, err := service.ConvertAndEncode(md)
	if err != nil {
		return postUpdate{}, err
	}

	return postUpdate{
		SEQ:          post.SEQ,
		Contents:     encoded,
		ContentsMD:   md,
		Summary:      summary,
		ThumbnailURL: thumbnail,
	}, nil
}
