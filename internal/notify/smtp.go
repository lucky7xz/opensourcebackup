package notify

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/health"
)

// smtpConfig holds the server-level SMTP relay settings. Credentials come from
// the environment only — never from the database, never on argv, never logged.
type smtpConfig struct {
	host   string
	port   string
	from   string
	user   string
	pass   string
	useTLS bool // implicit TLS (e.g. port 465); otherwise STARTTLS on 587
}

// loadSMTPConfig reads SMTP settings from the environment.
//
//	SMTP_HOST      relay hostname            (required to enable email)
//	SMTP_FROM      envelope/from address     (required to enable email)
//	SMTP_PORT      default 587
//	SMTP_USERNAME  optional — enables PLAIN auth when set
//	SMTP_PASSWORD  optional — secret, never logged
//	SMTP_TLS       "true" for implicit TLS (465); default STARTTLS
func loadSMTPConfig() smtpConfig {
	return smtpConfig{
		host:   os.Getenv("SMTP_HOST"),
		port:   envOr("SMTP_PORT", "587"),
		from:   os.Getenv("SMTP_FROM"),
		user:   os.Getenv("SMTP_USERNAME"),
		pass:   os.Getenv("SMTP_PASSWORD"),
		useTLS: strings.EqualFold(os.Getenv("SMTP_TLS"), "true"),
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (c smtpConfig) configured() bool { return c.host != "" && c.from != "" }

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func validEmail(s string) bool { return emailRe.MatchString(strings.TrimSpace(s)) }

// headerSafe strips CR/LF to prevent SMTP header injection via crafted values.
func headerSafe(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// buildAlertEmail renders a plain-text subject + body for a set of alerts.
func buildAlertEmail(alerts []health.Alert) (subject, body string) {
	crit := countBySeverity(alerts, "critical")
	warn := countBySeverity(alerts, "warning")
	switch {
	case crit > 0:
		subject = fmt.Sprintf("[OpenSourceBackup] %d kritische Warnung(en)", crit)
	case warn > 0:
		subject = fmt.Sprintf("[OpenSourceBackup] %d Warnung(en)", warn)
	default:
		subject = "[OpenSourceBackup] Hinweis"
	}

	var b strings.Builder
	b.WriteString("OpenSourceBackup — Health Alerts\n")
	b.WriteString(time.Now().UTC().Format(time.RFC1123) + "\n\n")
	for _, a := range alerts {
		fmt.Fprintf(&b, "- [%s] %s\n  %s\n", strings.ToUpper(a.Severity), a.Title, a.Description)
	}
	b.WriteString("\n— automatische Nachricht, bitte nicht antworten.\n")
	return subject, b.String()
}

// composeMessage builds an RFC 822 message. Header values are sanitised against
// CRLF injection; the body is normalised to CRLF line endings.
func composeMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", headerSafe(from))
	fmt.Fprintf(&b, "To: %s\r\n", headerSafe(to))
	fmt.Fprintf(&b, "Subject: %s\r\n", headerSafe(subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return []byte(b.String())
}

// sendEmail delivers the alert mail to the channel's recipient via SMTP.
func (n *Notifier) sendEmail(ch Channel, alerts []health.Alert) error {
	if !n.smtp.configured() {
		return fmt.Errorf("SMTP not configured — set SMTP_HOST and SMTP_FROM")
	}
	to := strings.TrimSpace(ch.Target)
	if !validEmail(to) {
		return fmt.Errorf("invalid recipient address")
	}

	subject, body := buildAlertEmail(alerts)
	msg := composeMessage(n.smtp.from, to, subject, body)
	addr := net.JoinHostPort(n.smtp.host, n.smtp.port)

	var auth smtp.Auth
	if n.smtp.user != "" {
		auth = smtp.PlainAuth("", n.smtp.user, n.smtp.pass, n.smtp.host)
	}

	if n.smtp.useTLS {
		return n.sendImplicitTLS(addr, auth, to, msg)
	}
	// STARTTLS path — net/smtp upgrades the connection when the server offers it.
	return smtp.SendMail(addr, auth, n.smtp.from, []string{to}, msg)
}

// sendImplicitTLS handles servers that expect TLS from the first byte (port 465).
func (n *Notifier) sendImplicitTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr,
		&tls.Config{ServerName: n.smtp.host},
	)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	c, err := smtp.NewClient(conn, n.smtp.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth failed: %w", err)
		}
	}
	if err := c.Mail(n.smtp.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp body close: %w", err)
	}
	return c.Quit()
}
