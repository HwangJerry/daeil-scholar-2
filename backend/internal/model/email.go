// email.go — Email message model for SMTP delivery
package model

// EmailMessage represents an outgoing email to be sent via the SMTP worker.
type EmailMessage struct {
	To      string
	Subject string
	Body    string // HTML content
}
