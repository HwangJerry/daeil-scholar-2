# 관리자 웹 애플리케이션 설계 (§15)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

---

## 15. 관리자 웹 애플리케이션 설계

### 15.1 개요

관리자 웹은 User SPA와 **별도 React 앱**으로 구현합니다. 동일한 Go 백엔드의 `/api/admin/*` 엔드포인트를 사용하며, Nginx에서 `/admin/*` 경로로 서빙합니다.

| 항목 | 내용 |
|------|------|
| 경로 | `https://alumni.example.com/admin` |
| 프레임워크 | React (Vite), TypeScript |
| UI 라이브러리 | Tailwind CSS v4 + Radix UI Primitives (User SPA와 동일) |
| Markdown 에디터 | `@uiw/react-md-editor` (~160KB gzip) |
| 상태 관리 | React Query (서버 상태) + Zustand (로컬 상태) |
| 빌드 | User SPA와 별도 빌드 (`/admin/dist/`) |
| 인가 | `AdminAuthMiddleware` — `USR_STATUS = 'ZZZ'` 확인 (운영자) |

### 15.2 인증/인가

관리자도 카카오 로그인 + 인증 브릿지를 통해 동일하게 로그인하되, Admin API 접근 시 추가 권한 검증을 수행합니다.

```go
// internal/middleware/admin_auth.go

func AdminAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromContext(r.Context())
        if user == nil {
            respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다")
            return
        }
        // 관리자 권한 확인 (USR_STATUS = 'ZZZ' 운영자)
        if user.USRStatus != "ZZZ" {
            respondError(w, 403, "FORBIDDEN", "관리자 권한이 필요합니다")
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

**라우터 등록:**
```go
r.Route("/api/admin", func(r chi.Router) {
    r.Use(middleware.AuthBridgeMiddleware(db))   // 인증
    r.Use(middleware.AdminAuthMiddleware)         // 관리자 인가

    // 대시보드
    r.Get("/dashboard", adminHandler.Dashboard)

    // 공지 관리
    r.Get("/feed", adminNoticeHandler.List)
    r.Get("/feed/{seq}", adminNoticeHandler.Detail)   // 편집용 (contentMd 포함)
    r.Post("/feed", adminNoticeHandler.Create)
    r.Put("/feed/{seq}", adminNoticeHandler.Update)
    r.Delete("/feed/{seq}", adminNoticeHandler.Delete)
    r.Put("/feed/{seq}/pin", adminNoticeHandler.TogglePin)

    // 이미지 업로드
    r.Post("/upload", adminUploadHandler.Upload)

    // 광고 관리
    r.Get("/ad", adminAdHandler.List)
    r.Post("/ad", adminAdHandler.Create)
    r.Put("/ad/{seq}", adminAdHandler.Update)
    r.Delete("/ad/{seq}", adminAdHandler.Delete)
    r.Get("/ad/stats", adminAdHandler.Stats)

    // 기부 설정
    r.Get("/donation/config", adminDonationHandler.GetConfig)
    r.Put("/donation/config", adminDonationHandler.UpdateConfig)
    r.Get("/donation/history", adminDonationHandler.History)

    // 회원 관리
    r.Get("/member", adminMemberHandler.List)
    r.Get("/member/{seq}", adminMemberHandler.Detail)
    r.Put("/member/{seq}", adminMemberHandler.Update)
    r.Get("/member/stats", adminMemberHandler.Stats)
})
```

### 15.3 화면 설계

#### 대시보드 (`/admin`)
```
┌──────────────────────────────────────────────────────────────┐
│  [로고]  관리자 페이지              [사용자 사이트] [로그아웃] │
├────────┬─────────────────────────────────────────────────────┤
│        │                                                     │
│ 📊 대시보드│  ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│ 📢 공지관리│  │ 총 회원    │ │ 기부 현황  │ │ 오늘 방문  │          │
│ 📺 광고관리│  │ 3,420명   │ │ ₩1.3억    │ │ 45명      │          │
│ 💰 기부설정│  └──────────┘ └──────────┘ └──────────┘          │
│ 👥 회원관리│                                                     │
│        │  ┌──────────────────────────────┐                   │
│        │  │ 최근 공지 5건                  │                   │
│        │  │ ① 2026 정기총회 안내  (2/10)   │                   │
│        │  │ ② 동문 장학금 선발 결과 (2/8)  │                   │
│        │  │ ...                           │                   │
│        │  └──────────────────────────────┘                   │
│        │                                                     │
│        │  [+ 새 공지 작성]  [+ 광고 등록]                      │
│        │                                                     │
└────────┴─────────────────────────────────────────────────────┘
```

#### 공지 작성/수정 (`/admin/notice/new`, `/admin/notice/:seq/edit`)
```
┌──────────────────────────────────────────────────────────────┐
│  [←뒤로]  공지 작성                           [사용자 사이트] │
├────────┬─────────────────────────────────────────────────────┤
│        │                                                     │
│ 사이드  │  제목: [_______________________________]             │
│ 바     │                                                     │
│        │  ┌─────────────────────────────────────────────┐    │
│        │  │  [B] [I] [링크] [이미지] [코드] [표] [인용]   │    │
│        │  ├──────────────────┬──────────────────────────┤    │
│        │  │   Markdown 편집   │   실시간 미리보기          │    │
│        │  │                  │                          │    │
│        │  │ # 정기총회 안내   │  정기총회 안내              │    │
│        │  │                  │                          │    │
│        │  │ 일시: 3월 15일    │  일시: 3월 15일            │    │
│        │  │                  │                          │    │
│        │  │ ![포스터](url)   │  [이미지 렌더링]           │    │
│        │  │                  │                          │    │
│        │  │ 참석 신청은...    │  참석 신청은...            │    │
│        │  │                  │                          │    │
│        │  └──────────────────┴──────────────────────────┘    │
│        │                                                     │
│        │  옵션: ☐ 상단 고정                                   │
│        │                                                     │
│        │  [임시저장]                    [미리보기] [저장]      │
│        │                                                     │
└────────┴─────────────────────────────────────────────────────┘
```

#### 회원 관리 (`/admin/member`)
```
┌──────────────────────────────────────────────────────────────┐
│  회원 관리                                    [사용자 사이트] │
├────────┬─────────────────────────────────────────────────────┤
│        │                                                     │
│ 사이드  │  검색: [이름____] [기수__▼] [상태__▼]  [검색]       │
│ 바     │                                                     │
│        │  ┌───┬────────┬────┬────┬──────┬──────┬──────┐     │
│        │  │ # │ 이름    │ 기수│ 학과│ 상태  │ 카카오│ 최근접속│     │
│        │  ├───┼────────┼────┼────┼──────┼──────┼──────┤     │
│        │  │ 1 │ 김동문  │ 20 │ 경영│ 활성  │ 연동  │ 2/10  │     │
│        │  │ 2 │ 이선배  │ 15 │ 법학│ 활성  │ 미연동│ 1/25  │     │
│        │  │ 3 │ 박후배  │ 25 │ 전자│ 휴면  │ 연동  │ 12/1  │     │
│        │  └───┴────────┴────┴────┴──────┴──────┴──────┘     │
│        │                                                     │
│        │  < 1 2 3 ... 10 >                                   │
│        │                                                     │
└────────┴─────────────────────────────────────────────────────┘
```

### 15.4 레거시 공지 편집 호환

기존 PHP로 작성된 공지(`CONTENT_FORMAT = 'LEGACY'`)는 Markdown 에디터 대신 read-only HTML 뷰어로 표시하고, 수정 시 안내 메시지를 노출합니다.

```tsx
// admin/src/pages/NoticeEditPage.tsx
function NoticeEditPage() {
  const { data: notice } = useNotice(seq);

  if (notice?.contentFormat === 'LEGACY') {
    return (
      <div>
        <div className="bg-yellow-50 p-4 rounded mb-4">
          ⚠️ 이 글은 기존 시스템에서 작성되었습니다.
          Markdown 에디터로 수정하려면 기존 내용을 복사하여 새 글로 작성해주세요.
        </div>
        <HtmlContent html={notice.contentHtml} />
      </div>
    );
  }

  return <MarkdownEditor value={notice?.contentMd || ''} onChange={...} />;
}
```

### 15.5 Admin SPA 보안

| 항목 | 방어 |
|------|------|
| 인가 | `/api/admin/*` 전체에 `AdminAuthMiddleware` 적용 |
| XSS | Markdown → HTML 변환 시 bluemonday 새니타이즈, 프론트에서 DOMPurify 2차 방어 |
| CSRF | CSRFMiddleware (Origin/Referer 검증) — Admin도 동일 적용 |
| 파일 업로드 | Content-Type 검증, 이미지만 허용, 리사이즈, UUID 파일명 |
| Rate limit | `/api/admin/*`는 API rate limit (30r/s) 동일 적용 |
| 접근 경로 | `/admin` SPA 자체는 누구나 접근 가능하나, API가 403을 반환하면 로그인 페이지로 리다이렉트 |

---
