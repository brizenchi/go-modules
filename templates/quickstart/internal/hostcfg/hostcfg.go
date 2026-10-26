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

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/brizenchi/go-modules/foundation/aix"
)

// Config 是当前 SaaS 的业务配置。
type Config struct {
	AI                 AIConfig                 `mapstructure:"ai"`
	ContentTranslation ContentTranslationConfig `mapstructure:"content_translation"`

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

// AIConfig contains deployment-only credentials; no API key is returned to the browser.
type AIConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	BaseURL         string `mapstructure:"base_url"`
	APIKey          string `mapstructure:"api_key" json:"-"`
	Model           string `mapstructure:"model"`
	MaxOutputTokens int    `mapstructure:"max_output_tokens"`
}

func (c AIConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	return (aix.Config{BaseURL: c.BaseURL, APIKey: c.APIKey, Model: c.Model, MaxOutputTokens: c.MaxOutputTokens}).Validate()
}

// ContentTranslationConfig owns the content AI translation integration. The
// Gemini-native protocol is separate from host.ai; the key only ever comes
// from APP_HOST_CONTENT_TRANSLATION_API_KEY or the uncommitted local .env.
type ContentTranslationConfig struct {
	Enabled               bool   `mapstructure:"enabled"`
	BaseURL               string `mapstructure:"base_url"`
	APIKey                string `mapstructure:"api_key" json:"-"`
	Model                 string `mapstructure:"model"`
	MaxOutputTokens       int    `mapstructure:"max_output_tokens"`
	MaxZhCharacters       int    `mapstructure:"max_zh_characters"`
	MaxEnWords            int    `mapstructure:"max_en_words"`
	RequestTimeoutSeconds int    `mapstructure:"request_timeout_seconds"`
	TaskTimeoutSeconds    int    `mapstructure:"task_timeout_seconds"`
}

func (c ContentTranslationConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	const name = "host.content_translation"
	key := strings.TrimSpace(c.APIKey)
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	model := strings.TrimSpace(c.Model)
	if key == "" || strings.ContainsAny(key, "\r\n\x00") || model == "" || len(model) > 200 ||
		c.MaxOutputTokens < 1 || c.MaxOutputTokens > 65536 || c.MaxZhCharacters < 1 || c.MaxEnWords < 1 ||
		c.RequestTimeoutSeconds < 1 || c.TaskTimeoutSeconds < c.RequestTimeoutSeconds {
		return fmt.Errorf("%s requires api key, model, positive limits and a request timeout within the task timeout", name)
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return fmt.Errorf("%s.base_url must be an https origin without credentials, query or fragment", name)
	}
	return nil
}
