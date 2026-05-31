package security

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
)

// HashIP returns a one-way SHA-256 hash of the IP address.
// This satisfies GDPR Art. 5 data-minimisation: we retain auditability
// (correlate events from the same source) without storing PII in clear text.
//
// The hash is NOT salted so that the same IP produces the same hash across
// log entries — enabling correlation for incident investigation.
// This is intentional: a salted hash would break the correlation use-case.
//
// If the IP needs to be identified (e.g. court order), the operator must
// cross-reference with their network/access logs. Document this in TOM.
func HashIP(ip string) string {
	h := sha256.Sum256([]byte(ip))
	return fmt.Sprintf("ip-sha256:%x", h[:8]) // 8 bytes → 16 hex chars — sufficient for correlation
}

// ClientIPHashed returns the hashed client IP for audit logging.
func ClientIPHashed(r *http.Request) string {
	return HashIP(ClientIP(r))
}

// isPrivateRange reports whether ip falls in RFC-1918 / loopback ranges.
// Used by ClientIP to decide whether to trust X-Forwarded-For.
func isPrivateRange(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"127.0.0.0/8", "::1/128", "fc00::/7",
	} {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(parsed) {
			return true
		}
	}
	return false
}

// ClientIP — re-exported here for single-import convenience.
// The implementation lives in ratelimit.go to avoid duplication.
func clientIPFromRequest(r *http.Request) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if isPrivateRange(remoteIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return xff[:i]
				}
			}
			return xff
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}
	return remoteIP
}
