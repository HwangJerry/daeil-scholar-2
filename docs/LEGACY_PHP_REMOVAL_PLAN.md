# 레거시 PHP 모듈 제거 작업 계획

- 작성일: 2026-09-26
- 범위: 운영 서버 `daeil-prod`의 Apache/PHP 설정, `dflh-saf-v2/deploy`, root crontab, EasyPay 모듈 경로
- 목표: 운영 서버에서 PHP(mod_php, php-fpm)를 제거하고, 메인 사이트(`/`)·관리자(`/admin/`)·Go API가 PHP와 무관하게 동작하게 한다.
- 상태: **계획. 아직 아무것도 실행하지 않았다.** 이 문서를 작성하며 운영 서버에서 한 작업은 읽기 전용 조회뿐이다.

## 0. 요약

- 실제로 PHP를 쓰는 사용자·기능은 없다. 남은 것은 **설정상의 결합**과 **거의 쓰지 않는 레거시 관리자(adms)**다.
- 그냥 PHP 모듈을 내리면 두 가지 사고가 난다.
  1. 메인·관리자 `index.html`이 `application/x-httpd-php` 타입으로 내려가 **사이트가 열리지 않는다** (`httpd.conf:283`).
  2. `/old/`·adms가 살아 있으면 `.php` **소스(DB 접속 정보 포함)가 노출**된다 (`php.conf`의 `AddType text/html .php`).
- 배포 파이프라인이 매 배포마다 `alumni.conf`와 PHP 보조 파일을 덮어쓰므로 **저장소 변경이 서버 변경보다 먼저**다.
- 모든 단계는 Phase 8(디렉터리 삭제) 전까지 설정 파일 복원 + reload로 즉시 되돌릴 수 있다.

## 0-1. 긴급: adms 무인증 쓰기 취약점 (2026-09-26 확인)

- `/var/www/dadms/_module/*`의 쓰기 엔드포인트(`mMember.df`, `mBoardProc.df`, `mBanner.df`, `mDonationCheck.df`, `mMemberFnCheck.df` 등)는 `SessionCheck`를 호출하지 않는다. 로그인 검사는 `index.df`에만 있다. 운영 파일과 로컬 v1 사본의 해시가 같음을 확인했다.
- 모든 SQL이 문자열 연결로 만들어져 SQL 인젝션도 가능하다.
- 즉 `adms.daeilfoundation.or.kr`에서 **로그인 없이 회원 정보·상태·비밀번호, 게시글, 배너, 기부 내역을 바꿀 수 있는 상태**다. 실제 악용 여부나 공격 가능성은 운영에서 시험하지 않았다.
- 대응: Phase 4(adms 비활성화)를 다른 단계와 무관하게 **먼저** 하거나, 최소한 adms vhost에 IP 제한(`Require ip <운영자 IP>`)을 즉시 건다. 대체 기능 검토(9절) 결과, 운영자가 Go 관리자로 옮기지 못하는 필수 업무는 회원 정보 수정뿐이다.

## 1. 현재 상태 (2026-09-26 조사)

| 항목 | 상태 | 근거 |
|---|---|---|
| PHP 실행 방식 | mod_php (`php7_module`, PHP 7.2.33) | `httpd -M`, `conf.modules.d/15-php.conf` |
| `.html`의 PHP 처리 | **전역**. 메인·관리자 `index.html`도 PHP 엔진 통과 | `httpd.conf:283` `AddType application/x-httpd-php .php .html .htm`, 응답 헤더 `X-Powered-By: PHP/7.2.33` |
| JS/CSS 번들, Go API | PHP 미경유 | 응답 헤더에 `X-Powered-By` 없음 |
| php-fpm | 실행 중이지만 사용하는 vhost 없음, 연결 0개 | `127.0.0.1:9000`, 설정에 `proxy:fcgi` 없음 |
| PHP 진입점 1: `/old/` | `alumni.conf`의 `Alias /old/ /var/www/html/`. 8~9월 접속은 전부 스캐너 | 메인 vhost 접속 로그 |
| PHP 진입점 2: adms | `adms.daeilfoundation.or.kr` → `/var/www/dadms`. 2026년 로그인 17회(월 1~2회), 운영자 2명 | `WEO_OPERATOR_LOG` |
| adms 사용 메뉴 (2026) | 회원관리(마지막 07-02), 장학사업 현황·회원승인관리(02-23), 행사 게시판, 배너관리(09-14), 통계 | `WEO_OPERATOR_PAGE_LOG` |
| adms의 `.html` PHP 의존 | `donation_excel/excel.html`에 PHP 코드 포함 (로컬 v1 사본 기준, 운영본 확인 필요) | `dflh-saf-v1/dadms` |
| PHP cron | root `0 9 * * *` `_profile_batch.php`. 2024-12-08 이후 결제 성공 0건, 미결제 주문만 생성 | root crontab, `WEO_ORDER` |
| 레거시 정기후원 | 활성 7건, 월 230,000원. Go `SUBSCRIPTION`과 겹치는 회원 0명 | `WEO_ORDER_PROFILE` |
| Go의 PHP 디렉터리 의존 | `EASYPAY_BIN_BASE=/var/www/html/_sys/payment` (`ep_cli`, 인증서) | `/etc/sysconfig/alumni-backend` |
| 배포 파이프라인 | 매 배포마다 `deploy/httpd-alumni.conf` → `/etc/httpd/conf.d/alumni.conf`, PHP 보조 파일 3개 → `/var/www/html`, 이어서 `httpd -t` | `deploy/remote_release.py:363-366` |
| 레거시 첨부 | 공개 게시글의 레거시 첨부 96개가 **이미 404** (PHP와 무관) | `WEO_FILES` `/upload/bbsFiles`, `/upload/adFiles` |

## 2. 범위

**포함**: cron 중지, `.html` PHP 처리 분리, `/old/`·adms 비활성화, mod_php 언로드, php-fpm 중지, EasyPay 모듈 이전, 레거시 디렉터리 보관 후 삭제, PHP 패키지 제거.

**제외**: 레거시 정기후원자 연락(재단 결정), 레거시 첨부 404 복구, DB 레거시 테이블 정리와 MariaDB 업그레이드(별도 DB 마이그레이션 계획).

## 3. 사전 조건 (Phase 4 이전에 모두 충족)

| # | 조건 | 담당 | 막는 단계 |
|---|---|---|---|
| G1 | adms 운영자 2명 확인. 대조 결과(9절): 회원 정보 수정·회원 검색 필터 외에는 대체됐거나 불필요 | 운영/재단 | Phase 4 (긴급 시 IP 제한으로 선조치) |
| G2 | 레거시 정기후원자 7명 조치 방향 결정 (Phase 1은 이 결정과 무관하게 진행 가능) | 재단 | Phase 8 (`WEO_ORDER_PROFILE` 원본 보관 방식) |
| G3 | 운영 서버 백업 공간 확인 (`/var/www/html`, `/var/www/dadms` 보관용) | 개발 | Phase 8 |
| G4 | 작업 창: 접속이 적은 시간대. Phase 3은 정식 배포라 httpd가 잠시 멈춤 | 개발 | Phase 3 |

## 4. 작업 단계

공통 규칙:
- 모든 서버 명령은 `sudo`로 실행하고, 변경 전 파일은 `/root/php-removal/` 아래에 복사해 둔다.
- 각 단계는 **검증(6절)을 통과해야** 다음 단계로 간다. 실패하면 그 단계의 롤백을 실행하고 멈춘다.
- 설정을 바꾼 뒤에는 반드시 `apachectl configtest`가 통과한 경우에만 `systemctl reload httpd`를 한다.

### Phase 0. 백업과 기준 측정 (위험 없음)

```bash
sudo mkdir -p /root/php-removal/phase0 && cd /root/php-removal/phase0
sudo cp -a /etc/httpd ./etc-httpd
sudo cp -a /etc/php-fpm.d /etc/php-fpm.conf ./ 2>/dev/null
sudo crontab -l -u root > ./root.crontab
sudo cp -a /etc/sysconfig/alumni-backend ./alumni-backend.env
sudo httpd -M > ./httpd-modules.txt; sudo httpd -S > ./httpd-vhosts.txt 2>&1
systemctl is-enabled php-fpm httpd > ./service-state.txt
# 운영 alumni.conf가 저장소 사본과 같은지 확인 (다르면 원인부터 파악)
diff /etc/httpd/conf.d/alumni.conf <(저장소의 deploy/httpd-alumni.conf 사본)
# 운영본에서 .html 안에 PHP 코드를 쓰는 파일 목록 (Phase 2 범위 결정용)
sudo grep -rl '<?' --include='*.htm' --include='*.html' /var/www/dadms /var/www/html > ./html-with-php.txt
# DirectoryIndex 정의 위치 (Phase 5에서 php.conf 제거 후에도 index.html이 남는지)
sudo grep -rn 'DirectoryIndex' /etc/httpd/conf /etc/httpd/conf.d > ./directoryindex.txt
```

6절 검증 스크립트를 한 번 돌려 **변경 전 결과**를 저장한다 (`X-Powered-By`가 보이는 것이 정상).

롤백: 해당 없음.

### Phase 1. PHP 정기결제 cron 중지 (위험 낮음)

```bash
sudo crontab -l -u root | sed 's|^\(0 9 \* \* \* /usr/bin/php /var/www/html/_sys/payment/profile/mobile/_profile_batch.php\)$|# disabled 2026-MM-DD php-removal: \1|' | sudo crontab -u root -
sudo crontab -l -u root | grep _profile_batch   # '#'로 시작해야 함
```

- 효과: 매일 쌓이던 미결제 주문(`O_GATE='P'`, `O_PAYMENT='N'`)이 멈춘다. 실제로 끊기는 결제는 없다 (2024-12 이후 성공 0건).
- 검증: 다음 결제일(2·4·6·8·9·16·18일) 다음 날, 새 배치 주문이 생기지 않았는지 확인.
  ```sql
  SELECT COUNT(*) FROM WEO_ORDER
  WHERE O_GATE IN ('P','F') AND HOUR(O_REGDATE)=9 AND COALESCE(REG_IPADDR,'')=''
    AND O_REGDATE >= '<Phase 1 실행일>';
  ```
- 롤백: `sudo crontab -u root /root/php-removal/phase0/root.crontab`

### Phase 2. `.html`의 PHP 처리를 레거시 디렉터리로 한정 (위험 낮음)

메인·관리자를 PHP에서 분리하되, `.html`에 PHP를 넣어 쓰는 레거시 파일(adms 기부 엑셀 등)은 제거 전까지 계속 동작하게 한다.

1. `httpd.conf:283`에서 `.html .htm`을 뺀다.
   ```bash
   sudo cp -a /etc/httpd/conf/httpd.conf /root/php-removal/httpd.conf.phase2
   sudo sed -i 's|^\(\s*AddType application/x-httpd-php\) \.php \.html \.htm\s*$|\1 .php|' /etc/httpd/conf/httpd.conf
   grep -n 'AddType application/x-httpd-php' /etc/httpd/conf/httpd.conf   # ".php"만 남아야 함
   ```
2. 레거시 디렉터리에만 `.html` PHP 처리를 다시 건다. 새 파일 `/etc/httpd/conf.d/zz-legacy-php-html.conf`:
   ```apache
   # Temporary: legacy PHP-in-.html files keep working until Phase 4/5 removes these sites.
   <Directory /var/www/dadms>
       AddType application/x-httpd-php .html .htm
   </Directory>
   <Directory /var/www/html>
       AddType application/x-httpd-php .html .htm
   </Directory>
   ```
3. `sudo apachectl configtest && sudo systemctl reload httpd`

- 검증: 6절 A·B 전부. 특히 `/`, `/admin/`, `/notice` 응답에 `X-Powered-By`가 **없어야** 하고 `Content-Type: text/html`이어야 한다. adms 로그인 화면과 `/old/`는 여전히 열려야 한다.
- 롤백:
  ```bash
  sudo cp -a /root/php-removal/httpd.conf.phase2 /etc/httpd/conf/httpd.conf
  sudo rm -f /etc/httpd/conf.d/zz-legacy-php-html.conf
  sudo apachectl configtest && sudo systemctl reload httpd
  ```

### Phase 3. 저장소 변경 후 정식 배포 (`/old/` 제거, PHP 보조 파일 제거)

저장소 변경 (`dflh-saf-v2`):

| 파일 | 변경 |
|---|---|
| `deploy/httpd-alumni.conf` | 63~98번 줄 `/old/` 블록 전체 삭제(`php_value` 포함). 대신 `RedirectMatch 410 ^/old/` 한 줄 추가. 140번 줄 `RewriteCond %{REQUEST_URI} !^/old/`는 410 규칙이 먼저 처리하므로 그대로 둬도 되지만 정리 권장 |
| `deploy/remote_release.py` | `SHIMS` 상수(30번 줄), 스냅샷 대상(334번 줄), 설치 루프(364~365번 줄) 제거 |
| `deploy/release_bundle.py` | 39번 줄 필수 파일 목록에서 PHP 보조 파일 3개 제거 |
| `deploy/_set_docroot.php`, `_legacy_docroot.php`, `_legacy_url_rewriter.php` | 삭제 |
| `deploy/tests/test_release.py` | 26번 줄 `remote.SHIMS` 참조 제거 |
| `CLAUDE.md` "Dual Auth System" 절 | PHP 공존 설명을 현재 상태로 갱신 (선택) |

검증(로컬): `python3 -m unittest discover deploy/tests` 통과, `httpd -t`는 운영 배포 과정에서 자동 실행된다.

배포: 평소처럼 `./deploy.sh`. 배포 스크립트가 기존 `alumni.conf`와 보조 파일을 `/app/releases/<release>/backup`에 스냅샷한 뒤 교체하고, `httpd -t` 실패나 헬스체크 실패 시 자동으로 원복한다.

- 효과: `/old/`는 410을 돌려준다. `/var/www/html`의 PHP 파일은 더 이상 웹으로 접근할 수 없다 (EasyPay 모듈과 cron 파일은 파일로만 남음).
- 검증: 6절 A·B, 그리고 `/old/index.php`, `/old/_sys/sys_config.php`가 **410**인지.
- 롤백: 이전 커밋으로 다시 배포하거나, 스냅샷에서 `alumni.conf`와 보조 파일을 복원 후 reload.
  ```bash
  B=/app/releases/<이번 release>/backup   # application-restore.json에 원래 경로가 기록됨
  sudo cp -a "$B"/<alumni.conf 스냅샷> /etc/httpd/conf.d/alumni.conf
  sudo apachectl configtest && sudo systemctl reload httpd
  ```

### Phase 4. adms 비활성화 (G1 충족 후)

```bash
sudo mkdir -p /root/php-removal/phase4
sudo mv /etc/httpd/conf.d/virthost.conf /etc/httpd/conf.d/virthost-le-ssl.conf /root/php-removal/phase4/
sudo apachectl configtest && sudo systemctl reload httpd
```

- 참고: `conf.d`에는 `*.conf`만 로드되므로 옮겨 두면 비활성화된다. `.bak`, `.rpmsave`, `.rpmnew` 파일은 원래 로드되지 않는다.
- 효과: `adms.daeilfoundation.or.kr` 요청이 기본 vhost(메인 사이트)로 넘어가 SPA가 열린다. PHP 소스는 노출되지 않는다 (메인 vhost에는 `/var/www/dadms` 경로가 없음).
- 검증: `curl -skI https://adms.daeilfoundation.or.kr/`가 adms 로그인 화면이 아닌지. 6절 A 전부.
- 롤백:
  ```bash
  sudo mv /root/php-removal/phase4/virthost*.conf /etc/httpd/conf.d/
  sudo apachectl configtest && sudo systemctl reload httpd
  ```

### Phase 5. mod_php 언로드, php-fpm 중지

전제: Phase 3·4 완료 (PHP 진입점 없음), Phase 0의 `directoryindex.txt`에서 `php.conf` 밖에 `DirectoryIndex index.html` 정의가 있음을 확인. 없으면 `alumni.conf`나 `httpd.conf`에 `DirectoryIndex index.html`을 먼저 추가한다.

```bash
sudo mkdir -p /root/php-removal/phase5
sudo cp -a /etc/httpd/conf.modules.d/15-php.conf /etc/httpd/conf.d/php.conf /etc/httpd/conf.d/zz-legacy-php-html.conf /root/php-removal/phase5/
sudo sed -i 's|^\(\s*LoadModule php7_module\)|#\1|' /etc/httpd/conf.modules.d/15-php.conf
sudo mv /etc/httpd/conf.d/php.conf /root/php-removal/phase5/php.conf.active
sudo rm -f /etc/httpd/conf.d/zz-legacy-php-html.conf
# httpd.conf의 AddType application/x-httpd-php .php 줄도 제거 (모듈이 없으면 .php가 이 타입으로 내려감)
sudo sed -i 's|^\(\s*AddType application/x-httpd-php .php\)\s*$|#\1|' /etc/httpd/conf/httpd.conf
sudo apachectl configtest && sudo systemctl restart httpd   # 모듈 변경은 restart
sudo systemctl stop php-fpm && sudo systemctl disable php-fpm
```

- 검증: 6절 A·B·C 전부. `httpd -M | grep php`가 비어야 하고, `Server` 헤더에서 `PHP/7.2.33`이 사라져야 한다.
- 롤백:
  ```bash
  sudo cp -a /root/php-removal/phase5/15-php.conf /etc/httpd/conf.modules.d/15-php.conf
  sudo cp -a /root/php-removal/phase5/php.conf.active /etc/httpd/conf.d/php.conf
  sudo cp -a /root/php-removal/phase5/zz-legacy-php-html.conf /etc/httpd/conf.d/
  sudo sed -i 's|^#\(\s*AddType application/x-httpd-php .php\)\s*$|\1|' /etc/httpd/conf/httpd.conf
  sudo apachectl configtest && sudo systemctl restart httpd
  sudo systemctl enable php-fpm && sudo systemctl start php-fpm
  ```

### Phase 6. 관찰 기간 (최소 2주)

- 매일: `/var/logs/alumni/httpd-error.log` 새 오류, 6절 A 검증, Go 헬스체크.
- 운영자 문의(adms 부재로 못 하는 업무) 수집. 생기면 Go 관리자에 기능을 추가하거나 Phase 4 롤백으로 임시 복구한다.
- 이 기간에는 Phase 8(삭제)을 하지 않는다.

### Phase 7. EasyPay 모듈을 레거시 디렉터리 밖으로 이전

```bash
sudo mkdir -p /opt/easypay && sudo cp -a /var/www/html/_sys/payment/. /opt/easypay/
sudo diff -r /var/www/html/_sys/payment /opt/easypay && echo SAME
sudo cp -a /etc/sysconfig/alumni-backend /root/php-removal/alumni-backend.env.phase7
sudo sed -i 's|^EASYPAY_BIN_BASE=.*|EASYPAY_BIN_BASE=/opt/easypay|' /etc/sysconfig/alumni-backend
sudo systemctl restart alumni-backend
```

- 주의: `ep_cli`가 쓰는 `log` 디렉터리 권한을 Go 서비스 실행 계정 기준으로 맞춘다 (`/opt/easypay/{immediately,profile}/log`).
- 배포 도구는 `/etc/sysconfig/alumni-backend`를 쓰지 않는다. `deploy/remote_release.py`는 `EASYPAY_BIN_BASE`가 비어 있지 않은지만 검사하고(`validate_activation`), 자기 설정은 `/app/backend/release-rollout.env`에 따로 둔다. 따라서 다음 배포가 값을 되돌리지 않는다.
- 검증: 소액 일시 기부 1건 결제 후 취소(또는 테스트 상점), `pg-audit.log`에 정상 응답 기록, `/api/health` 정상.
- 롤백: `sudo cp -a /root/php-removal/alumni-backend.env.phase7 /etc/sysconfig/alumni-backend && sudo systemctl restart alumni-backend` (원본 디렉터리는 Phase 8 전까지 그대로 있음).

### Phase 8. 레거시 보관 후 삭제 (되돌릴 수 없음)

전제: Phase 6 관찰 완료, Phase 7 결제 검증 완료, G2·G3 충족.

```bash
sudo mkdir -p /root/php-removal/archive
sudo tar -C /var/www -czf /root/php-removal/archive/legacy-www-$(date +%Y%m%d).tgz html dadms
sudo tar -tzf /root/php-removal/archive/legacy-www-*.tgz | wc -l      # 파일 수 확인
sha256sum /root/php-removal/archive/legacy-www-*.tgz > /root/php-removal/archive/SHA256SUMS
# 보관본을 서버 밖(재단 보관 위치)에도 1부 복사한 뒤
sudo rm -rf /var/www/html /var/www/dadms
sudo yum remove php php-common php-fpm php-cli   # 패키지 목록은 rpm -qa | grep -i php 로 확인 후
```

- `/var/www/html/upload`(레거시 파일 389개)는 탈퇴 파기 기록과 연결될 수 있으므로, 보관본의 보존 기간을 `docs/PRIVACY_RETENTION_REVIEW.md` 기준과 맞춘다.
- adms 도메인 DNS 레코드와 인증서(`certbot`) 정리, `conf.d`의 `.bak`/`.rpmsave`/`.rpmnew` 정리.
- 롤백: 보관본을 풀고(`tar -C /var/www -xzf ...`), Phase 5·4·3 롤백을 역순으로 실행. PHP 패키지를 지웠다면 `yum install`로 다시 설치.

## 5. 롤백 요약

| Phase | 되돌리는 방법 | 소요 | 되돌릴 수 없는 부분 |
|---|---|---|---|
| 1 cron | crontab 복원 | 1분 | 없음 |
| 2 AddType | `httpd.conf` 복원, 추가 파일 삭제, reload | 2분 | 없음 |
| 3 `/old/` | 이전 커밋 재배포 또는 스냅샷 복원 | 5~10분 | 없음 |
| 4 adms | vhost 파일 되돌리고 reload | 2분 | 없음 |
| 5 mod_php | 모듈·`php.conf` 복원, restart, php-fpm 재시작 | 5분 | 없음 |
| 7 EasyPay | env 복원, Go 재시작 | 2분 | 없음 |
| 8 삭제 | 보관본 복원 + 역순 롤백 | 30분 이상 | 보관본이 없으면 복구 불가 |

## 6. 검증 스크립트

`daeil-prod`에서 실행 (Apache에 직접 붙어 외부 경로 문제를 배제). 기대값과 다르면 즉시 해당 Phase 롤백.

```bash
H='--resolve daeilfoundation.or.kr:443:127.0.0.1'
chk() { printf '%-45s ' "$1"; curl -sk $H -o /dev/null -D - "https://daeilfoundation.or.kr$1" | tr -d '\r' | awk 'NR==1{s=$2} tolower($1)=="content-type:"{c=$2} tolower($1)=="x-powered-by:"{p=$2} END{print s, c, (p?p:"no-php")}'; }
# A. 서비스 (Phase 2 이후 항상 200, text/html 또는 json, no-php)
chk /; chk /admin/; chk /notice; chk /api/health; chk /api/donation/summary
# B. 정적 자원 (항상 200)
curl -sk $H https://daeilfoundation.or.kr/ | grep -oE '/assets/[^"]+\.js' | head -1 | while read -r js; do chk "$js"; done
# C. 레거시 경로 (Phase 3 이후 410, 소스 노출이면 즉시 롤백)
chk /old/index.php; chk /old/_sys/sys_config.php
curl -sk $H https://daeilfoundation.or.kr/old/_sys/sys_config.php | grep -c '<?' # 0이어야 함
```

| 검증 | Phase 0 기대값 | Phase 2~4 | Phase 5 이후 |
|---|---|---|---|
| A `/`, `/admin/`, `/notice` | 200 text/html PHP/7.2.33 | 200 text/html no-php | 200 text/html no-php |
| A `/api/*` | 200 json no-php | 같음 | 같음 |
| C `/old/*.php` | 200 (PHP 실행) | Phase 3 이후 410 | 410, `<?` 0건 |

## 7. 위험과 대응

| 위험 | 가능성 | 영향 | 대응 |
|---|---|---|---|
| mod_php만 내리고 AddType을 남김 → SPA가 다운로드됨 | 계획 미준수 시 | 사이트 전체 | Phase 2를 반드시 먼저. Phase 5 검증 A |
| `/old/`·adms를 남긴 채 mod_php 제거 → 소스·DB 정보 노출 | 계획 미준수 시 | 보안 사고 | Phase 3·4 완료가 Phase 5의 전제. 검증 C |
| 저장소 미변경 상태에서 서버만 변경 → 다음 배포 실패 | 중 | 배포 중단 | Phase 3을 Phase 5보다 먼저 |
| `php.conf` 제거로 `DirectoryIndex` 사라짐 → `/`, `/admin/`이 403 | 중 | 사이트 전체 | Phase 0에서 확인, 필요 시 `DirectoryIndex index.html` 추가 |
| adms 업무 공백 (기부 엑셀, 회원 승인 등) | 중 | 운영 | G1 확인, 관찰 기간, Phase 4 즉시 롤백 가능 |
| EasyPay 이전 후 결제 실패 | 낮음 | 결제 | Phase 7 검증, 원본 유지 중 env 원복 |
| 다음 배포가 `EASYPAY_BIN_BASE`를 되돌림 | 낮음 (배포 도구는 이 env 파일을 쓰지 않음) | 결제 | 배포 후 6절 A와 결제 확인 |

## 8. 미결 사항

- O1: adms 기능별 Go 관리자 대체 여부 (G1). 특히 기부 엑셀 내보내기, 장학사업 현황.
- O2: 레거시 정기후원자 7명 조치 (G2).
- O3: 운영본 `/var/www/dadms`, `/var/www/html`에서 `.html` 안에 PHP를 쓰는 파일 목록 (Phase 0에서 확정).
- O4: 레거시 첨부 96개 404 복구 여부 (범위 밖, 별도 과제).
- ~~O5: 배포 도구가 `/etc/sysconfig/alumni-backend`를 덮어쓰는지~~ → 쓰지 않음 (2026-09-26 코드 확인, Phase 7 참고).

## 9. adms 기능 대조 (2026-09-26)

근거: 레거시 소스 `dflh-saf-v1/dadms`, 운영 메뉴 설정·열람 기록(`WEO_JOBCODE`, `WEO_OPERATOR_PAGE_LOG`), 신규 관리자 `admin/src`, `backend/cmd/server/routes.go`.

| adms 메뉴 (2026 열람) | 레거시에서 실제로 되는 것 | 신규 관리자 | 판정 |
|---|---|---|---|
| 회원관리 (19) — 목록·검색 | 가입일/최근접속 기간, 학과, 기수, SMS·메일 수신, ID·이름·이메일·전화, 상태, 기부액·게시글 수 정렬 | 이름·전화 앞글자 + 상태만. 기수는 API만 지원, 화면 미연결 | **일부** |
| 회원관리 — 상세 | 프로필 + 작성 글·댓글 | 프로필(읽기 전용) | 일부 |
| 회원관리 — 정보 수정 | 전화, 학과, 기수, 생년월일, SMS 수신, 비밀번호 | 없음 | **없음** |
| 회원관리 — 상태 변경 | 7개 상태 모두 | 탈퇴·휴면·정지만. 승인은 가입 신청 화면. 화면 드롭다운은 7개를 보여 주지만 나머지는 409 (버그) | 일부 |
| 회원관리 — 회원 생성, SMS | 생성 가능. SMS는 "연동 준비중"으로 실제 발송 안 됨 | 없음 | 생성은 가입 절차로 대체, SMS 불필요 |
| 회원승인관리 (4) | 수정 창에서 상태를 CCC/BAA로 바꿈 | 가입 신청 큐: 승인, 사유 필수 반려, 회원 알림 | **대체됨** |
| 기수관리 (0, 마지막 2025-12) | `FUNDAMENTAL_NUMBERS` 추가·수정 | 없음. Go는 이 테이블을 쓰지 않음(학과는 `model/departments.go` 고정) | 불필요 |
| 졸업생 명단관리 (0) | 템플릿 파일이 없어 화면이 깨짐 | 없음 | 불필요 |
| 배너관리 (1) | `MAIN_AD` 등록·수정 | `MAIN_BANNER_AD` 관리. 사용자 화면은 `MAIN_BANNER_AD`만 노출, `MAIN_AD`는 대시보드 통계에만 쓰임 | 대체됨 (레거시 배너는 더 이상 노출 안 됨) |
| 장학사업 현황 (10) | `GATE='NOTICE'` 글 작성·수정, 작성자 지정 | 공지 관리 (`GATE='NOTICE'`) 작성·수정·삭제·고정 | **대체됨** |
| 행사 게시판 (2), 후원동문소개 (1) | `EVENT`, `RELAY` 글 관리 | 없음. 신규 사용자 화면도 이 게시판을 보여 주지 않음 | 불필요 |
| 문의 게시판 (0, 마지막 2025-11) | 답변 기능 깨짐(파일 누락) | 없음. 사용자 화면에도 없음 | 불필요 |
| 기부내역관리 | 목록(페이징 없음), 금액·구분·상태 수정 | 목록·필터·수동 등록·수정·설정·스냅샷 | **대체됨** |
| 동행 프로젝트 | 별도 기능 없음(기부내역 화면으로 연결) | 기부 구분 필터로 조회 | 대체됨 |
| 사용자 UV/PV (각 1) | 페이지가 404. 홈 대시보드에만 14일 UV/PV(`WEO_VISIT_TOT`, 2026-05 이후 갱신 중단) | 대시보드 DAU/MAU, 30일 차트. PV는 수집만 하고 미표시 | 대체됨 (PV 없음) |
| 기부 엑셀 (2026-01 4회) | 업로드 파일을 회원과 대조해 **화면에 출력만** 함. DB 저장 코드는 주석 처리 | 해피나눔 xlsx 미리보기 후 저장 | 대체됨 |

신규 관리자에 추가를 검토할 것: 회원 정보 수정, 회원 검색 필터(기수·학과·가입일), 상태 드롭다운 버그 수정. 나머지 adms 기능은 대체됐거나 현재 서비스에서 쓰이지 않는다.

→ 2026-09-26 `feature/admin-member-management` 브랜치에서 세 가지 모두 구현 (회원 상세의 "정보 수정", 목록의 기수·학과·가입일 필터와 학과 열, 상태 드롭다운은 탈퇴·휴면·정지만). 배포 후 G1 확인에 사용한다.
