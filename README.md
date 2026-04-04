# Weixin Chat Bot SDK

A comprehensive Go SDK for building WeChat and WeCom (Enterprise WeChat) bots via official channels.

## Features

### WeChat Bot
- **QR Code Login** - Easy account authorization via QR code scanning
- **Long-Polling** - Efficient message synchronization with server-side timeout
- **Rich Message Types** - Support for text, image, voice, file, and video messages
- **CDN Media Uploads** - AES-128-ECB encrypted media uploads/downloads
- **Account Management** - Persistent account storage and management
- **Block Streaming** - Simulated streaming via multiple message sends

### WeCom Bot
- **WebSocket Real-time** - Push-based message delivery via WebSocket
- **True Streaming** - Native streaming text response support
- **Interactive Cards** - Template cards with buttons and interactive elements
- **Rich Media** - Image, file, and video with AES-256-CBC encryption
- **Event Handling** - Support for chat events, card clicks, and feedback
- **Auto Reconnection** - Automatic reconnection with heartbeat monitoring

## Installation

```bash
go get github.com/tingly-dev/weixin
```

## Quick Start

### WeChat Bot Example

```go
package main

import (
    "context"
    "log"

    "github.com/tingly-dev/weixin/wechat"
    "github.com/tingly-dev/weixin/types"
)

func main() {
    // Initialize WeChat bot
    config := &wechat.WeChatConfig{
        BaseURL: "https://ilinkai.weixin.qq.com",
        BotType: "3",
    }
    bot := wechat.NewBot(config)

    // QR code login
    account, err := bot.Pair().Login(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Start message polling
    if err := bot.Gateway().Start(account); err != nil {
        log.Fatal(err)
    }

    // Send message
    msg := &types.OutboundMessage{
        ToUser: "userid",
        Items: []types.MessageItem{
            types.BuildTextItem("Hello from WeChat!"),
        },
    }
    bot.Actions().Send(context.Background(), account, msg)
}
```

### WeCom Bot Example

```go
package main

import (
    "context"
    "log"

    "github.com/tingly-dev/weixin/wecom"
    "github.com/tingly-dev/weixin/types"
)

func main() {
    // Initialize WeCom bot
    config := &wecom.WecomConfig{
        BotID:     "your_bot_id",
        BotSecret: "your_bot_secret",
    }
    bot := wecom.NewBot(config)

    // Start WebSocket connection
    account := &types.Account{
        ID:     config.BotID,
        Token:  config.BotSecret,
        Enabled: true,
    }
    if err := bot.Gateway().Start(account); err != nil {
        log.Fatal(err)
    }

    // Send message
    msg := &types.OutboundMessage{
        ToUser: "user_id",
        Items: []types.MessageItem{
            types.BuildTextItem("Hello from WeCom!"),
        },
    }
    bot.Actions().Send(context.Background(), account, msg)
}
```

## Architecture

The SDK uses an adapter pattern to provide a unified interface across different WeChat protocols:

```
types/          - Shared type definitions and interfaces
  ├── adapter.go - Core adapter interfaces
  └── types.go  - Message types and account models

wechat/         - WeChat ilink protocol (HTTP long-polling)
  ├── bot.go    - WeChat bot implementation
  └── pairing/  - QR code login flow

wecom/          - WeCom AI Bot (WebSocket)
  └── bot.go    - WeCom bot implementation

api/            - Low-level WeChat HTTP client
message/        - Message conversion and processing
storage/        - File system persistence
```

## Adapter Interfaces

The SDK provides six core adapters for bot operations:

| Adapter | Description | WeChat | WeCom |
|---------|-------------|--------|-------|
| **ConfigAdapter** | Account configuration | ✓ | ✓ |
| **ActionsAdapter** | Message sending (Send, SendStream, SendMedia) | ✓ | ✓ |
| **GatewayAdapter** | Connection lifecycle (Start/Stop) | ✓ | ✓ |
| **LongPollAdapter** | Message synchronization | ✓ | - |
| **UploadAdapter** | Media upload to CDN | ✓ | ✓ |
| **PairingAdapter** | QR code login flow | ✓ | - |

## Message Types

- **Text** - Plain text messages
- **Image** - Images with thumbnail support
- **Voice** - Audio messages (SILK format for WeChat)
- **File** - File attachments
- **Video** - Video with thumbnail support
- **Markdown** - Rich text formatting (WeCom)
- **Template Cards** - Interactive cards with buttons (WeCom)

## Examples

Complete examples are available in the `example/` directory:

```bash
# WeChat echo bot
cd example/weixin-echo-bot
go run main.go

# WeChat streaming bot
cd example/weixin-stream-bot
go run main.go

# WeCom echo bot
cd example/wecom-echo-bot
go run main.go
```

## Protocol Documentation

For detailed architecture and protocol documentation, see:
- [understand-tencent-weixin-openclaw-weixin](https://github.com/FFengIll/understand-tencent-weixin-openclaw-weixin)

## License

MIT
