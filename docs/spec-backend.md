# Go 백엔드 구조 및 에러 핸들링 (§10, §11)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

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
