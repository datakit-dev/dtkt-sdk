package network

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"

	corev1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/core/v1"
)

// TLSServerConfig builds the tls.Config for the Integration (Server Role)
// from a persisted TLSCertBundle and the current root CA certificate.
func TLSServerConfig(caCertPEM string, bundle *corev1.TLSCertBundle) (*tls.Config, error) {
	certPool, err := BundleCertPool(caCertPEM, bundle)
	if err != nil {
		return nil, err
	}

	serverTLS, err := tls.X509KeyPair(bundle.GetCertPem(), bundle.GetKeyPem())
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverTLS},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert, // Require Daemon to pass its client cert
		// Must advertise h2 via ALPN or the client's NextProtos offer has
		// nothing to negotiate against, leaving NegotiatedProtocol empty.
		NextProtos: []string{"h2"},
	}, nil
}

// TLSClientConfig builds the tls.Config for the Daemon Proxy (Client Role)
// from a persisted TLSCertBundle and the current root CA certificate.
func TLSClientConfig(caCertPEM string, bundle *corev1.TLSCertBundle) (*tls.Config, error) {
	certPool, err := BundleCertPool(caCertPEM, bundle)
	if err != nil {
		return nil, err
	}

	clientTLS, err := tls.X509KeyPair(bundle.GetCertPem(), bundle.GetKeyPem())
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientTLS},
		RootCAs:      certPool, // properly chained now, so no InsecureSkipVerify needed
		// tls.Client (used by DialContext) requires ServerName explicitly since
		// it can't infer one from a dial address; "localhost" is always a SAN
		// on the server leaf regardless of integrationID.
		ServerName: "localhost",
	}, nil
}

// BundleCertPool validates that bundle was signed by caCertPEM (guards
// against reusing a leaf cached from before a CA rotation) and returns the
// CA pool derived from it.
func BundleCertPool(caCertPEM string, bundle *corev1.TLSCertBundle) (*x509.CertPool, error) {
	if bundle == nil {
		return nil, errors.New("nil TLS cert bundle")
	}

	caFingerprint, err := CAFingerprint(caCertPEM)
	if err != nil {
		return nil, err
	}

	if bundle.GetCaFingerprint() != caFingerprint {
		return nil, errors.New("TLS cert bundle was signed by a different root CA")
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM([]byte(caCertPEM)) {
		return nil, errors.New("failed to parse root CA PEM")
	}

	return certPool, nil
}

// CAFingerprint returns the hex-encoded SHA-256 digest of the CA certificate's
// raw DER bytes. Leaf certs cached alongside this value (e.g. Integration's
// ca_fingerprint column) should be regenerated whenever it changes, since
// that means the root CA was rotated.
func CAFingerprint(caCertPEM string) (string, error) {
	certBlock, _ := pem.Decode([]byte(caCertPEM))
	if certBlock == nil {
		return "", errors.New("failed to decode root CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse root CA certificate: %w", err)
	}

	sum := sha256.Sum256(caCert.Raw)
	return hex.EncodeToString(sum[:]), nil
}
