package service

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTestEncryptionService creates a SystemSettingService with a known master key
// for testing encrypt/decrypt methods. The repo is nil since these tests do not
// touch the database.
func newTestEncryptionService(t *testing.T) *SystemSettingService {
	t.Helper()
	// 32-byte key encoded as 64-char hex.
	keyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	key, err := hex.DecodeString(keyHex)
	require.NoError(t, err)
	return &SystemSettingService{
		masterKey: key,
		logger:    zap.NewNop(),
	}
}

// Test_EncryptDecrypt_roundtrip verifies that encrypting a plaintext and then
// decrypting the ciphertext recovers the original plaintext.
func Test_EncryptDecrypt_roundtrip(t *testing.T) {
	svc := newTestEncryptionService(t)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple string", "hello world"},
		{"api key", "sk-proj-abc123def456ghi789"},
		{"empty string", ""},
		{"unicode", "你好世界"},
		{"special chars", "p@$$w0rd!#%^&*()"},
		{"long string", "a]b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := svc.encryptValue(tt.plaintext)
			require.NoError(t, err)

			// Empty plaintext returns empty (no encryption needed).
			if tt.plaintext == "" {
				assert.Equal(t, "", encrypted)
				return
			}

			// Encrypted value must start with "enc:" prefix.
			assert.True(t, len(encrypted) > 4 && encrypted[:4] == "enc:",
				"encrypted value should start with 'enc:' prefix, got: %q", encrypted[:min(20, len(encrypted))])

			// Decrypt back.
			decrypted, err := svc.decryptValue(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

// Test_Encrypt_different_inputs_different_ciphertext verifies that two different
// plaintexts produce different ciphertexts (non-deterministic due to random nonce).
func Test_Encrypt_different_inputs_different_ciphertext(t *testing.T) {
	svc := newTestEncryptionService(t)

	ct1, err := svc.encryptValue("secret-alpha")
	require.NoError(t, err)

	ct2, err := svc.encryptValue("secret-beta")
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "different plaintexts must produce different ciphertexts")
}

// Test_Encrypt_same_input_different_ciphertext verifies that encrypting the same
// plaintext twice yields different ciphertexts due to random nonces.
func Test_Encrypt_same_input_different_ciphertext(t *testing.T) {
	svc := newTestEncryptionService(t)

	ct1, err := svc.encryptValue("same-secret")
	require.NoError(t, err)

	ct2, err := svc.encryptValue("same-secret")
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "same plaintext encrypted twice must produce different ciphertexts (random nonce)")
}

// Test_Decrypt_no_prefix_passthrough verifies that values without the "enc:"
// prefix are returned as-is (backward compatibility).
func Test_Decrypt_no_prefix_passthrough(t *testing.T) {
	svc := newTestEncryptionService(t)

	plaintext := "not-encrypted-plaintext"
	result, err := svc.decryptValue(plaintext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, result)
}

// Test_Encrypt_no_key_passthrough verifies that when masterKey is nil,
// encryptValue returns the plaintext unchanged.
func Test_Encrypt_no_key_passthrough(t *testing.T) {
	svc := &SystemSettingService{
		masterKey: nil,
		logger:    zap.NewNop(),
	}

	result, err := svc.encryptValue("some-secret")
	require.NoError(t, err)
	assert.Equal(t, "some-secret", result)
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
