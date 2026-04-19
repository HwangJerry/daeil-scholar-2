# OAuth 개발자 앱 등록 가이드

## 개요

본 시스템은 3개 소셜 로그인 프로바이더를 지원합니다.

| 프로바이더 | DB 소셜 ID 코드 | OAuth 방식 |
|-----------|-----------------|------------|
| 카카오 | `KT` | OAuth 2.0 인가 코드 |
| 네이버 | `NV` | OAuth 2.0 인가 코드 |
| Facebook | `FB` | OAuth 2.0 인가 코드 |

모든 프로바이더는 **OAuth 2.0 Authorization Code Grant** 방식을 사용하며, 사용자가 소셜 로그인 후 기존 동문 회원 계정과 연동(linking)하는 흐름으로 동작합니다.

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
```

---

## 네이버 설정

### 1. 애플리케이션 등록

1. [Naver Developers](https://developers.naver.com)에 접속하여 로그인
2. **Application** > **애플리케이션 등록** 클릭
3. 애플리케이션 이름 입력

### 2. API 설정

1. **사용 API**에서 **네이버 로그인** 선택
2. **제공 정보 선택**에서 다음 항목 체크:
   - **회원이름** (필수)
   - **이메일** (필수)

### 3. 환경 설정

1. **로그인 오픈 API 서비스 환경**에서 **WEB** 선택
2. **서비스 URL** 입력:
   - 로컬: `http://localhost:8000`
   - 프로덕션: `https://{도메인}`
3. **Callback URL** 입력:
   - 로컬: `http://localhost:8000/api/auth/naver/callback`
   - 프로덕션: `https://{도메인}/api/auth/naver/callback`

### 4. 키 확인

- **애플리케이션 정보**에서 **Client ID** 복사 → `NAVER_CLIENT_ID`
- **Client Secret** 복사 → `NAVER_CLIENT_SECRET`

### 5. 환경변수

```bash
NAVER_CLIENT_ID=<Client ID>
NAVER_CLIENT_SECRET=<Client Secret>
NAVER_REDIRECT_URI=http://localhost:8000/api/auth/naver/callback
```

---

## Facebook 설정

### 1. 앱 생성

1. [Meta for Developers](https://developers.facebook.com)에 접속하여 로그인
2. **내 앱** > **앱 만들기** 클릭
3. 앱 유형으로 **소비자** 또는 **비즈니스** 선택
4. 앱 이름, 연락처 이메일 입력 후 생성

### 2. Facebook 로그인 제품 추가

1. 앱 대시보드에서 **제품 추가** > **Facebook 로그인** > **설정** 클릭
2. **웹** 플랫폼 선택
3. **설정** > **유효한 OAuth 리디렉션 URI**에 콜백 URL 등록:
   - 로컬: `http://localhost:8000/api/auth/facebook/callback`
   - 프로덕션: `https://{도메인}/api/auth/facebook/callback`

### 3. 권한

기본적으로 다음 권한이 사용됩니다 (추가 설정 불필요):

- `email`
- `public_profile`

### 4. 키 확인

1. **설정** > **기본 설정** 메뉴 진입
2. **앱 ID** 복사 → `FACEBOOK_CLIENT_ID`
3. **앱 시크릿 코드** > **보기** 클릭 후 복사 → `FACEBOOK_CLIENT_SECRET`

### 5. Graph API 버전

본 시스템은 **Graph API v19.0**을 사용합니다. 앱 대시보드의 **설정** > **기본 설정**에서 API 버전을 확인할 수 있습니다.

### 6. 환경변수

```bash
FACEBOOK_CLIENT_ID=<앱 ID>
FACEBOOK_CLIENT_SECRET=<앱 시크릿 코드>
FACEBOOK_REDIRECT_URI=http://localhost:8000/api/auth/facebook/callback
```

---

## 환경변수 요약

| 환경변수 | 프로바이더 | 설명 | 기본값 |
|---------|-----------|------|--------|
| `KAKAO_CLIENT_ID` | 카카오 | REST API 키 | (없음) |
| `KAKAO_CLIENT_SECRET` | 카카오 | Client Secret | (없음) |
| `KAKAO_REDIRECT_URI` | 카카오 | OAuth 콜백 URL | `http://localhost:8000/api/auth/kakao/callback` |
| `NAVER_CLIENT_ID` | 네이버 | Client ID | (없음) |
| `NAVER_CLIENT_SECRET` | 네이버 | Client Secret | (없음) |
| `NAVER_REDIRECT_URI` | 네이버 | OAuth 콜백 URL | `http://localhost:8000/api/auth/naver/callback` |
| `FACEBOOK_CLIENT_ID` | Facebook | 앱 ID | (없음) |
| `FACEBOOK_CLIENT_SECRET` | Facebook | 앱 시크릿 코드 | (없음) |
| `FACEBOOK_REDIRECT_URI` | Facebook | OAuth 콜백 URL | `http://localhost:8000/api/auth/facebook/callback` |

---

## 프로덕션 배포 체크리스트

- [ ] **HTTPS 필수**: 모든 프로바이더가 프로덕션 환경에서 HTTPS 콜백 URL을 요구합니다.
- [ ] **콜백 URL 변경**: 각 프로바이더 개발자 콘솔에서 콜백 URL을 프로덕션 도메인으로 변경합니다.
- [ ] **환경변수 업데이트**: `*_REDIRECT_URI` 값을 프로덕션 URL로 설정합니다.
- [ ] **카카오**: 앱 상태가 **활성화**인지 확인합니다. 비즈앱 전환이 필요할 수 있습니다.
- [ ] **네이버**: 서비스 URL을 프로덕션 도메인으로 변경합니다.
- [ ] **Facebook**: 앱 모드를 **개발** → **라이브**로 전환합니다. 라이브 전환 시 앱 검수(App Review)가 필요할 수 있으며, 개인정보처리방침 URL 등록이 필수입니다.
- [ ] **시크릿 관리**: Client Secret 값은 절대 소스 코드에 포함하지 않습니다. 환경변수 또는 시크릿 매니저를 사용합니다.
