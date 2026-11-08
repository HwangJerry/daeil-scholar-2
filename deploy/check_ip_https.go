// check_ip_https verifies the served mobile IP certificate and backend health.
package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

const certificatePath = "/etc/alumni-ip-tls/certificates/211.169.249.240.crt"
const minimumValidity = 48 * time.Hour

func verify() error {
	content, err := os.ReadFile(certificatePath)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(content)
	if block == nil {
		return fmt.Errorf("certificate PEM is missing")
	}
	expected, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		// The hosting network does not support connecting to its own public IP.
		// Keep the URL's IP identity and normal certificate verification intact.
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, "127.0.0.1:443")
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return fmt.Errorf("unexpected redirect from IP health endpoint")
	}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://211.169.249.240/api/health", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.TLS == nil {
		return fmt.Errorf("health returned %d", response.StatusCode)
	}
	actual := response.TLS.PeerCertificates[0]
	if sha256.Sum256(actual.Raw) != sha256.Sum256(expected.Raw) {
		return fmt.Errorf("Apache is not serving the current certificate")
	}
	if time.Until(actual.NotAfter) < minimumValidity {
		return fmt.Errorf("IP certificate has less than 48 hours remaining")
	}
	var health struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		return err
	}
	if health.Status != "ok" {
		return fmt.Errorf("backend status is %q", health.Status)
	}
	fmt.Printf("IP HTTPS verified; certificate expires %s\n", actual.NotAfter.UTC().Format(time.RFC3339))
	return nil
}

func main() {
	if err := verify(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
