# WeChat API Package

This package provides WeChat bot API implementations.

## Features

### QR Code Terminal Display

The package includes built-in QR code terminal display functionality for WeChat login:

```go
import "github.com/tingly-dev/agentchannel/pkg/channels/wechat/api"

// Display QR code response (recommended)
qrResp, err := client.GetBotQRCode(ctx, botType)
if err != nil {
    return err
}

// Automatically displays QR code in terminal with nice formatting
err = api.DisplayQRCodeResponse(qrResp.Qrcode, qrResp.QrcodeImgContent)
if err != nil {
    log.Printf("Warning: Failed to display QR code: %v", err)
}
```

Or display any QR code data directly:

```go
// Display plain text as QR code
err := api.DisplayQRCodeInTerminal("your-qr-code-data", false)

// Display base64-encoded image as QR code (will decode and fallback to text)
err := api.DisplayQRCodeInTerminal(base64ImageData, true)
```

The QR code is rendered using ASCII characters in the terminal, making it easy to scan with WeChat mobile app.

### Dependencies

The QR code display feature uses:
- `github.com/mdp/qrterminal/v3` - Terminal QR code rendering
- `github.com/skip2/go-qrcode` - QR code generation

These are automatically included when you import the package.

## Example

See [examples/wechat-echo-bot](../../examples/wechat-echo-bot) for a complete example of using the QR code login flow with terminal display.
