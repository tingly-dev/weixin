// Package cdn provides CDN utilities for WeChat media upload/download.
package cdn

import (
	"testing"
)

// TestBuildUploadURL tests CDN upload URL building.
func TestBuildUploadURL(t *testing.T) {
	testCases := []struct {
		name        string
		cdnBaseURL  string
		uploadParam  string
		filekey     string
		expectedURL string
	}{
		{
			name:        "Basic",
			cdnBaseURL:  "https://cdn.example.com",
			uploadParam: "param123",
			filekey:     "filekey456",
			expectedURL: "https://cdn.example.com/upload?encrypted_query_param=param123&filekey=filekey456",
		},
		{
			name:        "With special chars",
			cdnBaseURL:  "https://cdn.example.com/",
			uploadParam: "param=with=equals",
			filekey:     "key/with/slash",
			expectedURL: "https://cdn.example.com/upload?encrypted_query_param=param=with%3Dequals&filekey=key%2Fwith%2Fslash",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildUploadURL(tc.cdnBaseURL, tc.uploadParam, tc.filekey)
			if result != tc.expectedURL {
				t.Errorf("BuildUploadURL() = %s, want %s", result, tc.expectedURL)
			}
		})
	}
}

// TestBuildDownloadURL tests CDN download URL building.
func TestBuildDownloadURL(t *testing.T) {
	testCases := []struct {
		name              string
		cdnBaseURL        string
	encryptedQueryParam string
		expectedURL       string
	}{
		{
			name:              "Basic",
			cdnBaseURL:        "https://cdn.example.com",
			encryptedQueryParam: "param123",
			expectedURL:       "https://cdn.example.com/download?encrypted_query_param=param123",
		},
		{
			name:              "With special chars",
			cdnBaseURL:        "https://cdn.example.com/",
			encryptedQueryParam: "param=with=equals",
			expectedURL:       "https://cdn.example.com/download?encrypted_query_param=param%3Dwith%3Dequals",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildDownloadURL(tc.encryptedQueryParam, tc.cdnBaseURL)
			if result != tc.expectedURL {
				t.Errorf("BuildDownloadURL() = %s, want %s", result, tc.expectedURL)
			}
		})
	}
}