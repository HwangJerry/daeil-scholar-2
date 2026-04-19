// easypay_service.go — EasyPay PG gateway approval via ep_cli binary execution
package service

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/encoding/korean"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
)

type EasyPayService struct {
	cfg config.EasyPayConfig
}

func NewEasyPayService(cfg config.EasyPayConfig) *EasyPayService {
	return &EasyPayService{cfg: cfg}
}

func (s *EasyPayService) Approve(req model.ApproveRequest, gate string) (*model.ApproveResult, error) {
	binPath := filepath.Join(s.cfg.BinBase, gate, "bin", "linux_64", "ep_cli")
	certFile := filepath.Join(s.cfg.BinBase, gate, "cert", "pg_cert.pem")
	logDir := filepath.Join(s.cfg.BinBase, gate, "log")

	mallID := s.cfg.ImmediatelyMallID
	if gate == "profile" {
		mallID = s.cfg.ProfileMallID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := fmt.Sprintf(
		"order_no=%s,cert_file=%s,mall_id=%s,tr_cd=00101000,gw_url=%s,gw_port=%s,enc_data=%s,snd_key=%s,trace_no=%s,cust_ip=%s,log_dir=%s,log_level=1,opt=utf-8",
		req.OrderNo, certFile, mallID,
		s.cfg.GatewayURL, s.cfg.GatewayPort,
		req.EncryptData, req.SessionKey, req.TraceNo, req.ClientIP, logDir,
	)

	cmd := exec.CommandContext(ctx, binPath, "-h", args)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		return nil, fmt.Errorf("ep_cli execution failed: %w (stderr: %s)", err, stderr)
	}

	// ep_cli outputs Korean text fields in EUC-KR; decode to UTF-8
	utf8Output, decErr := korean.EUCKR.NewDecoder().Bytes(output)
	if decErr != nil {
		// Fallback: use raw output (ASCII fields like res_cd still work)
		return parseEasyPayResponse(string(output)), nil
	}
	return parseEasyPayResponse(string(utf8Output)), nil
}

func parseEasyPayResponse(raw string) *model.ApproveResult {
	fields := strings.Split(raw, "\x1F")
	m := make(map[string]string, len(fields))
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
