package http

import (
	"github.com/brizenchi/quickstart-template/internal/feature/account"
	"github.com/brizenchi/quickstart-template/internal/feature/note"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
)

// YOURS — edit freely.
//
// registerHostRoutes is where every host feature gets mounted. Build the
// feature from deps, then let it register itself on the group matching
// its access level:
//
//	g.Public  no auth
//	g.User    bearer token required; identity via authhttp.Authenticated(c)
//	g.Admin   bearer token + admin role (see auth.admin_emails)
//
// Shared auth / billing / referral routes are already mounted by the time
// this runs; do not re-declare them here.
func registerHostRoutes(deps hostapi.Deps, g hostapi.Groups) {
	account.New(deps.Users).Register(g)
	note.New(deps.DB).Register(g)
}
