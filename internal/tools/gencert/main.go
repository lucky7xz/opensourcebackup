// gencert generates a self-signed TLS certificate for local development.
//
// Usage:
//
//	go run ./internal/tools/gencert/                    # localhost only
//	go run ./internal/tools/gencert/ 192.168.1.100      # + LAN IP
//	go run ./internal/tools/gencert/ 192.168.1.100 10.0.0.5
//
// Output: certs/dev.crt + certs/dev.key
// Valid for: localhost, 127.0.0.1, and any IPs passed as arguments
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	// Collect IPs: always include localhost + 127.0.0.1, add any CLI args
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	}
	dns := []string{"localhost"}

	for _, arg := range os.Args[1:] {
		if ip := net.ParseIP(arg); ip != nil {
			ips = append(ips, ip)
		} else {
			fmt.Fprintf(os.Stderr, "warning: %q is not a valid IP address — skipping\n", arg)
		}
	}

	if err := os.MkdirAll("certs", 0700); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir certs:", err)
		os.Exit(1)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate key:", err)
		os.Exit(1)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "OpenSourceBackup Dev",
			Organization: []string{"OpenSourceBackup"},
		},
		NotBefore:             time.Now().Add(-time.Minute), // 1 min clock-skew tolerance
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true, // self-signed
		IPAddresses:           ips,
		DNSNames:              dns,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create certificate:", err)
		os.Exit(1)
	}

	if err := writePEM("certs/dev.crt", "CERTIFICATE", certDER); err != nil {
		fmt.Fprintln(os.Stderr, "write cert:", err)
		os.Exit(1)
	}

	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal key:", err)
		os.Exit(1)
	}
	if err := writePEM("certs/dev.key", "EC PRIVATE KEY", kb); err != nil {
		fmt.Fprintln(os.Stderr, "write key:", err)
		os.Exit(1)
	}

	ipStrings := make([]string, len(ips))
	for i, ip := range ips {
		ipStrings[i] = ip.String()
	}

	fmt.Println("✓ certs/dev.crt  — TLS certificate (365 days)")
	fmt.Println("✓ certs/dev.key  — private key (keep secret, in .gitignore)")
	fmt.Println()
	fmt.Printf("  Valid for: %s\n", strings.Join(append(dns, ipStrings...), ", "))
	fmt.Println()
	fmt.Println("  Start HTTPS server:")
	fmt.Println("    make run-https")
	fmt.Println()
	fmt.Println("  Or with redirect (HTTP :8080 → HTTPS :8443):")
	fmt.Println("    LISTEN_ADDR=:8443 HTTP_REDIRECT_ADDR=:8080 TLS_CERT_FILE=certs/dev.crt TLS_KEY_FILE=certs/dev.key ./control-plane")
	fmt.Println()
	fmt.Println("  Agent with self-signed cert:")
	fmt.Println("    CONTROL_PLANE_URL=https://localhost:8443 AGENT_TLS_SKIP_VERIFY=true ./agent")
	fmt.Println()
	fmt.Println("  Production (Let's Encrypt or real cert):")
	fmt.Println("    TLS_CERT_FILE=/etc/letsencrypt/live/domain/fullchain.pem \\")
	fmt.Println("    TLS_KEY_FILE=/etc/letsencrypt/live/domain/privkey.pem \\")
	fmt.Println("    TLS_REQUIRED=true ./control-plane")
}

func writePEM(path, blockType string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}
