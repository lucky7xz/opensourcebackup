package notify

import (
	"testing"
)

func TestValidateWebhookURL(t *testing.T) {
	type tc struct {
		url     string
		wantErr bool
		desc    string
	}
	cases := []tc{
		// valid
		{"https://hooks.slack.com/services/T00/B00/xxx", false, "valid https webhook"},
		{"http://my.webhook.example.com/notify", false, "valid http webhook"},
		{"https://webhook.site/abc123", false, "webhook.site"},

		// SSRF — private / loopback
		{"http://localhost/callback", true, "localhost rejected"},
		{"http://127.0.0.1/callback", true, "loopback IP rejected"},
		{"http://192.168.0.1/admin", true, "RFC-1918 class C rejected"},
		{"http://10.0.0.1/data", true, "RFC-1918 class A rejected"},
		{"http://172.16.0.1/data", true, "RFC-1918 class B rejected"},
		{"http://169.254.169.254/latest/meta-data/", true, "AWS metadata service rejected"},
		{"http://100.64.0.1/test", true, "shared address space rejected"},
		{"http://[::1]/callback", true, "IPv6 loopback rejected"},

		// bad scheme / format
		{"file:///etc/passwd", true, "file scheme rejected"},
		{"ftp://ftp.example.com/data", true, "ftp scheme rejected"},
		{"", true, "empty URL rejected"},
		{"not-a-url", true, "invalid URL rejected"},
		{"://missing-scheme", true, "missing scheme rejected"},
	}

	for _, c := range cases {
		err := ValidateWebhookURL(c.url)
		if (err != nil) != c.wantErr {
			if c.wantErr {
				t.Errorf("[%s] url=%q: expected error, got nil", c.desc, c.url)
			} else {
				t.Errorf("[%s] url=%q: unexpected error: %v", c.desc, c.url, err)
			}
		}
	}
}
