# 기부 결제 시스템 설계 (§16)

> 원본: TECH_DESIGN_DOC.md — 이 파일은 원본 설계서에서 분리된 상세 문서입니다.

---

## 16. 기부 결제 시스템 설계

V1의 EasyPay(KICC) PG 결제 기능을 V2(Go + React)로 이식합니다. 단계적으로 구현하며, Phase 1(일시후원 카드결제 MVP)부터 시작합니다.

### 16.1 결제 흐름 개요

```
[React SPA]                    [Go Backend]                    [EasyPay PG]
     │                              │                              │
  1. POST /api/donation/orders      │                              │
     │─────────────────────────────>│ INSERT WEO_ORDER (pending)   │
     │<─────────────────────────────│ { orderSeq, paymentParams }  │
     │                              │                              │
  2. EasyPayBridge 컴포넌트         │                              │
     SDK 로딩, form 세팅,           │                              │
     easypay_card_webpay() 호출     │                              │
     │── form POST ────────────────>│                              │
     │                              │ 3. POST /pg/easypay/relay    │
     │                              │    auto-submit HTML 응답     │
     │                              │────── form POST ────────────>│
     │                              │                              │
     │                     (사용자가 EasyPay UI에서 결제 완료)       │
     │                              │                              │
     │                              │<── POST /pg/easypay/return ──│
     │                              │ 4. sp_res_cd 파싱            │
     │                              │    ep_cli 바이너리 실행       │
     │                              │    INSERT WEO_PG_DATA        │
     │                              │    UPDATE WEO_ORDER          │
     │                              │    캐시 무효화               │
     │<──── 302 redirect ──────────│                              │
  5. /donation/result?status=       │                              │
     success&order=xxx              │                              │
```

> **핵심 발견사항:** EasyPay의 서버 사이드 승인은 HTTP API 호출이 아닙니다. **네이티브 Linux 바이너리** (`ep_cli`)를 셸아웃하여 `gw.easypay.co.kr`과 통신합니다. Go에서 `os/exec`로 래핑합니다. 바이너리는 V1 서버에 이미 존재합니다: `dflh-saf-v1/html/_sys/payment/*/bin/linux_64/ep_cli`

### 16.2 EasyPay 설정

#### Mall ID 및 상수

| 결제 유형 | Mall ID | sp_pay_type | sp_window_type | 용도 |
|-----------|---------|-------------|----------------|------|
| 일시후원 (immediately) | `05542574` | `11` (카드), `21` (계좌이체), `31` (휴대폰) | `iframe` | Phase 1 |
| 월정기후원 (profile) | `05543499` | `81` (정기결제) | `submit` | Phase 2 |

#### Gate/Type 매핑 (V1 컨벤션)

| 입력 (gate) | DB O_GATE | 설명 |
|-------------|-----------|------|
| `immediately` | `S` | 일시후원 (Single) |
| `profile` | `P` | 월정기후원 (Profile) |
| `project` | `F` | 프로젝트 기부 (Fund) |

| 입력 (payType) | DB O_PAY_TYPE | EasyPay sp_pay_type |
|----------------|---------------|---------------------|
| `CARD` | `CARD` | `11` |
| `BANK` | `BANK` | `21` |
| `HP` | `HP` | `31` |

#### 환경 변수

```
EASYPAY_IMMEDIATELY_MALL_ID=05542574
EASYPAY_PROFILE_MALL_ID=05543499
EASYPAY_GW_URL=gw.easypay.co.kr
EASYPAY_GW_PORT=80
EASYPAY_BIN_BASE=/var/www/html/_sys/payment
EASYPAY_RETURN_BASE_URL=https://www.daeilfoundation.or.kr
```

> **테스트 환경:** SDK는 `testsp.easypay.co.kr`, 게이트웨이는 `testgw.easypay.co.kr:80` 사용

#### Config 구조체

```go
// internal/config/config.go — EasyPayConfig 추가

type EasyPayConfig struct {
    ImmediatelyMallID string
    ProfileMallID     string
    GatewayURL        string
    GatewayPort       string
    BinBase           string        // ep_cli 바이너리 기본 경로
    ReturnBaseURL     string        // 콜백 베이스 URL
}
// CertFile, BinPath, LogDir는 EasyPayService.Approve() 내에서 BinBase로부터 파생

// Load()에서:
EasyPay: EasyPayConfig{
    ImmediatelyMallID:  getEnv("EASYPAY_IMMEDIATELY_MALL_ID", "05542574"),
    ProfileMallID:      getEnv("EASYPAY_PROFILE_MALL_ID", "05543499"),
    GatewayURL:         getEnv("EASYPAY_GW_URL", "testgw.easypay.co.kr"),
    GatewayPort:        getEnv("EASYPAY_GW_PORT", "80"),
    BinBase:            getEnv("EASYPAY_BIN_BASE", "/var/www/html/_sys/payment"),
    ReturnBaseURL:      getEnv("EASYPAY_RETURN_BASE_URL", "http://localhost:8080"),
},
PGAuditLogPath: getEnv("PG_AUDIT_LOG_PATH", "/var/logs/pg/pg-audit.log"),
```

### 16.3 백엔드 신규 파일

#### 파일 목록

| 신규 파일 | 용도 |
|-----------|------|
| `internal/handler/payment_handler.go` | `CreateOrder`, `GetOrder`, `EasyPayRelay`, `EasyPayReturn` |
| `internal/handler/templates/easypay_relay.html` | auto-submit HTML form → `sp.easypay.co.kr/ep8/MainAction.do` (`go:embed`로 바이너리에 포함) |
| `internal/service/easypay_service.go` | `ep_cli` 바이너리 래퍼 — `Approve()` 메서드 |
| `internal/service/donate_service.go` | `CreateOrder`, `ConfirmPayment` 비즈니스 로직 |
| `internal/service/pg_audit_logger.go` | PG 결제 감사 로거 — JSON-per-line, fsync 보장 |
| `internal/repository/donate_repo.go` | `InsertOrder`, `InsertPGData`, `UpdateOrderPayment`, `GetOrder` |
| `internal/model/payment.go` | `CreateOrderRequest`, `PaymentParams`, `PGData`, `ApproveResult` |

#### 수정 파일

| 파일 | 변경 내용 |
|------|-----------|
| `cmd/server/main.go` | `/pg/*` 라우트 등록 (CSRF 제외), `/api/donation/orders` 라우트 추가, 서비스 와이어링 |
| `internal/config/config.go` | `EasyPayConfig` 구조체 추가 |

#### 모델 정의

```go
// internal/model/payment.go

// 주문 생성 요청
type CreateOrderRequest struct {
    Amount  int    `json:"amount"`
    PayType string `json:"payType"`  // "CARD" (Phase 1)
    Gate    string `json:"gate"`     // "immediately" (Phase 1)
}

// EasyPay SDK에 전달할 파라미터
type PaymentParams struct {
    MallID      string `json:"mallId"`
    OrderNo     string `json:"orderNo"`
    ProductAmt  string `json:"productAmt"`
    ProductName string `json:"productName"`
    PayType     string `json:"payType"`      // "11", "21", "31", "81"
    ReturnURL   string `json:"returnUrl"`
    RelayURL    string `json:"relayUrl"`
    WindowType  string `json:"windowType"`   // "iframe" or "submit"
    UserName    string `json:"userName"`
    MallName    string `json:"mallName"`
    Currency    string `json:"currency"`     // "00" (KRW)
    Charset     string `json:"charset"`      // "UTF-8"
    LangFlag    string `json:"langFlag"`     // "KOR"
}

// 주문 생성 응답
type CreateOrderResponse struct {
    OrderSeq      int           `json:"orderSeq"`
    PaymentParams PaymentParams `json:"paymentParams"`
}

// 주문 조회 응답 (결과 페이지용)
type OrderDetail struct {
    OrderSeq int    `json:"orderSeq"`
    Amount   int    `json:"amount"`
    Status   string `json:"status"`  // "Y" or "N"
    PaidAt   string `json:"paidAt"`
}

// ep_cli 승인 요청
type ApproveRequest struct {
    OrderNo     string
    EncryptData string
    SessionKey  string
    TraceNo     string
    ClientIP    string
}

// ep_cli 승인 결과
type ApproveResult struct {
    ResCode      string // res_cd (0000 = 성공)
    ResMsg       string // res_msg
    CNO          string // PG 거래번호
    Amount       string // 승인 금액
    AuthNo       string // 승인번호
    TranDate     string // 거래일시
    CardNo       string // 마스킹된 카드번호
    PayType      string // 결제수단 코드 (11/21/31)
    IssuerName   string // 발급사명
    AcquirerName string // 매입사명
}

// PG 데이터 (WEO_PG_DATA 테이블)
type PGData struct {
    CNO      string
    ResCD    string
    ResMsg   string
    Amount   int
    NumCard  string
    TranDate string
    AuthNo   string
    PayType  string
    OSeq     int
}
```

#### 라우트 등록 (main.go 변경)

```go
// cmd/server/main.go — 라우터 재구성

// 1. 서비스 와이어링 추가
donateRepo := repository.NewDonateRepository(db)
easypayService := service.NewEasyPayService(cfg.EasyPay)
pgAuditLogger, err := service.NewPGAuditLogger(cfg.PGAuditLogPath)
if err != nil {
    logger.Fatal().Err(err).Msg("failed to open PG audit log")
}
defer pgAuditLogger.Close()
donateService := service.NewDonateService(donateRepo, easypayService, cacheStore, logger, pgAuditLogger)
paymentHandler := handler.NewPaymentHandler(donateService, cfg.EasyPay)

// 2. PG 콜백 라우트 — CSRF/Auth 미들웨어 없이 등록
//    ⚠️ 기존 전역 CSRF 미들웨어를 그룹 기반으로 변경 필요
mux := chi.NewMux()
mux.Use(chimw.Recoverer)
mux.Use(mw.RequestLogger(logger))
mux.Use(mw.CORSMiddleware(allowedOrigins))
mux.Use(mw.MaxBodySize(1 << 20))

// PG 콜백 라우트 — CSRF 없음, Auth 없음
mux.Route("/pg", func(r chi.Router) {
    r.Post("/easypay/relay", paymentHandler.EasyPayRelay)
    r.Post("/easypay/return", paymentHandler.EasyPayReturn)
})

// API 라우트 — CSRF 적용
mux.Group(func(r chi.Router) {
    r.Use(mw.CSRFMiddleware(allowedOrigins))

    // 공개 API (인증 불필요)
    r.Get("/api/health", healthHandler.Check)
    r.Get("/api/feed", feedHandler.GetFeed)
    r.Get("/api/donation/summary", donationHandler.GetSummary)
    // ... 기존 공개 라우트 ...

    // 인증 필수 API
    r.Group(func(r chi.Router) {
        r.Use(mw.AuthMiddleware(authService))
        r.Post("/api/donation/orders", paymentHandler.CreateOrder)
        r.Get("/api/donation/orders/{seq}", paymentHandler.GetOrder)
        // ... 기존 인증 라우트 ...
    })

    // 관리자 API
    r.Route("/api/admin", func(r chi.Router) {
        r.Use(mw.AuthMiddleware(authService))
        r.Use(mw.AdminAuthMiddleware)
        // ... 기존 관리자 라우트 ...
    })
})
```

> **⚠️ 주의:** 기존 `main.go`에서 CSRF 미들웨어가 전역으로 적용되어 있다면, `/pg/*` 라우트를 CSRF 그룹 밖에 배치하도록 리팩토링이 필요합니다.

#### Repository (donate_repo.go)

```go
// internal/repository/donate_repo.go

type DonateRepository struct {
    DB *sqlx.DB
}

func NewDonateRepository(db *sqlx.DB) *DonateRepository {
    return &DonateRepository{DB: db}
}

// InsertOrder — pending 상태 주문 생성, O_SEQ 반환
func (r *DonateRepository) InsertOrder(usrSeq int, gate string, payType string, price int, ip string) (int64, error) {
    result, err := r.DB.Exec(`
        INSERT INTO WEO_ORDER (USR_SEQ, O_GATE, O_PAY_TYPE, O_TYPE, O_REGDATE, O_PRICE, O_PAYMENT, REG_DATE, REG_IPADDR)
        VALUES (?, ?, ?, 'A', NOW(), ?, 'N', NOW(), ?)
    `, usrSeq, gate, payType, price, ip)
    if err != nil {
        return 0, err
    }
    return result.LastInsertId()
}

// InsertPGData — PG 승인 데이터 저장, PG_SEQ 반환
func (r *DonateRepository) InsertPGData(data *model.PGData) (int64, error) {
    result, err := r.DB.Exec(`
        INSERT INTO WEO_PG_DATA (CNO, RES_CD, RES_MSG, AMOUNT, NUM_CARD, TRAN_DATE, AUTH_NO, PAY_TYPE, O_SEQ)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, data.CNO, data.ResCD, data.ResMsg, data.Amount, data.NumCard, data.TranDate, data.AuthNo, data.PayType, data.OSeq)
    if err != nil {
        return 0, err
    }
    return result.LastInsertId()
}

// UpdateOrderPayment — 결제 완료 처리 (멱등: O_PAYMENT='N' 조건)
func (r *DonateRepository) UpdateOrderPayment(orderSeq int, amount int, pgSeq int64, ip string) (int64, error) {
    result, err := r.DB.Exec(`
        UPDATE WEO_ORDER
        SET O_PAYMENT = 'Y', O_PAY = ?, O_PAYDATE = NOW(), O_PG_SEQ = ?, O_STATUS = 'Y',
            EDT_DATE = NOW(), EDT_IPADDR = ?
        WHERE O_SEQ = ? AND O_PAYMENT = 'N'
    `, amount, pgSeq, ip, orderSeq)
    if err != nil {
        return 0, err
    }
    return result.RowsAffected()
}

// GetOrder — 주문 상세 조회
func (r *DonateRepository) GetOrder(orderSeq int, usrSeq int) (*model.OrderDetail, error) {
    var order model.OrderDetail
    err := r.DB.Get(&order, `
        SELECT O_SEQ AS OrderSeq, O_PRICE AS Amount, O_PAYMENT AS Status,
               IFNULL(DATE_FORMAT(O_PAYDATE, '%Y-%m-%d %H:%i:%s'), '') AS PaidAt
        FROM WEO_ORDER
        WHERE O_SEQ = ? AND USR_SEQ = ? AND O_TYPE = 'A'
    `, orderSeq, usrSeq)
    if err != nil {
        return nil, err
    }
    return &order, nil
}

// GetOrderPrice — 주문 금액 조회 (결제 검증용)
func (r *DonateRepository) GetOrderPrice(orderSeq int) (int, error) {
    var price int
    err := r.DB.Get(&price, `
        SELECT O_PRICE FROM WEO_ORDER WHERE O_SEQ = ? AND O_PAYMENT = 'N'
    `, orderSeq)
    return price, err
}
```

#### EasyPay Service (ep_cli 래퍼)

```go
// internal/service/easypay_service.go

type EasyPayService struct {
    cfg config.EasyPayConfig
}

func NewEasyPayService(cfg config.EasyPayConfig) *EasyPayService {
    return &EasyPayService{cfg: cfg}
}

// Approve — ep_cli 바이너리를 실행하여 서버 사이드 승인 처리
func (s *EasyPayService) Approve(req model.ApproveRequest) (*model.ApproveResult, error) {
    binPath := filepath.Join(s.cfg.BinBase, "immediately", "bin", "linux_64", "ep_cli")
    certFile := filepath.Join(s.cfg.BinBase, "immediately", "mobile", "cert", "pg_cert.pem")
    logDir := filepath.Join(s.cfg.BinBase, "immediately", "mobile", "log")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // ep_cli expects: -h key1=val1,key2=val2,...
    args := fmt.Sprintf(
        "order_no=%s,cert_file=%s,mall_id=%s,tr_cd=00101000,gw_url=%s,gw_port=%s,enc_data=%s,snd_key=%s,trace_no=%s,cust_ip=%s,log_dir=%s,log_level=1",
        req.OrderNo, certFile, s.cfg.ImmediatelyMallID,
        s.cfg.GatewayURL, s.cfg.GatewayPort,
        req.EncryptData, req.SessionKey, req.TraceNo, req.ClientIP, logDir,
    )

    cmd := exec.CommandContext(ctx, binPath, "-h", args)
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("ep_cli 실행 실패: %w", err)
    }

    // 응답 파싱: "res_cd=0000\x1Fres_msg=정상\x1Fcno=xxx\x1F..."
    return parseEasyPayResponse(string(output)), nil
}

// parseEasyPayResponse — \x1F (Unit Separator) 구분자로 key=value 파싱
func parseEasyPayResponse(raw string) *model.ApproveResult {
    fields := strings.Split(raw, "\x1F")
    m := make(map[string]string)
    for _, field := range fields {
        parts := strings.SplitN(field, "=", 2)
        if len(parts) == 2 {
            m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
        }
    }
    return &model.ApproveResult{
        ResCode:      m["res_cd"],
        ResMsg:       m["res_msg"],
        CNO:          m["cno"],
        Amount:       m["amount"],
        AuthNo:       m["auth_no"],
        TranDate:     m["tran_date"],
        CardNo:       m["card_no"],
        PayType:      m["pay_type"],
        IssuerName:   m["issuer_nm"],
        AcquirerName: m["acquirer_nm"],
    }
}
```

> **⚠️ 인코딩 주의:** `ep_cli` 출력이 EUC-KR일 수 있습니다. 필요 시 `golang.org/x/text/encoding/korean` 패키지로 변환합니다.

> **⚠️ 개발 환경:** `ep_cli`는 Linux 전용 바이너리로 macOS에서 실행 불가합니다. 로컬 개발 시 mock 응답을 반환하는 별도 로직이 필요합니다.

#### Donate Service (비즈니스 로직)

```go
// internal/service/donate_service.go

type DonateService struct {
    repo    *repository.DonateRepository
    cache   *cache.Cache
    epSvc   *EasyPayService
    logger  zerolog.Logger
    pgAudit *PGAuditLogger
}

func NewDonateService(
    repo *repository.DonateRepository,
    epSvc *EasyPayService,
    cacheStore *cache.Cache,
    logger zerolog.Logger,
    pgAudit *PGAuditLogger,
) *DonateService {
    return &DonateService{
        repo: repo, epSvc: epSvc, cache: cacheStore,
        logger: logger, pgAudit: pgAudit,
    }
}

// CreateOrder — pending 주문 생성, EasyPay 파라미터 반환
func (s *DonateService) CreateOrder(user *model.AuthUser, req model.CreateOrderRequest, ip string, cfg config.EasyPayConfig) (*model.CreateOrderResponse, error) {
    // 1. 검증
    if req.Amount < 10000 {
        return nil, errors.New("최소 기부 금액은 10,000원입니다")
    }
    if req.PayType != "CARD" {
        return nil, errors.New("현재 카드 결제만 지원합니다")
    }
    if req.Gate != "immediately" {
        return nil, errors.New("현재 일시후원만 지원합니다")
    }

    // 2. DB INSERT (gate "immediately" → O_GATE "S")
    orderSeq, err := s.repo.InsertOrder(user.USRSeq, "S", req.PayType, req.Amount, ip)
    if err != nil {
        return nil, fmt.Errorf("주문 생성 실패: %w", err)
    }

    // 3. EasyPay 파라미터 구성
    orderNo := strconv.FormatInt(orderSeq, 10)
    params := model.PaymentParams{
        MallID:      cfg.ImmediatelyMallID,
        OrderNo:     orderNo,
        ProductAmt:  strconv.Itoa(req.Amount),
        ProductName: fmt.Sprintf("%s님_일시후원_%s원", user.USRName, formatComma(req.Amount)),
        PayType:     "11", // 카드
        ReturnURL:   cfg.ReturnBaseURL + "/pg/easypay/return",
        RelayURL:    "/pg/easypay/relay",
        WindowType:  "iframe",
        UserName:    user.USRName,
        MallName:    "대일외국어고등학교 장학회",
        Currency:    "00",
        Charset:     "UTF-8",
        LangFlag:    "KOR",
    }

    return &model.CreateOrderResponse{
        OrderSeq:      int(orderSeq),
        PaymentParams: params,
    }, nil
}

// ConfirmPayment — ep_cli 승인 + DB 업데이트 + PG 감사 로깅
func (s *DonateService) ConfirmPayment(orderNo string, encData string, sessionKey string, traceNo string, clientIP string) error {
    orderSeq, err := strconv.Atoi(orderNo)
    if err != nil {
        return fmt.Errorf("invalid order number: %w", err)
    }

    // 1. ep_cli 승인
    result, err := s.epSvc.Approve(model.ApproveRequest{
        OrderNo: orderNo, EncryptData: encData,
        SessionKey: sessionKey, TraceNo: traceNo, ClientIP: clientIP,
    })
    if err != nil {
        s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("ep_cli approval failed")
        s.pgAudit.Log(orderNo, "approve_fail", nil, err)
        return err
    }
    if result.ResCode != "0000" {
        s.logger.Warn().Str("resCode", result.ResCode).Str("resMsg", result.ResMsg).Str("orderNo", orderNo).Msg("PG approval rejected")
        s.pgAudit.Log(orderNo, "approve_fail", result, nil)
        return fmt.Errorf("PG 승인 실패: %s (%s)", result.ResMsg, result.ResCode)
    }

    s.pgAudit.Log(orderNo, "approve_success", result, nil)

    // 2. 금액 검증 (주문 금액 vs PG 승인 금액)
    orderPrice, err := s.repo.GetOrderPrice(orderSeq)
    if err != nil {
        return fmt.Errorf("주문 조회 실패: %w", err)
    }
    approvedAmount, _ := strconv.Atoi(result.Amount)
    if orderPrice != approvedAmount {
        s.logger.Error().Int("orderPrice", orderPrice).Int("approvedAmount", approvedAmount).Str("orderNo", orderNo).Msg("amount mismatch")
        return fmt.Errorf("금액 불일치: 주문 %d, 승인 %d", orderPrice, approvedAmount)
    }

    // 3. PG 데이터 INSERT
    pgData := &model.PGData{
        CNO: result.CNO, ResCD: result.ResCode, ResMsg: result.ResMsg,
        Amount: approvedAmount, NumCard: result.CardNo, TranDate: result.TranDate,
        AuthNo: result.AuthNo, PayType: result.PayType, OSeq: orderSeq,
    }
    pgSeq, err := s.repo.InsertPGData(pgData)
    if err != nil {
        s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to insert PG data")
        s.pgAudit.Log(orderNo, "db_insert_fail", result, err)
        return fmt.Errorf("PG 데이터 저장 실패: %w", err)
    }

    // 4. 주문 결제 완료 (멱등: O_PAYMENT='N' 조건)
    affected, err := s.repo.UpdateOrderPayment(orderSeq, approvedAmount, pgSeq, clientIP)
    if err != nil {
        s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to update order payment")
        s.pgAudit.Log(orderNo, "db_update_fail", result, err)
        return fmt.Errorf("주문 업데이트 실패: %w", err)
    }
    if affected == 0 {
        s.logger.Info().Str("orderNo", orderNo).Msg("order already processed (idempotent)")
        return nil
    }

    // 5. 캐시 무효화
    s.cache.Delete("donation_summary")
    s.logger.Info().Str("orderNo", orderNo).Int("amount", approvedAmount).Msg("payment confirmed")
    return nil
}
```

> ⚠️ **NOTE (Cycle 3):** `InsertPGData` + `UpdateOrderPayment`를 DB 트랜잭션으로 묶어야 합니다. 현재 별도 실행 시 중간 실패 시 데이터 불일치 위험이 있습니다. 또한 `paymentStatus == "Y"` 멱등성 검사를 `ep_cli.Approve()` 호출 전으로 이동하여, 이미 결제된 주문에 대해 외부 PG 호출을 방지하십시오. 상세: `IMPLEMENTATION_CHECKLIST.md` IC-2 항목 참조.

#### Payment Handler

```go
// internal/handler/payment_handler.go

//go:embed templates/easypay_relay.html
var relayTemplateContent string

type PaymentHandler struct {
    donateService *service.DonateService
    easypayConfig config.EasyPayConfig
    relayTmpl     *template.Template  // go:embed로 바이너리에 포함된 relay 템플릿
}

func NewPaymentHandler(donateService *service.DonateService, cfg config.EasyPayConfig) *PaymentHandler {
    tmpl := template.Must(template.New("relay").Parse(relayTemplateContent))
    return &PaymentHandler{donateService: donateService, easypayConfig: cfg, relayTmpl: tmpl}
}

// CreateOrder — POST /api/donation/orders (인증 필수)
func (h *PaymentHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetAuthUser(r.Context())
    if user == nil {
        respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
        return
    }

    var req model.CreateOrderRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, http.StatusBadRequest, "INVALID_BODY", "잘못된 요청입니다")
        return
    }

    ip := r.Header.Get("X-Real-IP")
    if ip == "" {
        ip = r.RemoteAddr
    }

    result, err := h.donateService.CreateOrder(user, req, ip, h.easypayConfig)
    if err != nil {
        respondError(w, http.StatusBadRequest, "ORDER_FAILED", err.Error())
        return
    }
    respondJSON(w, http.StatusOK, result)
}

// EasyPayRelay — POST /pg/easypay/relay (CSRF 없음, Auth 없음)
// EasyPay SDK form POST → PG 서버로 auto-submit HTML 응답
func (h *PaymentHandler) EasyPayRelay(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    h.relayTmpl.Execute(w, r.Form)
}

// EasyPayReturn — POST /pg/easypay/return (CSRF 없음, Auth 없음)
// EasyPay 결제 완료 후 콜백 → ep_cli 승인 → DB 업데이트 → 리다이렉트
func (h *PaymentHandler) EasyPayReturn(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()

    resCode := r.FormValue("sp_res_cd")
    orderNo := r.FormValue("sp_order_no")

    // 1. PG 응답 코드 확인
    if resCode != "0000" {
        http.Redirect(w, r, "/donation/result?status=failed&reason=pg_error", http.StatusFound)
        return
    }

    // 2. ep_cli 승인 + DB 처리
    ip := r.Header.Get("X-Real-IP")
    if ip == "" {
        ip = r.RemoteAddr
    }

    err := h.donateService.ConfirmPayment(
        orderNo,
        r.FormValue("sp_encrypt_data"),
        r.FormValue("sp_sessionkey"),
        r.FormValue("sp_trace_no"),
        ip,
    )
    if err != nil {
        // 로그 기록 (결제는 PG에서 승인되었으나 DB 처리 실패 — 심각)
        http.Redirect(w, r, "/donation/result?status=failed&reason=server_error", http.StatusFound)
        return
    }

    http.Redirect(w, r, fmt.Sprintf("/donation/result?status=success&order=%s", orderNo), http.StatusFound)
}

// GetOrder — GET /api/donation/orders/{seq} (인증 필수)
func (h *PaymentHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
    user := middleware.GetAuthUser(r.Context())
    if user == nil {
        respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "로그인이 필요합니다")
        return
    }

    seq := parseIntParam(chi.URLParam(r, "seq"))
    order, err := h.donateService.GetOrder(seq, user.USRSeq)
    if err != nil {
        respondError(w, http.StatusNotFound, "NOT_FOUND", "주문을 찾을 수 없습니다")
        return
    }
    respondJSON(w, http.StatusOK, order)
}
```

#### EasyPay Relay 템플릿

```html
<!-- internal/handler/templates/easypay_relay.html (go:embed로 바이너리에 포함) -->
<!-- EasyPay SDK form POST → sp.easypay.co.kr/ep8/MainAction.do auto-submit -->
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body>
<form id="pgForm" method="post" action="https://sp.easypay.co.kr/ep8/MainAction.do">
  {{range $key, $values := .}}
    {{range $values}}
      <input type="hidden" name="{{$key}}" value="{{.}}">
    {{end}}
  {{end}}
</form>
<script>document.getElementById('pgForm').submit();</script>
</body>
</html>
```

### 16.4 프론트엔드 신규 파일

#### 파일 목록

| 신규 파일 | 용도 |
|-----------|------|
| `src/types/donate.ts` | 기부 결제 관련 TypeScript 타입 |
| `src/hooks/useDonateOrder.ts` | React Query mutation (주문 생성) |
| `src/components/donation/DonationForm.tsx` | 금액 + 결제수단 선택 + 확인 폼 |
| `src/components/donation/EasyPayBridge.tsx` | Hidden form + SDK 로딩 + 결제 호출 |
| `src/pages/DonationResultPage.tsx` | 결제 결과 (성공/실패) 화면 |

#### 수정 파일

| 파일 | 변경 내용 |
|------|-----------|
| `src/routes.tsx` | `/donation/result` 라우트 추가 |
| `src/pages/DonationPage.tsx` | `DonationForm` 컴포넌트 삽입 |

#### 타입 정의

```typescript
// src/types/donate.ts

export interface CreateOrderRequest {
  amount: number;
  payType: 'CARD';     // Phase 1
  gate: 'immediately'; // Phase 1
}

export interface PaymentParams {
  mallId: string;
  orderNo: string;
  productAmt: string;
  productName: string;
  payType: string;
  returnUrl: string;
  relayUrl: string;
  windowType: string;
  userName: string;
  mallName: string;
  currency: string;
  charset: string;
  langFlag: string;
}

export interface CreateOrderResponse {
  orderSeq: number;
  paymentParams: PaymentParams;
}

export interface OrderDetail {
  orderSeq: number;
  amount: number;
  status: 'Y' | 'N';
  paidAt: string;
}
```

#### 주문 생성 Hook

```typescript
// src/hooks/useDonateOrder.ts
import { useMutation } from '@tanstack/react-query';
import { api } from '../api/client';
import type { CreateOrderRequest, CreateOrderResponse } from '../types/donate';

export function useDonateOrder() {
  return useMutation({
    mutationFn: (req: CreateOrderRequest) =>
      api.post<CreateOrderResponse>('/api/donation/orders', req),
  });
}
```

#### DonationForm 컴포넌트

기존 `DonationPage.tsx`의 기부 현황 배너 아래에 삽입됩니다. Progressive Disclosure 패턴으로 각 단계를 순차적으로 표시합니다.

```
┌──────────────────────────────┐
│  누적 기부액: 1,234만원        │  ← 기존 기부 현황 (소셜 프루프)
│  ████████░░  67%  45명 참여   │
├──────────────────────────────┤
│  기부하기                      │  ← DonationForm
│                              │
│  기부방식                      │
│  [일시후원]  [월정기후원(준비중)]│
│                              │
│  기부금액                      │
│  [1만] [3만] [5만] [10만]     │  ← 2-col mobile, 4-col desktop
│  [30만] [50만] [100만] [200만] │
│  [직접 입력: ________원]       │  ← inputMode="numeric"
│                              │
│  결제수단                      │
│  ● 신용카드(체크카드)           │  ← Phase 1: 카드만 선택 가능
│  ○ 계좌이체 (준비중)           │
│  ○ 휴대폰 (준비중)            │
│                              │
│  ┌─ 기부내역 확인 ──────────┐ │  ← 모든 선택 완료 후 표시
│  │ 일시후원 / 50,000원 / 카드 │ │
│  └──────────────────────────┘ │
│  [ 기부하기 ]                 │  ← Button size="lg", shadow-primary-glow
├──────────────────────────────┤
│  기부 계좌 안내                │  ← 기존 계좌 정보 (폴백)
│  하나은행 393-910033-69205     │
└──────────────────────────────┘
```

**UX 규칙:**
- 금액 프리셋: `[10000, 30000, 50000, 100000, 300000, 500000, 1000000, 2000000]`
- 직접입력: `inputMode="numeric"`, 숫자만 허용, 콤마 포맷 표시
- 검증: 인라인 에러 메시지 (빨간 테두리 + 하단 텍스트), `alert()` 사용 금지
- 모바일: 하단 고정 CTA 버튼 (sticky bottom)
- 로그인 필수: 미로그인 시 "기부하기" 버튼 대신 "로그인 후 기부하기" 버튼 → `/login`으로 이동

#### EasyPayBridge 컴포넌트

주문 생성 성공 후 렌더링됩니다. SDK를 동적 로딩하고 `easypay_card_webpay()`를 호출합니다.

```tsx
// src/components/donation/EasyPayBridge.tsx
function EasyPayBridge({ params }: { params: PaymentParams }) {
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    // EasyPay SDK 동적 로딩
    const script = document.createElement('script');
    script.src = 'https://sp.easypay.co.kr/webpay/EasypayCard_Web.js';
    script.onload = () => {
      // SDK 호출: form, relayUrl, target, reserved1, reserved2, windowType, timeout
      window.easypay_card_webpay(
        formRef.current, params.relayUrl, "_self", "0", "0", params.windowType, 30
      );
    };
    document.head.appendChild(script);
    return () => document.head.removeChild(script);
  }, [params]);

  return (
    <form ref={formRef} name="payment" style={{ display: 'none' }}>
      <input type="hidden" name="sp_mall_id" value={params.mallId} />
      <input type="hidden" name="sp_order_no" value={params.orderNo} />
      <input type="hidden" name="sp_product_amt" value={params.productAmt} />
      <input type="hidden" name="sp_product_nm" value={params.productName} />
      <input type="hidden" name="sp_pay_type" value={params.payType} />
      <input type="hidden" name="sp_return_url" value={params.returnUrl} />
      <input type="hidden" name="sp_currency" value={params.currency} />
      <input type="hidden" name="sp_charset" value={params.charset} />
      <input type="hidden" name="sp_lang_flag" value={params.langFlag} />
      <input type="hidden" name="sp_user_nm" value={params.userName} />
      <input type="hidden" name="sp_mall_nm" value={params.mallName} />
      <input type="hidden" name="sp_window_type" value={params.windowType} />
      {/* EasyPay SDK가 채우는 응답 필드 */}
      <input type="hidden" name="sp_res_cd" value="" />
      <input type="hidden" name="sp_res_msg" value="" />
      <input type="hidden" name="sp_tr_cd" value="" />
      <input type="hidden" name="sp_sessionkey" value="" />
      <input type="hidden" name="sp_encrypt_data" value="" />
      <input type="hidden" name="sp_trace_no" value="" />
      <input type="hidden" name="sp_ret_pay_type" />
      <input type="hidden" name="sp_card_code" />
      <input type="hidden" name="sp_card_req_type" />
    </form>
  );
}
```

> **React vDOM 충돌 방지:** `EasyPayBridge`는 비제어 `ref`를 사용하며, SDK는 DOM을 직접 조작합니다. React의 가상 DOM과의 충돌을 피하기 위해 `style={{ display: 'none' }}`으로 숨기고, cleanup에서 script를 제거합니다.

#### DonationResultPage

라우트: `/donation/result?status=success|failed&order=X`

```
성공 시:
┌──────────────────────────────┐
│        ✅                    │
│  기부해 주셔서 감사합니다      │
│                              │
│  기부 금액: 50,000원          │
│  결제일시: 2026-02-12 14:30   │
│                              │
│  ※ 기부금은 연말정산 시       │
│    소득공제 혜택을 받을 수     │
│    있습니다.                  │
│                              │
│  [홈으로]  [기부 내역 보기]    │
└──────────────────────────────┘

실패 시:
┌──────────────────────────────┐
│        ❌                    │
│  결제에 실패했습니다           │
│                              │
│  사유: PG 결제 오류           │
│                              │
│  문의: 02-543-3558           │
│                              │
│  [다시 시도]                  │
└──────────────────────────────┘
```

### 16.5 DB 테이블

**Phase 1에서 신규 마이그레이션 불필요.** `WEO_ORDER`, `WEO_PG_DATA` 테이블은 V1에서 이미 존재합니다.

#### WEO_ORDER (기존 테이블, 결제 관련 컬럼)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `O_SEQ` | INT AUTO_INCREMENT | 주문 번호 (PK) |
| `USR_SEQ` | INT | 사용자 번호 |
| `O_GATE` | VARCHAR | 결제 방식 (S=일시, P=정기, F=프로젝트) |
| `O_PAY_TYPE` | VARCHAR | 결제 수단 (CARD, BANK, HP) |
| `O_TYPE` | CHAR(1) | 주문 유형 (A=기부) |
| `O_REGDATE` | DATETIME | 등록일시 |
| `O_PRICE` | INT | 주문 금액 |
| `O_PAYMENT` | CHAR(1) | 결제 완료 여부 (Y/N) |
| `O_PAY` | INT | 실 결제 금액 |
| `O_PAYDATE` | DATETIME | 결제 완료일시 |
| `O_PG_SEQ` | INT | PG 데이터 번호 (FK → WEO_PG_DATA) |
| `O_STATUS` | CHAR(1) | 주문 상태 (Y=완료) |
| `P_SEQ` | INT | 프로젝트 번호 (Phase 3) |

#### WEO_PG_DATA (기존 테이블)

| 컬럼 | 타입 | 설명 |
|------|------|------|
| `SEQ` | INT AUTO_INCREMENT | PG 데이터 번호 (PK) |
| `CNO` | VARCHAR | PG 거래번호 |
| `RES_CD` | VARCHAR | 응답 코드 |
| `RES_MSG` | VARCHAR | 응답 메시지 |
| `AMOUNT` | INT | 승인 금액 |
| `NUM_CARD` | VARCHAR | 마스킹된 카드번호 |
| `TRAN_DATE` | VARCHAR | 거래일시 |
| `AUTH_NO` | VARCHAR | 승인번호 |
| `PAY_TYPE` | VARCHAR | 결제수단 코드 (11/21/31) |
| `O_SEQ` | INT | 주문 번호 (FK → WEO_ORDER) |

### 16.6 보안 체크리스트

| 항목 | 방어 방법 |
|------|-----------|
| **SQL Injection** | sqlx `?` 플레이스홀더 사용 (V1의 문자열 연결 방식 복제 금지) |
| **CSRF** | `/pg/*`는 제외 (EasyPay cross-origin POST), `/api/*`는 CSRF 유지 |
| **멱등성** | `UPDATE ... WHERE O_PAYMENT='N'` — 중복 결제 방지 |
| **금액 검증** | `ep_cli` 응답 금액과 `WEO_ORDER.O_PRICE` 비교, 불일치 시 거부 |
| **카드정보 미저장** | EasyPay의 CNO 참조만 저장, 카드번호 원본은 절대 저장 금지 |
| **타임아웃** | `ep_cli` 호출에 30초 `context.WithTimeout` 적용 |
| **인증** | 주문 생성(`/api/donation/orders`)은 AuthMiddleware 필수 |
| **주문 소유권** | `GetOrder`에서 `USR_SEQ` 조건 — 타인의 주문 조회 불가 |

### 16.7 Phase 2: 월정기후원 (정기결제)

| 항목 | 내용 |
|------|------|
| Mall ID | `05543499` |
| sp_pay_type | `81` |
| sp_window_type | `submit` |
| 콜백 | `POST /pg/easypay/profile/return` |
| DB | `WEO_ORDER_PROFILE` (billing key `cno`, `OP_DAY`, `OP_AMOUNT`) |
| 배치 | `internal/job/billing_batch.go` (V1 `_profile_batch.php` 이식) |

**주요 고려사항:**
- 빌링키(`cno`)를 **암호화하여 저장** (V1은 평문 — 복제 금지)
- 월 최대 100,000원 제한
- V1+V2 동시 배치 실행 시 이중 청구 방지를 위한 **명확한 전환 계획** 필요

### 16.8 Phase 3: 추가 결제수단 + 프로젝트 기부

- 계좌이체 (`sp_pay_type=21`)
- 휴대폰 (`sp_pay_type=31`)
- 프로젝트별 기부 (`O_GATE='F'`, `P_SEQ` 연결)
- DonationForm에 결제수단 선택 활성화 + 프로젝트 선택 드롭다운 추가

### 16.9 PG 감사 로거 (PGAuditLogger)

PG 승인 후 DB 실패 시 금전 대사(reconciliation)를 위해 **전용 감사 로그**를 기록합니다.

#### 파일: `internal/service/pg_audit_logger.go`

```go
type PGAuditLogger struct {
    mu   sync.Mutex
    file *os.File
}

type PGAuditEntry struct {
    Timestamp string      `json:"ts"`
    OrderNo   string      `json:"order_no"`
    Event     string      `json:"event"`      // "approve_success", "approve_fail", "db_insert_fail", "db_update_fail"
    RawData   interface{} `json:"data"`
    Error     string      `json:"error,omitempty"`
}
```

- **포맷:** JSON-per-line, 매 write 후 `fsync` (durability 보장)
- **기록 시점:** approve 성공 (항상), approve 실패, DB INSERT 실패, DB UPDATE 실패
- **파일 경로:** `PG_AUDIT_LOG_PATH` 환경변수 (기본값: `/var/logs/pg/pg-audit.log`)
- **logrotate:** `deploy/pg-audit.logrotate` — 별도 디렉토리, 30일 보관
- **와이어링:** `main.go`에서 `NewPGAuditLogger(cfg.PGAuditLogPath)` → `NewDonateService(..., pgAudit)` 전달

### 16.10 리스크 및 대응

| 리스크 | 심각도 | 대응 |
|--------|--------|------|
| `ep_cli` macOS 실행 불가 | 🔴 높 | Go 테스트에서 mock 응답, Linux 스테이징에서 통합 테스트 |
| EasyPay SDK + React vDOM 충돌 | 🟡 중 | 비제어 `ref`, 동적 SDK 로딩, `display: none` 격리 |
| `ep_cli` 출력 EUC-KR 인코딩 | 🟡 중 | `golang.org/x/text/encoding/korean`으로 변환 (필요 시) |
| PG 승인 후 DB 실패 | 🟡 중 | `PGAuditLogger`로 전체 PG 응답 JSON 기록 + 관리자 수동 대사 |
| V1+V2 WEO_ORDER 동시 쓰기 | 🟢 낮 | AUTO_INCREMENT로 충돌 없음. Phase 2 배치는 전환 계획 필요 |
| 1-core 1GB 서버 리소스 | 🟢 낮 | 결제 빈도 낮음, `ep_cli` 실행 시간 짧음, 메모리 모니터링 |

### 16.11 Phase 1 검증 계획

1. **단위 테스트 (Go):** ep_cli mock, CreateOrder 검증, 응답 파싱 테스트
2. **EasyPay 샌드박스:** SDK `testsp.easypay.co.kr`, 게이트웨이 `testgw.easypay.co.kr:80`
3. **통합 테스트 (Linux 스테이징):**
   - 로그인 → `/donation` → 10,000원 → 신용카드 → 기부하기
   - EasyPay 열림 → 테스트 결제 완료 → 성공 화면
   - DB 확인: `WEO_ORDER.O_PAYMENT='Y'`, `WEO_PG_DATA` 생성
   - 캐시 만료 후 기부 현황 업데이트 확인
4. **엣지 케이스:**
   - 결제 중 브라우저 종료 → 주문 pending 유지
   - EasyPay 에러 → 실패 화면, 주문 `O_PAYMENT='N'`
   - 이중 콜백 → 멱등 처리 (중복 결제 없음)
   - 미로그인 → 로그인 페이지로 리다이렉트

---
