// Package media provides media handling utilities for weixin.
package media

import (
	"path/filepath"
	"strings"
)

// MIMEType detection based on file extension or Content-Type.

var extensionToMIME = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".mkv":  "video/x-matroska",
	".webm": "video/webm",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".silk": "audio/silk",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".txt":  "text/plain",
	".zip":  "application/zip",
	".rar":  "application/x-rar-compressed",
	".7z":   "application/x-7z-compressed",
}

var mimeToExtension = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/bmp":       ".bmp",
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"audio/mpeg":      ".mp3",
	"audio/wav":       ".wav",
	"audio/ogg":       ".ogg",
	"audio/silk":      ".silk",
	"application/pdf": ".pdf",
	"text/plain":      ".txt",
}

// GetMIMEFromFilename returns MIME type based on file extension.
func GetMIMEFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mime, ok := extensionToMIME[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// GetExtensionFromContentType returns file extension based on Content-Type.
func GetExtensionFromContentType(contentType string) string {
	// Remove charset if present
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	if ext, ok := mimeToExtension[contentType]; ok {
		return ext
	}
	return ".bin"
}

// GetExtensionFromContentTypeOrURL tries to get extension from Content-Type first, then URL.
func GetExtensionFromContentTypeOrURL(contentType, url string) string {
	if contentType != "" {
		ext := GetExtensionFromContentType(contentType)
		if ext != ".bin" {
			return ext
		}
	}

	// Try to extract extension from URL
	if url != "" {
		ext := strings.ToLower(filepath.Ext(url))
		if ext != "" && extensionToMIME[ext] != "" {
			return ext
		}
	}

	return ".bin"
}

// IsImageMIME checks if the MIME type is an image.
func IsImageMIME(mime string) bool {
	return strings.HasPrefix(mime, "image/")
}

// IsVideoMIME checks if the MIME type is a video.
func IsVideoMIME(mime string) bool {
	return strings.HasPrefix(mime, "video/")
}

// IsAudioMIME checks if the MIME type is audio.
func IsAudioMIME(mime string) bool {
	return strings.HasPrefix(mime, "audio/")
}
