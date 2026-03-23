// Package crypto provides encryption utilities for WeChat media.
package crypto

import (
	"bytes"
	"testing"
)

// TestEncryptAesEcb tests AES-128-ECB encryption.
func TestEncryptAesEcb(t *testing.T) {
	key := []byte("1234567890123456") // 16 bytes
	plaintext := []byte("Hello, WeChat!")

	ciphertext, err := EncryptAesEcb(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAesEcb failed: %v", err)
	}

	// Ciphertext should be longer due to PKCS7 padding
	if len(ciphertext) <= len(plaintext) {
		t.Errorf("Expected ciphertext to be longer than plaintext, got %d vs %d", len(ciphertext), len(plaintext))
	}

	// Ciphertext should be a multiple of block size (16)
	if len(ciphertext)%16 != 0 {
		t.Errorf("Expected ciphertext length to be multiple of 16, got %d", len(ciphertext))
	}
}

// TestDecryptAesEcb tests AES-128-ECB decryption.
func TestDecryptAesEcb(t *testing.T) {
	key := []byte("1234567890123456")
	plaintext := []byte("Hello, WeChat!")

	ciphertext, err := EncryptAesEcb(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptAesEcb failed: %v", err)
	}

	decrypted, err := DecryptAesEcb(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptAesEcb failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decryption failed: got %q, want %q", decrypted, plaintext)
	}
}

// TestEncryptDecryptRoundTrip tests round-trip encryption/decryption.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := []byte("1234567890123456")
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"Single byte", []byte{42}},
		{"Multiple blocks", []byte("This is a longer test that spans multiple AES blocks.")},
		{"Exactly one block", bytes.Repeat([]byte("X"), 15)},
		{"Exactly one block plus padding", bytes.Repeat([]byte("X"), 16)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, err := EncryptAesEcb(tc.data, key)
			if err != nil {
				t.Fatalf("EncryptAesEcb failed: %v", err)
			}

			decrypted, err := DecryptAesEcb(ciphertext, key)
			if err != nil {
				t.Fatalf("DecryptAesEcb failed: %v", err)
			}

			if !bytes.Equal(tc.data, decrypted) {
				t.Errorf("Round trip failed: got %q, want %q", decrypted, tc.data)
			}
		})
	}
}

// TestEncryptInvalidKeyLength tests encryption with invalid key length.
func TestEncryptInvalidKeyLength(t *testing.T) {
	plaintext := []byte("test")
	invalidKeys := [][]byte{
		{},                    // empty
		{1},                   // 1 byte
		{1, 2, 3},             // 3 bytes
		bytes.Repeat([]byte{0}, 15), // 15 bytes
		bytes.Repeat([]byte{0}, 17), // 17 bytes
	}

	for _, key := range invalidKeys {
		t.Run(fmt.Sprintf("keylen_%d", len(key)), func(t *testing.T) {
			_, err := EncryptAesEcb(plaintext, key)
			if err == nil {
				t.Error("Expected error for invalid key length, got nil")
			}
		})
	}
}

// TestAesEcbPaddedSize tests PKCS7 padding size calculation.
func TestAesEcbPaddedSize(t *testing.T) {
	testCases := []struct {
		plaintextSize int
		expectedSize  int
	}{
		{0, 16},
		{1, 16},
		{15, 16},
		{16, 32},
		{17, 32},
		{31, 32},
		{32, 48},
		{100, 112},
		{255, 256},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("size_%d", tc.plaintextSize), func(t *testing.T) {
			result := AesEcbPaddedSize(tc.plaintextSize)
			if result != tc.expectedSize {
				t.Errorf("AesEcbPaddedSize(%d) = %d, want %d",
					tc.plaintextSize, result, tc.expectedSize)
			}
		})
	}
}

// TestDecryptInvalidBlockSize tests decryption with invalid block size.
func TestDecryptInvalidBlockSize(t *testing.T) {
	key := []byte("1234567890123456")
	invalidCiphertexts := [][]byte{
		{},                   // empty
		{1, 2},               // 2 bytes
		{1, 2, 3},            // 3 bytes
		bytes.Repeat([]byte{0}, 15), // 15 bytes
	}

	for _, ciphertext := range invalidCiphertexts {
		t.Run(fmt.Sprintf("len_%d", len(ciphertext)), func(t *testing.T) {
			_, err := DecryptAesEcb(ciphertext, key)
			if err != ErrInvalidBlockSize {
				t.Errorf("Expected ErrInvalidBlockSize, got %v", err)
			}
		})
	}
}

// TestDecryptInvalidPadding tests decryption with invalid PKCS7 padding.
func TestDecryptInvalidPadding(t *testing.T) {
	key := []byte("1234567890123456")
	plaintext := []byte("test")
	ciphertext, _ := EncryptAesEcb(plaintext, key)

	// Corrupt the padding bytes
	ciphertext[len(ciphertext)-1] = 255

	_, err := DecryptAesEcb(ciphertext, key)
	if err != ErrInvalidPKCS7 {
		t.Errorf("Expected ErrInvalidPKCS7, got %v", err)
	}
}