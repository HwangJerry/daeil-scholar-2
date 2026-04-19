# UI Design System — dflh-saf-v2

Alumni community 차세대 웹 애플리케이션의 통합 UI 디자인 가이드라인입니다.
이 문서는 UI 관련 의사결정의 단일 참조점(Single Source of Truth)으로, **따뜻한 에디토리얼 스타일**을 기반으로 합니다.

---

## 1. 디자인 원칙

- **모바일 퍼스트**: 소셜 미디어 경험을 지향하며, 모바일 레이아웃을 기본으로 설계합니다.
- **웜 에디토리얼**: 다크 네이비 + 웜 오프화이트 계열의 따뜻한 에디토리얼 스타일을 적용합니다.
- **세리프 헤딩**: Pretendard로 헤딩, Instrument Sans로 본문을 구성하는 혼합 타이포그래피를 사용합니다.
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
| `cat-notice` | `#E8E5DF` (primary-light) | `#1a1a2e` (primary) | `#C8C5BF` (primary-muted) |

#### Hero Gradient

| 토큰 | 값 |
|------|-----|
| `--color-hero-from` | `#0f1b35` |
| `--color-hero-via` | `#1a2f5a` |
| `--color-hero-to` | `#1e3a6e` |

### 3.2 Typography

| 항목 | 값 |
|------|-----|
| 헤딩 폰트 | `Pretendard` (serif) — `font-serif` 클래스 |
| 본문 폰트 | `Instrument Sans` (sans) — `font-sans` 클래스 (기본) |
| `--font-sans` | `'Instrument Sans', 'Pretendard Variable', ui-sans-serif, system-ui, -apple-system, sans-serif` |
| `--font-serif` | `'Pretendard Variable', 'Pretendard', Georgia, serif` |
| 로딩 전략 | Instrument Sans: Google Fonts CDN / Pretendard: jsDelivr CDN (`display=swap`) |
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

#### Skeleton + Stagger 사용 패턴

스켈레톤 리스트에서는 각 행에 인라인 `animationDelay`를 사용해 shimmer 시작 타이밍을 분산시킨다
(동적 계산값이므로 `style={{}}` 예외 허용):

```tsx
{Array.from({ length: N }, (_, i) => (
  <div key={i} className="animate-fade-in-up" style={{ animationDelay: `${i * 0.05}s` }}>
    <RowSkeleton />
  </div>
))}
```

로드 완료 후 실제 콘텐츠는 `animate-fade-in-up` + `stagger-N` 클래스로 등장한다:

```tsx
const STAGGER_CLASSES = ['stagger-1', 'stagger-2', 'stagger-3', 'stagger-4', 'stagger-5'];

{items.map((item, i) => (
  <div key={item.id} className={`animate-fade-in-up ${STAGGER_CLASSES[i % 5]}`}>
    <ItemCard item={item} />
  </div>
))}
```

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

### 6.4 피드 카드 (FeedCard Compound Component)

`FeedCard` compound component가 공유 디자인 토큰을 강제하며, NoticeCard와 AdCard가 orchestrator로서 비즈니스 로직을 담당한다.

#### FeedCard Sub-components

| Sub-component | 역할 | 핵심 클래스 |
|---|---|---|
| `FeedCard` (Root) | `<article>` 컨테이너 | `rounded-2xl bg-surface border border-border-subtle shadow-card` + hover |
| `FeedCard.Body` | 패딩된 콘텐츠 영역 | `px-5 pt-5`, `hasImage ? 'pb-4' : 'pb-5'` |
| `FeedCard.Meta` | 메타 행 (children=좌측, action=우측) | `flex items-center justify-between mb-3` + `text-caption text-text-placeholder` |
| `FeedCard.MetaDot` | 메타 구분점 | `opacity-40` · |
| `FeedCard.Title` | 제목 `<h3>` | `line-clamp-2 text-body-md font-semibold font-serif` + `group-hover:text-primary` |
| `FeedCard.Description` | 본문/설명 텍스트 | `text-body-sm text-text-tertiary leading-relaxed line-clamp-3` |
| `FeedCard.Image` | `<img>` | `w-full aspect-video object-cover` |
| `FeedCard.Divider` | 구분선 | `border-t border-border-subtle mx-5` |
| `FeedCard.Footer` | Engagement 행 컨테이너 | `flex items-center px-5 py-2.5 text-xs text-text-placeholder` |

Root는 `ref` prop (impression 트래킹용)과 `className` prop (orchestrator 추가 스타일)을 지원한다.
`FeedCard.Title`의 `group-hover:text-primary`는 orchestrator가 링크에 `group` 클래스를 붙여야 동작한다.

#### NoticeCard (게시글 orchestrator)

 메타 행: `{카테고리} · {날짜}`, action=Pin 아이콘
 제목: `NoticeCardLink` + `FeedCard.Title`
 본문: `NoticeCardSummary` (expand/collapse 로직 보유, FeedCard.Description 미사용)
 하단: `FeedCard.Footer` 내 좋아요 + 댓글 (좌측) · 조회수 (`ml-auto`, 우측)

#### AdCard (광고 orchestrator)

 메타 행: `{스폰서명} · 광고 · {날짜}`, action=X(dismiss) 버튼
   "광고" 텍스트: `text-[10px] uppercase tracking-wider font-medium`
 제목/이미지: 외부 URL `<a>` 링크 (모달 대신)
 하단: `FeedCard.Footer` 내 좋아요 + 댓글 (좌측) · CTA 텍스트 링크 (`ml-auto`, 우측)
   CTA 링크: `text-body-sm font-medium text-text-tertiary hover:text-primary` + `ArrowRight` 아이콘(14px)

#### AdDetailModal (광고 상세 모달)

 스크롤 영역: `flex-1 overflow-y-auto` (이미지, 본문, 댓글 등)
 Sticky CTA 바: `border-t border-border-subtle p-4` (모달 최하단 고정)
   CTA 버튼: `w-full rounded-lg bg-primary text-white py-3 font-semibold` + `ArrowRight` 아이콘

### 6.5 AlumniCard (동문 카드)

- `rounded-[20px]` 카드
- 이름: `font-serif` bold
- hover: `border-border-hover` + 가벼운 shadow

### 6.6 ProfileHeader (프로필 헤더)

- 다크 네이비 그라디언트 배경 (HeroSection과 동일 팔레트)
- 이름: `font-serif` white

### 6.7 Skeleton Loading Patterns

모든 데이터 패칭 UI에 skeleton을 일관 적용한다. spinner는 load-more/pagination 재조회에만 유지.

#### `Bone` Primitive

`src/components/ui/Skeleton.tsx`의 `Bone` 컴포넌트를 사용한다:

```tsx
import { Bone } from '../ui/Skeleton';

// 사용 예
<Bone className="h-4 w-32" />               // 텍스트 라인
<Bone className="h-8 w-8 rounded-full" />   // 아바타 원형
<Bone className="h-5 w-14 rounded-full" />  // 필 배지
<Bone className="h-10 rounded-lg" />        // 버튼
```

#### `isLoading` vs `isFetching` 판단 기준

| 상태 | 언제 | 처리 |
|------|------|------|
| `isLoading` (`status === 'pending'`) | 캐시 미스 초기 1회 | Skeleton 표시 |
| `isFetching && data` | 페이지네이션·백그라운드 재조회 | 기존 데이터 유지 + 미니 스피너 |

```tsx
const { data, isLoading, isFetching } = useQuery({ ... });

if (isLoading) return <MySkeleton />;
// 이후 렌더: data 표시 + isFetching 시 미니 스피너
```

#### Row Skeleton 패턴 (테이블/리스트)

AlumniPage의 `AlumniTableSkeleton`처럼, 실제 행 그리드와 동일한 `grid-cols-[...]`를 사용해 형태를 충실히 흉내낸다:

```tsx
function RowSkeleton({ isLast }: { isLast: boolean }) {
  return (
    <div className={`grid items-center px-5 py-3.5 gap-2 grid-cols-[44px_1fr_88px] ${!isLast ? 'border-b border-border-subtle' : ''}`}>
      <Bone className="w-8 h-8 rounded-full" />
      <div className="space-y-1.5">
        <Bone className="h-3.5 w-24" />
        <Bone className="h-2.5 w-20" />
      </div>
      <Bone className="h-5 w-14 rounded-full" />
    </div>
  );
}
```

#### Card Skeleton 패턴

FeedCardSkeleton처럼 카드 내부 구조를 흉내낸다:

```tsx
function CardSkeleton() {
  return (
    <div className="rounded-2xl bg-surface border border-border-subtle p-4 space-y-3">
      <Bone className="h-4 w-16 rounded-full" />  {/* 배지 */}
      <div className="space-y-2">
        <Bone className="h-4 w-4/5" />             {/* 제목 1줄 */}
        <Bone className="h-4 w-3/5" />             {/* 제목 2줄 */}
      </div>
      <Bone className="h-3 w-full" />              {/* 본문 */}
      <Bone className="h-3 w-16" />                {/* 메타 */}
    </div>
  );
}
```

#### Banner Skeleton 패턴

ProfileHeaderSkeleton처럼 배너 색상은 실제 그라디언트를 유지하고, 아바타/텍스트 영역만 Bone으로 대체한다:

```tsx
function BannerSkeleton() {
  return (
    <>
      <div className="relative h-32 bg-gradient-to-br from-hero-from via-hero-via to-hero-to overflow-hidden">
        <div className="absolute -bottom-12 left-4">
          <Bone className="w-24 h-24 rounded-full ring-4 ring-surface" />
        </div>
      </div>
      <div className="px-4 pt-14 space-y-2">
        <Bone className="h-6 w-32" />
        <Bone className="h-4 w-16" />
      </div>
    </>
  );
}
```

---

## 7. Admin SPA 디자인

Admin SPA는 별도 디자인 토큰을 유지합니다 (`admin/src/index.css`).
User SPA의 웜 에디토리얼 스타일과 별개로 관리됩니다.
