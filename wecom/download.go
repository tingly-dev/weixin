package wecom

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Downloader handles HTTP file downloads for WeCom media.
type Downloader struct {
	httpClient *http.Client
}

// NewDownloader creates a new downloader.
func NewDownloader() *Downloader {
	return &Downloader{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// DownloadFile downloads a file from a URL and optionally decrypts it.
func (d *Downloader) DownloadFile(ctx context.Context, url string, aesKey string) (*DownloadResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	result := &DownloadResult{
		Buffer:   data,
		FileName: parseFilename(resp),
	}

	// Decrypt if aes_key is provided
	if aesKey != "" {
		decrypted, err := DecryptFile(data, aesKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt: %w", err)
		}
		result.Buffer = decrypted
		result.Decrypted = true
	}

	return result, nil
}

// DownloadResult contains the result of a file download.
type DownloadResult struct {
	Buffer    []byte
	FileName  string
	Decrypted bool
}

// parseFilename extracts the filename from Content-Disposition header.
func parseFilename(resp *http.Response) string {
	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return ""
	}

	// Simple parser for filename="..."
	for i := 0; i < len(cd)-9; i++ {
		if cd[i:i+9] == "filename=" {
			start := i + 9
			if start < len(cd) && cd[start] == '"' {
				start++
				end := start
				for end < len(cd) && cd[end] != '"' {
					end++
				}
				return cd[start:end]
			}
			// Unquoted filename
			end := start
			for end < len(cd) && cd[end] != ';' && cd[end] != ' ' {
				end++
			}
			return cd[start:end]
		}
	}

	return ""
}
