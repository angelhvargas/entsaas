package mail

import "fmt"

// ── Welcome ────────────────────────────────────────────────────────────────

type WelcomeData struct {
	AppName  string
	AppURL   string
	UserName string // email or display name
}

func SendWelcome(s Sender, to string, d WelcomeData) error {
	body, err := render(welcomeHTML, d)
	if err != nil {
		return err
	}
	return s.Send(to, fmt.Sprintf("Welcome to %s", d.AppName), body)
}

const welcomeHTML = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Welcome to {{.AppName}}</title></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:Inter,system-ui,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 0;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0"
        style="background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,.08);">
        <!-- Header -->
        <tr><td style="background:#6366f1;padding:32px 40px;text-align:center;">
          <h1 style="margin:0;color:#fff;font-size:24px;font-weight:700;letter-spacing:-0.02em;">{{.AppName}}</h1>
        </td></tr>
        <!-- Body -->
        <tr><td style="padding:40px;">
          <h2 style="margin:0 0 12px;font-size:20px;font-weight:600;color:#1a1a2e;">Welcome aboard! 🎉</h2>
          <p style="margin:0 0 24px;color:#6b7280;line-height:1.6;">
            Hi {{.UserName}}, your account is ready. You can now sign in and start building.
          </p>
          <a href="{{.AppURL}}/login"
            style="display:inline-block;padding:12px 28px;background:#6366f1;color:#fff;
                   border-radius:8px;font-weight:600;text-decoration:none;font-size:15px;">
            Sign in to {{.AppName}}
          </a>
        </td></tr>
        <!-- Footer -->
        <tr><td style="padding:24px 40px;border-top:1px solid #e5e7eb;text-align:center;">
          <p style="margin:0;font-size:12px;color:#9ca3af;">
            You received this because an account was created with this email address.<br>
            If this wasn't you, you can safely ignore this email.
          </p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`

// ── Password Reset ─────────────────────────────────────────────────────────

type PasswordResetData struct {
	AppName   string
	AppURL    string
	ResetURL  string
	ExpiresIn string // e.g. "1 hour"
}

func SendPasswordReset(s Sender, to string, d PasswordResetData) error {
	body, err := render(passwordResetHTML, d)
	if err != nil {
		return err
	}
	return s.Send(to, fmt.Sprintf("Reset your %s password", d.AppName), body)
}

const passwordResetHTML = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Reset your password</title></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:Inter,system-ui,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 0;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0"
        style="background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,.08);">
        <tr><td style="background:#6366f1;padding:32px 40px;text-align:center;">
          <h1 style="margin:0;color:#fff;font-size:24px;font-weight:700;letter-spacing:-0.02em;">{{.AppName}}</h1>
        </td></tr>
        <tr><td style="padding:40px;">
          <h2 style="margin:0 0 12px;font-size:20px;font-weight:600;color:#1a1a2e;">Password reset request</h2>
          <p style="margin:0 0 24px;color:#6b7280;line-height:1.6;">
            We received a request to reset your password. Click the button below — this link expires in {{.ExpiresIn}}.
          </p>
          <a href="{{.ResetURL}}"
            style="display:inline-block;padding:12px 28px;background:#6366f1;color:#fff;
                   border-radius:8px;font-weight:600;text-decoration:none;font-size:15px;">
            Reset password
          </a>
          <p style="margin:24px 0 0;font-size:13px;color:#9ca3af;">
            Or copy this link: <a href="{{.ResetURL}}" style="color:#6366f1;">{{.ResetURL}}</a>
          </p>
        </td></tr>
        <tr><td style="padding:24px 40px;border-top:1px solid #e5e7eb;text-align:center;">
          <p style="margin:0;font-size:12px;color:#9ca3af;">
            If you didn't request a password reset, you can ignore this email — your password won't change.
          </p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`

// ── Invite ─────────────────────────────────────────────────────────────────

type InviteData struct {
	AppName      string
	AppURL       string
	InviteURL    string
	OrgName      string
	InviterEmail string
	ExpiresIn    string
}

func SendInvite(s Sender, to string, d InviteData) error {
	body, err := render(inviteHTML, d)
	if err != nil {
		return err
	}
	return s.Send(to, fmt.Sprintf("You've been invited to join %s on %s", d.OrgName, d.AppName), body)
}

const inviteHTML = `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>You've been invited</title></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:Inter,system-ui,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 0;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0"
        style="background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 24px rgba(0,0,0,.08);">
        <tr><td style="background:#6366f1;padding:32px 40px;text-align:center;">
          <h1 style="margin:0;color:#fff;font-size:24px;font-weight:700;letter-spacing:-0.02em;">{{.AppName}}</h1>
        </td></tr>
        <tr><td style="padding:40px;">
          <h2 style="margin:0 0 12px;font-size:20px;font-weight:600;color:#1a1a2e;">You've been invited 🤝</h2>
          <p style="margin:0 0 24px;color:#6b7280;line-height:1.6;">
            <strong>{{.InviterEmail}}</strong> has invited you to join <strong>{{.OrgName}}</strong> on {{.AppName}}.
            This invite expires in {{.ExpiresIn}}.
          </p>
          <a href="{{.InviteURL}}"
            style="display:inline-block;padding:12px 28px;background:#6366f1;color:#fff;
                   border-radius:8px;font-weight:600;text-decoration:none;font-size:15px;">
            Accept invitation
          </a>
        </td></tr>
        <tr><td style="padding:24px 40px;border-top:1px solid #e5e7eb;text-align:center;">
          <p style="margin:0;font-size:12px;color:#9ca3af;">
            If you don't want to join, you can ignore this email. The invite will expire automatically.
          </p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
