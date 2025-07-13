package certs

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"winterflow-agent/pkg/log"

	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"

	"google.golang.org/grpc/credentials"
)

// GeneratePrivateKey generates a new ECDSA P-256 private key and saves it to the specified path
func GeneratePrivateKey(keyPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for private key: %v", err)
	}

	// Generate ECDSA P-256 private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ECDSA private key: %v", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key: %v", err)
	}

	// Convert to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
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

	// Parse private key (supports both ECDSA and RSA for backward compatibility)
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to parse private key PEM")
	}

	var parsedKey crypto.Signer
	switch block.Type {
	case "EC PRIVATE KEY":
		ecKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse EC private key: %v", err)
		}
		parsedKey = ecKey
	case "RSA PRIVATE KEY":
		rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse RSA private key: %v", err)
		}
		parsedKey = rsaKey
	default:
		return "", fmt.Errorf("unsupported private key type: %s", block.Type)
	}

	// Create CSR template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: certificateID,
		},
	}

	// Create CSR
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, parsedKey)
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
		return nil, fmt.Errorf("failed to read CA certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	// Create TLS configuration
	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// gRPC uses HTTP/2 under the hood, make sure we advertise it via ALPN
		NextProtos: []string{"h2"},
	}

	// Set ServerName only if host looks like a hostname (not an IP address). This avoids issues
	// when connecting via raw IPs that are not present in the certificateʼs SANs.
	if parsedIP := net.ParseIP(host); parsedIP == nil && host != "" {
		tlsConfig.ServerName = host
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

// DecryptWithPrivateKey decrypts base64-encoded data that was encrypted in the
// browser using the prime256v1 (P-256) ECDH + AES-GCM (256-bit) scheme.
//
// Encryption layout (all binary values are finally base64-encoded for
// transport):
//
//	┌───────────────────┬──────────────┬────────────────────────┐
//	│ 65 bytes          │ 12 bytes     │ remaining bytes        │
//	│ Ephemeral pub key │ AES-GCM IV   │ Ciphertext + auth tag  │
//	└───────────────────┴──────────────┴────────────────────────┘
//
// The 65-byte public key is the uncompressed form exported via
// `crypto.subtle.exportKey('raw', tmpKeyPair.publicKey)` in the browser and is
// always prefixed with 0x04.
//
// To derive the symmetric key we:
//  1. Perform ECDH with the agent's private key and the received public key.
//  2. Left-pad the X coordinate of the shared secret to 32 bytes (big-endian).
//  3. Hash it with SHA-256 and use the resulting 32 bytes as the AES-256-GCM
//     key.
//
// Only prime256v1 keys are supported – passing any other key type will return
// an explicit error.
func DecryptWithPrivateKey(privateKeyPath, encryptedBase64 string) (string, error) {
	// Load and parse the agent's private key (must be EC prime256v1).
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %v", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to decode private key PEM")
	}

	if block.Type != "EC PRIVATE KEY" {
		return "", fmt.Errorf("unsupported private key type %q – only EC (P-256) keys are supported", block.Type)
	}

	ecKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse EC private key: %v", err)
	}

	// Decode the payload.
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 payload: %v", err)
	}

	// Validate minimum length (65-byte pub key + 12-byte IV + 16-byte tag).
	const (
		rawPubKeyLen = 65 // 0x04 + X + Y
		ivLen        = 12
		minTotalLen  = rawPubKeyLen + ivLen + 16 // at least auth tag size
	)
	if len(encryptedData) < minTotalLen {
		return "", fmt.Errorf("encrypted payload too short: got %d bytes", len(encryptedData))
	}

	rawPubKey := encryptedData[:rawPubKeyLen]
	iv := encryptedData[rawPubKeyLen : rawPubKeyLen+ivLen]
	ciphertext := encryptedData[rawPubKeyLen+ivLen:]

	// Ensure uncompressed format.
	if rawPubKey[0] != 0x04 {
		return "", fmt.Errorf("unexpected EC public key format: first byte is 0x%02x, want 0x04 (uncompressed)", rawPubKey[0])
	}

	// Split X / Y coordinates.
	coordSize := 32 // for P-256
	x := new(big.Int).SetBytes(rawPubKey[1 : 1+coordSize])
	y := new(big.Int).SetBytes(rawPubKey[1+coordSize : 1+2*coordSize])

	curve := elliptic.P256()
	if !curve.IsOnCurve(x, y) {
		return "", fmt.Errorf("ephemeral public key is not on the P-256 curve")
	}

	// Derive shared secret (ECDH).
	sharedX, _ := curve.ScalarMult(x, y, ecKey.D.Bytes())
	if sharedX == nil {
		return "", fmt.Errorf("failed to derive shared secret")
	}

	// Left-pad X coordinate to 32 bytes.
	sharedXBytes := sharedX.Bytes()
	if len(sharedXBytes) < coordSize {
		padded := make([]byte, coordSize)
		copy(padded[coordSize-len(sharedXBytes):], sharedXBytes)
		sharedXBytes = padded
	}

	// Hash with SHA-256 to produce the AES-256 key.
	keyHash := sha256.Sum256(sharedXBytes)

	// Create AES-GCM cipher.
	blockCipher, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return "", fmt.Errorf("failed to create AES-GCM: %v", err)
	}

	// Decrypt.
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %v", err)
	}

	return string(plaintext), nil
}

// SignWithPrivateKey creates an ASN.1-encoded ECDSA signature over msg using
// the EC private key stored at keyPath.
func SignWithPrivateKey(keyPath string, msg []byte) (string, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return "", fmt.Errorf("failed to decode private key PEM")
	}

	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse EC private key: %w", err)
	}

	hash := sha256.Sum256(msg)

	sig, err := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}
