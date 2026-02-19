# API 설계 및 파일 마이그레이션 (§6, §7)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

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

> **인가:** 모든 `/api/admin/*` 엔드포인트는 `AdminAuthMiddleware`로 보호됩니다. `USR_STATUS = 'ZZZ'`(운영자) 확인. (`'AAA'`=탈퇴, `'BBB'`=승인대기, `'CCC'`=승인회원, `'ZZZ'`=운영자)

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
