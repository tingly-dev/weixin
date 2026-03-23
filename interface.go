// package weixin provides interfaces for the WeChat channel.
package weixin

// PluginInterface defines the interface for the WeChat plugin that adapters can use.
// This avoids import cycles between the wechat package and adapters package.
type PluginInterface interface {
	Accounts() *AccountManager
	WeChatConfig() *WeChatConfig
	SetRunning(accountID string, running bool)
	IsRunningByID(accountID string) bool
}
