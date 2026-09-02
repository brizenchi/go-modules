// Package hostcfg 保存当前 SaaS 自己拥有的业务配置。
//
// 这里可以自由修改。复制模板后，这个结构属于当前 SaaS；增加字段不会改变共享模块。
//
// 字段同时映射到 deploy/config.yaml 的 host 段和 APP_HOST_* 环境变量：
//
//	host:
//	  openai_api_key: "sk-..."     →  APP_HOST_OPENAI_API_KEY
//	  feature:
//	    enable_export: true        →  APP_HOST_FEATURE_ENABLE_EXPORT
package hostcfg

// Config 是当前 SaaS 的业务配置。
type Config struct {
	// SignupCredits 演示如何响应 UserSignedUp；设为 0 可关闭，
	// 也可以在 host_hooks.go 中替换成当前 SaaS 自己的规则。
	SignupCredits int64 `mapstructure:"signup_credits"`

	WelcomeEmail WelcomeEmailConfig `mapstructure:"welcome_email"`
}

type WelcomeEmailConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	OnlyProvider string `mapstructure:"only_provider"`
	Subject      string `mapstructure:"subject"`
	TextBody     string `mapstructure:"text_body"`
	HTMLBody     string `mapstructure:"html_body"`
}
