package seedancem

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
)

func DecryptVideoContent(encrypted []byte, encryptedDEK string, privateKeyPEM string) ([]byte, error) {
	privateKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	unescaped, err := url.QueryUnescape(strings.TrimSpace(encryptedDEK))
	if err != nil {
		return nil, fmt.Errorf("decode encrypted key: %w", err)
	}
	encryptedKey, err := base64.StdEncoding.DecodeString(unescaped)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted key base64: %w", err)
	}
	dek, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt encrypted key: %w", err)
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("encrypted content is too short")
	}
	nonce := encrypted[:nonceSize]
	ciphertext := encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt content: %w", err)
	}
	return plaintext, nil
}

func parseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(privateKeyPEM)))
	if block == nil {
		return nil, fmt.Errorf("decode private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		if pkcs1, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes); pkcs1Err == nil {
			return pkcs1, nil
		}
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return rsaKey, nil
}
