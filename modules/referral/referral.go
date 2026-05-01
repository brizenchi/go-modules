// Package referral is a portable, schema-owning referral module.
//
// Layering:
//
//	domain/   Code, Referral, Status, Stats, errors
//	event/    ReferralRegistered, ReferralActivated
//	port/     CodeRepository, ReferralRepository, CodeGenerator, EventBus
//	adapter/
//	  gormrepo/   GORM impl of both repositories + AutoMigrateModels()
//	  codegen/    deterministic + random CodeGenerator
//	  eventbus/   in-process synchronous bus
//	app/      CodeService, AttributeService, QueryService
//	http/     Gin handlers + Mount()
//	referral.go  Module wiring
//
// The host project provides:
//
//  1. CodeRepository + ReferralRepository (GORM impl ships in adapter/gormrepo)
//  2. CodeGenerator (deterministic or random; in adapter/codegen)
//  3. UserIDFunc — extract authenticated user from Gin context (host's auth scheme)
//  4. event listeners — react to ReferralActivated to grant rewards
//
// Two integration points the host calls explicitly:
//
//	module.Attribute.AttributeReferral(ctx, refereeID, code)
//	   — call from auth UserSignedUp listener when the user supplied a code
//
//	module.Attribute.ActivateReferral(ctx, refereeID, rewardCredits)
//	   — call from billing SubscriptionActivated listener (or whatever
//	     event signifies "qualified for reward")
package referral

import (
	"time"

	"github.com/brizenchi/go-modules/modules/referral/app"
	"github.com/brizenchi/go-modules/modules/referral/event"
	httpapi "github.com/brizenchi/go-modules/modules/referral/http"
	"github.com/brizenchi/go-modules/modules/referral/port"
)

// Module bundles the wired use cases + handler.
type Module struct {
	Deps Deps

	Code      *app.CodeService
	Attribute *app.AttributeService
	Query     *app.QueryService
	Handler   *httpapi.Handler
}

// Deps gathers host-supplied collaborators.
type Deps struct {
	Codes     port.CodeRepository
	Referrals port.ReferralRepository
	Generator port.CodeGenerator
	Bus       port.EventBus

	// GetUserID extracts the authenticated user from a Gin context.
	GetUserID httpapi.UserIDFunc

	// BaseLink is appended in front of a code to form an invite URL,
	// e.g. "https://app.example.com/invite?ref=".
	BaseLink string

	// ActivationWindow, if > 0, sets ExpiresAt on new referrals; the
	// host's billing/business logic must call ActivateReferral before
	// the deadline.
	ActivationWindow time.Duration
}

func New(d Deps) *Module {
	codeSvc := app.NewCodeService(d.Codes, d.Generator)
	attrSvc := app.NewAttributeService(app.AttributeDeps{
		Codes:            codeSvc,
		Referrals:        d.Referrals,
		Bus:              d.Bus,
		ActivationWindow: d.ActivationWindow,
	})
	querySvc := app.NewQueryService(d.Referrals)
	handler := httpapi.NewHandler(httpapi.Deps{
		Codes:     codeSvc,
		Attribute: attrSvc,
		Query:     querySvc,
		GetUserID: d.GetUserID,
		BaseLink:  d.BaseLink,
	})
	return &Module{
		Deps:      d,
		Code:      codeSvc,
		Attribute: attrSvc,
		Query:     querySvc,
		Handler:   handler,
	}
}

func (m *Module) Subscribe(kind event.Kind, fn port.Listener) {
	if m.Deps.Bus != nil {
		m.Deps.Bus.Subscribe(kind, fn)
	}
}
