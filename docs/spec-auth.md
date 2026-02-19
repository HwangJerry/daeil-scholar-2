# 인증 설계 (§4)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

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

> ⚠️ **NOTE (Cycle 3):** 위 코드의 `user.USRStatus == "BBB"` 조건에 보안 이슈가 있습니다. 상세 내용과 수정 방법은 `IMPLEMENTATION_CHECKLIST.md` IC-1 항목을 참조하십시오.

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
