// Package mailer renders and sends transactional emails over SMTP. In
// development this targets Mailhog; in production a real relay. The payload
// types double as the JSON schema for the corresponding outbox topics, so the
// worker can unmarshal a claimed message straight into them.
package mailer

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/kabanos/backend/internal/config"
)

// VerificationPayload is the body of an outbox `email.verification` message.
type VerificationPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
	Link string `json:"link"`
}

// PasswordResetPayload is the body of an outbox `email.password_reset` message.
type PasswordResetPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
	Link string `json:"link"`
}

type Mailer struct {
	cfg config.Mailer
}

func New(cfg config.Mailer) *Mailer { return &Mailer{cfg: cfg} }

func (m *Mailer) SendVerification(ctx context.Context, p VerificationPayload) error {
	body := renderTemplate("Подтвердите ваш email", greeting(p.Name),
		"Чтобы завершить регистрацию в Kabanos, подтвердите адрес электронной почты:",
		"Подтвердить email", p.Link,
		"Если вы не создавали аккаунт, просто проигнорируйте это письмо.")
	return m.send(ctx, p.To, "Kabanos — подтверждение email", body)
}

func (m *Mailer) SendPasswordReset(ctx context.Context, p PasswordResetPayload) error {
	body := renderTemplate("Сброс пароля", greeting(p.Name),
		"Мы получили запрос на сброc пароля. Нажмите кнопку ниже, чтобы задать новый пароль (ссылка действует 1 час):",
		"Сбросить пароль", p.Link,
		"Если вы не запрашивали сброс, ваш пароль остаётся без изменений.")
	return m.send(ctx, p.To, "Kabanos — сброс пароля", body)
}

func greeting(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Здравствуйте!"
	}
	return "Здравствуйте, " + name + "!"
}

// send delivers a multipart-free HTML email. It uses PlainAuth only when
// credentials are configured (Mailhog needs none).
func (m *Mailer) send(ctx context.Context, to, subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	from := fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromEmail)

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from)
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	// net/smtp has no context support; enforce the deadline with a goroutine so
	// a hung relay cannot block the worker indefinitely.
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.cfg.FromEmail, []string{to}, msg.Bytes())
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func renderTemplate(title, greeting, intro, cta, link, footer string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="ru"><body style="margin:0;background:#0f172a;font-family:Segoe UI,Roboto,Helvetica,Arial,sans-serif;">
  <div style="max-width:480px;margin:0 auto;padding:32px 24px;color:#e2e8f0;">
    <h1 style="font-size:22px;color:#38bdf8;margin:0 0 8px;">Kabanos</h1>
    <h2 style="font-size:18px;margin:16px 0 8px;">%s</h2>
    <p style="margin:0 0 8px;">%s</p>
    <p style="margin:0 0 24px;color:#94a3b8;">%s</p>
    <a href="%s" style="display:inline-block;background:#38bdf8;color:#0f172a;font-weight:600;
       text-decoration:none;padding:12px 20px;border-radius:10px;">%s</a>
    <p style="margin:24px 0 0;font-size:12px;color:#64748b;word-break:break-all;">%s</p>
    <p style="margin:24px 0 0;font-size:12px;color:#64748b;">%s</p>
  </div>
</body></html>`, title, greeting, intro, link, cta, link, footer)
}
