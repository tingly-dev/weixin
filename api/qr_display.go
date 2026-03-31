// Package api provides WeChat API implementations.
package api

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
)

// DisplayQRCodeInTerminal displays a QR code in the terminal.
// It supports both plain text URLs and base64-encoded image data.
//
// Parameters:
//   - data: Either a plain URL string or base64-encoded image data
//   - isBase64Image: If true, treats data as base64-encoded PNG/JPEG image
//
// For WeChat login, the QR code data is typically a simple string (not an HTTP URL),
// but it can still be rendered as a QR code for scanning.
func DisplayQRCodeInTerminal(data string, isBase64Image bool) error {
	if data == "" {
		return fmt.Errorf("QR code data is empty")
	}

	// If data is base64 image, we need to decode it first
	// However, for WeChat's case, we typically receive a simple qrcode string
	// that needs to be encoded into a QR image
	if isBase64Image && len(data) > 100 {
		// Try to decode base64 image
		imgData, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return fmt.Errorf("decode base64 image: %w", err)
		}

		// For now, just inform that we received image data
		// Most terminals can't display images directly
		fmt.Printf("Received QR code image (%d bytes)\n", len(imgData))
		fmt.Println("Falling back to text QR code generation...")
	}

	// Generate QR code in terminal
	// Use qrterminal to render the QR code with ASCII characters
	config := qrterminal.Config{
		Level:     qrterminal.M,
		Writer:    os.Stdout,
		BlackChar: qrterminal.WHITE,
		WhiteChar: qrterminal.BLACK,
		QuietZone: 1,
	}

	qrterminal.GenerateWithConfig(data, config)
	return nil
}

// DisplayQRCodeResponse is a convenience function for displaying QR code from API response.
// It automatically handles both qrcode string and base64 image content.
func DisplayQRCodeResponse(qrcode string, imgContent string) error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("SCAN THIS QR CODE WITH WECHAT:")
	fmt.Println(strings.Repeat("=", 60))

	// Try to display the QR code in terminal
	if err := DisplayQRCodeInTerminal(qrcode, false); err != nil {
		// Fallback: just print the QR code string
		fmt.Printf("\nQR Code Data: %s\n", qrcode)
		fmt.Println("(Failed to generate terminal QR code)")
		return err
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("QR Code: %s\n", qrcode)

	if imgContent != "" {
		fmt.Printf("QR Code Image (base64): %d bytes\n", len(imgContent))
	}

	fmt.Println(strings.Repeat("=", 60) + "\n")
	return nil
}
