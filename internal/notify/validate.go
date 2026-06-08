package notify

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateWebhookURL rejects URLs that could enable Server-Side Request Forgery (SSRF).
// It enforces http/https scheme and blocks private/loopback IP ranges.
//
// Security rationale: webhook targets are operator-supplied and later fetched by the
// server. Without validation an operator (or an attacker with operator access) could
// probe internal services (169.254.169.254 metadata, 192.168.x.x, localhost).
func ValidateWebhookURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("webhook URL must not be empty")
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("scheme %q not allowed — only http and https are permitted", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host in webhook URL")
	}
	// Reject bare loopback hostnames
	if host == "localhost" {
		return fmt.Errorf("loopback address %q not allowed as webhook target", host)
	}
	// Reject private / loopback IP addresses (SSRF protection)
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateOrLoopback(ip) {
			return fmt.Errorf("private or loopback IP %q not allowed as webhook target", host)
		}
	}
	return nil
}

// isPrivateOrLoopback returns true when ip falls in RFC-1918 / loopback / link-local ranges.
func isPrivateOrLoopback(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",  // RFC-6598 shared address space
		"169.254.0.0/16", // AWS/GCP metadata link-local
		"fc00::/7",       // IPv6 ULA
		"::1/128",
	} {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
