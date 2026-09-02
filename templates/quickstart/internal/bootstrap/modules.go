package bootstrap

import "github.com/brizenchi/quickstart-template/internal/platform"

func (c AppConfig) ModuleConfig() platform.Config {
	return platform.Config{
		ServiceName: c.Server.Name,
		Auth:        c.Auth,
		Email:       c.Email,
		Billing:     c.Billing,
		Referral:    c.Referral,
	}
}
