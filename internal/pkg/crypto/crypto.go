// Package crypto provides AES-256-GCM encryption helpers for sensitive fields at rest.
// The master key is loaded from the SREAGENT_SECRET_KEY environment variable (64-char hex = 32 bytes).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	// EncPrefix is prepended to AES-GCM ciphertext stored in the DB.
	// Format: "enc:" + base64(12-byte nonce + ciphertext)
	EncPrefix = "enc:"
)

var (
	masterKey []byte
	keyOnce   sync.Once
)

// loadKey loads the master key from SREAGENT_SECRET_KEY (once).
func loadKey() []byte {
	keyOnce.Do(func() {
		keyHex := os.Getenv("SREAGENT_SECRET_KEY")
		if keyHex == "" {
			fmt.Fprintf(os.Stderr, "[crypto] WARNING: SREAGENT_SECRET_KEY not set — encryption disabled, sensitive data will be stored in plaintext\n")
			return
		}
		key, err := hex.DecodeString(keyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[crypto] WARNING: SREAGENT_SECRET_KEY is not valid hex — encryption disabled\n")
			return
		}
		if len(key) != 32 {
			fmt.Fprintf(os.Stderr, "[crypto] WARNING: SREAGENT_SECRET_KEY must be 32 bytes (64 hex chars), got %d bytes — encryption disabled\n", len(key))
			return
		}
		masterKey = key
	})
	return masterKey
}

// IsEncrypted returns true if the value has the enc: prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, EncPrefix)
}

// EncryptString encrypts a plaintext string using AES-256-GCM.
// Returns "enc:<base64(nonce+ciphertext)>" or the original value if no key is configured.
func EncryptString(plaintext string) (string, error) {
	key := loadKey()
	if len(key) == 0 || plaintext == "" {
		return plaintext, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return EncPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a value encrypted by EncryptString.
// Values not starting with EncPrefix are returned as-is (backward compatible with plaintext).
func DecryptString(value string) (string, error) {
	key := loadKey()
	if len(key) == 0 || !strings.HasPrefix(value, EncPrefix) {
		return value, nil
	}

	data, err := base64.StdEncoding.DecodeString(value[len(EncPrefix):])
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", io.ErrUnexpectedEOF
	}

	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
