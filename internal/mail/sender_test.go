package mail

import (
	"os"
	"strings"
	"testing"
)

// mockSender records calls to Send.
type mockSender struct {
	to      string
	subject string
	body    string
	calls   int
}

func (m *mockSender) Send(to, subject, htmlBody string) error {
	m.to = to
	m.subject = subject
	m.body = htmlBody
	m.calls++
	return nil
}

func TestLogSender(t *testing.T) {
	s := LogSender{}
	err := s.Send("test@example.com", "Test Subject", "<h1>Test</h1>")
	if err != nil {
		t.Errorf("expected no error from LogSender.Send, got %v", err)
	}
}

func TestNewSender_DefaultToLogSender(t *testing.T) {
	os.Unsetenv("SMTP_HOST")

	s := New()
	if _, ok := s.(LogSender); !ok {
		t.Errorf("expected LogSender when SMTP_HOST is empty, got %T", s)
	}
}

func TestNewSender_SMTPSender(t *testing.T) {
	os.Setenv("SMTP_HOST", "smtp.test.dev")
	os.Setenv("SMTP_PORT", "25")
	os.Setenv("SMTP_USERNAME", "myuser")
	os.Setenv("SMTP_PASSWORD", "mypass")
	os.Setenv("SMTP_FROM", "test@entsaas.dev")

	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("SMTP_FROM")
	}()

	s := New()
	smtpS, ok := s.(*SMTPSender)
	if !ok {
		t.Fatalf("expected *SMTPSender when SMTP_HOST is set, got %T", s)
	}

	if smtpS.Host != "smtp.test.dev" {
		t.Errorf("expected Host to be smtp.test.dev, got %q", smtpS.Host)
	}
	if smtpS.Port != "25" {
		t.Errorf("expected Port to be 25, got %q", smtpS.Port)
	}
	if smtpS.Username != "myuser" {
		t.Errorf("expected Username to be myuser, got %q", smtpS.Username)
	}
	if smtpS.Password != "mypass" {
		t.Errorf("expected Password to be mypass, got %q", smtpS.Password)
	}
	if smtpS.From != "test@entsaas.dev" {
		t.Errorf("expected From to be test@entsaas.dev, got %q", smtpS.From)
	}
}

func TestSendWelcome(t *testing.T) {
	m := &mockSender{}
	data := WelcomeData{
		AppName:  "SaaSify",
		AppURL:   "https://saasify.dev",
		UserName: "Alice",
	}

	err := SendWelcome(m, "alice@test.com", data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.calls != 1 {
		t.Errorf("expected 1 call to Send, got %d", m.calls)
	}
	if m.to != "alice@test.com" {
		t.Errorf("expected to to be alice@test.com, got %q", m.to)
	}
	if m.subject != "Welcome to SaaSify" {
		t.Errorf("expected subject to be 'Welcome to SaaSify', got %q", m.subject)
	}

	// Verify template placeholder injection
	if !strings.Contains(m.body, "Alice") {
		t.Error("expected body to contain user name 'Alice'")
	}
	if !strings.Contains(m.body, "https://saasify.dev/login") {
		t.Error("expected body to contain app url '/login'")
	}
}

func TestSendPasswordReset(t *testing.T) {
	m := &mockSender{}
	data := PasswordResetData{
		AppName:   "SaaSify",
		AppURL:    "https://saasify.dev",
		ResetURL:  "https://saasify.dev/reset?token=123",
		ExpiresIn: "30 minutes",
	}

	err := SendPasswordReset(m, "bob@test.com", data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.calls != 1 {
		t.Errorf("expected 1 call, got %d", m.calls)
	}
	if m.subject != "Reset your SaaSify password" {
		t.Errorf("expected subject 'Reset your SaaSify password', got %q", m.subject)
	}
	if !strings.Contains(m.body, "https://saasify.dev/reset?token=123") {
		t.Error("expected body to contain ResetURL")
	}
	if !strings.Contains(m.body, "30 minutes") {
		t.Error("expected body to contain expires in time")
	}
}

func TestSendInvite(t *testing.T) {
	m := &mockSender{}
	data := InviteData{
		AppName:      "SaaSify",
		AppURL:       "https://saasify.dev",
		InviteURL:    "https://saasify.dev/invite/accept?token=999",
		OrgName:      "Acme Corp",
		InviterEmail: "owner@acme.com",
		ExpiresIn:    "7 days",
	}

	err := SendInvite(m, "new-member@gmail.com", data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.calls != 1 {
		t.Errorf("expected 1 call, got %d", m.calls)
	}
	if m.subject != "You've been invited to join Acme Corp on SaaSify" {
		t.Errorf("expected invite subject, got %q", m.subject)
	}
	if !strings.Contains(m.body, "owner@acme.com") {
		t.Error("expected body to show inviter email")
	}
	if !strings.Contains(m.body, "Acme Corp") {
		t.Error("expected body to show org name")
	}
	if !strings.Contains(m.body, "https://saasify.dev/invite/accept?token=999") {
		t.Error("expected body to show invite URL")
	}
}
