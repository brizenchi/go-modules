package saascore

import (
	"context"
	"strings"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
)

type configRoleResolver struct {
	adminEmails map[string]struct{}
}

func newConfigRoleResolver(adminEmails []string) *configRoleResolver {
	normalized := make(map[string]struct{}, len(adminEmails))
	for _, value := range adminEmails {
		email := strings.ToLower(strings.TrimSpace(value))
		if email != "" {
			normalized[email] = struct{}{}
		}
	}
	return &configRoleResolver{adminEmails: normalized}
}

func (r *configRoleResolver) Resolve(_ context.Context, id authdomain.Identity) (authdomain.Role, error) {
	email := strings.ToLower(strings.TrimSpace(id.Email))
	if email != "" {
		if _, ok := r.adminEmails[email]; ok {
			return authdomain.RoleAdmin, nil
		}
	}
	if id.Role != "" {
		return id.Role, nil
	}
	return authdomain.RoleUser, nil
}

var _ authport.RoleResolver = (*configRoleResolver)(nil)
