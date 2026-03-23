// Package channel provides the main Channel interface and metadata.
package channel

import (
	"context"
	"time"
)

// Meta represents metadata about a channel.
type Meta struct {
	// ID is the unique channel identifier.
	ID ChannelID `json:"id"`

	// Label is the human-readable name.
	Label string `json:"label"`

	// SelectionLabel is the name shown in selection UIs.
	SelectionLabel string `json:"selectionLabel"`

	// DetailLabel is a shorter label for details views.
	DetailLabel string `json:"detailLabel,omitempty"`

	// DocsPath is the path to documentation.
	DocsPath string `json:"docsPath"`

	// DocsLabel is the label for docs links.
	DocsLabel string `json:"docsLabel,omitempty"`

	// Blurb is a short description.
	Blurb string `json:"blurb"`

	// Order determines sorting order (lower is first).
	Order int `json:"order,omitempty"`

	// Aliases are alternative identifiers.
	Aliases []string `json:"aliases,omitempty"`

	// SystemImage is the SF Symbols icon name (macOS).
	SystemImage string `json:"systemImage,omitempty"`

	// Version of the channel implementation.
	Version string `json:"version,omitempty"`
}

// Plugin is the main interface for a messaging channel plugin.
//
// A Plugin represents a messaging platform (Telegram, Discord, Slack, etc.)
// and provides all the functionality needed to send and receive messages
// through that platform.
type Plugin interface {
	// ID returns the unique channel identifier.
	ID() ChannelID

	// Meta returns metadata about the channel.
	Meta() *Meta

	// Capabilities returns the channel's supported features.
	Capabilities() *Capabilities

	// Config returns the configuration adapter (required).
	Config() ConfigAdapter

	// Actions returns the message actions adapter.
	Actions() ActionsAdapter

	// Outbound returns the outbound adapter.
	Outbound() OutboundAdapter

	// Messaging returns the messaging adapter.
	Messaging() MessagingAdapter

	// Gateway returns the gateway adapter.
	Gateway() GatewayAdapter

	// Security returns the security adapter.
	Security() SecurityAdapter

	// Directory returns the directory adapter.
	Directory() DirectoryAdapter

	// Status returns the status adapter.
	Status() StatusAdapter

	// Pairing returns the pairing adapter.
	Pairing() PairingAdapter

	// Group returns the group adapter.
	Group() GroupAdapter

	// Threading returns the threading adapter.
	Threading() ThreadingAdapter

	// Upload returns the upload adapter (optional).
	Upload() UploadAdapter

	// LongPoll returns the long-polling adapter (optional).
	LongPoll() LongPollAdapter
}

// BasePlugin provides a default implementation of the Plugin interface.
//
// Channel implementations can embed BasePlugin and only override
// the methods they need.
type BasePlugin struct {
	meta         *Meta
	capabilities *Capabilities
	config       ConfigAdapter
	actions      ActionsAdapter
	outbound     OutboundAdapter
	messaging    MessagingAdapter
	gateway      GatewayAdapter
	security     SecurityAdapter
	directory    DirectoryAdapter
	status       StatusAdapter
	pairing      PairingAdapter
	group        GroupAdapter
	threading    ThreadingAdapter
	upload       UploadAdapter
	longPoll     LongPollAdapter
}

// NewBasePlugin creates a new base plugin.
func NewBasePlugin(meta *Meta, capabilities *Capabilities, config ConfigAdapter) *BasePlugin {
	return &BasePlugin{
		meta:         meta,
		capabilities: capabilities,
		config:       config,
	}
}

// ID returns the channel ID.
func (p *BasePlugin) ID() ChannelID {
	return p.meta.ID
}

// Meta returns the channel metadata.
func (p *BasePlugin) Meta() *Meta {
	return p.meta
}

// Capabilities returns the channel capabilities.
func (p *BasePlugin) Capabilities() *Capabilities {
	return p.capabilities
}

// Config returns the config adapter.
func (p *BasePlugin) Config() ConfigAdapter {
	return p.config
}

// Actions returns the actions adapter.
func (p *BasePlugin) Actions() ActionsAdapter {
	return p.actions
}

// Outbound returns the outbound adapter.
func (p *BasePlugin) Outbound() OutboundAdapter {
	return p.outbound
}

// Messaging returns the messaging adapter.
func (p *BasePlugin) Messaging() MessagingAdapter {
	return p.messaging
}

// Gateway returns the gateway adapter.
func (p *BasePlugin) Gateway() GatewayAdapter {
	return p.gateway
}

// Security returns the security adapter.
func (p *BasePlugin) Security() SecurityAdapter {
	return p.security
}

// Directory returns the directory adapter.
func (p *BasePlugin) Directory() DirectoryAdapter {
	return p.directory
}

// Status returns the status adapter.
func (p *BasePlugin) Status() StatusAdapter {
	return p.status
}

// Pairing returns the pairing adapter.
func (p *BasePlugin) Pairing() PairingAdapter {
	return p.pairing
}

// Group returns the group adapter.
func (p *BasePlugin) Group() GroupAdapter {
	return p.group
}

// Threading returns the threading adapter.
func (p *BasePlugin) Threading() ThreadingAdapter {
	return p.threading
}

// Upload returns the upload adapter.
func (p *BasePlugin) Upload() UploadAdapter {
	return p.upload
}

// LongPoll returns the long-polling adapter.
func (p *BasePlugin) LongPoll() LongPollAdapter {
	return p.longPoll
}

// SetActions sets the actions adapter.
func (p *BasePlugin) SetActions(actions ActionsAdapter) {
	p.actions = actions
}

// SetOutbound sets the outbound adapter.
func (p *BasePlugin) SetOutbound(outbound OutboundAdapter) {
	p.outbound = outbound
}

// SetMessaging sets the messaging adapter.
func (p *BasePlugin) SetMessaging(messaging MessagingAdapter) {
	p.messaging = messaging
}

// SetGateway sets the gateway adapter.
func (p *BasePlugin) SetGateway(gateway GatewayAdapter) {
	p.gateway = gateway
}

// SetSecurity sets the security adapter.
func (p *BasePlugin) SetSecurity(security SecurityAdapter) {
	p.security = security
}

// SetDirectory sets the directory adapter.
func (p *BasePlugin) SetDirectory(directory DirectoryAdapter) {
	p.directory = directory
}

// SetStatus sets the status adapter.
func (p *BasePlugin) SetStatus(status StatusAdapter) {
	p.status = status
}

// SetPairing sets the pairing adapter.
func (p *BasePlugin) SetPairing(pairing PairingAdapter) {
	p.pairing = pairing
}

// SetGroup sets the group adapter.
func (p *BasePlugin) SetGroup(group GroupAdapter) {
	p.group = group
}

// SetThreading sets the threading adapter.
func (p *BasePlugin) SetThreading(threading ThreadingAdapter) {
	p.threading = threading
}

// SetUpload sets the upload adapter.
func (p *BasePlugin) SetUpload(upload UploadAdapter) {
	p.upload = upload
}

// SetLongPoll sets the long-polling adapter.
func (p *BasePlugin) SetLongPoll(longPoll LongPollAdapter) {
	p.longPoll = longPoll
}

// ChannelInstance represents a running channel instance with an account.
type ChannelInstance struct {
	Plugin    Plugin
	AccountID string
	StartedAt time.Time
	StoppedAt *time.Time
	Running   bool
	abortChan chan struct{}
}

// Start starts the channel instance.
func (ci *ChannelInstance) Start(ctx context.Context) error {
	gateway := ci.Plugin.Gateway()
	if gateway == nil {
		return nil
	}

	ci.Running = true
	ci.StartedAt = time.Now()
	ci.StoppedAt = nil
	ci.abortChan = make(chan struct{})

	return gateway.StartAccount(ctx, ci.AccountID)
}

// Stop stops the channel instance.
func (ci *ChannelInstance) Stop(ctx context.Context) error {
	gateway := ci.Plugin.Gateway()
	if gateway == nil {
		ci.Running = false
		return nil
	}

	close(ci.abortChan)

	now := time.Now()
	ci.StoppedAt = &now
	ci.Running = false

	return gateway.StopAccount(ctx, ci.AccountID)
}

// IsRunning checks if the instance is running.
func (ci *ChannelInstance) IsRunning() bool {
	return ci.Running
}

// AbortChannel returns a channel that closes when the instance is aborted.
func (ci *ChannelInstance) AbortChannel() <-chan struct{} {
	return ci.abortChan
}

// Registry manages channel plugins.
type Registry struct {
	channels map[ChannelID]Plugin
}

// NewRegistry creates a new channel registry.
func NewRegistry() *Registry {
	return &Registry{
		channels: make(map[ChannelID]Plugin),
	}
}

// Register registers a channel plugin.
func (r *Registry) Register(plugin Plugin) error {
	id := plugin.ID()
	if _, exists := r.channels[id]; exists {
		return &ChannelError{
			Type:    ErrorDuplicateChannel,
			Message: "channel already registered: " + string(id),
			Channel: id,
		}
	}
	r.channels[id] = plugin
	return nil
}

// Unregister unregisters a channel plugin.
func (r *Registry) Unregister(id ChannelID) {
	delete(r.channels, id)
}

// Get retrieves a channel plugin by ID.
func (r *Registry) Get(id ChannelID) (Plugin, bool) {
	plugin, exists := r.channels[id]
	return plugin, exists
}

// List returns all registered channel plugins.
func (r *Registry) List() []Plugin {
	plugins := make([]Plugin, 0, len(r.channels))
	for _, plugin := range r.channels {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// ResolveByID resolves a channel by ID or alias.
func (r *Registry) ResolveByID(id string) (Plugin, bool) {
	// Try direct ID match
	if plugin, ok := r.Get(ChannelID(id)); ok {
		return plugin, true
	}

	// Try aliases
	for _, plugin := range r.channels {
		for _, alias := range plugin.Meta().Aliases {
			if alias == id {
				return plugin, true
			}
		}
	}

	return nil, false
}

// Error types
type ErrorType string

const (
	ErrorDuplicateChannel ErrorType = "duplicate_channel"
	ErrorChannelNotFound  ErrorType = "channel_not_found"
	ErrorAccountNotFound  ErrorType = "account_not_found"
	ErrorConfigInvalid    ErrorType = "config_invalid"
	ErrorSendFailed       ErrorType = "send_failed"
	ErrorNotSupported     ErrorType = "not_supported"
)

// ChannelError represents a channel-related error.
type ChannelError struct {
	Type    ErrorType
	Message string
	Channel ChannelID
	Err     error
}

func (e *ChannelError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ChannelError) Unwrap() error {
	return e.Err
}
