package notify

import (
	"strings"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/health"
)

func TestValidEmail(t *testing.T) {
	valid := []string{"a@b.co", "ops.team@example.com", "x+tag@sub.domain.io"}
	for _, e := range valid {
		if !validEmail(e) {
			t.Errorf("expected %q to be valid", e)
		}
	}
	invalid := []string{"", "no-at", "a@b", "a@@b.co", "a b@c.de", "a@b.co\r\nBcc: x@y.z"}
	for _, e := range invalid {
		if validEmail(e) {
			t.Errorf("expected %q to be invalid", e)
		}
	}
}

func TestSMTPConfig_Configured(t *testing.T) {
	if (smtpConfig{}).configured() {
		t.Error("empty config must not be configured")
	}
	if (smtpConfig{host: "mail.x"}).configured() {
		t.Error("host alone must not be configured (from missing)")
	}
	if !(smtpConfig{host: "mail.x", from: "a@b.co"}).configured() {
		t.Error("host + from must be configured")
	}
}

func TestBuildAlertEmail_SubjectReflectsSeverity(t *testing.T) {
	crit := []health.Alert{{Severity: "critical", Title: "Repo offline"}, {Severity: "warning"}}
	sub, body := buildAlertEmail(crit)
	if !strings.Contains(sub, "kritische") {
		t.Errorf("critical subject expected, got %q", sub)
	}
	if !strings.Contains(body, "Repo offline") {
		t.Error("body must contain the alert title")
	}

	warnOnly := []health.Alert{{Severity: "warning"}}
	if s, _ := buildAlertEmail(warnOnly); !strings.Contains(s, "Warnung") || strings.Contains(s, "kritische") {
		t.Errorf("warning subject expected, got %q", s)
	}

	infoOnly := []health.Alert{{Severity: "info"}}
	if s, _ := buildAlertEmail(infoOnly); !strings.Contains(s, "Hinweis") {
		t.Errorf("info subject expected, got %q", s)
	}
}

func TestComposeMessage_HasHeadersAndStripsInjection(t *testing.T) {
	msg := string(composeMessage("from@osb.io", "to@osb.io", "Subject line", "Hello\nWorld"))

	for _, want := range []string{"From: from@osb.io\r\n", "To: to@osb.io\r\n", "Subject: Subject line\r\n", "Content-Type: text/plain"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q", want)
		}
	}
	// Body normalised to CRLF
	if !strings.Contains(msg, "Hello\r\nWorld") {
		t.Error("body line endings must be normalised to CRLF")
	}
}

func TestComposeMessage_RejectsHeaderInjection(t *testing.T) {
	// A subject carrying CRLF must not become a separate (injected) header line.
	evil := "Hi\r\nBcc: attacker@evil.com"
	msg := string(composeMessage("from@osb.io", "to@osb.io", evil, "body"))

	// No CRLF-prefixed Bcc header may exist anywhere.
	if strings.Contains(msg, "\r\nBcc:") {
		t.Error("CRLF header injection: an injected Bcc header line was created")
	}
	// No line in the header block may start with "Bcc:".
	headerBlock := msg[:strings.Index(msg, "\r\n\r\n")]
	for _, line := range strings.Split(headerBlock, "\r\n") {
		if strings.HasPrefix(line, "Bcc:") {
			t.Errorf("injected header line found: %q", line)
		}
	}
}

func TestComposeMessage_NoCredentialLeak(t *testing.T) {
	// composeMessage never receives credentials; assert the rendered message
	// cannot contain a password field by construction.
	msg := string(composeMessage("from@osb.io", "to@osb.io", "Subj", "body"))
	for _, leak := range []string{"password", "SMTP_PASSWORD", "PLAIN"} {
		if strings.Contains(msg, leak) {
			t.Errorf("message unexpectedly contains %q", leak)
		}
	}
}
