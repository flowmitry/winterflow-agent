package certs

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	log "winterflow-agent/pkg/log"

	"google.golang.org/grpc/credentials"
)

// GeneratePrivateKey generates a new RSA private key and saves it to the specified path
func GeneratePrivateKey(keyPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for private key: %v", err)
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	// Convert to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Create file
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %v", err)
	}
	defer keyFile.Close()

	// Write PEM to file
	if err := pem.Encode(keyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key to file: %v", err)
	}

	log.Printf("[DEBUG] Generated private key at: %s", keyPath)
	return nil
}

// SaveCertificate saves the certificate data to the specified path
func SaveCertificate(certData, certPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for certificate: %v", err)
	}

	// Create file
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}
	defer certFile.Close()

	// Write certificate data to file
	if _, err := certFile.WriteString(certData); err != nil {
		return fmt.Errorf("failed to write certificate to file: %v", err)
	}

	log.Printf("[DEBUG] Saved certificate at: %s", certPath)
	return nil
}

// CreateCSR creates a Certificate Signing Request with the given private key and saves it to the specified path
func CreateCSR(certificateID string, privateKeyPath, csrPath string) (string, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(csrPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for CSR: %v", err)
	}

	// Read private key
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %v", err)
	}

	// Parse private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to parse private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Create CSR template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: certificateID,
		},
	}

	// Create CSR
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create CSR: %v", err)
	}

	// Convert to PEM format
	csrPEM := &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	}

	// Create file
	csrFile, err := os.OpenFile(csrPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create CSR file: %v", err)
	}
	defer csrFile.Close()

	// Write PEM to file
	if err := pem.Encode(csrFile, csrPEM); err != nil {
		return "", fmt.Errorf("failed to write CSR to file: %v", err)
	}

	// Convert CSR to string for API submission
	var csrBuffer bytes.Buffer
	if err := pem.Encode(&csrBuffer, csrPEM); err != nil {
		return "", fmt.Errorf("failed to encode CSR to string: %v", err)
	}

	log.Printf("[DEBUG] Created CSR at: %s with Common Name: %s", csrPath, certificateID)
	return csrBuffer.String(), nil
}

// LoadTLSCredentials loads TLS credentials from certificate and private key files
func LoadTLSCredentials(caCertPath, certPath, keyPath, host string) (credentials.TransportCredentials, error) {
	// Load certificate and private key
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate and private key: %v", err)
	}

	// Load your CA certificate
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCert)
	if !ok {
		log.Fatalf("failed to append CA certificate")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName:   host,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Create and return credentials
	creds := credentials.NewTLS(tlsConfig)
	log.Printf("[DEBUG] Loaded TLS credentials from certificate: %s and key: %s", certPath, keyPath)
	return creds, nil
}

// CertificateExists checks if a certificate file exists
func CertificateExists(certPath string) bool {
	_, err := os.Stat(certPath)
	return err == nil
}

// DecryptWithPrivateKey decrypts base64-encoded data using the RSA private key at the specified path
func DecryptWithPrivateKey(privateKeyPath, encryptedBase64 string) (string, error) {
	// Read private key
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %v", err)
	}

	// Parse private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to parse private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Decode base64 data
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 data: %v", err)
	}

	// Create a SHA-256 hash for OAEP
	hash := crypto.SHA256.New()

	// Decrypt data using RSA-OAEP with SHA-256
	decryptedData, err := rsa.DecryptOAEP(hash, rand.Reader, privateKey, encryptedData, nil)
	if err != nil {
		// Try with SHA-1 if SHA-256 fails (for backward compatibility)
		hash = crypto.SHA1.New()
		decryptedData, err = rsa.DecryptOAEP(hash, rand.Reader, privateKey, encryptedData, nil)
		if err != nil {
			// If both fail, try PKCS#1 v1.5 as a last resort
			decryptedData, err = rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedData)
			if err != nil {
				return "", fmt.Errorf("failed to decrypt data: %v", err)
			}
		}
	}

	return string(decryptedData), nil
}
