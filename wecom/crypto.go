package wecom

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// generateReqID creates a unique request ID: "{prefix}_{timestamp_ms}_{random_hex}".
func generateReqID(prefix string) string {
	ts := time.Now().UnixMilli()
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s_%d_%s", prefix, ts, hex.EncodeToString(b))
}

// DecryptFile decrypts AES-256-CBC encrypted content.
// The key is a base64-encoded 32-byte AES key.
// Uses manual PKCS#7 unpadding with 32-byte block support.
func DecryptFile(encrypted []byte, aesKey string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(aesKey)
	if err != nil {
		// Try raw hex key
		key, err = hex.DecodeString(aesKey)
		if err != nil {
			return nil, fmt.Errorf("decode aes key: %w", err)
		}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}

	if len(encrypted)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("encrypted data is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, key[:aes.BlockSize])
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// Manual PKCS#7 unpadding with 32-byte block support.
	// WeCom uses a non-standard padding that may use 32-byte block size.
	padding := int(decrypted[len(decrypted)-1])
	if padding == 0 || padding > 32 {
		return nil, fmt.Errorf("invalid padding size: %d", padding)
	}

	// Verify padding bytes
	for i := 0; i < padding; i++ {
		if decrypted[len(decrypted)-1-i] != byte(padding) {
			return nil, fmt.Errorf("invalid PKCS#7 padding")
		}
	}

	return decrypted[:len(decrypted)-padding], nil
}

// parseFrameBody parses a raw JSON body into a typed struct.
func parseFrameBody(raw interface{}, target interface{}) error {
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshal frame body: %w", err)
	}
	return json.Unmarshal(data, target)
}

// encodeFrame serializes a WsFrame to JSON bytes for sending over WebSocket.
func encodeFrame(frame *WsFrame) ([]byte, error) {
	return json.Marshal(frame)
}

// readFrame reads and deserializes a single JSON frame from a reader.
func readFrame(r io.Reader) (*WsFrame, error) {
	decoder := json.NewDecoder(r)
	var frame WsFrame
	if err := decoder.Decode(&frame); err != nil {
		return nil, err
	}
	return &frame, nil
}
