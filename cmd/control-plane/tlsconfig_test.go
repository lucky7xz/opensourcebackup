package main

import "testing"

func envFunc(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveServerTLS(t *testing.T) {
	cases := []struct {
		name        string
		env         map[string]string
		wantErr     bool
		wantEnabled bool
		wantReq     bool
		wantRedir   string
	}{
		{
			name:        "no TLS configured → plaintext, no error",
			env:         nil,
			wantEnabled: false,
		},
		{
			name:        "cert + key → enabled",
			env:         map[string]string{"TLS_CERT_FILE": "c.pem", "TLS_KEY_FILE": "k.pem"},
			wantEnabled: true,
		},
		{
			name:    "required but not configured → error (refuse plaintext)",
			env:     map[string]string{"TLS_REQUIRED": "true"},
			wantErr: true,
		},
		{
			name:    "required + cert only → still error",
			env:     map[string]string{"TLS_REQUIRED": "true", "TLS_CERT_FILE": "c.pem"},
			wantErr: true,
		},
		{
			name:        "required + cert + key → ok",
			env:         map[string]string{"TLS_REQUIRED": "true", "TLS_CERT_FILE": "c.pem", "TLS_KEY_FILE": "k.pem"},
			wantEnabled: true,
			wantReq:     true,
		},
		{
			name:        "cert only (not required) → not enabled, no error",
			env:         map[string]string{"TLS_CERT_FILE": "c.pem"},
			wantEnabled: false,
		},
		{
			name:        "redirect addr captured when enabled",
			env:         map[string]string{"TLS_CERT_FILE": "c.pem", "TLS_KEY_FILE": "k.pem", "HTTP_REDIRECT_ADDR": ":8080"},
			wantEnabled: true,
			wantRedir:   ":8080",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := resolveServerTLS(envFunc(c.env))
			if (err != nil) != c.wantErr {
				t.Fatalf("err: want %v, got %v", c.wantErr, err)
			}
			if c.wantErr {
				return
			}
			if s.enabled != c.wantEnabled {
				t.Errorf("enabled: want %v, got %v", c.wantEnabled, s.enabled)
			}
			if s.required != c.wantReq {
				t.Errorf("required: want %v, got %v", c.wantReq, s.required)
			}
			if s.redirectAddr != c.wantRedir {
				t.Errorf("redirectAddr: want %q, got %q", c.wantRedir, s.redirectAddr)
			}
		})
	}
}
