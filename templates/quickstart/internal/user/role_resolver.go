package user

import (
	"context"
	"strings"

	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
)

type RoleResolver struct {
	admins map[string]struct{}
}

func NewRoleResolver(adminEmails []string) *RoleResolver {
	resolver := &RoleResolver{admins: make(map[string]struct{}, len(adminEmails))}
	for _, email := range adminEmails {
		if normalized := normalizeEmail(email); normalized != "" {
			resolver.admins[normalized] = struct{}{}
		}
	}
	return resolver
}

func (r *RoleResolver) Resolve(_ context.Context, identity authdomain.Identity) (authdomain.Role, error) {
	if _, ok := r.admins[normalizeEmail(identity.Email)]; ok {
		return authdomain.RoleAdmin, nil
	}
	if strings.EqualFold(string(identity.Role), string(authdomain.RoleAdmin)) {
		return authdomain.RoleAdmin, nil
	}
	return authdomain.RoleUser, nil
}

var _ authport.RoleResolver = (*RoleResolver)(nil)
