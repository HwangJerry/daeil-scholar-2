# CentOS 7 APNs/FCM 운영 이력과 표준 절차

Last Updated: 2026-07-20
Owner: Backend / Server Operator
Applies To: `dflh-saf-v2/backend`, `alumni-backend.service`, `deploy.sh`

이 문서는 iOS APNs Sandbox/Production 이중 키와 Android FCM credential을 백엔드와 CentOS 7 운영 서버에 도입한 작업 이력, 장애 원인, 복구 과정, 이후 반복해서 사용할 표준 운영 절차를 기록한다.

비밀번호, `.p8` 내용, 실제 Key ID, DB 비밀번호, SMTP 비밀번호, device token 원문은 이 문서와 Git에 기록하지 않는다.

## 1. 작업 배경

기존 백엔드는 device token의 `apnsEnvironment`에 따라 APNs endpoint는 구분했지만, 서명에는 하나의 Key ID와 하나의 `.p8`만 사용했다.

Apple에서 발급한 환경 제한 키를 사용할 경우 다음 조합이 필요하다.

| Token environment | Endpoint | Signing credential |
|---|---|---|
| `sandbox` | `api.sandbox.push.apple.com` | Sandbox Key ID + Sandbox `.p8` |
| `production` | `api.push.apple.com` | Production Key ID + Production `.p8` |

Sandbox 키를 Production 요청에 재사용하거나 Production 키를 Sandbox 요청에 재사용하지 않는다.

## 2. 최종 구성

### 2.1 서버 파일 배치

```text
/app/backend/server
/app/backend/backfill

/etc/systemd/system/alumni-backend.service
/etc/sysconfig/alumni-backend
/etc/alumni-backend/firebase-service-account.json

/etc/alumni-backend/secrets/apns/
├── AuthKey_<SANDBOX_KEY_ID>.p8
└── AuthKey_<PRODUCTION_KEY_ID>.p8
```

역할을 다음처럼 분리한다.

- `/app/backend`: 배포되는 실행 바이너리만 저장
- `/etc/sysconfig/alumni-backend`: runtime 환경변수 저장
- `/etc/alumni-backend/firebase-service-account.json`: FCM 서비스 계정 JSON 저장
- `/etc/alumni-backend/secrets/apns`: APNs 개인키 저장
- `/etc/systemd/system`: systemd unit 저장

`.p8`을 `/app/backend`, `/var/www`, 저장소, systemd unit 본문에 저장하지 않는다.

FCM JSON과 APNs `.p8`은 모두 `root:alumni-backend`, `0640`으로 유지한다. 상위 디렉터리는 `alumni-backend` 그룹이 traversal할 수 있어야 한다.

### 2.2 systemd 실행 계정

서비스는 공용 `nobody` 계정 대신 전용 계정을 사용한다.

```ini
[Service]
User=alumni-backend
Group=alumni-backend
EnvironmentFile=/etc/sysconfig/alumni-backend
```

전용 계정은 다음 이유로 필요하다.

- APNs 키 읽기 권한을 백엔드 프로세스에만 부여
- upload, PG audit, EasyPay 권한을 경로별로 분리
- 다른 daemon이 `nobody` 계정을 공유하면서 비밀키를 읽는 상황 방지

### 2.3 APNs 환경변수

```dotenv
APNS_TEAM_ID=<APPLE_TEAM_ID>
APNS_BUNDLE_ID=com.daeil.dflhsafv2
APNS_ENVIRONMENT=production
APNS_REQUEST_TIMEOUT=5s

APNS_SANDBOX_KEY_ID=<SANDBOX_KEY_ID>
APNS_SANDBOX_PRIVATE_KEY_PATH=/etc/alumni-backend/secrets/apns/AuthKey_<SANDBOX_KEY_ID>.p8

APNS_PRODUCTION_KEY_ID=<PRODUCTION_KEY_ID>
APNS_PRODUCTION_PRIVATE_KEY_PATH=/etc/alumni-backend/secrets/apns/AuthKey_<PRODUCTION_KEY_ID>.p8
```

Production 키가 준비되지 않았다면 Production 두 항목은 빈 값으로 만들지 않고 생략한다. 단, active Production token 또는 Production outbox job이 존재하기 전에는 반드시 Production 키를 추가해야 한다.

### 2.4 FCM 환경변수

```dotenv
FCM_PROJECT_ID=<FIREBASE_PROJECT_ID>
FCM_CREDENTIALS_FILE=/etc/alumni-backend/firebase-service-account.json
FCM_REQUEST_TIMEOUT=5s
```

운영 systemd에서는 JSON 원문을 환경변수에 넣지 않고 `FCM_CREDENTIALS_FILE`을 사용한다. Firebase credential의 `project_id`, 환경파일의 `FCM_PROJECT_ID`, Android 앱의 `google-services.json` project id는 같아야 한다.

## 3. 백엔드 변경 내용

### 3.1 설정

- `APNsSandbox`, `APNsProduction` credential set을 분리했다.
- 환경별 변수가 기존 단일 키 변수보다 우선한다.
- 기존 `APNS_KEY_ID`, `APNS_PRIVATE_KEY*`, `PUSH_APNS_*`는 호환을 위해 유지한다.
- 기존 단일 키는 `APNS_ENVIRONMENT`가 가리키는 한 환경에만 배정한다.
- 기존 단일 키를 두 환경에 동시에 fallback하지 않는다.

### 3.2 provider

- Sandbox/Production `.p8`을 daemon 시작 시 읽고 파싱한다.
- 잘못된 파일, 읽기 실패, 부분 설정은 시작 오류로 처리한다.
- 환경별 APNs auth token cache를 분리한다.
- notification의 `apnsEnvironment`로 endpoint와 signing credential을 함께 선택한다.
- 설정되지 않은 환경으로 전송을 요청하면 다른 환경 키로 대체하지 않고 configuration error를 반환한다.

### 3.3 outbox 주의사항

APNs 설정 오류는 outbox에서 영구 오류로 분류되어 job이 `DEAD`가 될 수 있다. Sandbox만 설정된 기간에는 Production token/job이 없어야 한다.

확인 쿼리:

```sql
SELECT APNS_ENVIRONMENT, COUNT(*) AS token_count
FROM ALUMNI_MOBILE_DEVICE_TOKEN
WHERE PLATFORM = 'ios'
  AND STATUS = 'active'
GROUP BY APNS_ENVIRONMENT;
```

## 4. 서버 전환 작업 이력

### 4.1 APNs 키 설치

```bash
sudo install -d \
  -o root \
  -g alumni-backend \
  -m 0750 \
  /etc/alumni-backend/secrets/apns

sudo install \
  -o root \
  -g alumni-backend \
  -m 0640 \
  <UPLOADED_SANDBOX_P8> \
  /etc/alumni-backend/secrets/apns/AuthKey_<SANDBOX_KEY_ID>.p8
```

키 내용은 출력하지 않고 접근 가능 여부만 검사한다.

```bash
sudo -u alumni-backend test -r \
  /etc/alumni-backend/secrets/apns/AuthKey_<SANDBOX_KEY_ID>.p8 \
  && echo "OK sandbox key readable"
```

### 4.2 실행 계정 권한 점검 결과

전환 과정에서 다음 상태를 실제 서버에서 확인했다.

| 경로/기능 | 최초 결과 | 조치 후 결과 | 요구 권한 |
|---|---|---|---|
| `/app/backend/server` | OK | OK | 실행 |
| `/var/www/uploads` | FAIL | OK | 읽기/쓰기/하위 생성 |
| `/var/www/legacy/files` | OK | OK | 읽기/디렉터리 접근 |
| `/var/logs/pg` | FAIL | OK | 파일 생성/쓰기 |
| `/var/logs/pg/pg-audit.log` | 미확인 | OK | append 쓰기 |
| FCM service account JSON | FAIL | OK | 읽기 |
| EasyPay `immediately` binary/cert/log | OK | OK | 실행/읽기/쓰기 |
| EasyPay `profile` binary/cert/log | OK | OK | 실행/읽기/쓰기 |

`/var/www` 전체의 owner를 변경하지 않았다. 필요한 경로에만 ACL을 추가했다.

Upload ACL:

```bash
sudo setfacl -R -m u:alumni-backend:rwX /var/www/uploads

sudo find /var/www/uploads -type d \
  -exec setfacl -m d:u:alumni-backend:rwX {} +
```

PG audit ACL:

```bash
sudo setfacl -m u:alumni-backend:rwx /var/logs/pg
sudo setfacl -m d:u:alumni-backend:rwX /var/logs/pg

if sudo test -f /var/logs/pg/pg-audit.log; then
  sudo setfacl -m u:alumni-backend:rw /var/logs/pg/pg-audit.log
fi
```

실제 파일 열기 검사:

```bash
sudo -u alumni-backend sh -c \
  ': >> /var/logs/pg/pg-audit.log' \
  && echo "OK pg audit file writable"
```

### 4.3 EnvironmentFile 전환

기존 unit의 `Environment="KEY=value"`를 제거하고 `/etc/sysconfig/alumni-backend`로 이전했다.

중요한 운영 규칙:

- APNs 값만 넣고 기존 DB/JWT/SMTP/Kakao/EasyPay 값을 누락하지 않는다.
- `export KEY=value`를 사용하지 않는다.
- `KEY = value`처럼 `=` 주변에 공백을 넣지 않는다.
- 파일 권한은 `root:alumni-backend`, `0640`으로 유지한다.
- 변경 전 unit과 환경파일을 반드시 백업한다.

백업:

```bash
sudo cp -p /etc/systemd/system/alumni-backend.service \
  /etc/systemd/system/alumni-backend.service.before-apns-dual

if sudo test -f /etc/sysconfig/alumni-backend; then
  sudo cp -p /etc/sysconfig/alumni-backend \
    /etc/sysconfig/alumni-backend.before-apns-dual
fi
```

전환 중 환경파일에 APNs 값만 남아 `deploy.sh`가 운영 필수값 20개 누락을 감지한 사례가 있었다. 검사를 우회하지 않고 기존 unit 백업에서 환경변수를 복구했다.

복구 절차:

```bash
sudo cp -p /etc/sysconfig/alumni-backend \
  /etc/sysconfig/alumni-backend.apns-only

sudo install \
  -o root \
  -g root \
  -m 0600 \
  /dev/null \
  /etc/sysconfig/alumni-backend.migrated

sudo awk '
/^[[:space:]]*Environment="/ {
    line = $0
    sub(/^[[:space:]]*Environment="/, "", line)
    sub(/"[[:space:]]*$/, "", line)

    key = line
    sub(/=.*/, "", key)

    if (key ~ /^(APNS_|PUSH_APNS_)/) {
        next
    }

    print line
}
' /etc/systemd/system/alumni-backend.service.before-apns-dual |
  sudo tee /etc/sysconfig/alumni-backend.migrated >/dev/null

sudo awk '1' /etc/sysconfig/alumni-backend.apns-only |
  sudo tee -a /etc/sysconfig/alumni-backend.migrated >/dev/null

sudo chown root:alumni-backend \
  /etc/sysconfig/alumni-backend.migrated
sudo chmod 0640 \
  /etc/sysconfig/alumni-backend.migrated
sudo mv \
  /etc/sysconfig/alumni-backend.migrated \
  /etc/sysconfig/alumni-backend
```

값을 노출하지 않고 key name만 확인한다.

```bash
sudo awk -F= '
/^[A-Za-z_][A-Za-z0-9_]*=/ {
    print $1
}
' /etc/sysconfig/alumni-backend | sort
```

중복 key 확인:

```bash
sudo awk -F= '
/^[A-Za-z_][A-Za-z0-9_]*=/ {
    count[$1]++
}
END {
    for (key in count) {
        if (count[key] > 1) {
            print "DUPLICATE", key, count[key]
        }
    }
}
' /etc/sysconfig/alumni-backend
```

## 5. 배포·기동 장애 이력

### 5.1 macOS Bash 3.2 빈 배열 오류

증상:

```text
deploy.sh: line 201: SSH_OPTS[@]: unbound variable
```

원인:

- macOS 기본 `/bin/bash`는 3.2 계열이다.
- `set -u` 상태에서 빈 `SSH_OPTS=()`를 `"${SSH_OPTS[@]}"`로 확장하면 `unbound variable`이 발생한다.
- SSH port를 생략한 기본 배포 경로에서 발생했다.

수정:

- 빈 SSH/SCP option array 제거
- `run_ssh`, `run_scp` wrapper로 호출 통일
- `RSYNC_RSH`를 기본 port/custom port에 맞게 설정
- custom SSH port 숫자 검증 추가

첫 실패는 SSH 실행 전에 발생했으므로 서버 파일이나 daemon 상태를 변경하지 않았다.

### 5.2 필수 환경변수 20개 누락

증상:

```text
Env var validation failed.
Missing or empty (20)
```

감지된 key names:

```text
ALLOWED_ORIGIN
SITE_BASE_URL
DB_USER
DB_PASSWORD
DB_NAME
KAKAO_CLIENT_ID
KAKAO_CLIENT_SECRET
KAKAO_REDIRECT_URI
JWT_SECRET
UPLOAD_LEGACY_PATH
EASYPAY_IMMEDIATELY_MALL_ID
EASYPAY_PROFILE_MALL_ID
EASYPAY_GW_URL
EASYPAY_BIN_BASE
EASYPAY_RETURN_BASE_URL
SMTP_HOST
SMTP_USER
SMTP_PASSWORD
VISIT_IP_SALT
ENV
```

원인:

- sanitized systemd unit을 설치하면서 inline 환경변수는 제거했다.
- `/etc/sysconfig/alumni-backend`에는 APNs 값만 들어 있었다.
- DB/JWT/SMTP/Kakao/EasyPay 등 기존 운영값이 환경파일로 이전되지 않았다.

대응:

- `SKIP_ENV_CHECK=1`을 사용하지 않았다.
- daemon을 재시작하지 않았다.
- 기존 unit 백업에서 환경변수를 복구한 뒤 APNs 설정과 병합했다.

이 검사는 정상적인 fail-fast 동작이다. 필수값 누락 상태로 새 바이너리를 올리는 것보다 배포를 중단하는 것이 맞다.

### 5.3 SSH 개인키 교체

기존 `~/.ssh/daeil_deploy`의 passphrase를 확인할 수 없어 새 `daeil_deploy_v2` 키를 생성하고 `daeil-prod` alias의 `IdentityFile`을 교체했다.

구분:

- SSH key passphrase: Mac의 private key 암호
- `root@host` password: CentOS 원격 계정 암호

private key passphrase는 서버에서 조회하거나 복구할 수 없다. 새 키가 동작하기 전 기존 키를 삭제하지 않는다.

권장 SSH config:

```sshconfig
Host daeil-prod
    HostName <SERVER_HOST>
    User root
    IdentityFile ~/.ssh/daeil_deploy_v2
    IdentitiesOnly yes
    UseKeychain yes
    AddKeysToAgent yes
```

확인:

```bash
ssh -G daeil-prod |
  awk '$1 == "identityfile" { print $2 }'

ssh daeil-prod \
  'echo "OK SSH"; sudo -n cat /etc/sysconfig/alumni-backend >/dev/null && echo "OK sudo"'
```

### 5.4 FCM credential 권한으로 인한 재시작 루프

증상:

```text
initialize firebase messaging client: cannot read credentials file:
open /etc/alumni-backend/firebase-service-account.json: permission denied
```

초기 파일 상태는 `root:nobody`, `0640`이었다. systemd 서비스는 `alumni-backend` 사용자와 그룹으로 실행되므로 JSON을 읽지 못했고, dependency wiring 단계에서 status 1로 종료한 뒤 `Restart=on-failure`에 따라 5초마다 재시작했다.

복구:

```bash
sudo chown root:alumni-backend \
  /etc/alumni-backend/firebase-service-account.json

sudo chmod 0640 \
  /etc/alumni-backend/firebase-service-account.json

sudo -u alumni-backend test -r \
  /etc/alumni-backend/firebase-service-account.json \
  && echo "OK FCM credential readable"

sudo systemctl restart alumni-backend
```

복구 후 확인된 상태:

- FCM JSON 구조가 유효함
- credential `project_id`와 `FCM_PROJECT_ID`가 일치함
- Android `google-services.json` project id와도 일치함
- 서비스가 `active/running`으로 유지됨
- `/api/health`가 HTTP 200을 반환함

현재 `deploy.sh`는 APNs key 권한은 preflight에서 검사하지만 FCM credential 파일 권한은 강제하지 않는다. 배포 전 아래 검사를 별도로 수행한다.

```bash
sudo -u alumni-backend test -r \
  /etc/alumni-backend/firebase-service-account.json
```

## 6. 표준 배포 절차

### 6.1 배포 전 서버 백업

```bash
sudo cp -p /app/backend/server \
  /app/backend/server.before-current-deploy

sudo cp -p /etc/systemd/system/alumni-backend.service \
  /etc/systemd/system/alumni-backend.service.before-current-deploy

sudo cp -p /etc/sysconfig/alumni-backend \
  /etc/sysconfig/alumni-backend.before-current-deploy
```

### 6.2 설정과 권한 preflight

```bash
sudo systemd-analyze verify \
  /etc/systemd/system/alumni-backend.service

sudo -u alumni-backend test -x /app/backend/server
sudo -u alumni-backend test -w /var/www/uploads
sudo -u alumni-backend test -r /var/www/legacy/files
sudo -u alumni-backend test -w /var/logs/pg/pg-audit.log
sudo -u alumni-backend test -r \
  /etc/alumni-backend/firebase-service-account.json
```

APNs key:

```bash
sudo -u alumni-backend test -r \
  /etc/alumni-backend/secrets/apns/AuthKey_<SANDBOX_KEY_ID>.p8
```

### 6.3 로컬 검증

```bash
cd backend
go test ./...
go test -race ./internal/config ./internal/service ./internal/job
go vet ./...

cd ..
bash -n deploy.sh
git diff --check
```

### 6.4 배포

```bash
./deploy.sh daeil-prod --backend-only
```

`--backend-only`는 다음 frontend 인프라 작업을 건너뛴다.

- Apache configuration 업로드
- legacy PHP compatibility shim 업로드
- `httpd` reload
- legacy URL smoke test

전체/frontend 배포에서는 위 작업을 수행한다.

`deploy.sh`는 다음을 새 바이너리 교체 전에 검사한다.

- `/etc/sysconfig/alumni-backend` 존재와 권한
- 필수 운영 환경변수
- APNs credential set 완전성
- `.p8` 경로 제한
- daemon 사용자 기준 `.p8` 읽기 가능 여부
- migration drift

검사를 `SKIP_ENV_CHECK=1` 또는 `SKIP_MIGRATION_CHECK=1`로 우회하는 것은 장애 대응 중 명시적인 승인 없이 사용하지 않는다.

## 7. 배포 후 검증

### 7.1 daemon

```bash
sudo systemctl status alumni-backend --no-pager
sudo journalctl -u alumni-backend -n 150 --no-pager
```

기대 로그:

```text
push: APNs provider configured
push outbox worker started
```

2026-07-17 실제 기동 증빙:

```text
push: APNs provider configured environments=[sandbox, production]
push outbox worker started batch_size=50 poll_interval=5s max_attempts=8
```

같은 프로세스가 재시작 없이 유지됐고 다음 health 응답을 확인했다.

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"status":"ok"}
```

오류가 없어야 하는 항목:

```text
read apns sandbox key file
parse apns sandbox private key
apns team id is required
apns bundle id is required
```

### 7.2 outbox

실제 결제는 수행하지 않는다. 테스트 계정의 쪽지 또는 관리자 공지로 push event 한 건만 생성한다.

```sql
SELECT
  PO_SEQ,
  EVENT_TYPE,
  APNS_ENVIRONMENT,
  STATUS,
  ATTEMPT_COUNT,
  LAST_ERROR_CODE,
  LAST_ERROR_MESSAGE
FROM ALUMNI_PUSH_OUTBOX
ORDER BY PO_SEQ DESC
LIMIT 20;
```

Sandbox 개발 앱의 기대 결과:

```text
APNS_ENVIRONMENT = sandbox
STATUS = SENT
```

### 7.3 민감정보 로그 검사

다음 정보가 journal이나 애플리케이션 로그에 나타나면 안 된다.

- `.p8` 원문
- 환경파일 전체 내용
- device token 원문
- DB/JWT/SMTP 비밀번호
- 전체 push payload body

### 7.4 기동 검증과 실제 발송 검증의 구분

이번 기동 확인으로 증명된 범위:

- Sandbox/Production APNs `.p8` 읽기와 parsing 성공
- APNs provider 및 outbox worker 시작
- FCM credential JSON 읽기와 Firebase client 생성 성공
- backend와 DB health 정상

아직 실제 provider 전송을 증명하지는 않는다. 다음 결과는 실기기 smoke test로 별도 보관한다.

- Debug iOS token의 Sandbox outbox `SENT`
- TestFlight/Release iOS token의 Production outbox `SENT`
- Android 실기기의 FCM 수신과 `push_status=success`

## 8. Production 키 추가 절차

1. Apple Developer에서 Production용 topic-specific APNs key를 발급한다.
2. 새 `.p8`을 `/etc/alumni-backend/secrets/apns`에 `root:alumni-backend`, `0640`으로 설치한다.
3. `/etc/sysconfig/alumni-backend`에 Production Key ID와 path를 추가한다.
4. daemon 사용자로 파일 읽기를 검사한다.
5. daemon을 재시작한다.
6. TestFlight/Release token이 `production`으로 등록되는지 확인한다.
7. 실제 Production push 한 건을 검증한다.

환경파일 추가값:

```dotenv
APNS_PRODUCTION_KEY_ID=<PRODUCTION_KEY_ID>
APNS_PRODUCTION_PRIVATE_KEY_PATH=/etc/alumni-backend/secrets/apns/AuthKey_<PRODUCTION_KEY_ID>.p8
```

코드 재배포나 새 DB migration은 필요하지 않다.

## 9. APNs 키 교체 절차

1. 기존 키를 제거하기 전에 새 키를 발급한다.
2. 새 `.p8`을 새 파일명으로 설치한다.
3. 환경파일의 해당 환경 Key ID와 path를 동시에 교체한다.
4. `systemctl restart alumni-backend`를 수행한다.
5. startup parsing 성공과 push 한 건을 확인한다.
6. 검증 완료 후 Apple Developer에서 이전 키를 revoke한다.
7. 서버의 이전 `.p8`을 별도 보안 절차에 따라 폐기한다.

환경별 Key ID와 path는 반드시 같은 키 쌍으로 변경한다.

## 10. 롤백

새 daemon이 시작하지 못할 경우:

```bash
sudo cp -p /app/backend/server.before-current-deploy \
  /app/backend/server

sudo cp -p \
  /etc/systemd/system/alumni-backend.service.before-current-deploy \
  /etc/systemd/system/alumni-backend.service

sudo cp -p \
  /etc/sysconfig/alumni-backend.before-current-deploy \
  /etc/sysconfig/alumni-backend

sudo systemctl daemon-reload
sudo systemctl restart alumni-backend
sudo systemctl status alumni-backend --no-pager
```

이번 APNs 이중 키 변경은 새 DB migration을 추가하지 않으므로 DB rollback은 없다. 기존 push migrations `028`~`035`는 별도로 유지한다.

## 11. 장애 대응표

| 증상 | 가능한 원인 | 확인 | 조치 |
|---|---|---|---|
| `SSH_OPTS[@]: unbound variable` | 구형 `deploy.sh`, Bash 3.2 빈 배열 | `git diff`, script version | wrapper 적용 버전 사용 |
| 필수 env 20개 누락 | APNs 값만 EnvironmentFile로 이전 | key name만 출력 | 기존 unit/env 백업에서 복구 |
| `.p8` unreadable | owner/group/mode 또는 상위 dir 권한 | `sudo -u alumni-backend test -r` | `root:alumni-backend`, `0640`, dir `0750` |
| FCM JSON `permission denied` | 파일 group이 `nobody` 등 다른 계정 | daemon 사용자 `test -r`, `stat` | `root:alumni-backend`, `0640`으로 수정 후 재시작 |
| backend startup 실패 | `.p8` parse, PG audit, DB 설정 | `journalctl` | 첫 startup error부터 해결 |
| upload 실패 | `alumni-backend` ACL 없음 | create/delete permission test | upload 경로에 access/default ACL |
| `InvalidProviderToken` | Key ID/Team ID/환경/서버 시간 불일치 | APNs reason, `timedatectl` | 올바른 환경 키와 시간 동기화 확인 |
| `BadDeviceToken` | token 환경 또는 token 자체 오류 | DB environment, APNs reason | 올바른 endpoint 확인, token revoke |
| Sandbox만 성공 | Production credential 미설정 | env key names, provider startup log | Production key 설치 |
| job이 `DEAD` | config/auth/payload 영구 오류 | outbox error columns | 원인 수정 후 정책에 따라 replay |

## 12. 보안 및 운영 제약

- SSL/APNs 인증 오류를 우회하지 않는다.
- `.p8`, Firebase service account JSON, 환경파일을 Git에 커밋하지 않는다.
- 서비스 unit에 secret 값을 직접 넣지 않는다.
- key 내용을 `cat`, shell history, ticket, chat에 붙여 넣지 않는다.
- 운영값 검사를 우회하고 배포하지 않는다.
- 실제 결제는 push smoke 과정에서 수행하지 않는다.
- CentOS 7은 지원 종료 상태이므로 OS 업그레이드 계획을 별도 운영 과제로 유지한다.
- 과거 tracked systemd template에 들어 있던 자격정보는 Git history에 남을 수 있으므로 해당 credential을 교체한다.

## 13. 2026-07-17 상태

완료:

- APNs Sandbox/Production 이중 키 백엔드 구현
- 환경별 token cache와 endpoint/key 선택
- fail-fast `.p8` parsing
- CentOS 전용 daemon 사용자 준비
- Sandbox `.p8` 서버 배치 및 읽기 권한 검증
- Production `.p8` 서버 배치 및 읽기·parsing 검증
- FCM service account JSON의 프로젝트 일치 및 읽기 권한 복구
- upload/PG audit ACL 조정
- legacy/EasyPay 접근 검증
- EnvironmentFile 전환과 기존 운영값 복구
- macOS Bash 3.2 deploy wrapper 수정
- SSH deploy key를 `daeil_deploy_v2`로 교체
- backend `active/running`과 `/api/health` HTTP 200 확인
- APNs configured environments `[sandbox, production]` 확인
- push outbox worker 시작 확인

최종 증빙으로 계속 보관할 항목:

- 성공한 `deploy.sh` 실행 로그
- Sandbox outbox `SENT` 결과
- TestFlight/Release Production outbox `SENT` 결과
- Android 실기기 FCM 수신과 `push_status=success` 결과

## 14. 2026-07-19 알림 표시 정책과 배포 이력

### 14.1 공지와 쪽지 알림 문구

사용자에게 표시되는 문구의 단일 백엔드 정의 위치는
`backend/internal/service/push_notifier.go`다.

- `admin.notice`
  - 제목은 `새 소식`이다.
  - 본문은 공지 제목의 공백을 정규화해 사용한다.
  - 공지 제목이 비어 있으면 `새 공지가 등록되었습니다.`를 사용한다.
- `message.new`
  - 제목은 발신자 이름의 공백을 정규화해 사용한다.
  - 본문은 쪽지 내용의 공백을 정규화한 미리보기다.
  - 미리보기는 최대 80 Unicode 글자로 제한하고 초과분은 말줄임표로 끝낸다.
  - 이름이나 내용이 비어 있으면 일반 문구로 대체한다.

쪽지 발신자 이름과 내용 미리보기는 별도의 사용자 그룹별 마스킹 없이 모든
push 대상 수신자에게 같은 정책으로 표시한다. 다만 다음 기존 발송 조건은 계속
적용한다.

- 현재 로그인 계정에 연결된 active device token
- 사용자의 쪽지 또는 공지 push preference
- 이벤트별 수신 대상과 중복 방지 규칙

발신자 이름과 쪽지 본문은 APNs/FCM의 사용자 표시용 alert에만 포함한다. 라우팅용
custom data/`args`에는 넣지 않으며, 전체 payload를 애플리케이션 로그에 기록하지
않는다. 이 구분은 `docs/push/contract.md`와 `docs/push/policy.md`를 기준으로 유지한다.

### 14.2 Android heads-up 채널 정합성

Android heads-up 알림을 위해 백엔드 FCM notification의 channel id를
`dflh_push_v2`로 맞췄다. 이 값은 Android 앱의 manifest 기본 채널과 앱 시작 시
생성하는 high-importance 채널 id와 동일해야 한다. 채널 이름과 사용자에게 보이는
importance 설정은 Android 앱이 소유하며, 백엔드는 channel id만 지정한다.

해당 백엔드 변경은 운영 서버에 `./deploy.sh daeil-prod --backend-only` 경로로
배포한 이력이 있다. 현재 스크립트에서 이 경로는 backend binary와 backend service만
갱신하며 Apache 설정, legacy compatibility shim, `httpd` reload와 legacy URL smoke
test는 건너뛴다.

### 14.3 검증 범위와 남은 증빙

코드와 단위 테스트로 확인하는 항목:

- 공지 제목과 쪽지 발신자/미리보기 문구 생성
- 80 Unicode 글자 제한과 fallback 문구
- 민감한 표시 문구가 custom data에 복사되지 않음
- FCM channel id가 `dflh_push_v2`임

배포 성공이나 startup/health 확인만으로 실제 provider 수신 또는 heads-up 표시까지
증명되지는 않는다. 다음 결과는 기기별 smoke test 증빙으로 별도 보관한다.

- iOS Sandbox와 Production outbox의 실제 `SENT` 결과
- Android FCM provider의 `push_status=success`
- Android 실기기에서 알림 권한 허용 및 `dflh_push_v2` 중요도 `높음` 상태의
  heads-up 표시

이 표시 정책 및 채널 변경에는 새 DB migration이 없다. 기존 push migration
`028`~`035`가 운영 DB에 실제 반영되었는지는 daemon 기동 로그만으로 단정하지 않고,
`deploy.sh` migration drift 검사와 운영 DB schema 조회 결과로 별도 확인한다.
