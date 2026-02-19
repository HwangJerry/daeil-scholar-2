# 인프라 및 아키텍처 설계 (§2, §3, §12)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

---

## 2. MyISAM → InnoDB 전환 리스크 검토

### 2.1 전환 대상 테이블

현재 MyISAM을 사용하는 테이블:
- `FUNDAMENTAL_MEMBER`, `FUNDAMENTAL_NUMBERS`
- `WEO_BOARDBBS`, `WEO_BOARDCOMAND`, `WEO_BOARDLIKE`
- `WEO_MEMBER`, `WEO_MEMBER_LOG`, `WEO_MEMBER_PUSH`, `WEO_MEMBER_SOCIAL`
- `WEO_OPERATOR`, `WEO_OPERATOR_LOG`, `WEO_OPERATOR_PAGE_LOG`
- `WEO_ORDER`, `WEO_PG_DATA`
- `WEO_FILES`, `WEO_CODE`, `WEO_CONTENTS`, `WEO_JOBCODE`, `WEO_JOBCODE_GRANT`
- `MAIN_AD`, `WEO_POPUP`, `WEO_BOOKMARK`
- `WEO_VISIT_*` (4개), `WEO_PUSH_NOTI`

이미 InnoDB인 테이블: `WEO_APP_PUSH`, `WEO_APP_PUSHSAND`, `WEO_MEMBER_OUT`, `WEO_ORDER_PROFILE`, `WEO_SMS`, `WEO_SMSSAND`, `WEO_VISIT_LOGIN`

### 2.2 리스크 항목

| 리스크 | 심각도 | 설명 | 대응 방안 |
|--------|--------|------|-----------|
| **FULLTEXT 인덱스 비호환** | 🟡 중 | MyISAM의 FULLTEXT 인덱스가 있다면 InnoDB 전환 시 재생성 필요 (MySQL 5.6+에서는 InnoDB도 FULLTEXT 지원) | 현재 DDL에 FULLTEXT 인덱스 없음 → 리스크 없음 |
| **디스크 사용량 증가** | 🟡 중 | InnoDB는 MyISAM 대비 1.5~2배 디스크 사용. 50GB HDD에서 여유 확인 필요 | 현재 DB 용량 확인 후 판단. 통계 테이블(`WEO_VISIT_*`)은 용량이 클 수 있음 |
| **메모리 사용량 증가** | 🔴 높 | InnoDB는 buffer pool이 필요 (권장: 전체 RAM의 50~70%). **1GB RAM 서버에서 치명적** | `innodb_buffer_pool_size`를 256MB로 제한. Go + React 빌드 서버 동시 운영 시 OOM 위험 |
| **테이블 잠금 → 행 잠금** | 🟢 낮 | InnoDB는 행 단위 잠금으로 동시성 향상. 오히려 개선 | 긍정적 변화 |
| **AUTO_INCREMENT 동작 차이** | 🟡 중 | InnoDB는 서버 재시작 시 MAX(id)+1로 리셋. **MariaDB 10.1에서는 persistent counter 미지원** → ID 재사용 가능 | 신규 InnoDB 테이블의 ID를 외부 참조로 사용하지 않음. MariaDB 10.3+에서 해결 |
| **COUNT(*) 성능 저하** | 🟡 중 | MyISAM은 행 수를 메타데이터로 관리, InnoDB는 풀스캔 필요 | 통계/목록 페이지에서 `COUNT(*)` 대신 추정값 사용 또는 캐싱 |
| **외래키 제약조건** | 🟢 낮 | InnoDB 전환 후 FK 설정 가능하지만, 기존 데이터 정합성 문제 가능 | FK는 애플리케이션 레벨에서 유지 (기존 방식). 무리하게 DB FK 추가 안 함 |
| **전환 중 다운타임** | 🟡 중 | `ALTER TABLE ... ENGINE=InnoDB` 실행 시 테이블 잠금 발생 | 점검 시간 확보 (새벽). 큰 테이블(`WEO_VISIT_LOG` 등)은 시간 소요 |
| **PHP 레거시 호환성** | 🔴 높 | 전환 기간 동안 PHP와 Go가 동시에 같은 DB 접근. 트랜잭션 격리 수준 차이로 예기치 않은 동작 가능 | **단계적 전환 권장**: 먼저 Go 백엔드 완성 → PHP 완전 교체 후 → InnoDB 전환 |

### 2.3 권장 전략

```
Phase 1 (현재): MariaDB 10.1 + MyISAM 그대로 유지
    → Go 백엔드 개발, React 프론트엔드 개발
    → PHP 레거시와 공존 (인증 브릿지로 연결)

Phase 2 (PHP 완전 교체 후): MyISAM → InnoDB 전환
    → 서버 메모리 증설 검토 (최소 2GB 권장)
    → 점검 시간에 ALTER TABLE 실행
    → 통계 테이블(WEO_VISIT_*)은 데이터 규모에 따라 전환 여부 별도 판단

Phase 3 (선택): MariaDB → PostgreSQL 전환
    → Phase 2 안정화 이후 장기 로드맵으로 검토
```

**결론: Phase 1에서는 DB 변경 최소화. 신규 테이블만 InnoDB로 생성.**

### 2.4 MariaDB 10.1.38 제약사항 분석

현재 운영 DB가 **MariaDB 10.1.38** (2017년 릴리스, 2020년 EOL)로 확인되었습니다. Go 백엔드 SQL 작성 시 반드시 고려해야 할 제약사항입니다.

#### DB 엔진 정보

| 항목 | 값 |
|------|-----|
| 버전 | MariaDB 10.1.38 |
| EOL | 2020-10 (공식 지원 종료) |
| 기본 charset | utf8 (utf8mb4 지원은 되나 테이블별 확인 필요) |
| Go 드라이버 | `go-sql-driver/mysql` (MariaDB 호환) |

#### SQL 호환성 가이드라인

| 기능 | 지원 여부 | 대안 |
|------|----------|------|
| CTE (`WITH ... AS`) | ❌ 10.2부터 | 서브쿼리로 대체 |
| Window Functions (`ROW_NUMBER`, `RANK`) | ❌ 10.2부터 | `@변수` + `ORDER BY` 또는 `LIMIT` |
| JSON 컬럼 타입 | ❌ 10.2부터 | `TEXT` + 애플리케이션 레벨 JSON 파싱 |
| `DEFAULT` 표현식 (datetime) | ❌ 10.2부터 | 트리거 또는 애플리케이션에서 `NOW()` 설정 |
| System-versioned Tables | ❌ 10.3부터 | 수동 이력 관리 |
| `INSERT ... ON DUPLICATE KEY UPDATE` | ✅ | 스냅샷 UPSERT에 사용 가능 |
| `LIMIT` in subqueries | ✅ | — |
| FULLTEXT on InnoDB | ✅ 10.0.5부터 | — |
| `utf8mb4` | ✅ | 신규 테이블에 적용 |

#### Go SQL 작성 규칙

```go
// ❌ MariaDB 10.1에서 사용 불가
WITH recent AS (
    SELECT * FROM WEO_BOARDBBS WHERE GATE='NOTICE' ORDER BY SEQ DESC LIMIT 10
)
SELECT * FROM recent;

// ✅ 서브쿼리로 대체
SELECT * FROM (
    SELECT * FROM WEO_BOARDBBS WHERE GATE='NOTICE' ORDER BY SEQ DESC LIMIT 10
) AS recent;

// ❌ Window Function 사용 불가
SELECT *, ROW_NUMBER() OVER (PARTITION BY FM_FN ORDER BY FM_NAME) AS rn
FROM FUNDAMENTAL_MEMBER;

// ✅ LIMIT + OFFSET으로 대체 (페이지네이션)
SELECT * FROM FUNDAMENTAL_MEMBER WHERE FM_FN = ? ORDER BY FM_NAME LIMIT ? OFFSET ?;
```

#### InnoDB AUTO_INCREMENT 주의사항

MariaDB 10.1에서 InnoDB는 AUTO_INCREMENT 카운터가 **메모리에만 유지**됩니다. 서버 재시작 시 `MAX(id)+1`로 리셋되므로, 중간에 삭제된 레코드의 ID가 재사용될 수 있습니다. 신규 InnoDB 테이블(`DONATION_SNAPSHOT`, `USER_SESSION`, `WEO_AD_LOG`)에서 ID를 외부 참조(URL 등)로 사용하지 않도록 주의합니다.

#### 버전 업그레이드 권장

MariaDB 10.1은 보안 패치가 중단된 상태입니다. Phase 2(PHP 제거 후)에서 **MariaDB 10.6 LTS** (2026년까지 지원) 또는 **10.11 LTS** (2028년까지 지원)로 업그레이드를 강력 권장합니다. 업그레이드 시 CTE, Window Function 등을 활용하여 쿼리 최적화가 가능해집니다.

```
Phase 1 (현재): MariaDB 10.1.38 유지 → SQL 호환성 가이드라인 준수
Phase 2: MariaDB 10.6+ LTS 업그레이드 (InnoDB 전환과 동시 진행 권장)
Phase 3: PostgreSQL 전환 검토 (선택)
```

---

## 3. 아키텍처 설계

### 3.1 시스템 아키텍처 (단일 서버)

```
┌──────────────────────────────────────────────────────────────┐
│                     Gabia Cloud Server                       │
│                  (1 Core, 1GB RAM, 50GB HDD)                │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐    │
│  │                      Nginx                            │    │
│  │                   :80 → :443 redirect                 │    │
│  │                   :443 (Let's Encrypt)                │    │
│  │                                                       │    │
│  │   /              → User SPA (/var/www/app)            │    │
│  │   /admin/*       → Admin SPA (/var/www/admin)         │    │
│  │   /api/*         → Go Backend (127.0.0.1:8080)        │    │
│  │   /pg/*          → Go Backend (PG 결제 콜백)          │    │
│  │   /legacy/*      → PHP-FPM (127.0.0.1:9000)          │    │
│  │   /files/*       → 기존 업로드 디렉토리 직접 서빙       │    │
│  │   /uploads/*     → 신규 업로드 디렉토리 직접 서빙       │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │ Go Backend   │  │  PHP-FPM     │  │   MariaDB 10.1   │   │
│  │ :8080        │  │  :9000       │  │     :3306        │   │
│  │              │◀─┤─(인증 브릿지)─┤─▶│  (기존 유지)     │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
│                                                              │
│  certbot (Let's Encrypt 자동 갱신; cron 매월)                │
└──────────────────────────────────────────────────────────────┘
```

> **User SPA vs Admin SPA 분리:** 관리자 웹은 Markdown 에디터 등 무거운 의존성을 포함하므로 별도 빌드로 분리합니다. 일반 사용자는 관리자 코드를 다운로드하지 않습니다.

### 3.2 HTTPS / 도메인 설정

```nginx
# /etc/nginx/conf.d/rate_limit.conf (http {} 블록 레벨)
# ⚠️ limit_req_zone은 반드시 http {} 블록에 선언해야 함 (server {} 내부 불가)
limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s;
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;
```

```nginx
# /etc/nginx/conf.d/alumni.conf

# HTTP → HTTPS 강제 리다이렉트
server {
    listen 80;
    server_name alumni.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name alumni.example.com;

    # Let's Encrypt 인증서
    ssl_certificate     /etc/letsencrypt/live/alumni.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/alumni.example.com/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;

    # User SPA (일반 사용자)
    location / {
        root /var/www/app;
        try_files $uri $uri/ /index.html;
        gzip on;
        gzip_types text/css application/javascript application/json;
    }

    # Admin SPA (관리자 전용)
    location /admin {
        alias /var/www/admin;
        try_files $uri $uri/ /admin/index.html;
        gzip on;
        gzip_types text/css application/javascript application/json;
    }

    # Go API
    location /api/ {
        limit_req zone=api burst=50 nodelay;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # PG 결제 콜백 (EasyPay → Go 백엔드)
    # ⚠️ CSRF 미들웨어 제외 대상: EasyPay의 cross-origin form POST를 수신
    location /pg/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Go 로그인 엔드포인트 (더 엄격한 rate limit)
    location /api/auth/ {
        limit_req zone=login burst=3 nodelay;
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # PHP Legacy (공존 기간)
    location /legacy/ {
        fastcgi_pass 127.0.0.1:9000;
        include fastcgi_params;
    }

    # 기존 파일 서빙 (PHP 시절 업로드된 파일)
    location /files/ {
        alias /var/www/legacy/files/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # 신규 파일 서빙 (Go에서 업로드)
    location /uploads/ {
        alias /var/www/uploads/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

**Let's Encrypt 자동 갱신:**
```bash
# certbot 설치 및 초기 발급
sudo certbot --nginx -d alumni.example.com

# cron 자동 갱신 (이미 certbot이 설정하지만, 명시적으로)
# /etc/cron.d/certbot
0 3 1 * * root certbot renew --quiet --post-hook "systemctl reload nginx"
```

### 3.3 메모리 예산 및 OOM 방어 (1GB 서버)

#### 정상 상태 메모리 분배

| 프로세스 | 할당 | 설정 |
|----------|------|------|
| OS + 시스템 | ~200MB | — |
| MariaDB (buffer pool) | ~256MB | `innodb_buffer_pool_size=256M` |
| Go Backend | ~80MB | `GOMEMLIMIT=80MiB`, `GOMAXPROCS=1` |
| Nginx | ~30MB | `worker_processes 1`, `worker_connections 512` |
| PHP-FPM (공존 기간) | ~150MB | `pm.max_children=3` |
| 여유 | ~280MB | — |

#### 스파이크 시나리오 분석

| 시나리오 | 예상 동시 접속 | 대응 |
|----------|---------------|------|
| 평상시 | 5~10명 | 정상 운영 |
| 총회/행사 공지 | 50~100명 | Nginx rate limit로 큐잉, Go가 순차 처리 |
| 바이럴 공유 | 200명+ | `limit_req` 초과분 503 반환, 정적 파일은 Nginx 직접 서빙으로 Go 부하 감소 |

#### OOM Killer 보호 설정

```bash
# MariaDB를 OOM Killer로부터 보호 (가장 중요한 프로세스)
echo -1000 > /proc/$(pidof mysqld)/oom_score_adj

# Go 백엔드는 중간 우선순위
echo 0 > /proc/$(pidof alumni-server)/oom_score_adj

# PHP-FPM은 희생 우선 (공존 기간)
echo 300 > /proc/$(pidof php-fpm)/oom_score_adj
```

**systemd 서비스에 영구 설정:**
```ini
# /etc/systemd/system/alumni-backend.service
[Service]
Environment="GOMEMLIMIT=80MiB"
Environment="GOMAXPROCS=1"
OOMScoreAdjust=0
MemoryMax=120M
```

#### Go 커넥션 풀 설정

```go
db.SetMaxOpenConns(10)   // 동시 DB 연결 최대 10개
db.SetMaxIdleConns(5)    // 유휴 연결 5개 유지
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(3 * time.Minute)
```

> ⚠️ **1GB RAM은 매우 타이트합니다.** PHP 완전 제거 후에도 여유가 부족합니다. 최소 2GB 증설을 강력 권장합니다.

### 3.4 디렉토리 구조

```
/project-root
├── frontend/                  # User SPA — React (Vite)
│   ├── src/
│   │   ├── components/
│   │   │   ├── layout/        # Header, BottomNav, Sidebar
│   │   │   ├── feed/          # FeedCard, HeroSection, DonationBanner, AdCard
│   │   │   ├── alumni/        # AlumniSearch, AlumniCard
│   │   │   ├── profile/       # ProfileEdit
│   │   │   └── common/        # HtmlContent (sanitized HTML 렌더링)
│   │   ├── pages/
│   │   ├── hooks/             # useInfiniteScroll, useResponsive
│   │   ├── api/               # API client
│   │   └── store/             # Zustand or Context
│   └── public/
│
├── admin/                     # Admin SPA — React (Vite), 별도 빌드
│   ├── src/
│   │   ├── components/
│   │   │   ├── layout/        # AdminHeader, AdminSidebar
│   │   │   ├── editor/        # MarkdownEditor, ImageUploader, ContentPreview
│   │   │   ├── notice/        # NoticeForm, NoticeList, NoticeDetail
│   │   │   ├── ad/            # AdForm, AdList, AdStats
│   │   │   ├── donation/      # DonationConfigForm, DonationHistory
│   │   │   ├── member/        # MemberList, MemberDetail
│   │   │   └── dashboard/     # StatsCards, RecentActivity
│   │   ├── pages/
│   │   │   ├── DashboardPage.tsx
│   │   │   ├── NoticeListPage.tsx
│   │   │   ├── NoticeEditPage.tsx
│   │   │   ├── AdManagePage.tsx
│   │   │   ├── DonationConfigPage.tsx
│   │   │   └── MemberListPage.tsx
│   │   ├── hooks/
│   │   └── api/               # Admin API client
│   └── public/
│
├── backend/                   # Go
│   ├── cmd/server/            # main.go
│   ├── internal/
│   │   ├── handler/
│   │   │   ├── feed.go        # 피드 조회 (User)
│   │   │   ├── alumni.go      # 동문찾기 (User)
│   │   │   ├── auth.go        # 인증 (공통)
│   │   │   ├── profile.go     # 프로필 (User)
│   │   │   ├── admin_notice.go   # 공지 CRUD (Admin)
│   │   │   ├── admin_ad.go       # 광고 관리 (Admin)
│   │   │   ├── admin_donation.go # 기부 설정 (Admin)
│   │   │   ├── admin_member.go   # 회원 관리 (Admin)
│   │   │   ├── admin_upload.go   # 이미지 업로드 (Admin)
│   │   │   └── health.go
│   │   ├── service/
│   │   │   ├── content.go     # Markdown→HTML 변환 + 인코딩/디코딩
│   │   │   └── ...
│   │   ├── repository/
│   │   ├── model/
│   │   ├── middleware/        # Auth, AuthBridge, CORS, CSRF, AdminAuth, Logger
│   │   └── config/
│   ├── migrations/
│   └── go.mod
│
└── nginx/
    └── conf.d/
```

### 3.5 UI 디자인 시스템 (Frontend)

소셜 미디어 경험을 제공하기 위해 **Tailwind CSS v4** 기반의 모바일 퍼스트 디자인 시스템을 적용합니다.

#### 스타일 시스템 (Tailwind v4 Theme)

| 항목 | 값 (CSS Variable) | 설명 |
|------|-------------------|------|
| **Color: Primary** | `#4F46E5` (Royal Indigo) | 브랜드 아이덴티티, 주요 버튼, 활성 상태 (신뢰+세련) |
| **Color: Background** | `#F8FAFC` (Pale Slate) | 앱 배경색 (카드와 구분) |
| **Color: Surface** | `#FFFFFF` (White) | 카드, 내비게이션 바 등 컨텐츠 영역 |
| **Typography** | `Pretendard`, `Inter` | 모바일 가독성 최적화 (Sans-serif). CDN 로딩 + `font-display: swap` |
| **Border Radius** | `rounded-2xl` (16px) | iOS 스타일의 부드러운 곡선 |

> **폰트 로딩 전략:** Pretendard는 서브셋(~400KB woff2)을 `cdn.jsdelivr.net/gh/orioncactus/pretendard`에서 로딩합니다. 1GB 서버 자체 호스팅 대역폭 부담을 피하며, `font-display: swap`으로 FOUT를 허용합니다.

#### 핵심 UI 패턴

**1. App Shell (반응형 레이아웃)**

- **Mobile (<768px)**: 하단 고정 탭 바 (`BottomNav`) — Home, People, Donate, Me (4탭)
- **Desktop (≥768px)**: 상단 헤더 (`TopNav`) + 중앙 정렬 컨테이너 (`max-w-3xl`)

**2. Feed Card (v1.0 — 공지 피드 단위)**

공지글은 **관리자가 작성하는 콘텐츠**이므로, 개인 소셜 미디어와 달리 작성자 프로필을 표시하지 않습니다.

| 영역 | 구성 |
|------|------|
| **Header** | 카테고리 라벨 + 게시 시간 + 상단고정 배지 |
| **Media** | 16:9 또는 1:1 비율의 `AspectRatio` 썸네일 이미지 |
| **Content** | 제목(bold) + 요약 텍스트(2줄 clamp) |
| **Footer** | 조회수(Eye 아이콘) + 날짜 |

> **v1.1 확장:** 좋아요(Heart), 댓글(MessageCircle), 공유(Share) Action Bar는 v1.1 디자인 시스템에서 추가합니다. (§13 로드맵 참조)

**3. Donation Banner (인피드 기부 배너)**

피드 내에 기부 현황 + CTA(Call-to-Action) 버튼이 포함된 배너를 배치합니다.

| 영역 | 구성 |
|------|------|
| **Status** | 누적 기부액 (₩130,000,000) + 참여자 수 (342명) |
| **Progress** | 달성률 프로그레스 바 (65%) |
| **CTA** | "기부하기" 버튼 (Primary 컬러, 외부 기부 페이지 링크) |

> 현재 기부 기능은 외부 링크로 연결합니다. v2.0에서 온라인 기부 기능이 추가되면 인앱 결제 플로우로 전환합니다.

**4. Alumni List (동문 찾기)**

| 영역 | 구성 |
|------|------|
| **Search Bar** | 상단 고정 (Sticky), 실시간 필터링 |
| **List Item** | 썸네일(좌) + 이름/기수(중상) + 소속/직책(중하) + 연결 버튼(우) |

#### 기술 스택 및 유틸리티

| 도구 | 용도 | 비고 |
|------|------|------|
| `Tailwind CSS v4` | 스타일 시스템 | CSS-first `@theme` 설정, CSS 변수 기반 |
| `clsx` + `tailwind-merge` | 조건부 클래스 결합 | 중복 클래스 병합 |
| `class-variance-authority` (cva) | 컴포넌트 변형 관리 | Button, Badge 등 variant/size 조합 |
| `Lucide React` | 아이콘 | 일관된 스트로크 두께, 트리셰이킹 지원 |
| `Radix UI Primitives` | 접근성 컴포넌트 | Dialog, DropdownMenu, Tabs, Popover 등 개별 패키지 설치 |

> **Radix UI 선택 근거:** Headless UI v2.0은 트리셰이킹 미지원으로 Dialog + Menu만 사용해도 ~20KB gzip가 추가됩니다. Radix UI는 `@radix-ui/react-dialog` 등 개별 패키지로 필요한 컴포넌트만 설치하여 실사용 번들이 더 작고, 30+개 컴포넌트로 Admin SPA의 복잡한 UI도 추가 라이브러리 없이 구현 가능합니다.

---

## 12. 인프라 제약 사항 및 최적화

### 12.1 1GB RAM 서버 최적화 전략

| 영역 | 전략 |
|------|------|
| Go 빌드 | 서버에서 직접 빌드하지 않음. 로컬/CI에서 크로스 컴파일 후 바이너리 배포 |
| React 빌드 | 로컬/CI에서 `npm run build` 후 정적 파일만 배포 |
| MariaDB | `innodb_buffer_pool_size=256M`, `max_connections=50` |
| Go | `GOMAXPROCS=1`, `GOMEMLIMIT=80MiB`, 커넥션 풀 MaxOpen=10, MaxIdle=5 |
| Nginx | `worker_processes 1`, `worker_connections 512`, gzip 압축 |
| 이미지 | 업로드 시 리사이즈 (max 1200px), WebP 변환 검토 |
| API 응답 | 피드 페이지당 10건, JSON 최소화 |

### 12.2 Graceful Shutdown

배포 시 진행 중 요청이 끊기는 것을 방지합니다.

```go
// cmd/server/main.go

func main() {
    // ... 서버 초기화 (config, db, router, jobs) ...

    srv := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    // 서버 시작 (별도 goroutine)
    go func() {
        logger.Info().Msg("server starting on :8080")
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            logger.Fatal().Err(err).Msg("server failed")
        }
    }()

    // 배치 작업 시작
    donationJob.Start()
    sessionCleanupJob.Start()

    // Graceful shutdown 대기
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info().Msg("shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Error().Err(err).Msg("forced shutdown")
    }

    // DB 연결 정리
    db.Close()
    logger.Info().Msg("server stopped")
}
```

### 12.3 배포 방식

```bash
#!/bin/bash
# deploy.sh

# 1. 로컬에서 빌드
GOOS=linux GOARCH=amd64 go build -o server ./cmd/server  # Go 바이너리
cd frontend && npm run build && cd ..                      # User SPA
cd admin && npm run build && cd ..                         # Admin SPA

# 2. Go 바이너리 — atomic 교체
scp server user@gabia:/app/backend/server.new
ssh user@gabia 'mv /app/backend/server.new /app/backend/server'
ssh user@gabia 'sudo systemctl restart alumni-backend'
# → systemd가 SIGTERM 전송 → 10초 대기 (진행 중 요청 완료) → 새 프로세스 시작

# 3. User SPA 정적 파일 배포
scp -r frontend/dist/ user@gabia:/var/www/app/

# 4. Admin SPA 정적 파일 배포
scp -r admin/dist/ user@gabia:/var/www/admin/

# 5. Nginx 리로드
ssh user@gabia 'sudo systemctl reload nginx'
```

---
