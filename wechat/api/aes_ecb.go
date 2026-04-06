// Package crypto provides encryption utilities for WeChat media.
package api

import (
	"crypto/aes"
	"errors"
)

var (
	// ErrInvalidBlockSize is returned when the input is not a multiple of the block size.
	ErrInvalidBlockSize = errors.New("input is not a multiple of the block size")

	// ErrInvalidPKCS7 is returned when PKCS7 padding is invalid.
	ErrInvalidPKCS7 = errors.New("invalid PKCS7 padding")
)

// EncryptAesEcb encrypts plaintext using AES-128-ECB mode.
// key must be 16 bytes (AES-128).
func EncryptAesEcb(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(key) != 16 {
		return nil, errors.New("key must be 16 bytes for AES-128")
	}

	// Apply PKCS7 padding
	padded := pkcs7Pad(plaintext, aes.BlockSize)
	ciphertext := make([]byte, len(padded))

	// ECB mode: encrypt each block independently
	for i := 0; i < len(padded); i += aes.BlockSize {
		block.Encrypt(ciphertext[i:i+aes.BlockSize], padded[i:i+aes.BlockSize])
	}

	return ciphertext, nil
}

// DecryptAesEcb decrypts ciphertext using AES-128-ECB mode.
// key must be 16 bytes (AES-128).
func DecryptAesEcb(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, ErrInvalidBlockSize
	}

	plaintext := make([]byte, len(ciphertext))

	// ECB mode: decrypt each block independently
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(plaintext[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}

	// Remove PKCS7 padding
	return pkcs7Unpad(plaintext)
}

// AesEcbPaddedSize calculates the padded size for a given plaintext size.
func AesEcbPaddedSize(plaintextSize int) int {
	// PKCS7 padding: add padding bytes such that the total length is a multiple of block size
	// Each padding byte's value equals the number of padding bytes
	blockSize := aes.BlockSize
	padding := blockSize - (plaintextSize % blockSize)
	return plaintextSize + padding
}

// pkcs7Pad applies PKCS7 padding to the input.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// pkcs7Unpad removes PKCS7 padding from the input.
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrInvalidPKCS7
	}

	padding := int(data[len(data)-1])
	if padding < 1 || padding > aes.BlockSize {
		return nil, ErrInvalidPKCS7
	}

	// Verify all padding bytes are correct
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, ErrInvalidPKCS7
		}
	}

	return data[:len(data)-padding], nil
}
