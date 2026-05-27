package service

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
	"path/filepath"
	"time"

	"tunnelium/src/paths"
)

// GenerateGostCommand returns the gost command-line arguments for the given service parameters.
func GenerateGostCommand(params ServiceParams) []string {
	switch params.GostRole {
	case GostRoleClient:
		var args []string

		// SOCKS5 entry point with auth
		if params.GostSocksPort > 0 {
			socksPort := params.GostSocksPort
			args = append(args, "-L", fmt.Sprintf("socks5://:%d?auths=/etc/gost/auths.yaml", socksPort))
		}

		// HTTP entry point
		if params.GostHTTPPort > 0 {
			args = append(args, "-L", fmt.Sprintf("http://:%d", params.GostHTTPPort))
		}

		// Forwarder
		nextHopPort := params.GostNextHopPort
		if nextHopPort == 0 {
			nextHopPort = 443
		}
		args = append(args, "-F", fmt.Sprintf("relay+tls://%s:%d", params.GostNextHopHost, nextHopPort))

		return args
	case GostRoleServer:
		return []string{
			"-L", fmt.Sprintf("relay+tls://:%d?cert=/cert.pem&key=/key.pem", params.HostSystemPort),
		}
	default:
		return nil
	}
}

// GenerateGostVolumes returns the volume mounts for a gost server service.
// Returns nil for client role (no volumes needed).
func GenerateGostVolumes(params ServiceParams) []string {
	if params.GostRole != GostRoleServer {
		return nil
	}
	serviceName := fmt.Sprintf("%s-%s", params.ServiceType, params.InstanceName)
	tlsPath := filepath.Join(paths.ServiceDir(serviceName), "tls.pem")
	return []string{
		tlsPath + ":/cert.pem:ro",
		tlsPath + ":/key.pem:ro",
	}
}

// GenerateTLSCert generates a self-signed ECDSA P-256 TLS certificate
// and writes both the certificate and private key into a single combined PEM file.
func GenerateTLSCert(combinedPath string) error {
	// Generate ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating ECDSA key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"tunnelium"},
			CommonName:   "gost-relay",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add SANs: localhost and common local IPs
	template.IPAddresses = []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	}

	// Get host IPs for SAN
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip != nil && !ip.IsLoopback() {
					template.IPAddresses = append(template.IPAddresses, ip)
				}
			}
		}
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	// Marshal private key
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshaling private key: %w", err)
	}

	// Write combined PEM file: certificate first, then private key
	f, err := os.Create(combinedPath)
	if err != nil {
		return fmt.Errorf("creating combined PEM file %s: %w", combinedPath, err)
	}
	defer f.Close()

	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("encoding certificate: %w", err)
	}

	if err := pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return fmt.Errorf("encoding private key: %w", err)
	}

	return nil
}
