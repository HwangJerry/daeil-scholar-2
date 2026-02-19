# 대일외국어고등학교 장학회 차세대 웹 애플리케이션 설계서 v1.5

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
| v1.7 | 2026-02-20 | Cycle 3 아키텍처 리뷰: 문서 분리, §15.2/§6.7 admin status 'AAA'→'ZZZ' 수정, §4.1 레거시 인증 상태 필터 경고 추가, §16.3 결제 트랜잭션 경고 추가 |

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

## 상세 설계서 문서 구조

각 기능별 상세 설계는 `docs/` 디렉토리의 개별 문서로 분리되어 있습니다.

| 문서 | 포함 섹션 | 설명 |
|------|-----------|------|
| [docs/spec-infrastructure.md](docs/spec-infrastructure.md) | §2, §3, §12 | 인프라, 아키텍처, DB 전환, 메모리 예산, 배포, HTTPS |
| [docs/spec-auth.md](docs/spec-auth.md) | §4 | 인증 설계 (JWT + 레거시 브릿지, 카카오 OAuth, CSRF) |
| [docs/spec-database.md](docs/spec-database.md) | §5 | DB 스키마 변경, 신규 테이블, 콘텐츠 파이프라인 |
| [docs/spec-api.md](docs/spec-api.md) | §6, §7 | API 엔드포인트, 파일/이미지 마이그레이션 |
| [docs/spec-frontend.md](docs/spec-frontend.md) | §8, §9 | 화면 설계, React 컴포넌트 구조, Admin SPA |
| [docs/spec-backend.md](docs/spec-backend.md) | §10, §11 | Go 백엔드 구조, 에러 핸들링, 모니터링 |
| [docs/spec-admin.md](docs/spec-admin.md) | §15 | 관리자 웹 애플리케이션 설계 |
| [docs/spec-payment.md](docs/spec-payment.md) | §16 | 기부 결제 시스템 (EasyPay PG 연동) |

### 관련 문서

| 문서 | 용도 |
|------|------|
| [IMPLEMENTATION_CHECKLIST.md](IMPLEMENTATION_CHECKLIST.md) | 보안/검증 구현 체크리스트 (Cycle 3 리뷰 결과) |
| [STRATEGY_REPORT.md](STRATEGY_REPORT.md) | Cycle 3 아키텍처 리뷰 전체 보고서 |
| [UI_DESIGN_DOC.md](UI_DESIGN_DOC.md) | UI 디자인 시스템 상세 (Tailwind v4 테마, 컴포넌트 가이드라인) |

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
