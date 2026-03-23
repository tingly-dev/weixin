// Package storage provides persistence utilities for WeChat channel state.
package storage

import (
	"os"
	"path/filepath"
)

// GetStateDir returns the state directory for storing WeChat channel data.
func GetStateDir() (string, error) {
	// Check for custom state directory from environment
	if customDir := os.Getenv("OPENCLAW_STATE_DIR"); customDir != "" {
		return customDir, nil
	}

	// Default to ~/.agentchannel/state
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	stateDir := filepath.Join(homeDir, ".agentchannel", "state")
	return stateDir, nil
}

// GetWeixinStateDir returns the WeChat-specific state directory.
func GetWeixinStateDir() (string, error) {
	stateDir, err := GetStateDir()
	if err != nil {
		return "", err
	}

	weixinDir := filepath.Join(stateDir, "weixin")
	return weixinDir, nil
}

// EnsureDir ensures a directory exists, creating it if necessary.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0700)
}

// WriteFileAtomic writes data to a file atomically.
func WriteFileAtomic(path string, data []byte) error {
	// Write to temporary file
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	// Rename to final path (atomic on POSIX)
	return os.Rename(tmpPath, path)
}