// gencert generates a self-signed TLS certificate for local development.
// Usage: go run ./internal/tools/gencert/
// Output: certs/dev.crt + certs/dev.key
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

func main() {
	if err := os.MkdirAll("certs", 0700); err != nil {
		panic(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	cf, err := os.Create("certs/dev.crt")
	if err != nil {
		panic(err)
	}
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		panic(err)
	}
	if err := cf.Close(); err != nil {
		panic(err)
	}

	kf, err := os.Create("certs/dev.key")
	if err != nil {
		panic(err)
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}); err != nil {
		panic(err)
	}
	if err := kf.Close(); err != nil {
		panic(err)
	}

	println("✓ certs/dev.crt  — TLS certificate (valid 365 days, localhost + 127.0.0.1)")
	println("✓ certs/dev.key  — private key (keep secret, in .gitignore)")
	println("")
	println("Start HTTPS:  make run-https")
	println("Agent HTTPS:  CONTROL_PLANE_URL=https://localhost:8443 AGENT_TLS_SKIP_VERIFY=true ./agent")
}
