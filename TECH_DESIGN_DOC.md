# 동문 커뮤니티 차세대 웹 애플리케이션 설계서 v1.5

## 변경 이력

| 버전 | 일자 | 변경 사항 |
|------|------|-----------|
| v1.0 | 2026-02-11 | 초안 작성 |
| v1.1 | 2026-02-11 | 1차 리뷰 반영 — 인증 브릿지, 카카오 매핑 플로우, OOM 시나리오, 파일 마이그레이션, 광고 엣지케이스, HTTPS, 모니터링 추가 |
| v1.2 | 2026-02-11 | 2차 리뷰 + 레거시 코드 분석 반영 — 인증 브릿지 구현 확정, CSRF 방어, Nginx 설정 수정, OAuth state, 스냅샷 누락 방지, Graceful Shutdown |
| v1.3 | 2026-02-11 | 선행 확인 사항 반영 — MariaDB 10.1.38 제약사항, 세션 토큰 32자 hex 확정, DDusr 쿠키 HttpOnly=true 전환, SQL 호환성 가이드라인 |
| v1.4 | 2026-02-11 | Markdown 콘텐츠 시스템 추가 (Markdown→HTML→Encode 저장/Decode 조회, 인라인 이미지), 관리자 웹 애플리케이션 설계 추가 |
| v1.5 | 2026-02-12 | UI 디자인 시스템 확정 (Tailwind v4, Royal Indigo 테마, Radix UI, Donate 탭 추가, Feed Card 구조 확정), v1.1 디자인 로드맵 추가 |
| v1.6 | 2026-02-12 | 기부 결제 시스템 설계 추가 (EasyPay PG 연동, 일시후원 카드결제 MVP, ep_cli 바이너리 래퍼, 결제 API 설계, 프론트엔드 결제 폼/브릿지 컴포넌트, Phase 1→2→3 로드맵) |

---

## 1. 프로젝트 개요

| 항목 | 내용 |
|------|------|
| 목적 | Legacy PHP 기반 동문 웹사이트를 소셜 미디어 형태로 고도화 |
| 대상 사용자 | 졸업 동문 (일반 회원), 운영자 (관리자) |
| 핵심 기능 | 공지 피드(무한스크롤), 누적 기부액 표시, **온라인 기부 결제(EasyPay PG)**, 동문찾기, 프로필 관리, **관리자 웹** |
| 콘텐츠 작성 | Markdown 에디터로 작성 → HTML 변환 → Encode 후 DB 저장 → Decode 후 HTML 렌더링 |
| 기술 스택 | React (Frontend) + Go (Backend) + MariaDB 10.1.38 (기존 DB 유지) |
| 인프라 | Gabia 클라우드 서버 1대 (1-core, 1GB RAM, 50GB HDD) |

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

## 4. 인증 설계

### 4.1 PHP ↔ Go 인증 브릿지 (공존 기간)

PHP 레거시는 `DDusr*` 쿠키 5종 기반 인증을 사용하고, Go는 JWT 기반 인증을 사용합니다. 공존 기간 동안 두 체계를 연결하는 브릿지가 필요합니다.

#### 레거시 인증 구조 (PHP 코드 분석 결과)

- `session_start()`는 `sys_config.php`에서 호출되지만, `$_SESSION`은 로그인 판별에 **사용되지 않음**
- 로그인 상태 = `DDusrSession_id` 쿠키 존재 여부로 판별
- 세션 토큰 = `getUniqueTID()`로 자체 생성한 고유 값
- 동일 토큰이 쿠키(`DDusrSession_id`)와 DB(`WEO_MEMBER_LOG.SESSIONID`) 양쪽에 저장
- 로그인 시 쿠키 5종을 동시 발급:
  ```php
  SetCookie("DDusrSession_id", $session_id,       0, "/");
  SetCookie("DDusrSEQ",        $opData->USR_SEQ,  0, "/");
  SetCookie("DDusrID",         $opData->USR_ID,   0, "/");
  SetCookie("DDusrNAME",       $opData->USR_NAME, 0, "/");
  SetCookie("DDusrSTATUS",     $opData->USR_STATUS, 0, "/");
  ```
- `WEO_MEMBER_LOG`에 `SESSIONID`, 로그인 시간 등 로그 레코드 삽입

#### 선택한 전략: Go 로그인 시 레거시 쿠키 5종 + JWT 동시 발급

```
사용자 로그인 (카카오 or 직접)
         │
         ▼
   Go Backend 인증 처리
         │
         ├──▶ JWT 토큰 발급 (HttpOnly Cookie: alumni_token, SameSite=Lax)
         │
         ├──▶ 레거시 쿠키 5종 발급 (DDusrSession_id, DDusrSEQ, DDusrID, DDusrNAME, DDusrSTATUS)
         │
         └──▶ WEO_MEMBER_LOG에 로그인 레코드 INSERT
```

#### 브릿지 미들웨어 구현

```go
// internal/middleware/auth_bridge.go

func AuthBridgeMiddleware(db *sqlx.DB) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Go JWT 토큰 검증 (주 인증)
            claims, err := validateJWT(r)
            if err == nil {
                ctx := context.WithValue(r.Context(), "user", claims)
                next.ServeHTTP(w, r.WithContext(ctx))
                return
            }

            // 2. 레거시 세션 쿠키로 폴백 (공존 기간 한정)
            legacySession, err := r.Cookie("DDusrSession_id")
            if err == nil && legacySession.Value != "" {
                user, err := lookupLegacySession(db, legacySession.Value)
                if err == nil && user.USRStatus == "BBB" {
                    ctx := context.WithValue(r.Context(), "user", user)
                    next.ServeHTTP(w, r.WithContext(ctx))
                    return
                }
            }

            // 3. 미인증
            respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다")
        })
    }
}

// WEO_MEMBER_LOG에서 세션 조회 (레거시 인증 확인)
// 만료 기준: LOG_DATE 기준 24시간 이내 (Go JWT MaxAge와 일치, 환경설정으로 조정 가능)
func lookupLegacySession(db *sqlx.DB, sessionID string) (*User, error) {
    var user User
    err := db.Get(&user, `
        SELECT m.USR_SEQ, m.USR_ID, m.USR_NAME, m.USR_STATUS
        FROM WEO_MEMBER_LOG l
        JOIN WEO_MEMBER m ON l.USR_SEQ = m.USR_SEQ
        WHERE l.SESSIONID = ?
          AND l.LOG_DATE > DATE_SUB(NOW(), INTERVAL 24 HOUR)
        ORDER BY l.LOG_DATE DESC
        LIMIT 1
    `, sessionID)
    return &user, err
}
```

> **세션 만료 기준 (24시간):** `WEO_MEMBER_LOG`에는 `EXPIRES_AT` 컬럼이 없고, PHP에서도 쿠키 `MaxAge=0`(브라우저 세션 쿠키)으로 서버 측 만료 검증이 없습니다. Go에서는 `LOG_DATE` 기준 24시간을 만료 기준으로 사용하며, 이는 Go JWT의 `MaxAge: 86400`과 일치시킨 것입니다.

#### Go 로그인 시 레거시 쿠키 5종 + JWT 발급

```go
func (s *AuthService) LoginWithBridge(user *User, w http.ResponseWriter, r *http.Request) {
    sessionID := generateSessionID()  // 32자 lowercase hex (PHP getUniqueTID() 호환)
    // PHP: md5(uniqid(rand())) → 32자 hex
    // Go: hex.EncodeToString(md5.Sum(uuid)) → 32자 hex
    // SESSIONID 컬럼: varchar(40) → 32자 hex 저장 가능

    // 1. JWT 발급 (Go 신규 인증)
    token := generateJWT(user.USRSeq)
    http.SetCookie(w, &http.Cookie{
        Name:     "alumni_token",
        Value:    token,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   86400, // 24시간
    })

    // 2. 레거시 쿠키 5종 발급 (PHP 호환)
    s.SetLegacyCookies(w, user, sessionID)

    // 3. WEO_MEMBER_LOG에 로그인 레코드 삽입 (PHP 관리자 시스템 호환)
    s.InsertLoginLog(user.USRSeq, sessionID, r.RemoteAddr, r.UserAgent())

    // 4. WEO_MEMBER 최근 방문일자/누적 방문수 업데이트
    s.repo.UpdateLastLogin(user.USRSeq)
}

// 레거시 쿠키 5종 발급
func (s *AuthService) SetLegacyCookies(w http.ResponseWriter, user *User, sessionID string) {
    cookies := map[string]string{
        "DDusrSession_id": sessionID,
        "DDusrSEQ":        fmt.Sprintf("%d", user.USRSeq),
        "DDusrID":         user.USRID,
        "DDusrNAME":       user.USRName,
        "DDusrSTATUS":     user.USRStatus,
    }
    for name, value := range cookies {
        http.SetCookie(w, &http.Cookie{
            Name:     name,
            Value:    value,
            Path:     "/",        // PHP와 동일 (모든 경로에서 접근)
            HttpOnly: true,       // JS에서 DDusr 쿠키 직접 접근 코드 없음 확인 (서버 템플릿 치환만 사용)
            Secure:   true,
            MaxAge:   0,          // PHP와 동일: 브라우저 세션 쿠키
        })
    }
}

// WEO_MEMBER_LOG에 로그인 이력 삽입 (레거시 PHP 패턴과 동일)
// - REG_GEOCODE: NOT NULL 컬럼이지만 PHP에서도 INSERT에 포함하지 않음 (빈 문자열 기본값)
//   → 코드베이스 전수 조사 결과 REG_GEOCODE를 사용하는 코드 없음
// - WEO_MEMBER_LOG는 INSERT만 존재 (SELECT 없음) — 순수 로그 테이블
func (s *AuthService) InsertLoginLog(usrSeq int, sessionID, ipAddr, userAgent string) error {
    _, err := s.db.Exec(`
        INSERT INTO WEO_MEMBER_LOG 
            (USR_SEQ, LOG_DATE, REG_DATE, REG_IPADDR, SESSIONID, REG_AGENT)
        VALUES (?, NOW(), NOW(), ?, ?, ?)
    `, usrSeq, ipAddr, sessionID, userAgent)
    return err
}

// 세션 토큰 생성 (PHP getUniqueTID() 호환: 32자 lowercase hex)
// PHP: md5(uniqid(rand())) → 32자 hex
// Go: crypto/rand 기반 16바이트 → hex 인코딩 → 32자 hex
// WEO_MEMBER_LOG.SESSIONID: varchar(40) → 32자 저장 가능
func generateSessionID() string {
    b := make([]byte, 16)
    if _, err := crypto_rand.Read(b); err != nil {
        panic("failed to generate session ID")
    }
    return hex.EncodeToString(b) // 32자 lowercase hex
}
```

> **`HttpOnly: true` 전환 확인 완료:** 코드베이스 전수 조사 결과, `.htm` 템플릿이나 `.js` 파일에서 `DDusr*` 쿠키를 JavaScript로 직접 읽는 코드가 없었습니다. `DDusr` 관련 처리는 서버 측 템플릿 엔진에서 `{DDusrNAME}`, `{SetDDusrChk}` 등의 변수로 치환하는 방식으로만 사용됩니다. 따라서 공존 기간에도 `HttpOnly: true`로 설정하여 XSS를 통한 쿠키 탈취를 방지합니다.

#### CORS 정책 (공존 기간)

```go
// 같은 도메인의 다른 경로이므로 프로덕션에서 CORS 이슈는 없음.
// 개발 환경(localhost:3000 ↔ localhost:8080)에서는 CORS 설정 필요.
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        allowedOrigins := []string{
            "https://alumni.example.com",
            "http://localhost:3000",  // 개발 환경
        }
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }
        w.Header().Set("Access-Control-Allow-Credentials", "true")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        if r.Method == "OPTIONS" {
            w.WriteHeader(204)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 4.2 카카오 로그인 → 기존 회원 매핑 플로우

```
사용자가 "카카오 로그인" 클릭
         │
         ▼
   Go Backend: 랜덤 state 생성 → 인메모리 캐시에 저장 (TTL 5분)
   → 카카오 인가 URL로 리다이렉트 (?state=xxx 포함)
         │
         ▼
   카카오 OAuth2 인가 완료 → 콜백 (?code=xxx&state=xxx)
         │
         ▼
   Go Backend: state 검증 (캐시에서 조회 → 일치 확인 → 삭제)
   → 불일치 시 403 반환
         │
         ▼
   Go Backend: 인가 코드 → 카카오 Access Token 교환
         │
         ▼
   카카오 API로 사용자 정보 조회 (카카오 ID, 이메일, 닉네임)
         │
         ▼
   WEO_MEMBER_SOCIAL에서 NMS_GATE='KT', NMS_ID=카카오ID 조회
         │
    ┌────┴────┐
    │ 있음     │ 없음 (최초 카카오 로그인)
    │         │
    ▼         ▼
  USR_SEQ    본인 확인 플로우
  확보        │
    │         ▼
    │    "기존 회원이신가요?" 화면 표시
    │         │
    │    ┌────┴────────┐
    │    │ "예"         │ "아니오 (미가입자)"
    │    │             │
    │    ▼             ▼
    │  본인 확인       가입 불가 안내
    │  (이름 + 전화번호   "관리자에게 문의하세요"
    │   또는              (동문 인증이 필요한
    │   기수 + 이름)       사이트이므로 자유 가입 불가)
    │    │
    │    ▼
    │  WEO_MEMBER에서 매칭 조회
    │    │
    │  ┌─┴──────┐
    │  │ 매칭됨   │ 매칭 실패
    │  │        │
    │  ▼        ▼
    │ WEO_MEMBER_SOCIAL에    "정보가 일치하지 않습니다.
    │ 연동 레코드 INSERT      관리자에게 문의하세요"
    │ (NMS_GATE='KT',
    │  USR_SEQ=매칭된 회원,
    │  NMS_ID=카카오ID)
    │  │
    ├──┘
    │
    ▼
  JWT 발급 + PHP 세션 브릿지
  → 메인 피드로 리다이렉트
```

#### 본인 확인 매칭 쿼리

```sql
-- 방법 1: 이름 + 전화번호 (가장 정확)
SELECT USR_SEQ FROM WEO_MEMBER 
WHERE USR_NAME = ? AND USR_PHONE = ? AND USR_STATUS = 'BBB';

-- 방법 2: 기수 + 이름 (전화번호 없는 경우)
SELECT USR_SEQ FROM WEO_MEMBER 
WHERE USR_FN = ? AND USR_NAME = ? AND USR_STATUS = 'BBB';
```

#### 미가입자 정책

카카오 OAuth를 통한 **신규 회원가입이 가능**합니다. 카카오로 최초 로그인 시 이름/전화번호/기수를 입력하면:
- 전화번호가 기존 회원과 일치하고 이름도 일치하면 → 기존 회원과 카카오 계정 **자동 통합**
- 전화번호가 기존 회원과 일치하지만 이름이 다르면 → 오류 안내 (관리자 문의)
- 매칭되는 기존 회원이 없으면 → **신규 회원 자동 생성** (USR_ID = `K{kakaoId}`, USR_STATUS = `BBB`)

#### OAuth state 파라미터 (CSRF 방지)

```go
// 카카오 인가 요청 시 state 생성
func (h *AuthHandler) KakaoLogin(w http.ResponseWriter, r *http.Request) {
    state := generateRandomString(32)
    h.cache.Set("oauth_state:"+state, true, 5*time.Minute)

    redirectURL := fmt.Sprintf(
        "https://kauth.kakao.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
        h.config.KakaoClientID, h.config.KakaoRedirectURI, state,
    )
    http.Redirect(w, r, redirectURL, http.StatusFound)
}

// 카카오 콜백에서 state 검증
func (h *AuthHandler) KakaoCallback(w http.ResponseWriter, r *http.Request) {
    state := r.URL.Query().Get("state")
    if _, found := h.cache.Get("oauth_state:" + state); !found {
        respondError(w, 403, "INVALID_STATE", "OAuth state validation failed")
        return
    }
    h.cache.Delete("oauth_state:" + state)

    code := r.URL.Query().Get("code")
    // ... 이하 기존 토큰 교환 + 회원 매핑 플로우
}
```

### 4.3 세션 정리 배치

```go
// 만료된 세션 정리 (매 1시간)
func (s *SessionCleanupJob) Start() {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            deleted, err := s.repo.DeleteExpiredSessions()
            if err != nil {
                s.logger.Error("session cleanup failed", "error", err)
                continue
            }
            s.logger.Info("expired sessions cleaned", "count", deleted)
        }
    }()
}
```

```sql
DELETE FROM USER_SESSION WHERE EXPIRES_AT < NOW();
```

### 4.4 CSRF 방어

JWT를 `HttpOnly Cookie`로 발급하므로, 쿠키 기반 CSRF 공격에 대한 방어가 필요합니다. `SameSite=Lax`가 상당한 방어를 제공하지만, 명시적 방어 계층을 추가합니다.

```go
// internal/middleware/csrf.go

func CSRFMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // GET, HEAD, OPTIONS는 state-changing이 아니므로 통과
            if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
                next.ServeHTTP(w, r)
                return
            }

            // Origin 헤더 검증
            origin := r.Header.Get("Origin")
            if origin != "" && origin != allowedOrigin {
                respondError(w, 403, "CSRF_REJECTED", "Invalid origin")
                return
            }

            // Origin이 없는 경우 Referer로 폴백 검증
            if origin == "" {
                referer := r.Header.Get("Referer")
                if referer != "" && !strings.HasPrefix(referer, allowedOrigin) {
                    respondError(w, 403, "CSRF_REJECTED", "Invalid referer")
                    return
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**미들웨어 체인 등록:**
```go
r.Use(middleware.CORSMiddleware)
r.Use(middleware.CSRFMiddleware("https://alumni.example.com"))
r.Use(middleware.AuthBridgeMiddleware(db))
```

### 4.5 레거시 ID/PW 로그인

기존 PHP 시절의 아이디/비밀번호 로그인을 Go 백엔드에서 지원합니다.

```
사용자가 "기존 아이디로 로그인" 클릭
         │
         ▼
   /login/legacy 페이지
   (아이디 + 비밀번호 입력)
         │
         ▼
   POST /api/auth/login { usrId, password }
         │
         ▼
   Go Backend: MySQL Native Password 해시 계산
   (SHA1(SHA1(password)) → "*" + uppercase hex)
         │
         ▼
   WEO_MEMBER에서 USR_ID + USR_PWD + USR_STATUS 매칭 조회
         │
    ┌────┴────┐
    │ 매칭됨   │ 미매칭
    │         │
    ▼         ▼
  JWT 발급    401 INVALID_CREDENTIALS
  + PHP 브릿지
  → 메인으로
```

**비밀번호 해시:** WEO_MEMBER.USR_PWD는 MySQL의 `PASSWORD()` 함수 형식(`*` + 40자 uppercase hex)으로 저장되어 있습니다. Go에서 `crypto/sha1`로 이중 SHA1을 수행하여 동일한 해시를 생성합니다.

### 4.6 카카오 신규 회원가입 + 기존 회원 통합

카카오 OAuth 최초 로그인 시, 기존 회원 연동뿐 아니라 신규 회원가입도 지원합니다.

```
카카오 OAuth 콜백 → WEO_MEMBER_SOCIAL에 매칭 없음
         │
         ▼
   /login/link?token=xxx 리다이렉트
   (이름, 전화번호, 기수(선택) 입력)
         │
         ▼
   POST /api/auth/kakao/link { token, name, phone, fn }
         │
         ▼
   서버: 캐시에서 token으로 카카오 정보 조회
         │
         ▼
   WEO_MEMBER에서 전화번호로 기존 회원 조회
         │
    ┌────┴────────────┐
    │ 전화번호 매칭됨    │ 매칭 없음
    │                  │
    ▼                  ▼
  이름 일치 확인        신규 회원 생성
    │                  (USR_ID = K{kakaoId})
  ┌─┴──────┐          │
  │ 일치    │ 불일치    ▼
  │        │         WEO_MEMBER_SOCIAL
  ▼        ▼         연동 레코드 INSERT
 계정 통합  409 에러         │
 (소셜링크  "이름 불일치"     ▼
  INSERT)               JWT 발급 + 브릿지
    │                  → 메인으로
    ▼
  JWT 발급 + 브릿지
  → 메인으로
```

---

## 5. DB 스키마 변경 계획

### 5.1 기존 테이블 변경 사항

#### WEO_BOARDBBS — 컬럼 추가
```sql
-- 소셜 피드용 컬럼 추가
ALTER TABLE WEO_BOARDBBS 
  ADD COLUMN THUMBNAIL_URL VARCHAR(500) NULL COMMENT '대표 이미지 URL (피드 카드용)' AFTER FILES,
  ADD COLUMN SUMMARY VARCHAR(200) NULL COMMENT '본문 요약 (피드 미리보기용)' AFTER THUMBNAIL_URL,
  ADD COLUMN IS_PINNED ENUM('Y','N') DEFAULT 'N' COMMENT '상단 고정 여부' AFTER SUMMARY,
  ADD COLUMN CONTENTS_MD MEDIUMTEXT NULL COMMENT '원본 Markdown 텍스트 (편집용)' AFTER CONTENTS,
  ADD COLUMN CONTENT_FORMAT ENUM('LEGACY','MARKDOWN') DEFAULT 'LEGACY' COMMENT '콘텐츠 포맷' AFTER CONTENTS_MD;
```

**컬럼별 역할:**

| 컬럼 | 용도 | 포맷 |
|------|------|------|
| `CONTENTS` | **렌더링용 HTML** (인코딩 저장) | 신규 글: Base64(sanitized HTML), 기존 글: raw HTML |
| `CONTENTS_MD` | **편집용 원본** Markdown | 신규 글: raw Markdown, 기존 글: NULL |
| `CONTENT_FORMAT` | 콘텐츠 판별 플래그 | `LEGACY`=기존 raw HTML, `MARKDOWN`=Base64 인코딩된 HTML |
| `SUMMARY` | 피드 카드 미리보기 | 200자 plain text (태그 제거) |
| `THUMBNAIL_URL` | 피드 카드 대표 이미지 | Markdown 내 첫 번째 이미지 URL 자동 추출 |

#### MAIN_AD — 컬럼 추가
```sql
-- 인피드 광고 등급 추가
ALTER TABLE MAIN_AD 
  ADD COLUMN AD_TIER ENUM('PREMIUM','GOLD','NORMAL') DEFAULT 'NORMAL' 
      COMMENT '광고 등급 (PREMIUM:프리미엄, GOLD:골드, NORMAL:일반)' AFTER MA_TYPE,
  ADD COLUMN AD_TITLE_LABEL VARCHAR(50) DEFAULT '추천 동문 소식' 
      COMMENT '광고 카드 타이틀 라벨' AFTER AD_TIER;

-- 광고 배너 이미지 URL 추가
ALTER TABLE MAIN_AD
  ADD COLUMN MA_IMG VARCHAR(500) NULL COMMENT '광고 배너 이미지 URL' AFTER MA_URL;
```
- PREMIUM → "가장 핫한 동문 소식" (1순위)
- GOLD → "추천 동문 소식" (2순위)

> **레거시 스키마 참고:** MAIN_AD의 활성 여부 컬럼은 `OPEN_YN` (코드에서 maStatus로 노출),
> 정렬 순서 컬럼은 `INDX` (코드에서 maIndx로 노출).

#### FUNDAMENTAL_MEMBER — 인덱스 추가
```sql
-- 동문찾기 검색 조건 추가
CREATE INDEX INDX_COMPANY ON FUNDAMENTAL_MEMBER (FM_COMPANY);
CREATE INDEX INDX_POSITION ON FUNDAMENTAL_MEMBER (FM_POSITION);
```

### 5.2 신규 테이블 (InnoDB)

#### DONATION_SNAPSHOT — 기부액 일별 스냅샷
```sql
CREATE TABLE DONATION_SNAPSHOT (
    DS_SEQ        INT AUTO_INCREMENT PRIMARY KEY COMMENT 'SEQ',
    DS_DATE       DATE NOT NULL COMMENT '스냅샷 일자',
    DS_TOTAL      BIGINT DEFAULT 0 NOT NULL COMMENT '누적 기부 총액 (WEO_ORDER 합산)',
    DS_MANUAL_ADJ BIGINT DEFAULT 0 NOT NULL COMMENT '수동 조정액',
    DS_DONOR_CNT  INT DEFAULT 0 NOT NULL COMMENT '누적 기부자 수',
    DS_GOAL       BIGINT DEFAULT 0 NOT NULL COMMENT '목표 금액',
    REG_DATE      DATETIME NULL COMMENT '생성일시',
    UNIQUE KEY UK_DATE (DS_DATE)
) COMMENT '기부액 일별 스냅샷' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**기부액 계산 로직:**
```
표시 기부액 = DS_TOTAL + DS_MANUAL_ADJ
달성률 = 표시 기부액 / DS_GOAL * 100
```

#### DONATION_CONFIG — 기부 설정
```sql
CREATE TABLE DONATION_CONFIG (
    DC_SEQ        INT AUTO_INCREMENT PRIMARY KEY COMMENT 'SEQ',
    DC_GOAL       BIGINT DEFAULT 0 NOT NULL COMMENT '목표 금액',
    DC_MANUAL_ADJ BIGINT DEFAULT 0 NOT NULL COMMENT '수동 조정 누적액',
    DC_NOTE       VARCHAR(200) NULL COMMENT '조정 메모',
    IS_ACTIVE     ENUM('Y','N') DEFAULT 'Y' COMMENT '활성 설정 여부',
    REG_DATE      DATETIME NULL COMMENT '등록일시',
    REG_OPER      INT NULL COMMENT '등록 운영자'
) COMMENT '기부 설정 관리' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### USER_SESSION — Go 백엔드용 세션/토큰 관리
```sql
CREATE TABLE USER_SESSION (
    SESSION_ID    VARCHAR(64) PRIMARY KEY COMMENT '세션 토큰',
    USR_SEQ       INT NOT NULL COMMENT 'WEO_MEMBER USR_SEQ',
    PROVIDER      ENUM('KAKAO','DIRECT') DEFAULT 'DIRECT' COMMENT '인증 방식',
    EXPIRES_AT    DATETIME NOT NULL COMMENT '만료 시각',
    CREATED_AT    DATETIME NOT NULL COMMENT '생성 시각',
    INDEX IDX_USR (USR_SEQ),
    INDEX IDX_EXPIRES (EXPIRES_AT)
) COMMENT '사용자 세션 관리' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### WEO_AD_LOG — 광고 노출/클릭 로그
```sql
CREATE TABLE WEO_AD_LOG (
    AL_SEQ     INT AUTO_INCREMENT PRIMARY KEY COMMENT 'SEQ',
    MA_SEQ     INT NOT NULL COMMENT 'MAIN_AD SEQ',
    USR_SEQ    INT NULL COMMENT '사용자 SEQ (비로그인은 NULL)',
    AL_TYPE    ENUM('VIEW','CLICK') NOT NULL COMMENT '이벤트 유형',
    AL_DATE    DATETIME NOT NULL COMMENT '이벤트 일시',
    AL_IPADDR  VARCHAR(45) NULL COMMENT '접속 IP',
    INDEX IDX_MA_SEQ (MA_SEQ),
    INDEX IDX_DATE (AL_DATE)
) COMMENT '광고 노출/클릭 로그' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 5.3 기존 게시글 썸네일 백필 (Backfill)

기존 게시글의 `THUMBNAIL_URL`이 NULL이므로, 마이그레이션 배치가 필요합니다.

```sql
-- 기존 게시글에 첨부 이미지가 있는 경우 첫 번째 이미지를 썸네일로 설정
UPDATE WEO_BOARDBBS b
JOIN (
    SELECT F_JOIN_SEQ, MIN(F_SEQ) AS FIRST_FILE_SEQ
    FROM WEO_FILES 
    WHERE F_GATE = 'BB' 
      AND OPEN_YN = 'Y'
      AND FILE_NAME LIKE '%.jpg' OR FILE_NAME LIKE '%.jpeg' OR FILE_NAME LIKE '%.png' OR FILE_NAME LIKE '%.webp'
    GROUP BY F_JOIN_SEQ
) f ON b.SEQ = f.F_JOIN_SEQ
JOIN WEO_FILES wf ON wf.F_SEQ = f.FIRST_FILE_SEQ
SET b.THUMBNAIL_URL = CONCAT('/files/', wf.FILE_PATH, '/', wf.FILE_NAME)
WHERE b.GATE = 'NOTICE' 
  AND b.THUMBNAIL_URL IS NULL;

-- SUMMARY 백필: CONTENTS에서 태그 제거 후 앞 200자 추출 (Go 배치로 처리 권장)
-- HTML 태그 제거는 SQL로 어렵기 때문에 Go 스크립트에서 처리
```

### 5.4 콘텐츠 작성/저장/조회 플로우

공지글은 Markdown으로 작성하고, HTML로 변환 후 Base64 인코딩하여 DB에 저장합니다. 조회 시에는 디코딩하여 HTML을 반환합니다.

#### 저장 플로우 (Admin → DB)

```
관리자가 Markdown 에디터에서 작성
  │
  │  "# 제목\n\n본문 내용...\n\n![사진](/uploads/notice/abc.jpg)\n\n계속..."
  │
  ▼
에디터 내 이미지 업로드 ──▶ POST /api/admin/upload
  │                         └→ 응답: { "url": "/uploads/notice/abc.jpg" }
  │                         └→ 에디터에 ![](url) 삽입
  │
  ▼
관리자가 "저장" 클릭 ──▶ POST /api/admin/feed
  │                       Body: { "subject": "...", "contentMd": "# 제목\n..." }
  │
  ▼
Go Backend 처리:
  │
  ├── 1. Markdown → HTML 변환 (goldmark)
  │      "# 제목" → "<h1>제목</h1>"
  │      "![사진](/uploads/...)" → "<img src='/uploads/...' alt='사진'>"
  │
  ├── 2. <img> 태그에 loading="lazy" 자동 삽입 (Lazy Loading)
  │      → "<img loading='lazy' src='/uploads/...' alt='사진'>"
  │
  ├── 3. HTML 새니타이즈 (bluemonday) — XSS 방어
  │      <script>, onerror= 등 위험 태그/속성 제거
  │      <img>, <a>, <h1>~<h6>, <p>, <ul>, <ol>, <li>, <blockquote>,
  │      <code>, <pre>, <em>, <strong>, <br>, <hr>, <table> 등 허용
  │      loading="lazy" 속성 허용
  │
  ├── 4. Base64 인코딩 → CONTENTS에 저장
  │      base64.StdEncoding.EncodeToString([]byte(sanitizedHTML))
  │
  ├── 5. 원본 Markdown → CONTENTS_MD에 저장
  │
  ├── 6. CONTENT_FORMAT = 'MARKDOWN'
  │
  ├── 7. SUMMARY 자동 생성: HTML 태그 제거 → 앞 200자
  │
  └── 8. THUMBNAIL_URL 자동 추출: Markdown 내 첫 번째 이미지 URL
```

#### 조회 플로우 (DB → User)

```
User가 피드 또는 상세 페이지 요청
  │
  ▼
Go Backend:
  │
  ├── CONTENT_FORMAT 확인
  │     │
  │     ├── 'MARKDOWN' (신규 글)
  │     │     → CONTENTS를 Base64 디코딩 → sanitized HTML 반환
  │     │
  │     └── 'LEGACY' (기존 글)
  │           → CONTENTS를 raw HTML 그대로 반환
  │
  ▼
API 응답: { "contentHtml": "<h1>제목</h1><p>본문...</p><img src='...'>" }
  │
  ▼
React (User SPA):
  └── <HtmlContent html={contentHtml} />
      → DOMPurify로 2차 새니타이즈 후 렌더링
```

#### Go 서비스 구현

```go
// internal/service/content.go

import (
    "encoding/base64"
    "github.com/yuin/goldmark"
    "github.com/microcosm-cc/bluemonday"
    "regexp"
    "strings"
)

var (
    md       = goldmark.New(goldmark.WithExtensions(/* table, strikethrough */))
    sanitize = bluemonday.UGCPolicy() // User Generated Content 정책
)

func init() {
    // 이미지 태그 허용 (인라인 이미지)
    sanitize.AllowImages()
    // loading="lazy" 속성 허용 (Lazy Loading)
    sanitize.AllowAttrs("loading").Matching(regexp.MustCompile(`^lazy$`)).OnElements("img")
    // iframe 등은 차단 유지
}

// ConvertAndEncode: Markdown → sanitized HTML → Base64
func ConvertAndEncode(markdownText string) (encoded string, summary string, thumbnail string, err error) {
    // 1. Markdown → HTML
    var buf bytes.Buffer
    if err := md.Convert([]byte(markdownText), &buf); err != nil {
        return "", "", "", err
    }
    rawHTML := buf.String()

    // 2. <img> 태그에 loading="lazy" 자동 삽입 (Lazy Loading)
    rawHTML = addLazyLoading(rawHTML)

    // 3. HTML 새니타이즈 (XSS 방어)
    safeHTML := sanitize.Sanitize(rawHTML)

    // 4. Base64 인코딩
    encoded = base64.StdEncoding.EncodeToString([]byte(safeHTML))

    // 5. Summary 추출: 태그 제거 후 200자
    plain := stripHTMLTags(safeHTML)
    summary = truncate(plain, 200)

    // 6. Thumbnail 추출: 첫 번째 <img> src
    thumbnail = extractFirstImageURL(safeHTML)

    return encoded, summary, thumbnail, nil
}

// DecodeContent: DB에서 읽은 콘텐츠를 HTML로 디코딩
func DecodeContent(contents string, format string) string {
    if format == "MARKDOWN" {
        decoded, err := base64.StdEncoding.DecodeString(contents)
        if err != nil {
            return contents // 디코딩 실패 시 원본 반환
        }
        return string(decoded)
    }
    // LEGACY: raw HTML 그대로 반환
    return contents
}

// 첫 번째 이미지 URL 추출
func extractFirstImageURL(html string) string {
    re := regexp.MustCompile(`<img[^>]+src="([^"]+)"`)
    matches := re.FindStringSubmatch(html)
    if len(matches) > 1 {
        return matches[1]
    }
    return ""
}

// <img> 태그에 loading="lazy" 속성 자동 삽입
// 이미 loading 속성이 있는 경우 건너뜀
var imgTagRe = regexp.MustCompile(`<img(?P<attrs>[^>]*)>`)

func addLazyLoading(html string) string {
    return imgTagRe.ReplaceAllStringFunc(html, func(match string) string {
        if strings.Contains(match, `loading=`) {
            return match // 이미 있으면 건너뜀
        }
        return strings.Replace(match, "<img", `<img loading="lazy"`, 1)
    })
}
```

#### React (User SPA) — HTML 렌더링 컴포넌트

```tsx
// frontend/src/components/common/HtmlContent.tsx
import DOMPurify from 'dompurify';

interface Props {
  html: string;
  className?: string;
}

export function HtmlContent({ html, className }: Props) {
  // 서버에서 이미 새니타이즈되었지만, 2차 방어
  const clean = DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['h1','h2','h3','h4','h5','h6','p','a','img','ul','ol','li',
                   'blockquote','code','pre','em','strong','br','hr','table',
                   'thead','tbody','tr','th','td','del','sup','sub'],
    ALLOWED_ATTR: ['href','src','alt','title','class','target','rel','loading'],
  });

  return (
    <div
      className={`prose prose-lg max-w-none ${className || ''}`}
      dangerouslySetInnerHTML={{ __html: clean }}
    />
  );
}
```

#### 인라인 이미지 업로드 (Markdown 에디터 내)

관리자가 에디터에서 이미지를 삽입하면, 즉시 서버에 업로드하고 반환된 URL을 Markdown에 삽입합니다.

```
[에디터에서 이미지 선택/붙여넣기]
        │
        ▼
POST /api/admin/upload
  Body: multipart/form-data (file)
  Query: ?type=notice
        │
        ▼
Go: 파일 저장 (/var/www/uploads/notice/{uuid}.{ext})
    이미지 리사이즈 (max 1200px width)
    WEO_FILES 레코드 삽입
        │
        ▼
응답: { "url": "/uploads/notice/abc123.jpg" }
        │
        ▼
에디터에 자동 삽입: ![이미지 설명](/uploads/notice/abc123.jpg)
```

---

## 6. API 설계

### 6.1 인증

| Method | Endpoint | 설명 |
|--------|----------|------|
| POST | `/api/auth/login` | 레거시 ID/PW 로그인 (MySQL native password 해시 검증) |
| POST | `/api/auth/kakao` | 카카오 로그인 (Authorization Code → JWT + PHP세션 발급) |
| POST | `/api/auth/kakao/link` | 카카오 계정 연동 또는 신규 회원가입 |
| POST | `/api/auth/logout` | 로그아웃 (JWT + PHP세션 모두 무효화) |
| GET | `/api/auth/me` | 현재 로그인 사용자 정보 |

### 6.2 피드 (공지글)

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/feed` | 피드 목록 (무한스크롤, cursor-based pagination) |
| GET | `/api/feed/hero` | Hero 섹션용 최신 공지 1건 |
| GET | `/api/feed/{seq}` | 공지 상세 |

**GET /api/feed/{seq} 응답 구조 (상세):**
```json
{
  "seq": 1234,
  "subject": "제목",
  "contentHtml": "<h1>제목</h1><p>본문 내용...</p><img src='/uploads/notice/abc.jpg'>",
  "contentFormat": "MARKDOWN",
  "regDate": "2026-02-10T10:00:00",
  "regName": "관리자",
  "hit": 150,
  "likeCnt": 23,
  "commentCnt": 5,
  "files": [
    { "fileSeq": 1, "fileName": "첨부파일.pdf", "fileUrl": "/files/..." }
  ]
}
```

> **`contentHtml` 반환 규칙:**
> - `CONTENT_FORMAT='MARKDOWN'`: CONTENTS를 Base64 디코딩 → sanitized HTML 반환
> - `CONTENT_FORMAT='LEGACY'`: CONTENTS를 raw HTML 그대로 반환
> - 프론트에서는 `contentFormat`을 구분할 필요 없이, `contentHtml`을 DOMPurify로 새니타이즈 후 렌더링

**GET /api/feed 쿼리 파라미터:**

| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `cursor` | string | 마지막으로 본 게시글 SEQ (없으면 최신부터) |
| `size` | int | 한 페이지 크기 (기본 10, 최대 20) |
| `exclude_ads` | string | 이미 노출된 광고 MA_SEQ 목록 (콤마 구분) |

**GET /api/feed 응답 구조:**
```json
{
  "items": [
    {
      "type": "notice",
      "seq": 1234,
      "subject": "제목",
      "summary": "본문 요약 200자",
      "thumbnailUrl": "/files/thumb/xxx.jpg",
      "regDate": "2026-02-10T10:00:00",
      "regName": "관리자",
      "hit": 150,
      "likeCnt": 23,
      "commentCnt": 5
    },
    {
      "type": "ad",
      "adTier": "PREMIUM",
      "titleLabel": "가장 핫한 동문 소식",
      "maSeq": 10,
      "maName": "광고명",
      "maUrl": "/ad/xxx",
      "imageUrl": "/files/ad/xxx.jpg"
    }
  ],
  "nextCursor": "seq_1230",
  "hasMore": true
}
```

**피드 조합 로직 (서버 사이드):**
```
1. WEO_BOARDBBS에서 GATE='NOTICE', OPEN_YN='Y' 조건으로 최신순 조회
2. Hero 게시글(최신 1건)은 피드 목록에서 제외하여 중복 방지
3. 4개 게시글마다 1개 광고 삽입
4. 광고 삽입 우선순위: PREMIUM → GOLD → NORMAL (INDX 정렬)
5. exclude_ads 파라미터에 있는 광고는 제외 (중복 방지)
6. 활성 광고가 모두 소진되면 광고 슬롯 건너뜀 (빈 공간 없음)
7. 게시글이 4개 미만이면 광고 삽입하지 않음
```

### 6.3 광고 추적

| Method | Endpoint | 설명 |
|--------|----------|------|
| POST | `/api/ad/{maSeq}/view` | 광고 노출 로그 (피드에 나타날 때 호출) |
| POST | `/api/ad/{maSeq}/click` | 광고 클릭 로그 |

```go
// 광고 노출은 fire-and-forget (비동기)
func (h *AdHandler) TrackView(w http.ResponseWriter, r *http.Request) {
    maSeq := chi.URLParam(r, "maSeq")
    usrSeq := getUserSeqOrNil(r.Context())
    go h.adService.LogEvent(maSeq, usrSeq, "VIEW", r.RemoteAddr)
    w.WriteHeader(204)
}
```

### 6.4 기부액 및 기부 결제

#### 기부 현황 조회

| Method | Endpoint | 인증 | 설명 |
|--------|----------|------|------|
| GET | `/api/donation/summary` | 불필요 | 최신 기부 요약 (총액, 기부자 수, 달성률) |

**폴백 로직:** 오늘 스냅샷이 없으면 가장 최근 스냅샷을 반환합니다.

```go
func (s *DonationService) GetSummary() (*DonationSummary, error) {
    // 오늘 스냅샷 조회
    snapshot, err := s.repo.GetSnapshotByDate(time.Now())
    if err != nil || snapshot == nil {
        // 폴백: 가장 최근 스냅샷
        snapshot, err = s.repo.GetLatestSnapshot()
        if err != nil || snapshot == nil {
            return &DonationSummary{}, nil // 데이터 없음 → 0 표시
        }
    }
    displayAmount := snapshot.Total + snapshot.ManualAdj
    rate := float64(0)
    if snapshot.Goal > 0 {
        rate = float64(displayAmount) / float64(snapshot.Goal) * 100
    }
    return &DonationSummary{
        TotalAmount:     snapshot.Total,
        ManualAdj:       snapshot.ManualAdj,
        DisplayAmount:   displayAmount,
        DonorCount:      snapshot.DonorCnt,
        GoalAmount:      snapshot.Goal,
        AchievementRate: rate,
        SnapshotDate:    snapshot.Date,
    }, nil
}
```

#### 기부 결제 API (Phase 1: 일시후원 카드결제)

| Method | Endpoint | 인증 | CSRF | 설명 |
|--------|----------|------|------|------|
| POST | `/api/donation/orders` | 필수 | 적용 | 기부 주문 생성 (pending 상태) |
| GET | `/api/donation/orders/{seq}` | 필수 | 적용 | 주문 상세 조회 (결과 페이지용) |
| POST | `/pg/easypay/relay` | 불필요 | **제외** | EasyPay SDK → PG 서버 중계 (auto-submit HTML) |
| POST | `/pg/easypay/return` | 불필요 | **제외** | EasyPay 결제 완료 후 콜백 (ep_cli 승인 처리) |

> **`/pg/*` CSRF 제외 이유:** EasyPay PG 서버에서 cross-origin form POST로 호출하므로 Origin/Referer가 `sp.easypay.co.kr`입니다. `SameSite=Lax` 쿠키도 전달되지 않으므로 인증 없이 주문번호 기반으로 처리합니다.

**POST /api/donation/orders — 주문 생성**

```
Request:  { "amount": 50000, "payType": "CARD", "gate": "immediately" }

Response: {
  "orderSeq": 39740,
  "paymentParams": {
    "mallId": "05542574",
    "orderNo": "39740",
    "productAmt": "50000",
    "productName": "홍길동님_일시후원_50,000원",
    "payType": "11",
    "returnUrl": "https://www.daeilfoundation.or.kr/pg/easypay/return",
    "relayUrl": "/pg/easypay/relay",
    "windowType": "iframe",
    "userName": "홍길동",
    "mallName": "대일외국어고등학교 장학회",
    "currency": "00",
    "charset": "UTF-8",
    "langFlag": "KOR"
  }
}
```

**검증 규칙:**
- `amount >= 10000` (최소 1만원)
- `payType = "CARD"` (Phase 1에서는 카드만 지원)
- `gate = "immediately"` (Phase 1에서는 일시후원만 지원)
- 로그인 필수

**주문 생성 SQL:**
```sql
INSERT INTO WEO_ORDER (USR_SEQ, O_GATE, O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR)
VALUES (?, 'S', 'CARD', 'A', NOW(), ?, 'N', NOW(), ?)
-- O_GATE='S' (일시후원, V1 컨벤션), O_TYPE='A' (기부), O_PAYMENT='N' (미결제)
```

**GET /api/donation/orders/{seq} — 주문 조회**

```
Response: {
  "orderSeq": 39740,
  "amount": 50000,
  "status": "Y",
  "paidAt": "2026-02-12 14:30:00"
}
```

**POST /pg/easypay/relay — PG 중계**

EasyPay SDK에서 form POST로 호출됩니다. Go 서버는 `sp.easypay.co.kr/ep8/MainAction.do`로 auto-submit하는 HTML 페이지를 응답합니다. V1의 `order_req.php`와 동일한 역할입니다.

**POST /pg/easypay/return — 결제 완료 콜백**

EasyPay에서 사용자 결제 완료 후 리다이렉트합니다 (`sp_return_url`).

처리 순서:
1. `sp_res_cd` 파싱 — `"0000"`이 아니면 실패 페이지로 리다이렉트
2. `ep_cli` 바이너리 실행 (서버 사이드 승인, 30초 타임아웃)
3. `ep_cli` 응답 파싱 (`key=value` 쌍, `\x1F` 구분자)
4. 승인 성공 시 DB 트랜잭션:
   ```sql
   -- PG 데이터 저장
   INSERT INTO WEO_PG_DATA (CNO, RES_CD, RES_MSG, AMOUNT, NUM_CARD, TRAN_DATE, AUTH_NO, PAY_TYPE, O_SEQ)
   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)

   -- 주문 결제 완료 처리 (멱등성: O_PAYMENT='N' 조건)
   UPDATE WEO_ORDER SET O_PAYMENT='Y', O_PAY=?, O_PAYDATE=NOW(), O_PG_SEQ=?, O_STATUS='Y'
   WHERE O_SEQ=? AND O_PAYMENT='N'
   ```
5. 캐시 무효화: `cache.Delete("donation_summary")`
6. `302 redirect` → `/donation/result?status=success&order=X`

> **상세 설계:** §16 기부 결제 시스템 설계 참조

### 6.5 동문찾기

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/alumni` | 동문 검색 (로그인 필수) |
| GET | `/api/alumni/filters` | 검색 필터 옵션 (기수 목록, 학과 목록) |

**검색 쿼리 파라미터:**

| 파라미터 | 타입 | 설명 |
|----------|------|------|
| `fn` | int | 기수 |
| `dept` | string | 학과 |
| `name` | string | 이름 (전방 일치) |
| `company` | string | 회사 (전방 일치) |
| `position` | string | 직책 (전방 일치) |
| `page` | int | 페이지 번호 (기본 1) |
| `size` | int | 페이지 크기 (기본 20) |

**검색 성능 주의사항:**

```sql
-- ✅ 전방 일치: 인덱스 사용 가능
WHERE FM_NAME LIKE '홍%'
WHERE FM_COMPANY LIKE '삼성%'

-- ❌ 중간 일치: 인덱스 사용 불가 → 풀스캔
WHERE FM_NAME LIKE '%길동%'
WHERE FM_COMPANY LIKE '%전자%'
```

**설계 결정:** 이름과 회사는 **전방 일치(`keyword%`)** 로 검색합니다. 중간 일치가 필요한 경우 데이터 규모가 커지면 별도 검색 인덱스(MariaDB FULLTEXT 또는 외부 검색 엔진) 도입을 검토합니다.

### 6.6 마이페이지

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/profile` | 내 프로필 조회 |
| PUT | `/api/profile` | 프로필 수정 |

### 6.7 관리자 API (운영자 전용)

> **인가:** 모든 `/api/admin/*` 엔드포인트는 `AdminAuthMiddleware`로 보호됩니다. `USR_STATUS = 'AAA'`(관리자) 또는 지정된 관리자 권한 레벨을 가진 사용자만 접근 가능합니다.

#### 공지 관리 (Markdown 콘텐츠)

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/admin/feed` | 공지 목록 (페이지네이션, 상태 필터) |
| GET | `/api/admin/feed/{seq}` | 공지 상세 (편집용: Markdown 원본 포함) |
| POST | `/api/admin/feed` | 공지 등록 |
| PUT | `/api/admin/feed/{seq}` | 공지 수정 |
| DELETE | `/api/admin/feed/{seq}` | 공지 삭제 (soft delete) |
| PUT | `/api/admin/feed/{seq}/pin` | 상단 고정 토글 |

**POST/PUT /api/admin/feed 요청:**
```json
{
  "subject": "2026년 정기총회 안내",
  "contentMd": "# 정기총회\n\n일시: 2026년 3월 15일\n\n![포스터](/uploads/notice/abc.jpg)\n\n참석 신청...",
  "boardId": "NOTICE",
  "isPinned": "N"
}
```

**Go 핸들러 처리 흐름:**
```go
func (h *AdminNoticeHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateNoticeRequest
    // 1. 요청 파싱
    json.NewDecoder(r.Body).Decode(&req)

    // 2. Markdown → HTML → Base64 인코딩 + Summary/Thumbnail 자동 추출
    encoded, summary, thumbnail, err := content.ConvertAndEncode(req.ContentMd)

    // 3. DB INSERT
    notice := &model.Notice{
        Subject:      req.Subject,
        Contents:     encoded,        // Base64(sanitized HTML)
        ContentsMD:   req.ContentMd,  // 원본 Markdown
        ContentFormat: "MARKDOWN",
        Summary:      summary,
        ThumbnailURL: thumbnail,
        IsPinned:     req.IsPinned,
    }
    seq, err := h.service.CreateNotice(notice)

    respondJSON(w, 201, map[string]int{"seq": seq})
}
```

**GET /api/admin/feed/{seq} 응답 (편집용):**
```json
{
  "seq": 1234,
  "subject": "2026년 정기총회 안내",
  "contentMd": "# 정기총회\n\n일시: 2026년 3월 15일...",
  "contentHtml": "<h1>정기총회</h1><p>일시: ...",
  "contentFormat": "MARKDOWN",
  "isPinned": "N",
  "regDate": "2026-02-10T10:00:00",
  "hit": 150
}
```
> 일반 피드 API(`/api/feed/{seq}`)와 달리, 관리자 API는 `contentMd`(원본 Markdown)를 추가로 반환하여 에디터에 로드할 수 있게 합니다. `contentFormat`이 `LEGACY`인 기존 글은 `contentMd`가 null이므로, 에디터 대신 레거시 HTML 편집 모드로 전환합니다.

#### 이미지 업로드

| Method | Endpoint | 설명 |
|--------|----------|------|
| POST | `/api/admin/upload` | 이미지 업로드 (Markdown 에디터용 + 광고 배너) |

**요청:** `multipart/form-data` (`file` 필드 + `type` 쿼리: `notice`, `ad`, `profile`)

**응답:**
```json
{
  "url": "/uploads/notice/20260211_a1b2c3d4.jpg",
  "width": 1200,
  "height": 800
}
```

**처리:** 이미지 리사이즈 (max 1200px width), 파일명 UUID 변환, `WEO_FILES` INSERT, 저장 경로 `/var/www/uploads/{type}/`

#### 기부 설정

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/admin/donation/config` | 현재 기부 설정 조회 |
| PUT | `/api/admin/donation/config` | 기부 설정 수정 (목표액, 수동 조정) |
| GET | `/api/admin/donation/history` | 스냅샷 이력 조회 (최근 30일) |

#### 광고 관리

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/admin/ad` | 광고 목록 |
| POST | `/api/admin/ad` | 광고 등록 |
| PUT | `/api/admin/ad/{seq}` | 광고 수정 |
| DELETE | `/api/admin/ad/{seq}` | 광고 삭제 |
| GET | `/api/admin/ad/stats` | 광고 노출/클릭 통계 (기간별) |

#### 회원 관리

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/admin/member` | 회원 목록 (검색, 페이지네이션) |
| GET | `/api/admin/member/{seq}` | 회원 상세 정보 |
| PUT | `/api/admin/member/{seq}` | 회원 정보 수정 (상태 변경 등) |
| GET | `/api/admin/member/stats` | 회원 통계 (가입 현황, 카카오 연동률) |

#### 대시보드 통계

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/admin/dashboard` | 대시보드 종합 통계 |

**응답:**
```json
{
  "totalMembers": 3420,
  "kakaoLinkedMembers": 1280,
  "recentLoginCount": 156,
  "totalNotices": 890,
  "donation": {
    "total": 130000000,
    "donorCount": 342,
    "goalPercent": 65
  },
  "adStats": {
    "totalImpressions": 12400,
    "totalClicks": 890,
    "ctr": 7.18
  }
}
```

### 6.8 헬스체크 / 모니터링

| Method | Endpoint | 설명 |
|--------|----------|------|
| GET | `/api/health` | 서버 상태 (DB 연결 확인 포함) |

```go
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
    status := map[string]string{"status": "ok"}

    // DB 연결 확인
    if err := h.db.PingContext(r.Context()); err != nil {
        status["status"] = "degraded"
        status["db"] = "unreachable"
        w.WriteHeader(503)
    }

    json.NewEncoder(w).Encode(status)
}
```

**외부 모니터링:** UptimeRobot (무료) 등으로 `https://alumni.example.com/api/health`를 1분 간격 모니터링하여, 장애 시 Slack/이메일 알림을 설정합니다.

---

## 7. 파일/이미지 마이그레이션 계획

### 7.1 기존 파일 유지 전략

```
/var/www/legacy/files/       ← PHP 시절 업로드 파일 (변경하지 않음)
    └── WEO_FILES.FILE_PATH + FILE_NAME으로 참조

/var/www/uploads/            ← Go에서 새로 업로드하는 파일
    ├── notice/              ← 공지글 이미지
    ├── ad/                  ← 광고 이미지
    └── profile/             ← 프로필 이미지
```

**핵심 원칙:** 기존 파일은 절대 이동하지 않습니다. Nginx location 블록으로 기존 경로를 그대로 서빙합니다.

### 7.2 WEO_FILES와의 호환

Go에서 새 파일 업로드 시에도 `WEO_FILES` 테이블에 레코드를 삽입하여 기존 PHP 관리 도구와의 호환성을 유지합니다.

```go
func (s *FileService) Upload(file multipart.File, header *multipart.FileHeader, gate string, joinSeq int) (*FileRecord, error) {
    // 1. 파일 저장 (/var/www/uploads/{gate}/{filename})
    filename := generateUniqueFilename(header.Filename)
    path := filepath.Join("/var/www/uploads", gate, filename)
    // ... 파일 저장 로직

    // 2. 이미지인 경우 리사이즈 (max 1200px width)
    if isImage(header.Header.Get("Content-Type")) {
        resizeImage(path, 1200)
    }

    // 3. WEO_FILES에 레코드 삽입
    record := &FileRecord{
        FGate:       gate,
        FJoinSeq:    joinSeq,
        TypeName:    header.Filename,
        FileName:    filename,
        FileSize:    fmt.Sprintf("%d", header.Size),
        FilePath:    fmt.Sprintf("/uploads/%s", gate),
        FileOrgName: header.Filename,
        OpenYN:      "Y",
    }
    return s.repo.InsertFile(record)
}
```

### 7.3 썸네일 백필 배치

Go 스크립트로 기존 게시글의 썸네일과 요약을 채웁니다.

```go
// cmd/backfill/main.go
func main() {
    // 1. THUMBNAIL_URL이 NULL인 NOTICE 게시글 조회
    posts := repo.GetNoticesWithoutThumbnail()

    for _, post := range posts {
        // 2. WEO_FILES에서 해당 게시글의 첫 이미지 조회
        file := repo.GetFirstImageFile("BB", post.SEQ)
        if file != nil {
            thumbnail := fmt.Sprintf("/files/%s/%s", file.FilePath, file.FileName)
            repo.UpdateThumbnail(post.SEQ, thumbnail)
        }

        // 3. CONTENTS에서 HTML 태그 제거 후 200자 추출
        if post.Summary == "" && post.Contents != "" {
            plain := stripHTML(post.Contents)
            summary := truncate(plain, 200)
            repo.UpdateSummary(post.SEQ, summary)
        }
    }
}
```

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

## 10. Go 백엔드 주요 구조

### 10.1 패키지 구성

```go
// cmd/server/main.go        — 엔트리포인트
// cmd/backfill/main.go      — 기존 데이터 백필 도구
// internal/config/          — 환경 설정 (DB, 카카오 OAuth, 서버 포트, EasyPay PG)
// internal/handler/         — HTTP 핸들러 (feed, alumni, auth, profile, payment, admin, health)
// internal/service/         — 비즈니스 로직 (donate, easypay 포함)
// internal/repository/      — SQL 쿼리 (database/sql + sqlx)
// internal/model/           — 도메인 모델 구조체 (payment 모델 포함)
// internal/middleware/      — Auth, AuthBridge, CORS, CSRF, AdminAuth, Logger, RateLimit
// internal/job/             — 배치 작업 (스냅샷, 세션 정리)
// templates/                — HTML 템플릿 (EasyPay relay 등)
```

### 10.2 기술 선택

| 항목 | 선택 | 이유 |
|------|------|------|
| HTTP 라우터 | `chi` | 경량, 미들웨어 체인, 표준 라이브러리 호환 |
| DB 드라이버 | `go-sql-driver/mysql` + `sqlx` | MariaDB 10.1 호환. CTE/Window Function 미사용 주의 (§2.4) |
| 인증 | JWT (access token) + DB 세션 | 카카오 OAuth2 연동 |
| 로깅 | `zerolog` | 경량 구조화 로깅, 메모리 할당 최소 |
| Markdown→HTML | `yuin/goldmark` | GFM 지원 (table, strikethrough), 확장 용이, 표준적 |
| HTML 새니타이즈 | `microcosm-cc/bluemonday` | UGC 정책 기반 XSS 방어, `<img>` 허용 설정 가능 |
| 이미지 처리 | `disintegration/imaging` | 리사이즈, Go 네이티브 |
| 캐시 | `patrickmn/go-cache` | 인메모리 (기부액, 필터 옵션, OAuth state 등) |

### 10.3 로깅 및 로그 관리

```go
// 구조화 로깅 (zerolog)
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

// 요청 로깅 미들웨어
func RequestLogger(logger zerolog.Logger) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
            next.ServeHTTP(ww, r)
            logger.Info().
                Str("method", r.Method).
                Str("path", r.URL.Path).
                Int("status", ww.Status()).
                Dur("duration", time.Since(start)).
                Msg("request")
        })
    }
}
```

**logrotate 설정:**
```
# /etc/logrotate.d/alumni-backend
/var/logs/alumni/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    maxsize 50M
}
```

### 10.4 기부액 스냅샷 배치

```go
func (s *DonationSnapshotJob) Start() {
    // 서버 시작 시 오늘 스냅샷 존재 여부 확인 → 없으면 즉시 생성
    // (서버가 자정 직전 재시작되어 당일 스냅샷이 누락되는 경우 방지)
    if exists, _ := s.repo.HasSnapshotForDate(time.Now()); !exists {
        if err := s.createSnapshot(); err != nil {
            s.logger.Error().Err(err).Msg("startup snapshot backfill failed")
        } else {
            s.logger.Info().Msg("startup snapshot backfill completed")
        }
    }

    // 매일 자정 00:05에 스냅샷 생성
    go func() {
        for {
            now := time.Now()
            next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, now.Location())
            time.Sleep(time.Until(next))

            if err := s.createSnapshot(); err != nil {
                s.logger.Error().Err(err).Msg("donation snapshot failed")
                // 실패해도 프론트는 직전 스냅샷을 폴백으로 사용
            } else {
                s.logger.Info().Msg("donation snapshot created")
            }
        }
    }()
}

func (s *DonationSnapshotJob) createSnapshot() error {
    // 1. WEO_ORDER에서 O_TYPE='A', O_PAYMENT='Y' 합산
    total, err := s.repo.SumDonations()
    if err != nil { return err }
    donorCount, err := s.repo.CountDonors()
    if err != nil { return err }

    // 2. DONATION_CONFIG에서 수동 조정액, 목표금액 조회
    config, err := s.repo.GetActiveConfig()
    if err != nil { return err }

    // 3. DONATION_SNAPSHOT에 UPSERT
    return s.repo.UpsertSnapshot(time.Now(), total, config.ManualAdj, donorCount, config.Goal)
}
```

---

## 11. 에러 핸들링 / 모니터링 전략

### 11.1 Go 에러 응답 표준

```go
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// 표준 에러 응답
func respondError(w http.ResponseWriter, status int, code string, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(APIError{Code: code, Message: msg})
}

// 사용 예시
respondError(w, 401, "UNAUTHORIZED", "로그인이 필요합니다")
respondError(w, 404, "NOT_FOUND", "게시글을 찾을 수 없습니다")
respondError(w, 429, "RATE_LIMITED", "요청이 너무 많습니다. 잠시 후 다시 시도하세요")
```

### 11.2 모니터링 체크리스트

| 항목 | 도구 | 비용 |
|------|------|------|
| 업타임 모니터링 | UptimeRobot | 무료 (50 모니터) |
| 헬스체크 | `GET /api/health` (DB 연결 포함) | — |
| 로그 | zerolog → 파일 → logrotate (7일 보관) | — |
| 디스크 | cron으로 `df -h` 체크, 80% 초과 시 알림 | — |
| 메모리 | cron으로 `free -m` 체크, 90% 초과 시 알림 | — |

**간단한 디스크/메모리 알림 스크립트:**
```bash
#!/bin/bash
# /etc/cron.d/server-monitor (매 10분)
DISK_USAGE=$(df / | awk 'NR==2 {print $5}' | tr -d '%')
MEM_USAGE=$(free | awk 'NR==2 {printf "%.0f", $3/$2*100}')

if [ "$DISK_USAGE" -gt 80 ] || [ "$MEM_USAGE" -gt 90 ]; then
    # 슬랙 웹훅 또는 이메일 발송
    curl -X POST -H 'Content-type: application/json' \
      --data "{\"text\":\"⚠️ 서버 경고: Disk ${DISK_USAGE}%, Memory ${MEM_USAGE}%\"}" \
      $SLACK_WEBHOOK_URL
fi
```

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

## 13. 향후 확장 로드맵

```
v1.0 — MVP
  ✅ 공지 피드 (무한스크롤)
  ✅ Hero 섹션
  ✅ 누적 기부액 (일별 스냅샷 + 수동 조정 + 시작 시 누락분 체크)
  ✅ 인피드 기부 배너 (기부 현황 + CTA 버튼) + Donate 탭
  ✅ 인피드 광고 (PREMIUM/GOLD 등급, 중복 방지, 노출/클릭 추적)
  ✅ 동문찾기 (전방 일치 검색)
  ✅ 카카오 로그인 (기존 회원 연동 + 신규 가입 + OAuth state 검증)
  ✅ 레거시 ID/PW 로그인 (MySQL native password 해시 호환)
  ✅ PHP ↔ Go 인증 브릿지 (레거시 쿠키 5종 + WEO_MEMBER_LOG 호환)
  ✅ Markdown 콘텐츠 시스템 (Markdown→HTML→Encode 저장/Decode 조회)
  ✅ 관리자 웹 (대시보드, 공지 CRUD + Markdown 에디터, 광고, 기부, 회원 관리)
  ✅ UI 디자인 시스템 (Tailwind v4, Royal Indigo, Radix UI, cva, Lucide)
  ✅ CSRF 방어 (Origin/Referer 검증)
  ✅ 프로필 수정
  ✅ HTTPS (Let's Encrypt)
  ✅ Graceful Shutdown
  ✅ 헬스체크 / 기본 모니터링

v1.1 — 기부 결제 + 개선
  □ 온라인 기부: 일시후원 카드결제 (EasyPay PG, §16 Phase 1)
  □ 좋아요 기능 (WEO_BOARDLIKE 연동)
  □ 댓글 보기 (WEO_BOARDCOMAND 연동)
  □ Feed Card Action Bar 확장: 좋아요(Heart), 댓글(MessageCircle), 공유(Share) 추가
  □ 이미지 갤러리 (다중 이미지 슬라이드)
  □ PHP 레거시 완전 제거 → 인증 브릿지 제거 → DDusr* 쿠키 중단 → alumni_token 전용

v1.2 — 기부 확장
  □ 월정기후원 카드결제 (EasyPay 정기결제, §16 Phase 2)
  □ 계좌이체 / 휴대폰 결제 추가 (§16 Phase 3)
  □ 프로젝트별 기부 (O_GATE='F', P_SEQ)

v2.0 — 인프라 확장
  □ MariaDB 10.6+ LTS 업그레이드 (InnoDB 전환과 동시 진행 권장)
  □ MyISAM → InnoDB 전환 (PHP 제거 후)
  □ 서버 메모리 증설 (최소 2GB)
  □ 웹 푸시 알림
  □ PostgreSQL 전환 검토
  □ 동문찾기 FULLTEXT 검색 도입
```

---

## 14. 선행 확인 사항 (확인 완료)

개발 착수 전 확인이 필요했던 5개 항목이 모두 확인 완료되었습니다.

| # | 확인 항목 | 결과 | 설계 반영 |
|---|-----------|------|-----------|
| 1 | `getUniqueTID()` 토큰 포맷 | ✅ `md5(uniqid(rand()))` → 32자 lowercase hex | `generateSessionID()`: `crypto/rand` 16바이트 → hex → 32자. varchar(40) 컬럼에 저장 가능 |
| 2 | `WEO_MEMBER_LOG.SESSIONID` 길이 | ✅ `varchar(40)` — 32자 hex 저장 충분 | 64자 hex 토큰 대신 32자 hex 사용 확정 |
| 3 | JS에서 `DDusr*` 쿠키 직접 접근 | ✅ **JS 접근 없음** — 서버 템플릿 치환만 사용 | `HttpOnly: true`로 전환 (XSS 방어 강화) |
| 4 | dadms의 `WEO_MEMBER_LOG` 사용 | ✅ **INSERT만** (SELECT 없음). `REG_GEOCODE` 미사용 | `InsertLoginLog()`에서 `REG_GEOCODE` 제외 (레거시 PHP와 동일) |
| 5 | DB 버전 | ✅ **MariaDB 10.1.38** (2017, EOL 2020) | §2.4 SQL 호환성 가이드라인 추가. CTE/Window Function 사용 금지. Phase 2에서 10.6+ 업그레이드 권장 |

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
| 인가 | `AdminAuthMiddleware` — `USR_STATUS = 'AAA'` 확인 |

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
        // 관리자 권한 확인 (USR_STATUS = 'AAA')
        if user.USRStatus != "AAA" {
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

## 16. 기부 결제 시스템 설계

V1의 EasyPay(KICC) PG 결제 기능을 V2(Go + React)로 이식합니다. 단계적으로 구현하며, Phase 1(일시후원 카드결제 MVP)부터 시작합니다.

### 16.1 결제 흐름 개요

```
[React SPA]                    [Go Backend]                    [EasyPay PG]
     │                              │                              │
  1. POST /api/donation/orders      │                              │
     │─────────────────────────────>│ INSERT WEO_ORDER (pending)   │
     │<─────────────────────────────│ { orderSeq, paymentParams }  │
     │                              │                              │
  2. EasyPayBridge 컴포넌트         │                              │
     SDK 로딩, form 세팅,           │                              │
     easypay_card_webpay() 호출     │                              │
     │── form POST ────────────────>│                              │
     │                              │ 3. POST /pg/easypay/relay    │
     │                              │    auto-submit HTML 응답     │
     │                              │────── form POST ────────────>│
     │                              │                              │
     │                     (사용자가 EasyPay UI에서 결제 완료)       │
     │                              │                              │
     │                              │<── POST /pg/easypay/return ──│
     │                              │ 4. sp_res_cd 파싱            │
     │                              │    ep_cli 바이너리 실행       │
     │                              │    INSERT WEO_PG_DATA        │
     │                              │    UPDATE WEO_ORDER          │
     │                              │    캐시 무효화               │
     │<──── 302 redirect ──────────│                              │
  5. /donation/result?status=       │                              │
     success&order=xxx              │                              │
```

> **핵심 발견사항:** EasyPay의 서버 사이드 승인은 HTTP API 호출이 아닙니다. **네이티브 Linux 바이너리** (`ep_cli`)를 셸아웃하여 `gw.easypay.co.kr`과 통신합니다. Go에서 `os/exec`로 래핑합니다. 바이너리는 V1 서버에 이미 존재합니다: `dflh-saf-v1/html/_sys/payment/*/bin/linux_64/ep_cli`

### 16.2 EasyPay 설정

#### Mall ID 및 상수

| 결제 유형 | Mall ID | sp_pay_type | sp_window_type | 용도 |
|-----------|---------|-------------|----------------|------|
| 일시후원 (immediately) | `05542574` | `11` (카드), `21` (계좌이체), `31` (휴대폰) | `iframe` | Phase 1 |
| 월정기후원 (profile) | `05543499` | `81` (정기결제) | `submit` | Phase 2 |

#### Gate/Type 매핑 (V1 컨벤션)

| 입력 (gate) | DB O_GATE | 설명 |
|-------------|-----------|------|
| `immediately` | `S` | 일시후원 (Single) |
| `profile` | `P` | 월정기후원 (Profile) |
| `project` | `F` | 프로젝트 기부 (Fund) |

| 입력 (payType) | DB O_PAY_TYPE | EasyPay sp_pay_type |
|----------------|---------------|---------------------|
| `CARD` | `CARD` | `11` |
| `BANK` | `BANK` | `21` |
| `HP` | `HP` | `31` |

#### 환경 변수

```
EASYPAY_IMMEDIATELY_MALL_ID=05542574
EASYPAY_PROFILE_MALL_ID=05543499
EASYPAY_GW_URL=gw.easypay.co.kr
EASYPAY_GW_PORT=80
EASYPAY_BIN_BASE=/var/www/html/_sys/payment
EASYPAY_RETURN_BASE_URL=https://www.daeilfoundation.or.kr
```

> **테스트 환경:** SDK는 `testsp.easypay.co.kr`, 게이트웨이는 `testgw.easypay.co.kr:80` 사용

#### Config 구조체

```go
// internal/config/config.go — EasyPayConfig 추가

type EasyPayConfig struct {
    ImmediatelyMallID string
    ProfileMallID     string
    GatewayURL        string
    GatewayPort       string
    BinBase           string        // ep_cli 바이너리 기본 경로
    ReturnBaseURL     string        // 콜백 베이스 URL
}
// CertFile, BinPath, LogDir는 EasyPayService.Approve() 내에서 BinBase로부터 파생

// Load()에서:
EasyPay: EasyPayConfig{
    ImmediatelyMallID:  getEnv("EASYPAY_IMMEDIATELY_MALL_ID", "05542574"),
    ProfileMallID:      getEnv("EASYPAY_PROFILE_MALL_ID", "05543499"),
    GatewayURL:         getEnv("EASYPAY_GW_URL", "testgw.easypay.co.kr"),
    GatewayPort:        getEnv("EASYPAY_GW_PORT", "80"),
    BinBase:            getEnv("EASYPAY_BIN_BASE", "/var/www/html/_sys/payment"),
    ReturnBaseURL:      getEnv("EASYPAY_RETURN_BASE_URL", "http://localhost:8080"),
},
PGAuditLogPath: getEnv("PG_AUDIT_LOG_PATH", "/var/logs/pg/pg-audit.log"),
```

### 16.3 백엔드 신규 파일

#### 파일 목록

| 신규 파일 | 용도 |
|-----------|------|
| `internal/handler/payment_handler.go` | `CreateOrder`, `GetOrder`, `EasyPayRelay`, `EasyPayReturn` |
| `internal/handler/templates/easypay_relay.html` | auto-submit HTML form → `sp.easypay.co.kr/ep8/MainAction.do` (`go:embed`로 바이너리에 포함) |
| `internal/service/easypay_service.go` | `ep_cli` 바이너리 래퍼 — `Approve()` 메서드 |
| `internal/service/donate_service.go` | `CreateOrder`, `ConfirmPayment` 비즈니스 로직 |
| `internal/service/pg_audit_logger.go` | PG 결제 감사 로거 — JSON-per-line, fsync 보장 |
| `internal/repository/donate_repo.go` | `InsertOrder`, `InsertPGData`, `UpdateOrderPayment`, `GetOrder` |
| `internal/model/payment.go` | `CreateOrderRequest`, `PaymentParams`, `PGData`, `ApproveResult` |

#### 수정 파일

| 파일 | 변경 내용 |
|------|-----------|
| `cmd/server/main.go` | `/pg/*` 라우트 등록 (CSRF 제외), `/api/donation/orders` 라우트 추가, 서비스 와이어링 |
| `internal/config/config.go` | `EasyPayConfig` 구조체 추가 |

#### 모델 정의

```go
// internal/model/payment.go

// 주문 생성 요청
type CreateOrderRequest struct {
    Amount  int    `json:"amount"`
    PayType string `json:"payType"`  // "CARD" (Phase 1)
    Gate    string `json:"gate"`     // "immediately" (Phase 1)
}

// EasyPay SDK에 전달할 파라미터
type PaymentParams struct {
    MallID      string `json:"mallId"`
    OrderNo     string `json:"orderNo"`
    ProductAmt  string `json:"productAmt"`
    ProductName string `json:"productName"`
    PayType     string `json:"payType"`      // "11", "21", "31", "81"
    ReturnURL   string `json:"returnUrl"`
    RelayURL    string `json:"relayUrl"`
    WindowType  string `json:"windowType"`   // "iframe" or "submit"
    UserName    string `json:"userName"`
    MallName    string `json:"mallName"`
    Currency    string `json:"currency"`     // "00" (KRW)
    Charset     string `json:"charset"`      // "UTF-8"
    LangFlag    string `json:"langFlag"`     // "KOR"
}

// 주문 생성 응답
type CreateOrderResponse struct {
    OrderSeq      int           `json:"orderSeq"`
    PaymentParams PaymentParams `json:"paymentParams"`
}

// 주문 조회 응답 (결과 페이지용)
type OrderDetail struct {
    OrderSeq int    `json:"orderSeq"`
    Amount   int    `json:"amount"`
    Status   string `json:"status"`  // "Y" or "N"
    PaidAt   string `json:"paidAt"`
}

// ep_cli 승인 요청
type ApproveRequest struct {
    OrderNo     string
    EncryptData string
    SessionKey  string
    TraceNo     string
    ClientIP    string
}

// ep_cli 승인 결과
type ApproveResult struct {
    ResCode      string // res_cd (0000 = 성공)
    ResMsg       string // res_msg
    CNO          string // PG 거래번호
    Amount       string // 승인 금액
    AuthNo       string // 승인번호
    TranDate     string // 거래일시
    CardNo       string // 마스킹된 카드번호
    PayType      string // 결제수단 코드 (11/21/31)
    IssuerName   string // 발급사명
    AcquirerName string // 매입사명
}

// PG 데이터 (WEO_PG_DATA 테이블)
type PGData struct {
    CNO      string
    ResCD    string
    ResMsg   string
    Amount   int
    NumCard  string
    TranDate string
    AuthNo   string
    PayType  string
    OSeq     int
}
```

#### 라우트 등록 (main.go 변경)

```go
// cmd/server/main.go — 라우터 재구성

// 1. 서비스 와이어링 추가
donateRepo := repository.NewDonateRepository(db)
easypayService := service.NewEasyPayService(cfg.EasyPay)
pgAuditLogger, err := service.NewPGAuditLogger(cfg.PGAuditLogPath)
if err != nil {
    logger.Fatal().Err(err).Msg("failed to open PG audit log")
}
defer pgAuditLogger.Close()
donateService := service.NewDonateService(donateRepo, easypayService, cacheStore, logger, pgAuditLogger)
paymentHandler := handler.NewPaymentHandler(donateService, cfg.EasyPay)

// 2. PG 콜백 라우트 — CSRF/Auth 미들웨어 없이 등록
//    ⚠️ 기존 전역 CSRF 미들웨어를 그룹 기반으로 변경 필요
mux := chi.NewMux()
mux.Use(chimw.Recoverer)
mux.Use(mw.RequestLogger(logger))
mux.Use(mw.CORSMiddleware(allowedOrigins))
mux.Use(mw.MaxBodySize(1 << 20))

// PG 콜백 라우트 — CSRF 없음, Auth 없음
mux.Route("/pg", func(r chi.Router) {
    r.Post("/easypay/relay", paymentHandler.EasyPayRelay)
    r.Post("/easypay/return", paymentHandler.EasyPayReturn)
})

// API 라우트 — CSRF 적용
mux.Group(func(r chi.Router) {
    r.Use(mw.CSRFMiddleware(allowedOrigins))

    // 공개 API (인증 불필요)
    r.Get("/api/health", healthHandler.Check)
    r.Get("/api/feed", feedHandler.GetFeed)
    r.Get("/api/donation/summary", donationHandler.GetSummary)
    // ... 기존 공개 라우트 ...

    // 인증 필수 API
    r.Group(func(r chi.Router) {
        r.Use(mw.AuthMiddleware(authService))
        r.Post("/api/donation/orders", paymentHandler.CreateOrder)
        r.Get("/api/donation/orders/{seq}", paymentHandler.GetOrder)
        // ... 기존 인증 라우트 ...
    })

    // 관리자 API
    r.Route("/api/admin", func(r chi.Router) {
        r.Use(mw.AuthMiddleware(authService))
        r.Use(mw.AdminAuthMiddleware)
        // ... 기존 관리자 라우트 ...
    })
})
```

> **⚠️ 주의:** 기존 `main.go`에서 CSRF 미들웨어가 전역으로 적용되어 있다면, `/pg/*` 라우트를 CSRF 그룹 밖에 배치하도록 리팩토링이 필요합니다.

#### Repository (donate_repo.go)

```go
// internal/repository/donate_repo.go

type DonateRepository struct {
    DB *sqlx.DB
}

func NewDonateRepository(db *sqlx.DB) *DonateRepository {
    return &DonateRepository{DB: db}
}

// InsertOrder — pending 상태 주문 생성, O_SEQ 반환
func (r *DonateRepository) InsertOrder(usrSeq int, gate string, payType string, price int, ip string) (int64, error) {
    result, err := r.DB.Exec(`
        INSERT INTO WEO_ORDER (USR_SEQ, O_GATE, O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR)
        VALUES (?, ?, ?, 'A', NOW(), ?, 'N', NOW(), ?)
    `, usrSeq, gate, payType, price, ip)
    if err != nil {
        return 0, err
    }
    return result.LastInsertId()
}

// InsertPGData — PG 승인 데이터 저장, PG_SEQ 반환
func (r *DonateRepository) InsertPGData(data *model.PGData) (int64, error) {
    result, err := r.DB.Exec(`
        INSERT INTO WEO_PG_DATA (CNO, RES_CD, RES_MSG, AMOUNT, NUM_CARD, TRAN_DATE, AUTH_NO, PAY_TYPE, O_SEQ)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, data.CNO, data.ResCD, data.ResMsg, data.Amount, data.NumCard, data.TranDate, data.AuthNo, data.PayType, data.OSeq)
    if err != nil {
        return 0, err
    }
    return result.LastInsertId()
}

// UpdateOrderPayment — 결제 완료 처리 (멱등: O_PAYMENT='N' 조건)
func (r *DonateRepository) UpdateOrderPayment(orderSeq int, amount int, pgSeq int64, ip string) (int64, error) {
    result, err := r.DB.Exec(`
        UPDATE WEO_ORDER
        SET O_PAYMENT = 'Y', O_PAY = ?, O_PAYDATE = NOW(), O_PG_SEQ = ?, O_STATUS = 'Y',
            EDT_DATE = NOW(), EDT_IPADDR = ?
        WHERE O_SEQ = ? AND O_PAYMENT = 'N'
    `, amount, pgSeq, ip, orderSeq)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// GetOrder — 주문 상세 조회
func (r *DonateRepository) GetOrder(orderSeq int, usrSeq int) (*model.OrderDetail, error) {
    var order model.OrderDetail
    err := r.DB.Get(&order, `
        SELECT O_SEQ AS OrderSeq, O_PRICE AS Amount, O_PAYMENT AS Status,
               IFNULL(DATE_FORMAT(O_PAYDATE, '%Y-%m-%d %H:%i:%s'), '') AS PaidAt
        FROM WEO_ORDER
        WHERE O_SEQ = ? AND USR_SEQ = ? AND O_TYPE = 'A'
    `, orderSeq, usrSeq)
    if err != nil {
        return nil, err
    }
    return &order, nil
}

// GetOrderPrice — 주문 금액 조회 (결제 검증용)
func (r *DonateRepository) GetOrderPrice(orderSeq int) (int, error) {
    var price int
    err := r.DB.Get(&price, `
        SELECT O_PRICE FROM WEO_ORDER WHERE O_SEQ = ? AND O_PAYMENT = 'N'
    `, orderSeq)
    return price, err
}
```

#### EasyPay Service (ep_cli 래퍼)

```go
// internal/service/easypay_service.go

type EasyPayService struct {
    cfg config.EasyPayConfig
}

func NewEasyPayService(cfg config.EasyPayConfig) *EasyPayService {
    return &EasyPayService{cfg: cfg}
}

// Approve — ep_cli 바이너리를 실행하여 서버 사이드 승인 처리
func (s *EasyPayService) Approve(req model.ApproveRequest) (*model.ApproveResult, error) {
    binPath := filepath.Join(s.cfg.BinBase, "immediately", "bin", "linux_64", "ep_cli")
    certFile := filepath.Join(s.cfg.BinBase, "immediately", "mobile", "cert", "pg_cert.pem")
    logDir := filepath.Join(s.cfg.BinBase, "immediately", "mobile", "log")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // ep_cli expects: -h key1=val1,key2=val2,...
    args := fmt.Sprintf(
        "order_no=%s,cert_file=%s,mall_id=%s,tr_cd=00101000,gw_url=%s,gw_port=%s,enc_data=%s,snd_key=%s,trace_no=%s,cust_ip=%s,log_dir=%s,log_level=1",
        req.OrderNo, certFile, s.cfg.ImmediatelyMallID,
        s.cfg.GatewayURL, s.cfg.GatewayPort,
        req.EncryptData, req.SessionKey, req.TraceNo, req.ClientIP, logDir,
    )

    cmd := exec.CommandContext(ctx, binPath, "-h", args)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("ep_cli 실행 실패: %w", err)
    }

    // 응답 파싱: "res_cd=0000\x1Fres_msg=정상\x1Fcno=xxx\x1F..."
    return parseEasyPayResponse(string(output)), nil
}

// parseEasyPayResponse — \x1F (Unit Separator) 구분자로 key=value 파싱
func parseEasyPayResponse(raw string) *model.ApproveResult {
    fields := strings.Split(raw, "\x1F")
    m := make(map[string]string)
    for _, field := range fields {
        parts := strings.SplitN(field, "=", 2)
        if len(parts) == 2 {
            m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
        }
    }
    return &model.ApproveResult{
        ResCode:      m["res_cd"],
        ResMsg:       m["res_msg"],
        CNO:          m["cno"],
        Amount:       m["amount"],
        AuthNo:       m["auth_no"],
        TranDate:     m["tran_date"],
        CardNo:       m["card_no"],
        PayType:      m["pay_type"],
        IssuerName:   m["issuer_nm"],
        AcquirerName: m["acquirer_nm"],
    }
}
```

> **⚠️ 인코딩 주의:** `ep_cli` 출력이 EUC-KR일 수 있습니다. 필요 시 `golang.org/x/text/encoding/korean` 패키지로 변환합니다.

> **⚠️ 개발 환경:** `ep_cli`는 Linux 전용 바이너리로 macOS에서 실행 불가합니다. 로컬 개발 시 mock 응답을 반환하는 별도 로직이 필요합니다.

#### Donate Service (비즈니스 로직)

```go
// internal/service/donate_service.go

type DonateService struct {
    repo    *repository.DonateRepository
    cache   *cache.Cache
    epSvc   *EasyPayService
    logger  zerolog.Logger
    pgAudit *PGAuditLogger
}

func NewDonateService(
    repo *repository.DonateRepository,
    epSvc *EasyPayService,
    cacheStore *cache.Cache,
    logger zerolog.Logger,
    pgAudit *PGAuditLogger,
) *DonateService {
    return &DonateService{
        repo: repo, epSvc: epSvc, cache: cacheStore,
        logger: logger, pgAudit: pgAudit,
    }
}

// CreateOrder — pending 주문 생성, EasyPay 파라미터 반환
func (s *DonateService) CreateOrder(user *model.AuthUser, req model.CreateOrderRequest, ip string, cfg config.EasyPayConfig) (*model.CreateOrderResponse, error) {
    // 1. 검증
    if req.Amount < 10000 {
        return nil, errors.New("최소 기부 금액은 10,000원입니다")
    }
    if req.PayType != "CARD" {
        return nil, errors.New("현재 카드 결제만 지원합니다")
    }
    if req.Gate != "immediately" {
        return nil, errors.New("현재 일시후원만 지원합니다")
    }

    // 2. DB INSERT (gate "immediately" → O_GATE "S")
    orderSeq, err := s.repo.InsertOrder(user.USRSeq, "S", req.PayType, req.Amount, ip)
    if err != nil {
        return nil, fmt.Errorf("주문 생성 실패: %w", err)
    }

    // 3. EasyPay 파라미터 구성
    orderNo := strconv.FormatInt(orderSeq, 10)
    params := model.PaymentParams{
        MallID:      cfg.ImmediatelyMallID,
        OrderNo:     orderNo,
        ProductAmt:  strconv.Itoa(req.Amount),
        ProductName: fmt.Sprintf("%s님_일시후원_%s원", user.USRName, formatComma(req.Amount)),
        PayType:     "11", // 카드
        ReturnURL:   cfg.ReturnBaseURL + "/pg/easypay/return",
        RelayURL:    "/pg/easypay/relay",
        WindowType:  "iframe",
        UserName:    user.USRName,
        MallName:    "대일외국어고등학교 장학회",
        Currency:    "00",
        Charset:     "UTF-8",
        LangFlag:    "KOR",
    }

    return &model.CreateOrderResponse{
        OrderSeq:      int(orderSeq),
        PaymentParams: params,
    }, nil
}

// ConfirmPayment — ep_cli 승인 + DB 업데이트 + PG 감사 로깅
func (s *DonateService) ConfirmPayment(orderNo string, encData string, sessionKey string, traceNo string, clientIP string) error {
    orderSeq, err := strconv.Atoi(orderNo)
    if err != nil {
        return fmt.Errorf("invalid order number: %w", err)
    }

    // 1. ep_cli 승인
    result, err := s.epSvc.Approve(model.ApproveRequest{
        OrderNo: orderNo, EncryptData: encData,
        SessionKey: sessionKey, TraceNo: traceNo, ClientIP: clientIP,
    })
    if err != nil {
        s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("ep_cli approval failed")
        s.pgAudit.Log(orderNo, "approve_fail", nil, err)
        return err
    }
    if result.ResCode != "0000" {
        s.logger.Warn().Str("resCode", result.ResCode).Str("resMsg", result.ResMsg).Str("orderNo", orderNo).Msg("PG approval rejected")
        s.pgAudit.Log(orderNo, "approve_fail", result, nil)
        return fmt.Errorf("PG 승인 실패: %s (%s)", result.ResMsg, result.ResCode)
    }

    s.pgAudit.Log(orderNo, "approve_success", result, nil)

    // 2. 금액 검증 (주문 금액 vs PG 승인 금액)
    orderPrice, err := s.repo.GetOrderPrice(orderSeq)
    if err != nil {
        return fmt.Errorf("주문 조회 실패: %w", err)
    }
    approvedAmount, _ := strconv.Atoi(result.Amount)
    if orderPrice != approvedAmount {
        s.logger.Error().Int("orderPrice", orderPrice).Int("approvedAmount", approvedAmount).Str("orderNo", orderNo).Msg("amount mismatch")
        return fmt.Errorf("금액 불일치: 주문 %d, 승인 %d", orderPrice, approvedAmount)
    }

    // 3. PG 데이터 INSERT
    pgData := &model.PGData{
        CNO: result.CNO, ResCD: result.ResCode, ResMsg: result.ResMsg,
        Amount: approvedAmount, NumCard: result.CardNo, TranDate: result.TranDate,
        AuthNo: result.AuthNo, PayType: result.PayType, OSeq: orderSeq,
    }
    pgSeq, err := s.repo.InsertPGData(pgData)
    if err != nil {
        s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to insert PG data")
        s.pgAudit.Log(orderNo, "db_insert_fail", result, err)
        return fmt.Errorf("PG 데이터 저장 실패: %w", err)
    }

    // 4. 주문 결제 완료 (멱등: O_PAYMENT='N' 조건)
    affected, err := s.repo.UpdateOrderPayment(orderSeq, approvedAmount, pgSeq, clientIP)
    if err != nil {
        s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to update order payment")
        s.pgAudit.Log(orderNo, "db_update_fail", result, err)
        return fmt.Errorf("주문 업데이트 실패: %w", err)
    }
    if affected == 0 {
        s.logger.Info().Str("orderNo", orderNo).Msg("order already processed (idempotent)")
        return nil
    }

    // 5. 캐시 무효화
    s.cache.Delete("donation_summary")
    s.logger.Info().Str("orderNo", orderNo).Int("amount", approvedAmount).Msg("payment confirmed")
    return nil
}
```

#### Payment Handler

```go
// internal/handler/payment_handler.go

//go:embed templates/easypay_relay.html
var relayTemplateContent string

type PaymentHandler struct {
    donateService *service.DonateService
    easypayConfig config.EasyPayConfig
    relayTmpl     *template.Template  // go:embed로 바이너리에 포함된 relay 템플릿
}

func NewPaymentHandler(donateService *service.DonateService, cfg config.EasyPayConfig) *PaymentHandler {
    tmpl := template.Must(template.New("relay").Parse(relayTemplateContent))
    return &PaymentHandler{donateService: donateService, easypayConfig: cfg, relayTmpl: tmpl}
}

// CreateOrder — POST /api/donation/orders (인증 필수)
func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetAuthUser(r.Context())
    if user == nil {
        respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
        return
    }

    var req model.CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "INVALID_BODY", "잘못된 요청입니다")
        return
    }

    ip := r.Header.Get("X-Real-IP")
    if ip == "" {
        ip = r.RemoteAddr
    }

    result, err := h.donateService.CreateOrder(user, req, ip, h.easypayConfig)
    if err != nil {
        respondError(w, http.StatusBadRequest, "ORDER_FAILED", err.Error())
        return
    }
    respondJSON(w, http.StatusOK, result)
}

// EasyPayRelay — POST /pg/easypay/relay (CSRF 없음, Auth 없음)
// EasyPay SDK form POST → PG 서버로 auto-submit HTML 응답
func (h *PaymentHandler) EasyPayRelay(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    h.relayTmpl.Execute(w, r.Form)
}

// EasyPayReturn — POST /pg/easypay/return (CSRF 없음, Auth 없음)
// EasyPay 결제 완료 후 콜백 → ep_cli 승인 → DB 업데이트 → 리다이렉트
func (h *PaymentHandler) EasyPayReturn(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()

    resCode := r.FormValue("sp_res_cd")
    orderNo := r.FormValue("sp_order_no")

    // 1. PG 응답 코드 확인
    if resCode != "0000" {
        http.Redirect(w, r, "/donation/result?status=failed&reason=pg_error", http.StatusFound)
        return
    }

    // 2. ep_cli 승인 + DB 처리
    ip := r.Header.Get("X-Real-IP")
    if ip == "" {
        ip = r.RemoteAddr
    }

    err := h.donateService.ConfirmPayment(
        orderNo,
        r.FormValue("sp_encrypt_data"),
        r.FormValue("sp_sessionkey"),
        r.FormValue("sp_trace_no"),
        ip,
    )
    if err != nil {
        // 로그 기록 (결제는 PG에서 승인되었으나 DB 처리 실패 — 심각)
        http.Redirect(w, r, "/donation/result?status=failed&reason=server_error", http.StatusFound)
        return
    }

    http.Redirect(w, r, fmt.Sprintf("/donation/result?status=success&order=%s", orderNo), http.StatusFound)
}

// GetOrder — GET /api/donation/orders/{seq} (인증 필수)
func (h *PaymentHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetAuthUser(r.Context())
    if user == nil {
        respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
        return
    }

    seq := parseIntParam(chi.URLParam(r, "seq"))
    order, err := h.donateService.GetOrder(seq, user.USRSeq)
    if err != nil {
        respondError(w, http.StatusNotFound, "NOT_FOUND", "주문을 찾을 수 없습니다")
        return
    }
    respondJSON(w, http.StatusOK, order)
}
```

#### EasyPay Relay 템플릿

```html
<!-- internal/handler/templates/easypay_relay.html (go:embed로 바이너리에 포함) -->
<!-- EasyPay SDK form POST → sp.easypay.co.kr/ep8/MainAction.do auto-submit -->
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
<form id="pgForm" method="post" action="https://sp.easypay.co.kr/ep8/MainAction.do">
  {{range $key, $values := .}}
    {{range $values}}
      <input type="hidden" name="{{$key}}" value="{{.}}">
    {{end}}
  {{end}}
</form>
<script>document.getElementById('pgForm').submit();</script>
</body>
</html>
```

### 16.4 프론트엔드 신규 파일

#### 파일 목록

| 신규 파일 | 용도 |
|-----------|------|
| `src/types/donate.ts` | 기부 결제 관련 TypeScript 타입 |
| `src/hooks/useDonateOrder.ts` | React Query mutation (주문 생성) |
| `src/components/donation/DonationForm.tsx` | 금액 + 결제수단 선택 + 확인 폼 |
| `src/components/donation/EasyPayBridge.tsx` | Hidden form + SDK 로딩 + 결제 호출 |
| `src/pages/DonationResultPage.tsx` | 결제 결과 (성공/실패) 화면 |

#### 수정 파일

| 파일 | 변경 내용 |
|------|-----------|
| `src/routes.tsx` | `/donation/result` 라우트 추가 |
| `src/pages/DonationPage.tsx` | `DonationForm` 컴포넌트 삽입 |

#### 타입 정의

```typescript
// src/types/donate.ts

export interface CreateOrderRequest {
  amount: number;
  payType: 'CARD';     // Phase 1
  gate: 'immediately'; // Phase 1
}

export interface PaymentParams {
  mallId: string;
  orderNo: string;
  productAmt: string;
  productName: string;
  payType: string;
  returnUrl: string;
  relayUrl: string;
  windowType: string;
  userName: string;
  mallName: string;
  currency: string;
  charset: string;
  langFlag: string;
}

export interface CreateOrderResponse {
  orderSeq: number;
  paymentParams: PaymentParams;
}

export interface OrderDetail {
  orderSeq: number;
  amount: number;
  status: 'Y' | 'N';
  paidAt: string;
}
```

#### 주문 생성 Hook

```typescript
// src/hooks/useDonateOrder.ts
import { useMutation } from '@tanstack/react-query';
import { api } from '../api/client';
import type { CreateOrderRequest, CreateOrderResponse } from '../types/donate';

export function useDonateOrder() {
  return useMutation({
    mutationFn: (req: CreateOrderRequest) =>
      api.post<CreateOrderResponse>('/api/donation/orders', req),
  });
}
```

#### DonationForm 컴포넌트

기존 `DonationPage.tsx`의 기부 현황 배너 아래에 삽입됩니다. Progressive Disclosure 패턴으로 각 단계를 순차적으로 표시합니다.

```
┌──────────────────────────────┐
│  누적 기부액: 1,234만원        │  ← 기존 기부 현황 (소셜 프루프)
│  ████████░░  67%  45명 참여   │
├──────────────────────────────┤
│  기부하기                      │  ← DonationForm
│                              │
│  기부방식                      │
│  [일시후원]  [월정기후원(준비중)]│
│                              │
│  기부금액                      │
│  [1만] [3만] [5만] [10만]     │  ← 2-col mobile, 4-col desktop
│  [30만] [50만] [100만] [200만] │
│  [직접 입력: ________원]       │  ← inputMode="numeric"
│                              │
│  결제수단                      │
│  ● 신용카드(체크카드)           │  ← Phase 1: 카드만 선택 가능
│  ○ 계좌이체 (준비중)           │
│  ○ 휴대폰 (준비중)            │
│                              │
│  ┌─ 기부내역 확인 ──────────┐ │  ← 모든 선택 완료 후 표시
│  │ 일시후원 / 50,000원 / 카드 │ │
│  └──────────────────────────┘ │
│  [ 기부하기 ]                 │  ← Button size="lg", shadow-primary-glow
├──────────────────────────────┤
│  기부 계좌 안내                │  ← 기존 계좌 정보 (폴백)
│  하나은행 393-910033-69205     │
└──────────────────────────────┘
```

**UX 규칙:**
- 금액 프리셋: `[10000, 30000, 50000, 100000, 300000, 500000, 1000000, 2000000]`
- 직접입력: `inputMode="numeric"`, 숫자만 허용, 콤마 포맷 표시
- 검증: 인라인 에러 메시지 (빨간 테두리 + 하단 텍스트), `alert()` 사용 금지
- 모바일: 하단 고정 CTA 버튼 (sticky bottom)
- 로그인 필수: 미로그인 시 "기부하기" 버튼 대신 "로그인 후 기부하기" 버튼 → `/login`으로 이동

#### EasyPayBridge 컴포넌트

주문 생성 성공 후 렌더링됩니다. SDK를 동적 로딩하고 `easypay_card_webpay()`를 호출합니다.

```tsx
// src/components/donation/EasyPayBridge.tsx
function EasyPayBridge({ params }: { params: PaymentParams }) {
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    // EasyPay SDK 동적 로딩
    const script = document.createElement('script');
    script.src = 'https://sp.easypay.co.kr/webpay/EasypayCard_Web.js';
    script.onload = () => {
      // SDK 호출: form, relayUrl, target, reserved1, reserved2, windowType, timeout
      window.easypay_card_webpay(
        formRef.current, params.relayUrl, "_self", "0", "0", params.windowType, 30
      );
    };
    document.head.appendChild(script);
    return () => document.head.removeChild(script);
  }, [params]);

  return (
    <form ref={formRef} name="payment" style={{ display: 'none' }}>
      <input type="hidden" name="sp_mall_id" value={params.mallId} />
      <input type="hidden" name="sp_order_no" value={params.orderNo} />
      <input type="hidden" name="sp_product_amt" value={params.productAmt} />
      <input type="hidden" name="sp_product_nm" value={params.productName} />
      <input type="hidden" name="sp_pay_type" value={params.payType} />
      <input type="hidden" name="sp_return_url" value={params.returnUrl} />
      <input type="hidden" name="sp_currency" value={params.currency} />
      <input type="hidden" name="sp_charset" value={params.charset} />
      <input type="hidden" name="sp_lang_flag" value={params.langFlag} />
      <input type="hidden" name="sp_user_nm" value={params.userName} />
      <input type="hidden" name="sp_mall_nm" value={params.mallName} />
      <input type="hidden" name="sp_window_type" value={params.windowType} />
      {/* EasyPay SDK가 채우는 응답 필드 */}
      <input type="hidden" name="sp_res_cd" value="" />
      <input type="hidden" name="sp_res_msg" value="" />
      <input type="hidden" name="sp_tr_cd" value="" />
      <input type="hidden" name="sp_sessionkey" value="" />
      <input type="hidden" name="sp_encrypt_data" value="" />
      <input type="hidden" name="sp_trace_no" value="" />
      <input type="hidden" name="sp_ret_pay_type" />
      <input type="hidden" name="sp_card_code" />
      <input type="hidden" name="sp_card_req_type" />
    </form>
  );
}
```

> **React vDOM 충돌 방지:** `EasyPayBridge`는 비제어 `ref`를 사용하며, SDK는 DOM을 직접 조작합니다. React의 가상 DOM과의 충돌을 피하기 위해 `style={{ display: 'none' }}`으로 숨기고, cleanup에서 script를 제거합니다.

#### DonationResultPage

라우트: `/donation/result?status=success|failed&order=X`

```
성공 시:
┌──────────────────────────────┐
│        ✅                    │
│  기부해 주셔서 감사합니다      │
│                              │
│  기부 금액: 50,000원          │
│  결제일시: 2026-02-12 14:30   │
│                              │
│  ※ 기부금은 연말정산 시       │
│    소득공제 혜택을 받을 수     │
│    있습니다.                  │
│                              │
│  [홈으로]  [기부 내역 보기]    │
└──────────────────────────────┘

실패 시:
┌──────────────────────────────┐
│        ❌                    │
│  결제에 실패했습니다           │
│                              │
│  사유: PG 결제 오류           │
│                              │
│  문의: 02-543-3558           │
│                              │
│  [다시 시도]                  │
└──────────────────────────────┘
```

### 16.5 DB 테이블

**Phase 1에서 신규 마이그레이션 불필요.** `WEO_ORDER`, `WEO_PG_DATA` 테이블은 V1에서 이미 존재합니다.

#### WEO_ORDER (기존 테이블, 결제 관련 컬럼)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `O_SEQ` | INT AUTO_INCREMENT | 주문 번호 (PK) |
| `USR_SEQ` | INT | 사용자 번호 |
| `O_GATE` | VARCHAR | 결제 방식 (S=일시, P=정기, F=프로젝트) |
| `O_PAY_TYPE` | VARCHAR | 결제 수단 (CARD, BANK, HP) |
| `O_TYPE` | CHAR(1) | 주문 유형 (A=기부) |
| `O_REGDATE` | DATETIME | 등록일시 |
| `O_PRICE` | INT | 주문 금액 |
| `O_PAYMENT` | CHAR(1) | 결제 완료 여부 (Y/N) |
| `O_PAY` | INT | 실 결제 금액 |
| `O_PAYDATE` | DATETIME | 결제 완료일시 |
| `O_PG_SEQ` | INT | PG 데이터 번호 (FK → WEO_PG_DATA) |
| `O_STATUS` | CHAR(1) | 주문 상태 (Y=완료) |
| `P_SEQ` | INT | 프로젝트 번호 (Phase 3) |

#### WEO_PG_DATA (기존 테이블)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `SEQ` | INT AUTO_INCREMENT | PG 데이터 번호 (PK) |
| `CNO` | VARCHAR | PG 거래번호 |
| `RES_CD` | VARCHAR | 응답 코드 |
| `RES_MSG` | VARCHAR | 응답 메시지 |
| `AMOUNT` | INT | 승인 금액 |
| `NUM_CARD` | VARCHAR | 마스킹된 카드번호 |
| `TRAN_DATE` | VARCHAR | 거래일시 |
| `AUTH_NO` | VARCHAR | 승인번호 |
| `PAY_TYPE` | VARCHAR | 결제수단 코드 (11/21/31) |
| `O_SEQ` | INT | 주문 번호 (FK → WEO_ORDER) |

### 16.6 보안 체크리스트

| 항목 | 방어 방법 |
|------|-----------|
| **SQL Injection** | sqlx `?` 플레이스홀더 사용 (V1의 문자열 연결 방식 복제 금지) |
| **CSRF** | `/pg/*`는 제외 (EasyPay cross-origin POST), `/api/*`는 CSRF 유지 |
| **멱등성** | `UPDATE ... WHERE O_PAYMENT='N'` — 중복 결제 방지 |
| **금액 검증** | `ep_cli` 응답 금액과 `WEO_ORDER.O_PRICE` 비교, 불일치 시 거부 |
| **카드정보 미저장** | EasyPay의 CNO 참조만 저장, 카드번호 원본은 절대 저장 금지 |
| **타임아웃** | `ep_cli` 호출에 30초 `context.WithTimeout` 적용 |
| **인증** | 주문 생성(`/api/donation/orders`)은 AuthMiddleware 필수 |
| **주문 소유권** | `GetOrder`에서 `USR_SEQ` 조건 — 타인의 주문 조회 불가 |

### 16.7 Phase 2: 월정기후원 (정기결제)

| 항목 | 내용 |
|------|------|
| Mall ID | `05543499` |
| sp_pay_type | `81` |
| sp_window_type | `submit` |
| 콜백 | `POST /pg/easypay/profile/return` |
| DB | `WEO_ORDER_PROFILE` (billing key `cno`, `OP_DAY`, `OP_AMOUNT`) |
| 배치 | `internal/job/billing_batch.go` (V1 `_profile_batch.php` 이식) |

**주요 고려사항:**
- 빌링키(`cno`)를 **암호화하여 저장** (V1은 평문 — 복제 금지)
- 월 최대 100,000원 제한
- V1+V2 동시 배치 실행 시 이중 청구 방지를 위한 **명확한 전환 계획** 필요

### 16.8 Phase 3: 추가 결제수단 + 프로젝트 기부

- 계좌이체 (`sp_pay_type=21`)
- 휴대폰 (`sp_pay_type=31`)
- 프로젝트별 기부 (`O_GATE='F'`, `P_SEQ` 연결)
- DonationForm에 결제수단 선택 활성화 + 프로젝트 선택 드롭다운 추가

### 16.9 PG 감사 로거 (PGAuditLogger)

PG 승인 후 DB 실패 시 금전 대사(reconciliation)를 위해 **전용 감사 로그**를 기록합니다.

#### 파일: `internal/service/pg_audit_logger.go`

```go
type PGAuditLogger struct {
    mu   sync.Mutex
    file *os.File
}

type PGAuditEntry struct {
    Timestamp string      `json:"ts"`
    OrderNo   string      `json:"order_no"`
    Event     string      `json:"event"`      // "approve_success", "approve_fail", "db_insert_fail", "db_update_fail"
    RawData   interface{} `json:"data"`
    Error     string      `json:"error,omitempty"`
}
```

- **포맷:** JSON-per-line, 매 write 후 `fsync` (durability 보장)
- **기록 시점:** approve 성공 (항상), approve 실패, DB INSERT 실패, DB UPDATE 실패
- **파일 경로:** `PG_AUDIT_LOG_PATH` 환경변수 (기본값: `/var/logs/pg/pg-audit.log`)
- **logrotate:** `deploy/pg-audit.logrotate` — 별도 디렉토리, 30일 보관
- **와이어링:** `main.go`에서 `NewPGAuditLogger(cfg.PGAuditLogPath)` → `NewDonateService(..., pgAudit)` 전달

### 16.10 리스크 및 대응

| 리스크 | 심각도 | 대응 |
|--------|--------|------|
| `ep_cli` macOS 실행 불가 | 🔴 높 | Go 테스트에서 mock 응답, Linux 스테이징에서 통합 테스트 |
| EasyPay SDK + React vDOM 충돌 | 🟡 중 | 비제어 `ref`, 동적 SDK 로딩, `display: none` 격리 |
| `ep_cli` 출력 EUC-KR 인코딩 | 🟡 중 | `golang.org/x/text/encoding/korean`으로 변환 (필요 시) |
| PG 승인 후 DB 실패 | 🟡 중 | `PGAuditLogger`로 전체 PG 응답 JSON 기록 + 관리자 수동 대사 |
| V1+V2 WEO_ORDER 동시 쓰기 | 🟢 낮 | AUTO_INCREMENT로 충돌 없음. Phase 2 배치는 전환 계획 필요 |
| 1-core 1GB 서버 리소스 | 🟢 낮 | 결제 빈도 낮음, `ep_cli` 실행 시간 짧음, 메모리 모니터링 |

### 16.11 Phase 1 검증 계획

1. **단위 테스트 (Go):** ep_cli mock, CreateOrder 검증, 응답 파싱 테스트
2. **EasyPay 샌드박스:** SDK `testsp.easypay.co.kr`, 게이트웨이 `testgw.easypay.co.kr:80`
3. **통합 테스트 (Linux 스테이징):**
   - 로그인 → `/donation` → 10,000원 → 신용카드 → 기부하기
   - EasyPay 열림 → 테스트 결제 완료 → 성공 화면
   - DB 확인: `WEO_ORDER.O_PAYMENT='Y'`, `WEO_PG_DATA` 생성
   - 캐시 만료 후 기부 현황 업데이트 확인
4. **엣지 케이스:**
   - 결제 중 브라우저 종료 → 주문 pending 유지
   - EasyPay 에러 → 실패 화면, 주문 `O_PAYMENT='N'`
   - 이중 콜백 → 멱등 처리 (중복 결제 없음)
   - 미로그인 → 로그인 페이지로 리다이렉트
   - 금액 변조 → 서버에서 불일치 거부