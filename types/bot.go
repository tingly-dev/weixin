// Package bot provides minimal shared bot functionality.
package types

// BaseBot provides shared bot functionality.
// Contains only metadata and capabilities - no adapter indirection.
type BaseBot struct {
	meta         *Meta
	capabilities *Capabilities
}

// NewBaseBot creates a new base bot.
func NewBaseBot(meta *Meta, capabilities *Capabilities) *BaseBot {
	return &BaseBot{
		meta:         meta,
		capabilities: capabilities,
	}
}

// Meta returns the bot metadata.
func (b *BaseBot) Meta() *Meta {
	return b.meta
}

// Capabilities returns the bot capabilities.
func (b *BaseBot) Capabilities() *Capabilities {
	return b.capabilities
}
