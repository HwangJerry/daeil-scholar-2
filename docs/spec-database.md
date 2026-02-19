# DB 스키마 및 콘텐츠 파이프라인 (§5)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

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
