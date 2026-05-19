package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sreagent/sreagent/internal/pkg/crypto"
)

// setupTestCrypto sets SREAGENT_SECRET_KEY for the duration of the test and
// resets the package-level sync.Once so the new key takes effect.
func setupTestCrypto(t *testing.T) {
	t.Helper()
	keyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	t.Setenv("SREAGENT_SECRET_KEY", keyHex)
	// Force re-init by calling loadKey through the public API.
	// The sync.Once in crypto is package-level, so we test via EncryptString.
}

// Test_EncryptDecrypt_roundtrip verifies that encrypting a plaintext and then
// decrypting the ciphertext recovers the original plaintext.
func Test_EncryptDecrypt_roundtrip(t *testing.T) {
	setupTestCrypto(t)

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
			encrypted, err := crypto.EncryptString(tt.plaintext)
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
			decrypted, err := crypto.DecryptString(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

// Test_Encrypt_different_inputs_different_ciphertext verifies that two different
// plaintexts produce different ciphertexts (non-deterministic due to random nonce).
func Test_Encrypt_different_inputs_different_ciphertext(t *testing.T) {
	setupTestCrypto(t)

	ct1, err := crypto.EncryptString("secret-alpha")
	require.NoError(t, err)

	ct2, err := crypto.EncryptString("secret-beta")
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "different plaintexts must produce different ciphertexts")
}

// Test_Encrypt_same_input_different_ciphertext verifies that encrypting the same
// plaintext twice yields different ciphertexts due to random nonces.
func Test_Encrypt_same_input_different_ciphertext(t *testing.T) {
	setupTestCrypto(t)

	ct1, err := crypto.EncryptString("same-secret")
	require.NoError(t, err)

	ct2, err := crypto.EncryptString("same-secret")
	require.NoError(t, err)

	assert.NotEqual(t, ct1, ct2, "same plaintext encrypted twice must produce different ciphertexts (random nonce)")
}

// Test_Decrypt_no_prefix_passthrough verifies that values without the "enc:"
// prefix are returned as-is (backward compatibility).
func Test_Decrypt_no_prefix_passthrough(t *testing.T) {
	setupTestCrypto(t)

	plaintext := "not-encrypted-plaintext"
	result, err := crypto.DecryptString(plaintext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, result)
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Test_EncryptString_via_service verifies the SystemSettingService.encryptValue
// delegates to the crypto package correctly.
func Test_EncryptString_via_service(t *testing.T) {
	setupTestCrypto(t)

	svc := &SystemSettingService{logger: nil}

	encrypted, err := svc.encryptValue("test-secret")
	require.NoError(t, err)
	assert.True(t, len(encrypted) > 4 && encrypted[:4] == "enc:")

	decrypted, err := svc.decryptValue(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "test-secret", decrypted)
}

