package service

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

func TestEmailServiceSendHasFiniteSMTPDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	service := newEmailServiceWithTimeout(config.SMTPConfig{
		Host: host,
		Port: port,
		From: "sender@example.test",
	}, zerolog.Nop(), 50*time.Millisecond)

	started := time.Now()
	err = service.Send(model.EmailMessage{
		To:      "receiver@example.test",
		Subject: "deadline contract",
		Body:    "body",
	})
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("SMTP hang returned success")
	}
	if elapsed >= time.Second {
		t.Fatalf("SMTP send exceeded finite deadline: %s", elapsed)
	}

	select {
	case conn := <-accepted:
		conn.Close()
	case <-time.After(time.Second):
		t.Fatal("SMTP fixture did not accept connection")
	}
}

func TestEmailServiceDoesNotRetryAfterSMTPAcceptedData(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverDone <- acceptErr
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(time.Second))
		reader := bufio.NewReader(conn)
		if _, writeErr := fmt.Fprint(conn, "220 fixture ESMTP\r\n"); writeErr != nil {
			serverDone <- writeErr
			return
		}
		inData := false
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				serverDone <- readErr
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if inData {
				if line == "." {
					if _, writeErr := fmt.Fprint(conn, "250 accepted\r\n"); writeErr != nil {
						serverDone <- writeErr
						return
					}
					inData = false
				}
				continue
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				fmt.Fprint(conn, "250 fixture\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"):
				fmt.Fprint(conn, "250 sender ok\r\n")
			case strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(conn, "250 recipient ok\r\n")
			case line == "DATA":
				fmt.Fprint(conn, "354 continue\r\n")
				inData = true
			case line == "QUIT":
				// Simulate transport ambiguity after the server accepted DATA.
				time.Sleep(200 * time.Millisecond)
				serverDone <- nil
				return
			default:
				serverDone <- fmt.Errorf("unexpected SMTP command %q", line)
				return
			}
		}
	}()

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	service := newEmailServiceWithTimeout(config.SMTPConfig{
		Host: host,
		Port: port,
		From: "sender@example.test",
	}, zerolog.Nop(), 100*time.Millisecond)
	err = service.Send(model.EmailMessage{
		To:      "receiver@example.test",
		Subject: "accepted data contract",
		Body:    "body",
	})
	if err != nil {
		t.Fatalf("post-accept QUIT ambiguity must not trigger a retry: %v", err)
	}
	if serverErr := <-serverDone; serverErr != nil {
		t.Fatal(serverErr)
	}
}
