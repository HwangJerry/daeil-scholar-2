# 레거시 PHP 제거 작업 결과 보고서

- 작업일: 2026-09-26
- 대상: 운영 서버 `daeil-prod` (Apache, PHP, root crontab), `dflh-saf-v2` 저장소
- 계획서: `docs/LEGACY_PHP_REMOVAL_PLAN.md` (실행 순서 0-2절, 단계별 명령·롤백 4절·10절)
- 검토 요청: 7절 "미확인·검토 필요 사항"을 우선 확인해 주세요.

## 1. 요약

- 운영 서버에서 PHP가 완전히 빠졌다. mod_php를 내리고 php-fpm을 중지·비활성화했다. 응답 헤더에서 `PHP/7.2.33` 표기도 사라졌다.
- 메인 사이트(`/`)·관리자(`/admin/`)·Go API는 모든 단계 후 정상(200)이었다. 백엔드 오류 로그는 없었다.
- 조사 중 발견한 **adms 무인증 쓰기 취약점**은 당일 차단했고, 이후 adms 사이트 자체를 내렸다.
- adms 기능 중 새 관리자에 없던 것(회원 정보 수정, 회원 검색 필터, 운영자 지정)을 구현해 배포했다.
- 되돌릴 수 없는 단계(레거시 파일 삭제, PHP 패키지 제거)는 하지 않았다. 레거시 파일은 서버에 그대로 있고, 웹으로는 접근할 수 없다.

## 2. 배경 조사 결과

| 항목 | 결과 |
|---|---|
| 앱·웹의 PHP 사용 | 웹·관리자·iOS·Android 모두 PHP를 호출하지 않음 |
| PHP 진입점 | `/old/`(v1 사이트)와 `adms.daeilfoundation.or.kr`(레거시 관리자) 두 곳. `/old/` 접속은 전환 후 스캐너뿐 |
| 설정상 결합 | `httpd.conf`가 모든 `.html`을 PHP로 처리해, React 앱의 `index.html`도 PHP 엔진을 통과하고 있었음 (기능 영향은 없고 PHP 제거 시 사이트가 다운되는 결합) |
| PHP 정기결제 cron | 매일 09:00 실행. 2024-12-08 이후 결제 성공 0건, PG 요청 기록도 없음. 미결제 주문만 매달 쌓임 |
| 레거시 정기후원자 | 활성 7명, 월 230,000원. Go 정기후원과 겹치는 회원 없음 |
| adms 보안 | `_module/*` 쓰기 엔드포인트에 로그인 검사가 없고 SQL을 문자열로 조합함 (운영 파일 해시가 로컬 사본과 동일함을 확인. 실제 공격은 시험하지 않음) |
| adms 사용량 | 2026년 로그인 17회, 운영자 계정 2개 |
| Go의 PHP 디렉터리 의존 | EasyPay 결제 모듈(`ep_cli`, 인증서)이 `/var/www/html/_sys/payment`에 있음. PHP를 멈추는 건 무관하지만 디렉터리를 지우면 결제가 멈춤 |

## 3. 새 관리자 기능 (adms 대체)

| 기능 | 내용 | 커밋 |
|---|---|---|
| 회원 정보 수정 | 이름·휴대폰·연락용 이메일·기수·학과. 휴대폰은 중복 방지 기록(`AUTH_PHONE_CLAIM`)과 함께 트랜잭션으로 변경. 승인 회원은 동문 인증 기록의 기수·학과도 함께 반영 | `41491d8` |
| 회원 검색 필터 | 기수·학과·가입일 기간, 목록에 학과 열 | `41491d8` |
| 상태 드롭다운 수정 | 서버가 허용하는 탈퇴·휴면·정지만 표시 (기존에는 7개를 보여 주고 4개는 409 오류) | `41491d8` |
| 운영자 관리 (root 전용) | 관리자 지정·권한 변경(root ↔ 일반)·해제. 본인 변경 금지, root 최소 1명 유지(행 잠금), 탈퇴·휴면·정지 회원 지정 불가 | `ca08912` |

adms 메뉴별 대조표는 계획서 9절·9-1절에 있다.

## 4. 운영 작업 기록

| 단계 | 시각(KST) | 내용 | 결과 |
|---|---|---|---|
| A | 18:57 | adms에 Basic 인증, `/old/` 차단 (`/etc/httpd/conf.d/zz-legacy-guard.conf`) | adms 무인증 401, `/old/` 403. 처음 적용 때 비밀번호 파일 권한 문제로 인증 후 500이 났으나(차단은 유지됨) httpd 실행 계정 `nobody`로 그룹을 고쳐 해결 |
| B | 19:00 | 새 관리자 기능 배포 `b01d46e` (release `20260926T100033Z-b01d46ea70ff`) | 정상. 새 API 무인증 401, 번들에 새 화면 포함 |
| D-0 | 19:0x | 설정 백업 `/root/php-removal/phase0` | 완료 |
| D-1 | 19:0x | PHP 정기결제 cron 주석 처리 | 다른 cron 줄 변경 없음 |
| D-2 | 19:0x | `.html` PHP 처리를 레거시 디렉터리로 한정 | 메인·관리자 `X-Powered-By` 사라짐, 응답이 디스크 `index.html`과 해시 동일 |
| E | 19:10 | 저장소에서 `/old/` 설정·PHP 보조 파일 제거 후 배포 `8978a90` (release `20260926T101058Z-8978a901ce37`) | `/old/` 410, 메인·관리자·API 200 |
| F | 19:1x | adms vhost 비활성화 (설정 파일을 `/root/php-removal/phase4`로 이동) | adms 도메인은 메인 사이트가 응답(React 앱, PHP 흔적 없음). 인증서에 adms 도메인이 포함돼 경고 없음 |
| G | 19:1x | mod_php 언로드, php-fpm 중지·비활성화 (백업 `/root/php-removal/phase5`) | PHP 모듈 없음, `Server` 헤더에서 PHP 표기 사라짐, 테스트한 어떤 경로에서도 PHP 소스 노출 없음, 업로드 파일 정상 |

B·E 배포는 배포 도구 규칙대로 탈퇴 처리 기능이 멈춘 상태로 끝나므로, 각각 `--activate-all-users`로 배포 전과 같은 상태(탈퇴 접수·워커 ON, 보관 만료 OFF)를 복원했다.

## 5. 현재 운영 구성

- 웹 서버: Apache에 vhost 하나(`alumni.conf`). PHP 모듈 없음, php-fpm 비활성.
- `/old/` → 410. `/index.php` → `/`로 301.
- `adms.daeilfoundation.or.kr` → 메인 사이트가 응답.
- 남아 있는 임시 설정: `zz-legacy-guard.conf` (adms Basic 인증과 `/old/` 차단. 지금은 해당 경로가 서비스되지 않아 실효는 없고, 실수로 레거시 설정이 되살아날 때를 대비한 안전장치로 남겨 둠)
- 레거시 파일: `/var/www/html`, `/var/www/dadms`는 서버에 그대로 있음. 웹으로는 접근 불가.
- Go 결제 모듈: 여전히 `/var/www/html/_sys/payment`를 사용.

## 6. 되돌리는 방법

| 단계 | 방법 |
|---|---|
| G | `/root/php-removal/phase5`의 `15-php.conf`, `php.conf.active`(→`php.conf`), `zz-legacy-php-html.conf.active`(→`.conf`), `httpd.conf` 복원 → `apachectl configtest` → `systemctl restart httpd` → `systemctl enable --now php-fpm` |
| F | `/root/php-removal/phase4/virthost*.conf`를 `/etc/httpd/conf.d/`로 되돌리고 reload |
| E | 이전 release(`20260926T100033Z-b01d46ea70ff`) 묶음 재배포 후 `--activate-all-users` |
| D-2 | `/root/php-removal/httpd.conf.phase2` 복원, `zz-legacy-php-html.conf` 삭제, reload |
| D-1 | `crontab -u root /root/php-removal/phase0/root.crontab` |
| A | `zz-legacy-guard.conf` 삭제 후 reload |
| B | 이전 release(`20260925T021708Z-93f1a8c475d0`) 묶음 재배포 후 `--activate-all-users` (스키마 변경 없음) |

G·F를 되돌리면 adms가 다시 열린다. adms의 무인증 취약점이 그대로이므로, 되돌릴 때는 A의 차단 설정을 유지해야 한다.

## 7. 미확인·검토 필요 사항

1. **새 관리자 화면을 사람이 직접 써 보지 않았다.** 확인한 것은 자동 테스트(백엔드 전체, 관리자 147개), 배포 후 API·번들 점검까지다. 운영자 확인 단계(C)는 운영자 본인 결정으로 생략했다. root 계정으로 아래를 확인해 주세요.
   - 설정 → 운영자 관리: 목록이 보이고 본인 행이 잠겨 있는지
   - 회원 관리: 기수·학과·가입일 필터
   - 회원 상세 → 정보 수정: 창이 열리는지(저장은 시험 계정으로만)
   - 회원 상세: 상태 드롭다운에 탈퇴·휴면·정지만 있는지
2. **adms 로그인 기록에 운영자 계정이 2개 있다.** 2026년에 서로 다른 계정 2개가 로그인했고 9월에도 두 계정 모두 기록이 있다. 두 계정이 모두 본인이 아니라면 다른 사용자에게 adms 종료와 새 관리자 사용법을 알려야 한다.
3. **PHP cron 중지 사후 확인**: 다음 레거시 결제일(10-02) 다음 날인 10-03에 새 배치 주문이 0건인지 확인한다(계획서 4절 Phase 1 쿼리).
4. **레거시 정기후원자 7명 조치**(사용자 담당): 명단 `prod-db-backups/legacy_recurring_donors_20260926.tsv`. 조치 후 파일을 삭제한다.
5. **adms Basic 인증 비밀번호**: adms를 내렸으므로 운영자에게 전달할 필요는 없어졌다. 서버의 `/root/php-removal/stepA/adms-ops.password`와 `/etc/httpd/adms.htpasswd`는 Phase 8에서 `zz-legacy-guard.conf`와 함께 지운다.
6. **Phase 8 전 준비**: `httpd.conf`의 서버 기본 `DocumentRoot`가 아직 `/var/www/html`이다. 모든 요청은 `alumni.conf` vhost가 받으므로 현재 영향은 없지만, 디렉터리를 지우기 전에 비어 있는 디렉터리나 `/var/www/app`으로 바꾼다.
7. **기존 문제 (이번 작업과 무관)**: 공개 게시글의 레거시 첨부파일 96개가 이전부터 404다(`/upload/bbsFiles`, `/upload/adFiles`, 해당 경로에 파일 없음).

## 8. 남은 단계

| 단계 | 내용 | 시점 |
|---|---|---|
| H | 관찰: 오류 로그, 메인·관리자·API, 운영자 문의 | 2026-10-10까지 (최소 2주) |
| I | EasyPay 모듈을 `/opt/easypay`로 이전, `EASYPAY_BIN_BASE` 변경, 결제 확인 | H 기간 중 |
| J | `/var/www/html`·`/var/www/dadms` 보관(서버 밖 사본 포함) 후 삭제, PHP 패키지 제거, `zz-legacy-guard.conf`·htpasswd·비밀번호 파일 삭제, adms DNS·인증서 정리, `conf.d`의 `.bak`/`.rpm*` 정리 | H·I 완료 후, 7절 4·6번 처리 후 |
