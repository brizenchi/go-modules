package authstore

import (
	"context"
	"strings"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/spf13/viper"
)

// ConfigRoleResolver assigns admin role based on auth.admin_emails config.
type ConfigRoleResolver struct{}

func NewConfigRoleResolver() *ConfigRoleResolver { return &ConfigRoleResolver{} }

func (r *ConfigRoleResolver) Resolve(_ context.Context, id authdomain.Identity) (authdomain.Role, error) {
	if isAdminEmail(id.Email) {
		return authdomain.RoleAdmin, nil
	}
	if id.Role != "" {
		return id.Role, nil
	}
	return authdomain.RoleUser, nil
}

func isAdminEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	configured := viper.GetStringSlice("auth.admin_emails")
	if len(configured) == 0 {
		if raw := strings.TrimSpace(viper.GetString("auth.admin_emails")); raw != "" {
			configured = strings.Split(raw, ",")
		}
	}
	for _, value := range configured {
		if strings.EqualFold(strings.TrimSpace(value), email) {
			return true
		}
	}
	return false
}

var _ authport.RoleResolver = (*ConfigRoleResolver)(nil)
