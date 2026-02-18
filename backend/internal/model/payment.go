// payment.go — Domain models for donation payment and EasyPay integration
package model

// CreateOrderRequest is the request body for POST /api/donation/orders.
type CreateOrderRequest struct {
	Amount  int    `json:"amount"`
	PayType string `json:"payType"`
	Gate    string `json:"gate"`
}

// PaymentParams holds EasyPay SDK form parameters returned to the frontend.
type PaymentParams struct {
	MallID      string `json:"mallId"`
	OrderNo     string `json:"orderNo"`
	ProductAmt  string `json:"productAmt"`
	ProductName string `json:"productName"`
	PayType     string `json:"payType"`
	ReturnURL   string `json:"returnUrl"`
	RelayURL    string `json:"relayUrl"`
	WindowType  string `json:"windowType"`
	UserName    string `json:"userName"`
	MallName    string `json:"mallName"`
	Currency    string `json:"currency"`
	Charset     string `json:"charset"`
	LangFlag    string `json:"langFlag"`
}

// CreateOrderResponse is the response for POST /api/donation/orders.
type CreateOrderResponse struct {
	OrderSeq      int            `json:"orderSeq"`
	PaymentParams *PaymentParams `json:"paymentParams"`
}

// OrderDetail is the response for GET /api/donation/orders/{seq}.
type OrderDetail struct {
	OrderSeq int    `json:"orderSeq" db:"OrderSeq"`
	Amount   int    `json:"amount" db:"Amount"`
	Status   string `json:"status" db:"Status"`
	PaidAt   string `json:"paidAt" db:"PaidAt"`
}

// ApproveRequest holds parameters for calling the ep_cli binary.
type ApproveRequest struct {
	OrderNo     string
	EncryptData string
	SessionKey  string
	TraceNo     string
	ClientIP    string
}

// ApproveResult holds parsed output from the ep_cli binary.
type ApproveResult struct {
	ResCode      string
	ResMsg       string
	CNO          string
	Amount       string
	AuthNo       string
	TranDate     string
	CardNo       string
	PayType      string
	IssuerName   string
	AcquirerName string
}

// PGData represents a row to insert into WEO_PG_DATA.
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
