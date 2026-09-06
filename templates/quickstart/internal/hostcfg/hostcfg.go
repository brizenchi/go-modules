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
	Uploads      UploadConfig       `mapstructure:"uploads"`
}

// UploadConfig is deployment-owned; credentials never enter editable site settings.
type UploadConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Provider        string `mapstructure:"provider"` // local (development), s3 (S3/R2)
	Directory       string `mapstructure:"directory"`
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	Endpoint        string `mapstructure:"endpoint"`
	UsePathStyle    bool   `mapstructure:"use_path_style"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

type WelcomeEmailConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	OnlyProvider string `mapstructure:"only_provider"`
	Subject      string `mapstructure:"subject"`
	TextBody     string `mapstructure:"text_body"`
	HTMLBody     string `mapstructure:"html_body"`
}
