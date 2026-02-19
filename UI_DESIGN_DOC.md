# UI Design System — dflh-saf-v2

Alumni community 차세대 웹 애플리케이션의 통합 UI 디자인 가이드라인입니다.
이 문서는 UI 관련 의사결정의 단일 참조점(Single Source of Truth)으로, **따뜻한 에디토리얼 스타일**을 기반으로 합니다.

---

## 1. 디자인 원칙

- **모바일 퍼스트**: 소셜 미디어 경험을 지향하며, 모바일 레이아웃을 기본으로 설계합니다.
- **웜 에디토리얼**: 다크 네이비 + 웜 오프화이트 계열의 따뜻한 에디토리얼 스타일을 적용합니다.
- **세리프 헤딩**: Noto Serif KR로 헤딩, DM Sans로 본문을 구성하는 혼합 타이포그래피를 사용합니다.
- **20px 라운드**: 메인 카드 20px, 피드 카드 16px radius를 기본으로 합니다.

---

## 절대 규칙

- **인라인 `style={{}}` 절대 사용 금지** — 동적 계산값(예: 프로그레스 바 width %)은 예외.

- **색상은 `index.css` `@theme` 토큰만 사용** — `bg-[#xxx]`, `text-indigo-600` 등 임의 색상·Tailwind 기본 팔레트 직접 사용 금지. 새 색상이 필요하면 `@theme` 블록에 토큰을 먼저 추가한다.

- **새 컴포넌트를 만들기 전에 `src/components/ui/` 기존 컴포넌트 먼저 확인** — Button, Card, Badge, Input 등 기존 CVA 컴포넌트를 재사용한다.

- **클래스 결합은 반드시 `cn()` 유틸리티 사용** (`src/lib/utils.ts`).

- **반응형은 모바일 퍼스트** — 기본값이 모바일 레이아웃이며, `md:` → `lg:` 순서로 확장한다.

- **아이콘은 `lucide-react`만 사용**.

- **HTML 렌더링은 반드시 `HtmlContent` 컴포넌트 경유** (`src/components/common/HtmlContent.tsx`).

- **애니메이션은 `index.css`의 `@utility` 클래스만 사용**.

---

## 2. 기술 스택

| 도구 | 용도 | 비고 |
|------|------|------|
| `Tailwind CSS v4` | 스타일 시스템 | CSS-first `@theme` 설정, CSS 변수 기반 |
| `clsx` + `tailwind-merge` | 조건부 클래스 결합 | 중복 클래스 병합 |
| `class-variance-authority` (CVA) | 컴포넌트 변형 관리 | Button, Card, Badge 등 variant/size 조합 |
| `Lucide React` | 아이콘 | 일관된 스트로크 두께, 트리셰이킹 지원 |
| `Radix UI Primitives` | 접근성 컴포넌트 | Dialog, DropdownMenu, Tabs, Popover 등 개별 패키지 설치 |

---

## 3. 디자인 토큰

### 3.1 Color Palette

`frontend/src/index.css` `@theme` 블록에서 정의합니다.

#### Primary — 다크 네이비

| 토큰 | 값 | 용도 |
|------|-----|------|
| `--color-primary` | `#1a1a2e` (다크 네이비) | 브랜드 아이덴티티, 주요 버튼, 활성 상태 |
| `--color-primary-hover` | `#0f1b35` | Primary hover 상태 |
| `--color-primary-light` | `#E8E5DF` | Primary 배경 (웜 오프화이트) |
| `--color-primary-muted` | `#C8C5BF` | 비활성/약한 Primary |

#### Surface / Background — 웜 톤

| 토큰 | 값 | 용도 |
|------|-----|------|
| `--color-surface` | `#FAFAF8` (웜 화이트) | 카드, 내비게이션 바 등 컨텐츠 영역 |
| `--color-background` | `#F5F4F0` (웜 오프화이트) | 앱 배경색 (카드와 구분) |
| `--color-border` | `#E8E5DF` (웜 그레이) | 기본 border |
| `--color-border-subtle` | `#F0EDE8` | 약한 border (카드 등) |
| `--color-border-hover` | `#C8C5BF` | hover 시 border |

#### Text

| 토큰 | 값 | 용도 |
|------|-----|------|
| `--color-text-primary` | `#1a1a2e` | 제목, 헤딩 |
| `--color-text-secondary` | `#555555` | 본문 텍스트 |
| `--color-text-tertiary` | `#888888` | 보조 텍스트 |
| `--color-text-placeholder` | `#AAAAAA` | 입력 필드 placeholder |

#### 카테고리 배지

| 토큰 | 배경 | 텍스트 | 테두리 |
|------|------|--------|--------|
| `cat-notice` | `#EFF2FF` | `#3B5BDB` | `#C5D0FA` |
| `cat-event` | `#FFF9DB` | `#E67700` | `#FFE066` |
| `cat-scholarship` | `#EBFBEE` | `#2B8A3E` | `#8CE99A` |
| `cat-career` | `#F3F0FF` | `#6741D9` | `#D0BFFF` |

#### Hero Gradient

| 토큰 | 값 |
|------|-----|
| `--color-hero-from` | `#0f1b35` |
| `--color-hero-via` | `#1a2f5a` |
| `--color-hero-to` | `#1e3a6e` |

### 3.2 Typography

| 항목 | 값 |
|------|-----|
| 헤딩 폰트 | `Noto Serif KR` (serif) — `font-serif` 클래스 |
| 본문 폰트 | `DM Sans` (sans) — `font-sans` 클래스 (기본) |
| `--font-sans` | `'DM Sans', ui-sans-serif, system-ui, -apple-system, sans-serif` |
| `--font-serif` | `'Noto Serif KR', Georgia, serif` |
| 로딩 전략 | Google Fonts CDN (`display=swap`) |
| 기본 본문 크기 | `text-sm` (14px) |
| 헤딩 letter-spacing | `-0.015em` |

### 3.3 Border Radius

| 토큰 | 값 | 용도 |
|------|-----|------|
| `--radius-sm` | `0.5rem` (8px) | 소형 요소 |
| `--radius-md` | `0.75rem` (12px) | 중형 요소 |
| `--radius-lg` | `1rem` (16px) | 피드 카드 |
| `--radius-xl` | `1.25rem` (20px) | 메인 카드 |
| `--radius-2xl` | `1.5rem` (24px) | 대형 모달 |
| `--radius-full` | `9999px` | 원형/필 배지 |

### 3.4 Shadows

| 토큰 | 용도 |
|------|------|
| `--shadow-xs` | 미세한 깊이감 |
| `--shadow-card` | 기본 카드 그림자 |
| `--shadow-card-hover` | 카드 hover 시 |
| `--shadow-primary-glow` | Primary 버튼 glow (네이비 기반) |
| `--shadow-nav` | 내비게이션 바 |
| `--shadow-float` | 플로팅 요소 |

### 3.5 Animations

| 유틸리티 클래스 | 애니메이션 |
|----------------|-----------|
| `animate-fade-in-up` | 아래에서 위로 페이드인 |
| `animate-fade-in` | 페이드인 |
| `animate-progress` | 프로그레스 바 채움 |
| `animate-fill-bar` | 프로그레스 바 (delay 0.5s) |
| `skeleton-shimmer` | 스켈레톤 로딩 시머 |
| `stagger-1` ~ `stagger-5` | 순차 딜레이 (50ms~250ms) |

---

## 4. 레이아웃 시스템

### 4.1 App Shell (반응형)

| 뷰포트 | 레이아웃 | 내비게이션 |
|---------|---------|-----------|
| **Mobile** (`<768px`) | 풀 너비 | `BottomNav` — Home, People, Donate, Message, Me (5탭) |
| **Desktop** (`≥768px`) | 중앙 정렬 (`max-w-[1080px]`) | `TopNav` 상단 헤더 |

### 4.2 Feed 2컬럼 레이아웃 (데스크탑)

- 좌: 고정 공지(HeroSection) + 피드 리스트 (`1fr`)
- 우: DonationBanner(sticky) + NetworkWidget(sticky) (`360px`)

---

## 5. 공유 UI 컴포넌트

### 5.1 Button (CVA)

| variant | 스타일 | 용도 |
|---------|--------|------|
| `default` | `bg-primary text-white` (네이비) + hover glow | 주요 CTA |
| `destructive` | `bg-error text-white` | 삭제/위험 액션 |
| `outline` | `border border-border bg-surface` | 보조 액션 |
| `secondary` | `bg-primary-light text-primary` | 부차적 액션 |
| `ghost` | `hover:bg-background` | 최소 강조 |
| `link` | `underline text-primary` | 인라인 링크 |

### 5.2 Card (CVA)

`rounded-xl` (20px) 기반. variant: `default`, `elevated`, `interactive`, `ghost`.

### 5.3 Badge (CVA)

`rounded-full` 필 형태. 카테고리 배지는 `cat-*` 토큰 사용.

---

## 6. 핵심 UI 패턴

### 6.1 HeroPinnedCard (고정 공지 카드)

- 다크 네이비 그라디언트 배경 (`from-hero-from via-hero-via to-hero-to`)
- Ambient glow (radial gradient 장식)
- 제목: `font-serif` 흰색 24px
- `min-h-[260px]`, `rounded-[20px]`

### 6.2 DonationWidget (사이드바 기부 현황)

- 배경: `bg-surface border-border`
- `rounded-[20px]`, `p-7`
- 프로그레스 바: `bg-border` 트랙, 다크 네이비 그라디언트 필
- `animate-fill-bar` 애니메이션

### 6.3 NetworkWidget (동문 현황 위젯)

- 동문 아바타 스택 + 총 동문 수
- 통계 그리드 (전체 동문, 이번 주 가입)
- "동문 찾기 →" 버튼

### 6.4 NoticeCard (피드 카드)

- `bg-surface border-border rounded-2xl` (16px)
- 카테고리 배지: `rounded-full` 필 형태 + 카테고리별 색상
- 제목: `font-serif` 15px
- hover: `border-border-hover` + 화살표 아이콘 나타남

### 6.5 AlumniCard (동문 카드)

- `rounded-[20px]` 카드
- 이름: `font-serif` bold
- hover: `border-border-hover` + 가벼운 shadow

### 6.6 ProfileHeader (프로필 헤더)

- 다크 네이비 그라디언트 배경 (HeroSection과 동일 팔레트)
- 이름: `font-serif` white

---

## 7. Admin SPA 디자인

Admin SPA는 별도 디자인 토큰을 유지합니다 (`admin/src/index.css`).
User SPA의 웜 에디토리얼 스타일과 별개로 관리됩니다.
