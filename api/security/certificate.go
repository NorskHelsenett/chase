// certificate.go
package security

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"strings"
	"time"
)

func (s *Scanner) checkCertificate(domain string) (*CertificateAnalysis, error) {
	host := strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}

	conn, err := tls.Dial("tcp", host, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	cert := conn.ConnectionState().PeerCertificates[0]
	analysis := &CertificateAnalysis{
		ValidUntil: cert.NotAfter,
		Issuer:     cert.Issuer.CommonName,
		Findings:   make([]Finding, 0),
		Warnings:   make([]Finding, 0),
		Risk:       RiskLow,
	}

	// Check certificate expiration
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

	// Check key strength
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

	// Determine grade based on findings
	if len(analysis.Findings) == 0 && len(analysis.Warnings) == 0 {
		analysis.Grade = "A+"
	} else if len(analysis.Findings) == 0 {
		analysis.Grade = "A"
	} else if analysis.Risk == RiskHigh {
		analysis.Grade = "C"
	} else {
		analysis.Grade = "B"
	}

	return analysis, nil
}
