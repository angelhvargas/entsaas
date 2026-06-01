package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// Sender is the mail sending interface. Swap implementations per environment.
type Sender interface {
	Send(to, subject, htmlBody string) error
}

// ── Log Sender (dev/test) ──────────────────────────────────────────────────

// LogSender prints emails to the structured logger. Used in development.
type LogSender struct{}

func (LogSender) Send(to, subject, htmlBody string) error {
	log.Info().Str("to", to).Str("subject", subject).Msg("📧 [mail:log] email sent")
	return nil
}

// ── SMTP Sender (prod) ─────────────────────────────────────────────────────

// SMTPSender sends real emails via SMTP (works with SendGrid, Postmark, Mailgun SMTP relay).
type SMTPSender struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// sanitizeHeader strips CR and LF characters from SMTP header values
// to prevent header injection attacks (SEC-04).
func sanitizeHeader(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return v
}

func (s *SMTPSender) Send(to, subject, htmlBody string) error {
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		sanitizeHeader(s.From), sanitizeHeader(to), sanitizeHeader(subject), htmlBody,
	))
	return smtp.SendMail(s.Host+":"+s.Port, auth, s.From, []string{sanitizeHeader(to)}, msg)
}

// ── Factory ────────────────────────────────────────────────────────────────

// New returns a Sender based on environment. If SMTP_HOST is set, uses SMTP;
// otherwise falls back to LogSender. Safe to call at startup.
func New() Sender {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		log.Info().Msg("mail: no SMTP_HOST set — using log sender (dev mode)")
		return LogSender{}
	}
	return &SMTPSender{
		Host:     host,
		Port:     envOrDefault("SMTP_PORT", "587"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     envOrDefault("SMTP_FROM", "noreply@entsaas.dev"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ── Template helpers ───────────────────────────────────────────────────────

// render parses a Go HTML template string and executes it with data.
func render(tmplStr string, data any) (string, error) {
	t, err := template.New("mail").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("mail: template parse error: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("mail: template execute error: %w", err)
	}
	return buf.String(), nil
}
