# OAuth 개발자 앱 등록 가이드

## 개요

현재 운영 코드가 지원하는 소셜 로그인 프로바이더는 다음과 같습니다.

| 프로바이더 | DB 소셜 ID 코드 | OAuth 방식 |
|-----------|-----------------|------------|
| 카카오 | `KT` | 웹 인가 코드, iOS SDK 액세스 토큰 |
| Apple | `AP` | iOS Sign in with Apple + 서버 코드 교환 |

네이버/Facebook 값은 레거시 데이터에 남아 있을 수 있지만 현재 로그인 라우트에는 연결되어 있지 않습니다. 이메일만 일치한다는 이유로 기존 계정을 자동 연결하지 않으며, 기존 계정 연결에는 ID/비밀번호 재인증이 필요합니다.

## 공통 사항

### 콜백 URL 패턴

```
https://{도메인}/api/auth/{provider}/callback
```

- `{provider}`: `kakao`, `naver`, `facebook`
- 로컬 개발 시 기본값: `http://localhost:8000/api/auth/{provider}/callback`

### 인증 흐름

1. 프론트엔드에서 소셜 로그인 버튼 클릭
2. 백엔드가 해당 프로바이더의 인가 URL로 리다이렉트
3. 사용자가 프로바이더에서 로그인 및 동의
4. 프로바이더가 콜백 URL로 인가 코드 전달
5. 백엔드가 인가 코드로 액세스 토큰 교환
6. 액세스 토큰으로 사용자 정보 조회
7. JWT 발급 및 세션 생성

---

## 카카오 설정

### 1. 앱 생성

1. [Kakao Developers](https://developers.kakao.com)에 접속하여 로그인
2. **내 애플리케이션** > **애플리케이션 추가하기** 클릭
3. 앱 이름, 사업자명 입력 후 저장

### 2. 카카오 로그인 활성화

1. **제품 설정** > **카카오 로그인** 메뉴 진입
2. **활성화 설정**을 **ON**으로 변경
3. **Redirect URI**에 콜백 URL 등록:
   - 로컬: `http://localhost:8000/api/auth/kakao/callback`
   - 프로덕션: `https://{도메인}/api/auth/kakao/callback`

### 3. 동의항목 설정

1. **제품 설정** > **카카오 로그인** > **동의항목** 메뉴 진입
2. 다음 항목을 **필수 동의** 또는 **선택 동의**로 설정:
   - **닉네임** (프로필)
   - **이메일**

### 4. 키 확인

1. **앱 설정** > **앱 키** 메뉴에서 **REST API 키** 복사 → `KAKAO_CLIENT_ID`
2. **제품 설정** > **카카오 로그인** > **보안** 메뉴에서 **Client Secret** 발급 → `KAKAO_CLIENT_SECRET`
   - Client Secret 상태를 **사용함**으로 설정

### 5. 환경변수

```bash
KAKAO_CLIENT_ID=<REST API 키>
KAKAO_CLIENT_SECRET=<Client Secret>
KAKAO_REDIRECT_URI=http://localhost:8000/api/auth/kakao/callback
KAKAO_ALLOWED_REDIRECT_URIS=http://localhost:8000/api/auth/kakao/callback,dflhsafv2://oauth
```

---

## Sign in with Apple 설정

### 1. Apple Developer 구성

1. App ID에서 Sign in with Apple capability를 활성화합니다.
2. iOS bundle ID를 `APPLE_BUNDLE_ID` 및 허용 audience에 등록합니다.
3. 웹 Service ID를 사용하는 환경이면 Service ID도 허용 audience에 추가합니다.
4. Sign in with Apple 키를 만들고 Team ID, Key ID, `.p8` private key를 보관합니다.
5. Apple server-to-server notification URL을 `POST /api/auth/apple/notifications`로 등록합니다.
   - Apple이 보내는 JSON의 `payload` 필드에는 서명된 JWS가 들어옵니다.
   - 서버는 JWS의 Apple issuer, 허용 audience, `iat`, `jti`, event subject/type과 JWKS 서명을 검증합니다.

### 2. iOS capability

Debug/Release entitlements 모두 `com.apple.developer.applesignin = Default`를 포함해야 합니다. 실제 기기 서명 프로파일에도 동일 capability가 포함됐는지 확인합니다.

### 3. 백엔드 환경변수

```bash
APPLE_TEAM_ID=<10자리 Team ID>
APPLE_KEY_ID=<Sign in with Apple Key ID>
APPLE_CLIENT_ID=<iOS bundle ID 또는 Service ID>
APPLE_BUNDLE_ID=<iOS bundle ID>
APPLE_PRIVATE_KEY_PATH=/run/secrets/AuthKey_XXXXXXXXXX.p8
APPLE_ALLOWED_AUDIENCES=com.daeil.dflhsafv2,<선택적 Service ID>
SOCIAL_CREDENTIAL_ENCRYPTION_KEY=<32바이트 키의 표준 base64>
```

`APPLE_PRIVATE_KEY`로 PEM 내용을 직접 전달할 수도 있지만 secret file/manager 사용을 권장합니다. private key와 credential encryption key는 소스, 로그, 앱 번들에 넣지 않습니다.

### 4. 검증 동작

- 앱은 서버에서 일회용 nonce challenge를 받고 SHA-256 nonce를 Apple 요청에 전달합니다.
- 서버는 Apple JWKS의 `kid`, RS256 서명, `iss`, 허용 audience, `exp`, nonce를 검증합니다.
- authorization code는 서버에서 Apple token endpoint로 한 번만 교환하며 재사용을 거부합니다.
- Apple이 이름/이메일을 다시 주지 않는 로그인도 정상 처리합니다. Private Relay 이메일은 유효한 이메일로 취급합니다.
- 앱은 저장한 Apple user identifier로 시작/foreground 시 credential state를 확인하고 revoked notification도 처리합니다.

## 환경변수 요약

| 환경변수 | 프로바이더 | 설명 | 기본값 |
|---------|-----------|------|--------|
| `KAKAO_CLIENT_ID` | 카카오 | REST API 키 | (없음) |
| `KAKAO_CLIENT_SECRET` | 카카오 | Client Secret | (없음) |
| `KAKAO_REDIRECT_URI` | 카카오 | OAuth 콜백 URL | `http://localhost:8000/api/auth/kakao/callback` |
| `KAKAO_ALLOWED_REDIRECT_URIS` | 카카오 | 코드 교환 시 허용할 정확한 redirect URI 목록 | `KAKAO_REDIRECT_URI` |
| `APPLE_TEAM_ID` / `APPLE_KEY_ID` | Apple | client secret 서명자 식별값 | (없음) |
| `APPLE_CLIENT_ID` | Apple | token/revoke endpoint의 client ID | (없음) |
| `APPLE_ALLOWED_AUDIENCES` | Apple | ID token 허용 audience CSV | (없음) |
| `APPLE_PRIVATE_KEY_PATH` | Apple | ES256 `.p8` 키 경로 | (없음) |
| `SOCIAL_CREDENTIAL_ENCRYPTION_KEY` | 공통 | revocation credential AES-256-GCM 키 | (없음) |
| `ACCESS_TOKEN_TTL` | 모바일 세션 | access token 수명 | `15m` |
| `REFRESH_TOKEN_TTL` | 모바일 세션 | rotating refresh token 수명 | `720h` |

---

## 프로덕션 배포 체크리스트

- [ ] **HTTPS 필수**: 모든 프로바이더가 프로덕션 환경에서 HTTPS 콜백 URL을 요구합니다.
- [ ] **콜백 URL 변경**: 각 프로바이더 개발자 콘솔에서 콜백 URL을 프로덕션 도메인으로 변경합니다.
- [ ] **환경변수 업데이트**: `*_REDIRECT_URI` 값을 프로덕션 URL로 설정합니다.
- [ ] **허용목록 고정**: `KAKAO_ALLOWED_REDIRECT_URIS`에 실제 등록 URI만 포함합니다.
- [ ] **카카오**: 앱 상태가 **활성화**인지 확인합니다. 비즈앱 전환이 필요할 수 있습니다.
- [ ] **Apple**: capability, provisioning profile, audience, `.p8` 키와 server notification URL을 확인합니다.
- [ ] **암호화 키**: `SOCIAL_CREDENTIAL_ENCRYPTION_KEY`를 생성하고 복구 가능한 secret manager에 보관합니다.
- [ ] **시크릿 관리**: Client Secret 값은 절대 소스 코드에 포함하지 않습니다. 환경변수 또는 시크릿 매니저를 사용합니다.
- [ ] **DB 적용**: `028_social_auth_security.sql`을 백업/검증 절차에 따라 적용합니다.
- [ ] **운영 절차**: [소셜 인증 운영 런북](social-auth-runbook.md)의 사전/사후 검증을 수행합니다.
