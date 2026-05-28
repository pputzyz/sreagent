package crypto

import (
	"os"
	"sync"
	"testing"
)

// setTestKey sets the SREAGENT_SECRET_KEY env var and resets the sync.Once
// so that loadKey() picks up the new value for this test.
func setTestKey(key string) {
	os.Setenv("SREAGENT_SECRET_KEY", key)
	masterKey = nil
	keyOnce = sync.Once{}
}

// cleanupKey unsets the env var and resets state so other tests start fresh.
func cleanupKey() {
	os.Unsetenv("SREAGENT_SECRET_KEY")
	masterKey = nil
	keyOnce = sync.Once{}
}

// validHexKey is a 64-char hex string = 32 bytes for AES-256.
const validHexKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	plaintext := `{"username":"admin","password":"s3cret"}`

	encrypted, err := EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	if encrypted == plaintext {
		t.Fatal("encrypted text should differ from plaintext")
	}

	if !IsEncrypted(encrypted) {
		t.Fatal("IsEncrypted should return true for encrypted text")
	}

	decrypted, err := DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptString_NoKey_ReturnsError(t *testing.T) {
	cleanupKey()

	_, err := EncryptString("test")
	if err == nil {
		t.Fatal("expected error when SREAGENT_SECRET_KEY is not set")
	}
}

func TestDecryptString_NoKey_ReturnsError(t *testing.T) {
	cleanupKey()

	// Decrypting an enc: prefixed value without a key should error.
	_, err := DecryptString("enc:YWJj")
	if err == nil {
		t.Fatal("expected error when SREAGENT_SECRET_KEY is not set")
	}
}

func TestDecryptString_InvalidCiphertext(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	_, err := DecryptString("enc:garbage_data_that_is_too_short")
	if err == nil {
		t.Fatal("expected error for invalid ciphertext")
	}
}

func TestDecryptString_InvalidBase64(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	_, err := DecryptString("enc:!!!not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptString_PlaintextPassthrough(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	// If input is not encrypted (no "enc:" prefix), should return as-is.
	plaintext := `{"username":"admin"}`
	result, err := DecryptString(plaintext)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}
	if result != plaintext {
		t.Fatalf("expected plaintext passthrough, got %q", result)
	}
}

func TestDecryptString_WrongKey(t *testing.T) {
	// Encrypt with one key, decrypt with another -- should fail.
	setTestKey(validHexKey)

	encrypted, err := EncryptString("secret-value")
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	// Switch to a different valid key.
	differentKey := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	setTestKey(differentKey)

	_, err = DecryptString(encrypted)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}

	cleanupKey()
}

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"enc:abc123", true},
		{"enc:", true},
		{"not_encrypted", false},
		{"", false},
		{"enc", false},
		{"ENC:abc", false},
	}
	for _, tt := range tests {
		got := IsEncrypted(tt.input)
		if got != tt.want {
			t.Errorf("IsEncrypted(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	encrypted, err := EncryptString("")
	if err != nil {
		t.Fatalf("EncryptString('') failed: %v", err)
	}

	// Empty plaintext should return empty string, not encrypted form.
	if encrypted != "" {
		t.Fatalf("expected empty encrypted string, got %q", encrypted)
	}

	decrypted, err := DecryptString("")
	if err != nil {
		t.Fatalf("DecryptString('') failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("expected empty string, got %q", decrypted)
	}
}

func TestEncryptString_BadKeyLength(t *testing.T) {
	setTestKey("tooshort")
	defer cleanupKey()

	_, err := EncryptString("test")
	if err == nil {
		t.Fatal("expected error for bad key length")
	}
}

func TestEncryptString_NonHexKey(t *testing.T) {
	// 64 chars but not valid hex.
	setTestKey("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	defer cleanupKey()

	_, err := EncryptString("test")
	if err == nil {
		t.Fatal("expected error for non-hex key")
	}
}

func TestEncryptDeterminism_DifferentCiphertexts(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	// AES-GCM uses random nonces, so encrypting the same plaintext twice
	// should produce different ciphertexts.
	enc1, err := EncryptString("same-input")
	if err != nil {
		t.Fatalf("first EncryptString failed: %v", err)
	}
	enc2, err := EncryptString("same-input")
	if err != nil {
		t.Fatalf("second EncryptString failed: %v", err)
	}

	if enc1 == enc2 {
		t.Fatal("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}

	// Both should decrypt to the same value.
	for i, enc := range []string{enc1, enc2} {
		dec, err := DecryptString(enc)
		if err != nil {
			t.Fatalf("DecryptString #%d failed: %v", i+1, err)
		}
		if dec != "same-input" {
			t.Fatalf("DecryptString #%d = %q, want %q", i+1, dec, "same-input")
		}
	}
}

func TestEncryptDecrypt_UnicodeContent(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	plaintext := "Hello, 世界! 🔑 Données chiffrées"

	encrypted, err := EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	decrypted, err := DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_LongValue(t *testing.T) {
	setTestKey(validHexKey)
	defer cleanupKey()

	// Build a long plaintext (~10 KB).
	long := make([]byte, 10000)
	for i := range long {
		long[i] = 'A' + byte(i%26)
	}
	plaintext := string(long)

	encrypted, err := EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	decrypted, err := DecryptString(encrypted)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted length = %d, want %d", len(decrypted), len(plaintext))
	}
}
