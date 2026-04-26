// Package billing is a portable, payment-provider-agnostic billing module.
//
// Layering (DDD-ish):
//
//	domain/   pure types (no I/O): enums, errors, snapshots, persistence model
//	event/    domain event definitions (subscription.activated, ...)
//	port/     interfaces the module depends on (Provider, EventBus, Repository, ...)
//	adapter/  concrete implementations of ports (stripe, gorm, in-process bus)
//	app/      use cases (Checkout, CancelSubscription, ProcessWebhook, ...)
//	http/     Gin handlers + Mount()
//
// The module is host-agnostic: it never imports project-specific models
// (e.g. user, bot, relay). Hosts integrate via three pluggable points:
//
//  1. port.CustomerStore   — load/save provider customer IDs against the host's user table
//  2. port.UserResolver    — resolve a user ID from webhook hints
//  3. event listeners      — apply provider-specific side effects (grant quota, send email)
//
// To copy this module into another project:
//
//   - cp -r pkg/billing/ <other-project>/pkg/billing/
//   - implement port.CustomerStore + port.UserResolver against your user table
//   - register listeners for the events you care about
//   - call billing.New(...) and billing.Mount(...) from your bootstrap
package billing

import (
	"github.com/brizenchi/go-modules/billing/app"
	"github.com/brizenchi/go-modules/billing/event"
	httpapi "github.com/brizenchi/go-modules/billing/http"
	"github.com/brizenchi/go-modules/billing/port"
)

// Module is the assembled billing system: use cases + handlers, ready to mount.
type Module struct {
	Provider     port.Provider
	Bus          port.EventBus
	Customers    port.CustomerStore
	EventRepo    port.BillingEventRepository
	UserResolver port.UserResolver

	Checkout     *app.CheckoutService
	Subscription *app.SubscriptionService
	Webhook      *app.WebhookService
	Query        *app.QueryService

	Handler *httpapi.Handler
}

// Deps describes the host-supplied collaborators.
type Deps struct {
	Provider     port.Provider
	Bus          port.EventBus
	Customers    port.CustomerStore
	EventRepo    port.BillingEventRepository
	UserResolver port.UserResolver
	GetUserID    httpapi.UserIDFunc
}

// New wires the module from its dependencies.
//
// All Deps fields are required except UserResolver (used as a fallback
// when a webhook payload doesn't carry the user ID directly). Pass nil
// to disable resolution.
func New(d Deps) *Module {
	checkout := app.NewCheckoutService(d.Provider, d.Customers)
	subs := app.NewSubscriptionService(d.Provider, d.Customers, d.Bus)
	webhook := app.NewWebhookService(d.Provider, d.EventRepo, d.UserResolver, d.Bus)
	query := app.NewQueryService(d.Provider, d.Customers)

	handler := httpapi.NewHandler(httpapi.Deps{
		Checkout:     checkout,
		Subscription: subs,
		Webhook:      webhook,
		Query:        query,
		GetUserID:    d.GetUserID,
	})

	return &Module{
		Provider:     d.Provider,
		Bus:          d.Bus,
		Customers:    d.Customers,
		EventRepo:    d.EventRepo,
		UserResolver: d.UserResolver,
		Checkout:     checkout,
		Subscription: subs,
		Webhook:      webhook,
		Query:        query,
		Handler:      handler,
	}
}

// Subscribe is a thin pass-through to the bus, exposed for ergonomics.
func (m *Module) Subscribe(kind event.Kind, fn port.Listener) {
	m.Bus.Subscribe(kind, fn)
}
