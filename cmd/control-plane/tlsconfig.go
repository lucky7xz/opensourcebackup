package main

import "fmt"

// tlsSettings is the resolved TLS configuration for the control-plane server.
type tlsSettings struct {
	certFile     string
	keyFile      string
	enabled      bool   // both cert and key provided
	required     bool   // TLS_REQUIRED=true — refuse to serve plaintext
	redirectAddr string // optional HTTP→HTTPS redirect listener
}

// resolveServerTLS derives the TLS configuration from environment values.
//
// It enforces one security guardrail: when TLS is *required* but a cert/key
// pair is not fully provided, it returns an error so the server refuses to
// start in plaintext — preventing an accidental unencrypted production
// deployment. getenv is injected so the decision is unit-testable.
func resolveServerTLS(getenv func(string) string) (tlsSettings, error) {
	s := tlsSettings{
		certFile:     getenv("TLS_CERT_FILE"),
		keyFile:      getenv("TLS_KEY_FILE"),
		required:     getenv("TLS_REQUIRED") == "true",
		redirectAddr: getenv("HTTP_REDIRECT_ADDR"),
	}
	s.enabled = s.certFile != "" && s.keyFile != ""

	if s.required && !s.enabled {
		return s, fmt.Errorf("TLS_REQUIRED=true but TLS_CERT_FILE/TLS_KEY_FILE not set — refusing to start in HTTP mode")
	}
	return s, nil
}
