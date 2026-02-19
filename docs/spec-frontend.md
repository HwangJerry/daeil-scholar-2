# 프론트엔드 화면/컴포넌트 설계 (§8, §9)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

---

## 8. 화면 설계

### 8.1 웹 뷰 (Desktop, ≥768px)

```
┌──────────────────────────────────────────────────────────────┐
│  [Logo + Title]                  뉴스피드  동문찾기  기부  마이페이지 │  ← 상단 메뉴바
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐    │
│  │                   HERO SECTION                       │    │
│  │   최신 공지 1건 (큰 이미지 + 제목 + 요약)             │    │
│  │   [전체보기 →]                                        │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐    │
│  │              💰 누적 기부액 섹션                       │    │
│  │   ₩130,000,000  |  342명 참여  |  달성률 65%          │    │
│  │   [프로그레스 바 ████████░░░░░ ]                       │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─────────────────────────────┐  ┌───────────────────────┐  │
│  │  공지 카드 1                 │  │                       │  │
│  │  [이미지]                    │  │   사이드 영역          │  │
│  │  제목 / 요약 / 날짜          │  │   (추후 확장 가능)     │  │
│  ├─────────────────────────────┤  │                       │  │
│  │  공지 카드 2                 │  │                       │  │
│  ├─────────────────────────────┤  │                       │  │
│  │  공지 카드 3                 │  │                       │  │
│  ├─────────────────────────────┤  │                       │  │
│  │  공지 카드 4                 │  │                       │  │
│  ├─────────────────────────────┤  └───────────────────────┘  │
│  │  🔥 가장 핫한 동문 소식       │                            │
│  │  [광고 카드 - PREMIUM]       │                            │
│  ├─────────────────────────────┤                            │
│  │  공지 카드 5                 │                            │
│  │  ...                         │                            │
│  │  (무한스크롤)                │                            │
│  └─────────────────────────────┘                            │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**웹 뷰 비율:** 피드 영역 ~65%, 사이드 영역 ~35%

### 8.2 모바일 뷰 (<768px)

```
┌──────────────────────────┐
│     [Logo + Title]       │  ← 상단 타이틀만 표시
├──────────────────────────┤
│                          │
│  ┌────────────────────┐  │
│  │   HERO SECTION     │  │
│  │  최신 공지 (풀폭)   │  │
│  │  [큰 이미지]        │  │
│  │  제목 / 요약        │  │
│  └────────────────────┘  │
│                          │
│  ┌────────────────────┐  │
│  │  💰 누적 기부액     │  │
│  │  ₩130,000,000      │  │
│  │  342명 | 65%        │  │
│  │  [프로그레스 바]     │  │
│  └────────────────────┘  │
│                          │
│  ┌────────────────────┐  │
│  │  공지 카드 1 (풀폭) │  │
│  ├────────────────────┤  │
│  │  공지 카드 2        │  │
│  ├────────────────────┤  │
│  │  공지 카드 3        │  │
│  ├────────────────────┤  │
│  │  공지 카드 4        │  │
│  ├────────────────────┤  │
│  │  🔥 광고 (풀폭)     │  │
│  ├────────────────────┤  │
│  │  공지 카드 5        │  │
│  │  ...                │  │
│  └────────────────────┘  │
│                          │
├──────────────────────────┤
│  🏠홈  🔍동문찾기  💰기부  👤MY │  ← 하단 네비게이션 바 (4탭)
└──────────────────────────┘
```

**모바일 뷰:** 풀폭 단일 컬럼, 사이드 영역 없음

### 8.3 동문찾기 화면

```
┌─────────────────────────────────────┐
│  검색 필터                           │
│  [기수 ▼] [학과 ▼] [이름 입력]       │
│  [회사 입력] [직책 입력] [검색]       │
├─────────────────────────────────────┤
│                                     │
│  ┌────────────────────────────────┐ │
│  │ 홍길동 | 15기 | 컴퓨터공학과    │ │
│  │ 삼성전자 | 부장                 │ │
│  │ 📱 010-****-5678  📧 h***@... │ │
│  ├────────────────────────────────┤ │
│  │ 김철수 | 15기 | 경영학과        │ │
│  │ ...                            │ │
│  └────────────────────────────────┘ │
└─────────────────────────────────────┘
```
- FM_SMS='N' → 연락처 비공개 (마스킹)
- FM_SPAM='N' → 이메일 비공개 (마스킹)

---

## 9. 컴포넌트 구조 (React)

### 9.1 User SPA 라우팅

```
/                → FeedPage (메인/뉴스피드)
/post/:seq       → PostDetailPage (공지 상세 — HtmlContent로 렌더링)
/alumni          → AlumniPage (동문찾기, 로그인 필수)
/donation        → DonationPage (기부 현황 + 기부 결제 폼, 로그인 시 결제 가능)
/donation/result → DonationResultPage (결제 결과 — 성공/실패 안내)
/me              → MyPage (프로필 수정, 로그인 필수)
/login           → LoginPage (카카오 로그인)
/login/link      → AccountLinkPage (카카오 ↔ 기존 회원 연동)
```

### 9.2 User SPA 주요 컴포넌트 트리

```
App
├── Layout
│   ├── DesktopHeader          # 웹 뷰 상단 메뉴바 (≥768px)
│   │   ├── Logo
│   │   └── NavLinks (뉴스피드, 동문찾기, 기부, 마이페이지)
│   ├── MobileBottomNav        # 모바일 하단 바 (<768px)
│   │   └── NavButton × 4 (홈, 동문찾기, 기부, MY)
│   └── PageContent
│
├── FeedPage
│   ├── HeroSection            # 최신 공지 1건
│   │   ├── HeroImage
│   │   └── HeroContent (제목, 요약, 링크)
│   ├── DonationBanner         # 누적 기부액 + CTA
│   │   ├── AmountDisplay
│   │   ├── DonorCount
│   │   ├── ProgressBar (달성률)
│   │   └── DonateButton ("기부하기" → /donate 이동)
│   └── FeedList               # 무한스크롤
│       ├── NoticeCard         # 공지 카드 (반복)
│       │   ├── CardHeader (카테고리 라벨 + 게시 시간 + 고정 배지)
│       │   ├── CardImage (AspectRatio 썸네일)
│       │   ├── CardContent (제목 + 요약 2줄 clamp)
│       │   └── CardFooter (조회수 + 날짜)
│       └── AdCard             # 광고 카드 (4개마다 삽입)
│           ├── AdBadge ("가장 핫한 동문 소식" / "추천 동문 소식")
│           ├── AdContent
│           └── AdTracker (노출 시 POST /api/ad/{maSeq}/view)
│
├── PostDetailPage             # 공지 상세 보기
│   ├── PostHeader (제목, 작성자, 날짜, 조회수)
│   ├── HtmlContent            # ★ sanitized HTML 렌더링 (DOMPurify)
│   │   └── (인라인 이미지, 표, 코드블록 등 포함)
│   ├── FileAttachments (첨부파일 목록)
│   └── PostFooter (조회수, 이전/다음글)
│
├── DonationPage               # 기부 현황 + 결제 폼
│   ├── DonationSummary        # 기부 현황 (총액, 참여자, 달성률 프로그레스 바)
│   ├── DonationForm           # 기부 결제 폼 (로그인 시 표시)
│   │   ├── GateSelector       # 기부방식 선택 (일시후원 / 월정기후원)
│   │   ├── AmountSelector     # 금액 선택 (프리셋 버튼 + 직접입력)
│   │   ├── PayMethodSelector  # 결제수단 선택 (카드/계좌이체/휴대폰)
│   │   ├── OrderSummary       # 기부내역 확인 (선택 완료 후 표시)
│   │   └── SubmitButton       # "기부하기" 버튼
│   ├── EasyPayBridge          # Hidden form + SDK 로딩 + easypay_card_webpay() 호출
│   └── BankAccountInfo        # 기부 계좌 안내 (폴백)
│
├── DonationResultPage         # 결제 결과
│   ├── SuccessResult          # 성공: 체크마크, 금액, 감사 메시지
│   └── FailureResult          # 실패: 에러 사유, 재시도 버튼
│
├── AlumniPage
│   ├── SearchFilter           # 검색 조건
│   │   ├── SelectBox (기수, 학과)
│   │   └── TextInput (이름, 회사, 직책)
│   └── AlumniList
│       └── AlumniCard         # 동문 카드 (반복)
│           ├── BasicInfo (이름, 기수, 학과)
│           ├── WorkInfo (회사, 직책)
│           └── ContactInfo (마스킹 적용)
│
├── MyPage
│   └── ProfileEditForm
│       ├── AvatarUpload
│       ├── BasicFields (이름, 닉네임, 생년월일)
│       └── ContactFields (연락처, 이메일)
│
├── LoginPage
│   └── KakaoLoginButton
│
└── AccountLinkPage            # 카카오 ↔ 기존 회원 연동
    ├── LinkPrompt ("기존 회원이신가요?")
    └── VerificationForm (이름 + 전화번호 or 기수 + 이름)
```

### 9.3 핵심 커스텀 훅

```typescript
// 무한스크롤
useInfiniteScroll({ fetchFn, cursor, threshold })

// 반응형 뷰 감지
useResponsive() → { isMobile: boolean, isDesktop: boolean }

// 인증 상태
useAuth() → { user, isLoggedIn, login, logout }

// 광고 노출 추적 (Intersection Observer)
useAdImpression(maSeq: number) → { ref: RefObject }

// 기부 주문 생성 (React Query mutation)
useDonateOrder() → { mutateAsync, isPending, data, error }
```

### 9.4 Frontend 기술 선택

| 항목 | 선택 | 이유 |
|------|------|------|
| 빌드 | Vite | 빠른 HMR, ESM 네이티브 |
| 스타일 | Tailwind CSS v4 | CSS-first `@theme`, 유틸리티 퍼스트 |
| 클래스 유틸 | `clsx` + `tailwind-merge` | 조건부 클래스 결합, 중복 병합 |
| 컴포넌트 변형 | `class-variance-authority` (cva) | Button, Badge 등 variant/size 조합 관리 |
| 접근성 컴포넌트 | `Radix UI Primitives` | 개별 패키지 설치로 번들 최적화, 30+ 컴포넌트 |
| 아이콘 | `Lucide React` | 일관된 스트로크, 트리셰이킹 지원 |
| 상태 관리 | `Zustand` (로컬) + `@tanstack/react-query` (서버) | 경량, 캐싱/무효화 자동화 |
| HTML 렌더링 | `dompurify` | XSS 2차 방어 (서버 새니타이즈 이후 클라이언트 보강) |
| 폰트 | Pretendard (CDN), Inter | `font-display: swap`, jsdelivr CDN |

### 9.5 Admin SPA 라우팅

```
/admin                  → DashboardPage (대시보드)
/admin/notice           → NoticeListPage (공지 목록)
/admin/notice/new       → NoticeEditPage (공지 작성 — Markdown 에디터)
/admin/notice/:seq/edit → NoticeEditPage (공지 수정)
/admin/ad               → AdManagePage (광고 관리)
/admin/donation         → DonationConfigPage (기부 설정)
/admin/member           → MemberListPage (회원 관리)
/admin/member/:seq      → MemberDetailPage (회원 상세)
/admin/login            → AdminLoginPage (관리자 로그인)
```

### 9.6 Admin SPA 주요 컴포넌트 트리

```
AdminApp
├── AdminLayout
│   ├── AdminHeader            # 상단 바 (사이트명, 로그아웃, 사용자 사이트 링크)
│   ├── AdminSidebar           # 좌측 네비게이션
│   │   ├── NavItem (대시보드)
│   │   ├── NavItem (공지 관리)
│   │   ├── NavItem (광고 관리)
│   │   ├── NavItem (기부 설정)
│   │   └── NavItem (회원 관리)
│   └── AdminContent
│
├── DashboardPage              # 대시보드
│   ├── StatsCards             # 주요 지표 카드 (회원 수, 기부액, 방문자 등)
│   ├── RecentNotices          # 최근 공지 5건
│   └── QuickActions           # 빠른 작업 (글 쓰기, 광고 등록)
│
├── NoticeListPage             # 공지 목록
│   ├── NoticeToolbar (검색, 필터, 새 글 쓰기 버튼)
│   ├── NoticeTable            # 게시글 테이블
│   │   └── NoticeRow (제목, 작성일, 조회수, 상단고정, 수정/삭제)
│   └── Pagination
│
├── NoticeEditPage             # 공지 작성/수정 ★
│   ├── SubjectInput           # 제목 입력
│   ├── MarkdownEditor         # ★ Markdown 에디터 (react-md-editor 또는 toast-ui)
│   │   ├── Toolbar            # Bold, Italic, Link, Image, Code 등
│   │   ├── EditorPane         # Markdown 작성 영역
│   │   └── PreviewPane        # 실시간 HTML 미리보기 (split view)
│   ├── ImageUploader          # 이미지 업로드 (드래그앤드롭/붙여넣기)
│   │   └── (업로드 → URL 반환 → 에디터에 ![](url) 자동 삽입)
│   ├── NoticeOptions          # 옵션 (상단 고정, 카테고리)
│   └── SubmitActions          # 저장 / 미리보기 / 취소
│
├── AdManagePage               # 광고 관리
│   ├── AdList                 # 광고 목록 (등급, 상태, 노출수, 클릭수)
│   ├── AdForm                 # 광고 등록/수정 폼
│   │   ├── AdImageUpload      # 배너 이미지
│   │   ├── AdFields           # 이름, URL, 등급, 기간
│   │   └── AdPreview          # 피드 내 미리보기
│   └── AdStats                # 통계 차트 (일별 노출/클릭)
│
├── DonationConfigPage         # 기부 설정
│   ├── DonationConfigForm     # 목표액, 수동 조정 입력
│   ├── CurrentStatus          # 현재 기부 현황
│   └── SnapshotHistory        # 스냅샷 이력 (최근 30일 그래프)
│
├── MemberListPage             # 회원 관리
│   ├── MemberSearchBar        # 이름, 기수, 상태 필터
│   ├── MemberTable            # 회원 테이블
│   │   └── MemberRow (이름, 기수, 상태, 카카오 연동 여부, 최근 로그인)
│   └── Pagination
│
└── MemberDetailPage           # 회원 상세
    ├── MemberProfile          # 기본 정보
    ├── MemberLoginHistory     # 최근 로그인 이력
    └── MemberActions          # 상태 변경, 카카오 연동 해제
```

### 9.7 Admin Markdown 에디터 기술 선택

| 후보 | 번들 크기 | 특징 | 결정 |
|------|----------|------|------|
| `@uiw/react-md-editor` | ~160KB gzip | 심플, split view, 이미지 붙여넣기 지원 | ✅ **선택** |
| `@toast-ui/react-editor` | ~300KB gzip | 기능 풍부, WYSIWYG 전환 | 번들 크기 과대 |
| `react-simplemde-editor` | ~80KB gzip | 경량이나 커스텀 어려움 | 이미지 삽입 제한 |

> Admin SPA는 관리자만 접속하므로 번들 크기가 일반 사용자에게 영향을 주지 않습니다. 따라서 DX가 좋은 `@uiw/react-md-editor`를 선택합니다.

**에디터 내 이미지 업로드 통합:**

```tsx
// admin/src/components/editor/MarkdownEditor.tsx
import MDEditor from '@uiw/react-md-editor';

interface Props {
  value: string;
  onChange: (val: string) => void;
}

export function MarkdownEditor({ value, onChange }: Props) {
  // 이미지 붙여넣기/드래그앤드롭 핸들러
  const handleImageUpload = async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    const res = await fetch('/api/admin/upload?type=notice', {
      method: 'POST',
      body: formData,
      credentials: 'include',
    });
    const { url } = await res.json();
    return url;
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (file?.type.startsWith('image/')) {
      const url = await handleImageUpload(file);
      onChange(value + `\n![이미지](${url})\n`);
    }
  };

  const handlePaste = async (e: React.ClipboardEvent) => {
    const items = e.clipboardData.items;
    for (const item of items) {
      if (item.type.startsWith('image/')) {
        e.preventDefault();
        const file = item.getAsFile();
        if (file) {
          const url = await handleImageUpload(file);
          onChange(value + `\n![이미지](${url})\n`);
        }
      }
    }
  };

  return (
    <div onDrop={handleDrop} onPaste={handlePaste}>
      <MDEditor
        value={value}
        onChange={(val) => onChange(val || '')}
        height={500}
        preview="live"     // split view: 에디터 | 미리보기
      />
    </div>
  );
}
```

---
