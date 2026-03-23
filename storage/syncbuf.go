// Package storage provides persistence utilities for WeChat channel state.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetSyncBufFilePath returns the path to the sync buffer file for an account.
func GetSyncBufFilePath(accountID string) (string, error) {
	weixinDir, err := GetWeixinStateDir()
	if err != nil {
		return "", fmt.Errorf("get state dir: %w", err)
	}

	if err := EnsureDir(weixinDir); err != nil {
		return "", fmt.Errorf("ensure dir: %w", err)
	}

	return filepath.Join(weixinDir, accountID+"-syncbuf.dat"), nil
}

// LoadSyncBuf loads the sync buffer for an account.
// Returns empty string if file doesn't exist.
func LoadSyncBuf(accountID string) (string, error) {
	path, err := GetSyncBufFilePath(accountID)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	return string(data), nil
}

// SaveSyncBuf saves the sync buffer for an account.
func SaveSyncBuf(accountID, buf string) error {
	path, err := GetSyncBufFilePath(accountID)
	if err != nil {
		return err
	}

	return WriteFileAtomic(path, []byte(buf))
}
