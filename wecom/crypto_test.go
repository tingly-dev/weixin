package wecom

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

// encryptForTest is the inverse of DecryptFile, used to build fixtures.
func encryptForTest(t *testing.T, plaintext []byte, key []byte, blockSize int) []byte {
	t.Helper()
	padding := blockSize - len(plaintext)%blockSize
	if padding == 0 {
		padding = blockSize
	}
	padded := append(append([]byte{}, plaintext...), makePadding(padding)...)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	mode := cipher.NewCBCEncrypter(block, key[:aes.BlockSize])
	encrypted := make([]byte, len(padded))
	mode.CryptBlocks(encrypted, padded)
	return encrypted
}

func makePadding(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(n)
	}
	return b
}

func randomAESKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("random key: %v", err)
	}
	return key
}

func TestDecryptFile_RoundTrip(t *testing.T) {
	key := randomAESKey(t)
	plaintext := []byte("hello wecom media content")
	encrypted := encryptForTest(t, plaintext, key, 32)

	decrypted, err := DecryptFile(encrypted, base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptFile_EmptyInput_NoPanic(t *testing.T) {
	key := randomAESKey(t)
	_, err := DecryptFile([]byte{}, base64.StdEncoding.EncodeToString(key))
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestDecryptFile_PaddingExceedsLength_NoPanic(t *testing.T) {
	key := randomAESKey(t)
	// One AES block (16 bytes) whose last byte claims 32 bytes of padding —
	// larger than the whole buffer. Must return an error, not panic.
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	plaintext := make([]byte, 16)
	plaintext[15] = 32
	mode := cipher.NewCBCEncrypter(block, key[:aes.BlockSize])
	encrypted := make([]byte, 16)
	mode.CryptBlocks(encrypted, plaintext)

	_, err = DecryptFile(encrypted, base64.StdEncoding.EncodeToString(key))
	if err == nil {
		t.Fatal("expected error for out-of-range padding, got nil")
	}
}

func TestDecryptFile_InvalidPaddingBytes(t *testing.T) {
	key := randomAESKey(t)
	// Valid-length padding claim, but the padding bytes themselves don't match.
	plaintext := make([]byte, 16)
	plaintext[15] = 4
	plaintext[14] = 9 // should be 4 to be valid PKCS#7
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	mode := cipher.NewCBCEncrypter(block, key[:aes.BlockSize])
	encrypted := make([]byte, 16)
	mode.CryptBlocks(encrypted, plaintext)

	_, err = DecryptFile(encrypted, base64.StdEncoding.EncodeToString(key))
	if err == nil {
		t.Fatal("expected error for invalid PKCS#7 padding, got nil")
	}
}
