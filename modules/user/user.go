// Package user is the standard shared user-domain module for SaaS hosts
// that intentionally share the same users schema across projects.
//
// Layering:
//
//	domain/   pure user types + enums
//	adapter/
//	  gormrepo/      GORM schema + repository + migration
//	  authstore/     auth.UserStore + RoleResolver
//	  billingstore/  billing.CustomerStore + UserResolver + sync helpers
//
// This module is optional for hosts that already have an existing user
// table. Those hosts can continue implementing auth/billing ports
// directly against their own schema.
package user

