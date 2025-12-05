package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"
)

func (s *Scanner) checkCertificate(ctx context.Context, domain string) (*CertificateAnalysis, error) {
	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	// Test different TLS versions
	tlsVersions := make([]string, 0)
	versions := map[uint16]string{
		tls.VersionTLS10: "TLS 1.0",
		tls.VersionTLS11: "TLS 1.1",
		tls.VersionTLS12: "TLS 1.2",
		tls.VersionTLS13: "TLS 1.3",
	}

	for version, name := range versions {
		conn, err := s.dialTLS(ctx, host, true, version)
		if err == nil {
			tlsVersions = append(tlsVersions, name)
			conn.Close()
		}
	}

	// Main connection for full analysis
	conn, err := s.dialTLS(ctx, host, false, 0)
	var tlsVerificationErr error
	if err != nil {
		if isTLSError(err) {
			tlsVerificationErr = err
			conn, err = s.dialTLS(ctx, host, true, 0)
		}
		if err != nil {
			return nil, err
		}
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0]
	// Get public key information
	var keyType string
	var keyBits int
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		keyType = "RSA"
		keyBits = pub.N.BitLen()
	case *ecdsa.PublicKey:
		keyType = "ECDSA"
		keyBits = pub.Curve.Params().BitSize
	default:
		keyType = fmt.Sprintf("%T", cert.PublicKey)
		keyBits = 0
	}

	analysis := &CertificateAnalysis{
		ValidFrom:          cert.NotBefore,
		ValidUntil:         cert.NotAfter,
		Issuer:             cert.Issuer.CommonName,
		Organization:       GetOrganization(cert),
		SubjectDNS:         cert.DNSNames,
		SerialNumber:       cert.SerialNumber.Text(16),
		SignatureAlg:       cert.SignatureAlgorithm.String(),
		PublicKeyType:      keyType,
		PublicKeyBits:      keyBits,
		Findings:           make([]Finding, 0),
		Warnings:           make([]Finding, 0),
		Risk:               RiskLow,
		TLSVersions:        tlsVersions,
		NegotiatedProtocol: conn.ConnectionState().NegotiatedProtocol,
		PreferredCipher:    tls.CipherSuiteName(conn.ConnectionState().CipherSuite),
	}

	if len(tlsVersions) == 0 {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "Unable to negotiate any modern TLS versions",
			Risk:        RiskCritical,
			Evidence:    "Server rejected TLS 1.0 - 1.3 handshakes",
			Mitigation:  "Enable TLS 1.2+ on the edge load balancer",
		})
		analysis.Risk = RiskCritical
	} else if !containsString(tlsVersions, "TLS 1.3") {
		analysis.Warnings = append(analysis.Warnings, Finding{
			Description: "TLS 1.3 not supported",
			Risk:        RiskMedium,
			Evidence:    fmt.Sprintf("Supported versions: %s", strings.Join(tlsVersions, ", ")),
			Mitigation:  "Enable TLS 1.3 to prevent downgrade attacks",
		})
	}

	if tlsVerificationErr != nil {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "TLS verification failed",
			Risk:        RiskHigh,
			Evidence:    tlsVerificationErr.Error(),
			Mitigation:  "Install a publicly trusted certificate matching the hostname",
		})
		analysis.Risk = RiskHigh
	}

	// Check TLS version support
	for _, version := range tlsVersions {
		if version == "TLS 1.0" || version == "TLS 1.1" {
			analysis.Warnings = append(analysis.Warnings, Finding{
				Description: "Outdated TLS version supported",
				Risk:        RiskMedium,
				Evidence:    fmt.Sprintf("Server supports %s", version),
				Mitigation:  "Disable TLS 1.0 and 1.1 support",
			})
		}
	}

	// Original expiration check
	expirationDays := time.Until(cert.NotAfter).Hours() / 24
	if expirationDays < 0 {
		analysis.Findings = append(analysis.Findings, Finding{
			Description: "Certificate has expired",
			Risk:        RiskCritical,
			Evidence:    fmt.Sprintf("Expired on %s", cert.NotAfter.Format("2006-01-02")),
			Mitigation:  "Renew SSL certificate immediately",
		})
		analysis.Risk = RiskCritical
	} else if expirationDays < 30 {
		analysis.Warnings = append(analysis.Warnings, Finding{
			Description: "Certificate expiring soon",
			Risk:        RiskHigh,
			Evidence:    fmt.Sprintf("Expires on %s", cert.NotAfter.Format("2006-01-02")),
			Mitigation:  "Plan to renew SSL certificate",
		})
		analysis.Risk = RiskHigh
	}

	// Original key strength check
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		keyBits := pub.N.BitLen()
		if keyBits < 2048 {
			analysis.Findings = append(analysis.Findings, Finding{
				Description: "Weak certificate key strength",
				Risk:        RiskHigh,
				Evidence:    fmt.Sprintf("Key size: %d bits", keyBits),
				Mitigation:  "Use at least 2048-bit RSA key",
			})
			if analysis.Risk < RiskHigh {
				analysis.Risk = RiskHigh
			}
		}
	case *ecdsa.PublicKey:
		curveBits := pub.Curve.Params().BitSize
		if curveBits < 256 {
			analysis.Findings = append(analysis.Findings, Finding{
				Description: "Weak ECDSA curve size",
				Risk:        RiskHigh,
				Evidence:    fmt.Sprintf("Curve size: %d bits", curveBits),
				Mitigation:  "Use at least a 256-bit ECDSA curve",
			})
			if analysis.Risk < RiskHigh {
				analysis.Risk = RiskHigh
			}
		}
	default:
		return nil, fmt.Errorf("unsupported public key type: %T", cert.PublicKey)
	}

	// Updated grade calculation
	analysis.Grade = calculateGradeCertificate(analysis)

	return analysis, nil
}

func containsString(values []string, candidate string) bool {
	for _, v := range values {
		if v == candidate {
			return true
		}
	}
	return false
}

func (s *Scanner) dialTLS(ctx context.Context, host string, insecure bool, version uint16) (*tls.Conn, error) {
	dialer := &net.Dialer{
		Timeout: s.timeout,
	}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	}

	serverName := host
	if strings.Contains(host, ":") {
		serverName = strings.Split(host, ":")[0]
	}

	config := &tls.Config{
		InsecureSkipVerify: insecure,
		ServerName:         serverName,
	}
	if version != 0 {
		config.MinVersion = version
		config.MaxVersion = version
	}

	return tls.DialWithDialer(dialer, "tcp", host, config)
}

func GetOrganization(cert *x509.Certificate) string {
	// Try different certificate fields for organization info
	if len(cert.Subject.Organization) > 0 {
		return cert.Subject.Organization[0]
	}

	if len(cert.Issuer.Organization) > 0 {
		return cert.Issuer.Organization[0]
	}

	// Try CommonName if no Organization is set
	if cert.Subject.CommonName != "" {
		// Remove any wildcard prefixes
		cn := strings.TrimPrefix(cert.Subject.CommonName, "*.")
		// If it looks like a domain, don't use it as organization
		if !strings.Contains(cn, ".") {
			return cn
		}
	}

	// Try OrganizationalUnit if available
	if len(cert.Subject.OrganizationalUnit) > 0 {
		return cert.Subject.OrganizationalUnit[0]
	}

	// Try Subject Alternative Names for organization info
	for _, name := range cert.Subject.Names {
		// OID 2.5.4.10 is Organization Name
		if name.Type.Equal([]int{2, 5, 4, 10}) {
			if str, ok := name.Value.(string); ok {
				return str
			}
		}
	}

	return "Unknown Organization"
}

func calculateGradeCertificate(analysis *CertificateAnalysis) string {
	if len(analysis.Findings) > 0 {
		return "C"
	}

	// Cap grade at B for TLS 1.0/1.1 support
	for _, version := range analysis.TLSVersions {
		if version == "TLS 1.0" || version == "TLS 1.1" {
			return "B"
		}
	}

	if len(analysis.Warnings) > 0 {
		return "A"
	}

	return "A+"
}
