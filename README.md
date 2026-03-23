# Weixin Chat Bot SDK

A WeChat Chat Bot SDK via Official Channel.

## Features

- **QR Code Login** - Easy account authorization via QR code scanning
- **Long-Polling** - Efficient message synchronization with server-side timeout
- **Rich Message Types** - Support for text, image, voice, file, and video messages
- **CDN Media Uploads** - AES-128-ECB encrypted media uploads/downloads
- **Account Management** - Persistent account storage and management
- **Plugin Architecture** - Flexible integration with AgentChannel

## Installation

```bash
go get github.com/tingly-dev/weixin
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/tingly-dev/weixin"
    "github.com/tingly-dev/weixin/api"
    "github.com/tingly-dev/weixin/message"
)

func main() {
    // Initialize plugin
    config := &weixin.WeChatConfig{
        BaseURL: "https://ilinkai.weixin.qq.com",
        BotType: "3",
    }
    plugin := weixin.NewPlugin(config)

    // Perform QR code login
    client := api.NewClient(config.BaseURL, "")
    qrResp, _ := client.GetBotQRCode(context.Background(), config.BotType)
    // Display QR code...
    statusResp, _ := client.GetQRStatus(context.Background(), qrResp.Qrcode)

    // Save account
    account := &weixin.WeChatAccount{
        ID:         statusResp.IlinkBotID,
        BotToken:   statusResp.BotToken,
        BotID:      statusResp.IlinkBotID,
        UserID:     statusResp.IlinkUserID,
        BaseURL:    statusResp.BaseURL,
        Enabled:    true,
        Configured: true,
    }
    plugin.Accounts().Save(account)

    // Send message
    client = api.NewClient(account.BaseURL, account.BotToken)
    items := []weixin.MessageItem{
        message.BuildTextItem("Hello!"),
    }
    client.SendMessage(context.Background(), "userid", "context_token", items)
}
```

## Documentation

For detailed architecture and protocol documentation, see:
- [understand-tencent-weixin-openclaw-weixin](https://github.com/FFengIll/understand-tencent-weixin-openclaw-weixin)

## Example

A complete echo bot example is available in `example/weixin-echo-bot/`:

```bash
cd example/weixin-echo-bot
go run main.go
```

## License

MIT