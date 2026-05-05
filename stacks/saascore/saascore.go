package saascore

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/brizenchi/go-modules/modules/auth"
	"github.com/brizenchi/go-modules/modules/auth/adapter/emailcode"
	autheventbus "github.com/brizenchi/go-modules/modules/auth/adapter/eventbus"
	"github.com/brizenchi/go-modules/modules/auth/adapter/google"
	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	authhttp "github.com/brizenchi/go-modules/modules/auth/http"
	authport "github.com/brizenchi/go-modules/modules/auth/port"
	"github.com/brizenchi/go-modules/modules/billing"
	billingeventbus "github.com/brizenchi/go-modules/modules/billing/adapter/eventbus"
	billingrepo "github.com/brizenchi/go-modules/modules/billing/adapter/repo"
	stripeadapter "github.com/brizenchi/go-modules/modules/billing/adapter/stripe"
	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	"github.com/brizenchi/go-modules/modules/email"
	"github.com/brizenchi/go-modules/modules/email/adapter/brevo"
	logsender "github.com/brizenchi/go-modules/modules/email/adapter/log"
	"github.com/brizenchi/go-modules/modules/email/adapter/resend"
	emaildomain "github.com/brizenchi/go-modules/modules/email/domain"
	"github.com/brizenchi/go-modules/modules/referral"
	"github.com/brizenchi/go-modules/modules/referral/adapter/codegen"
	referraleventbus "github.com/brizenchi/go-modules/modules/referral/adapter/eventbus"
	referralgormrepo "github.com/brizenchi/go-modules/modules/referral/adapter/gormrepo"
	referraldomain "github.com/brizenchi/go-modules/modules/referral/domain"
	referralevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/go-modules/modules/user/adapter/authstore"
	"github.com/brizenchi/go-modules/modules/user/adapter/billingstore"
	usergormrepo "github.com/brizenchi/go-modules/modules/user/adapter/gormrepo"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Stack struct {
	Config   Config
	DB       *gorm.DB
	Users    *usergormrepo.Repo
	Email    *email.Module
	Auth     *auth.Module
	Billing  *billing.Module
	Referral *referral.Module

	hostHooks   HostHooks
	policyHooks PolicyHooks
}

func New(db *gorm.DB, cfg Config, hostHooks HostHooks, policyHooks PolicyHooks) (*Stack, error) {
	if db == nil {
		return nil, fmt.Errorf("saascore: db required")
	}
	cfg = withDefaults(cfg)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	usersRepo := usergormrepo.New(db)
	if err := usergormrepo.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("saascore: migrate users: %w", err)
	}
	if err := autoMigrateAuthStore(db); err != nil {
		return nil, fmt.Errorf("saascore: migrate auth store: %w", err)
	}
	if err := db.AutoMigrate(&billingdomain.BillingEvent{}); err != nil {
		return nil, fmt.Errorf("saascore: migrate billing events: %w", err)
	}
	if err := db.AutoMigrate(referralgormrepo.AutoMigrateModels()...); err != nil {
		return nil, fmt.Errorf("saascore: migrate referral: %w", err)
	}

	stack := &Stack{
		Config:      cfg,
		DB:          db,
		Users:       usersRepo,
		hostHooks:   hostHooks,
		policyHooks: policyHooks,
	}

	emailMod, err := initEmail(cfg.Email)
	if err != nil {
		return nil, err
	}
	stack.Email = emailMod

	authMod, err := stack.initAuth()
	if err != nil {
		return nil, err
	}
	stack.Auth = authMod

	stack.Billing = stack.initBilling()
	stack.Referral = stack.initReferral()
	stack.subscribeStandardListeners()

	return stack, nil
}

func withDefaults(cfg Config) Config {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "saascore"
	}

	if cfg.Auth.UserJWTExpire <= 0 {
		cfg.Auth.UserJWTExpire = 7 * 24 * time.Hour
	}
	if cfg.Auth.WSTicketTTL <= 0 {
		cfg.Auth.WSTicketTTL = 5 * time.Minute
	}

	if cfg.Auth.EmailCode.TTL <= 0 {
		cfg.Auth.EmailCode.TTL = 10 * time.Minute
	}
	if cfg.Auth.EmailCode.MinResendGap <= 0 {
		cfg.Auth.EmailCode.MinResendGap = time.Minute
	}
	if cfg.Auth.EmailCode.DailyCap <= 0 {
		cfg.Auth.EmailCode.DailyCap = 10
	}
	if cfg.Auth.EmailCode.MaxAttempts <= 0 {
		cfg.Auth.EmailCode.MaxAttempts = 5
	}

	if strings.TrimSpace(cfg.Email.Provider) == "" {
		cfg.Email.Provider = "log"
	}
	if cfg.Referral.Prefix == "" {
		cfg.Referral.Prefix = "INV"
	}
	if cfg.Billing.Stripe.CreditsPerPackage <= 0 {
		cfg.Billing.Stripe.CreditsPerPackage = 100
	}
	return cfg
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Auth.UserJWTSecret) == "" {
		return fmt.Errorf("saascore: auth.user_jwt_secret required")
	}
	if cfg.Billing.Stripe.Enabled {
		if strings.TrimSpace(cfg.Billing.Stripe.SecretKey) == "" {
			return fmt.Errorf("saascore: billing.stripe.secret_key required when stripe enabled")
		}
		if strings.TrimSpace(cfg.Billing.Stripe.WebhookSecret) == "" {
			return fmt.Errorf("saascore: billing.stripe.webhook_secret required when stripe enabled")
		}
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Email.Provider), "brevo") {
		if strings.TrimSpace(cfg.Email.Brevo.APIKey) == "" || strings.TrimSpace(cfg.Email.Brevo.SenderEmail) == "" {
			return fmt.Errorf("saascore: email brevo api key and sender email required when provider=brevo")
		}
	}
	if strings.EqualFold(strings.TrimSpace(cfg.Email.Provider), "resend") {
		if strings.TrimSpace(cfg.Email.Resend.APIKey) == "" || strings.TrimSpace(cfg.Email.Resend.SenderEmail) == "" {
			return fmt.Errorf("saascore: email resend api key and sender email required when provider=resend")
		}
	}
	if strings.TrimSpace(cfg.Auth.Google.ClientID) != "" ||
		strings.TrimSpace(cfg.Auth.Google.ClientSecret) != "" {
		if strings.TrimSpace(cfg.Auth.Google.ClientID) == "" ||
			strings.TrimSpace(cfg.Auth.Google.ClientSecret) == "" ||
			strings.TrimSpace(cfg.Auth.Google.RedirectURL) == "" ||
			strings.TrimSpace(cfg.Auth.Google.StateSecret) == "" {
			return fmt.Errorf("saascore: google oauth requires client_id, client_secret, redirect_url, state_secret together")
		}
	}
	return nil
}

func initEmail(cfg EmailConfig) (*email.Module, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "log":
		slog.Info("saascore: email provider not configured, using log sender")
		return email.New(logsender.New(nil), nil), nil
	case "brevo":
		sender, err := brevo.New(brevo.Config{
			APIKey: cfg.Brevo.APIKey,
			Sender: emaildomain.Address{
				Email: cfg.Brevo.SenderEmail,
				Name:  cfg.Brevo.SenderName,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("saascore: init brevo sender: %w", err)
		}
		slog.Info("saascore: email provider registered", "provider", "brevo")
		return email.New(sender, nil), nil
	case "resend":
		sender, err := resend.New(resend.Config{
			APIKey: cfg.Resend.APIKey,
			Sender: emaildomain.Address{
				Email: cfg.Resend.SenderEmail,
				Name:  cfg.Resend.SenderName,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("saascore: init resend sender: %w", err)
		}
		slog.Info("saascore: email provider registered", "provider", "resend")
		return email.New(sender, nil), nil
	default:
		return nil, fmt.Errorf("saascore: unsupported email provider %q", cfg.Provider)
	}
}

func (s *Stack) initAuth() (*auth.Module, error) {
	signer, err := authjwt.NewSigner(authjwt.Config{
		Secret:  s.Config.Auth.UserJWTSecret,
		Issuer:  s.Config.ServiceName,
		UserTTL: s.Config.Auth.UserJWTExpire,
	})
	if err != nil {
		return nil, fmt.Errorf("saascore: init jwt signer: %w", err)
	}
	ticketSigner, err := authjwt.NewTicketSigner(authjwt.Config{
		Secret:    s.Config.Auth.UserJWTSecret,
		Issuer:    s.Config.ServiceName + "-ws",
		TicketTTL: s.Config.Auth.WSTicketTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("saascore: init ws ticket signer: %w", err)
	}

	store := newAuthStore(s.DB)
	issuer := emailcode.NewIssuer(emailcode.Config{
		TTL:          s.Config.Auth.EmailCode.TTL,
		MinResendGap: s.Config.Auth.EmailCode.MinResendGap,
		DailyCap:     s.Config.Auth.EmailCode.DailyCap,
		MaxAttempts:  s.Config.Auth.EmailCode.MaxAttempts,
		TemplateRef:  defaultEmailCodeTemplateRef(s.Config.Auth.EmailCode.VerificationTemplate),
		Debug:        s.Config.Auth.EmailCode.Debug,
	}, store, mailerWrapper{mod: s.Email, serviceName: s.Config.ServiceName, emailCfg: s.Config.Email})
	verifier := emailcode.NewVerifier(emailcode.Config{
		MaxAttempts: s.Config.Auth.EmailCode.MaxAttempts,
	}, store)

	providers := map[string]authport.IdentityProvider{}
	if provider, err := buildGoogleProvider(s.Config.Auth.Google); err != nil {
		return nil, err
	} else if provider != nil {
		providers[string(authdomain.ProviderGoogle)] = provider
	}

	return auth.New(auth.Deps{
		UserStore:         authstore.New(s.Users),
		RoleResolver:      newConfigRoleResolver(s.Config.Auth.AdminEmails),
		TokenSigner:       signer,
		WSTicketSigner:    ticketSigner,
		ExchangeCodeStore: store,
		EmailCodeIssuer:   issuer,
		EmailCodeVerifier: verifier,
		IdentityProviders: providers,
		Bus:               autheventbus.NewInProc(),
		FrontendURL:       s.Config.Auth.FrontendRedirect,
	}), nil
}

func buildGoogleProvider(cfg GoogleOAuthConfig) (authport.IdentityProvider, error) {
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, nil
	}
	provider, err := google.New(google.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		StateSecret:  cfg.StateSecret,
		Scope:        cfg.Scope,
	})
	if err != nil {
		return nil, fmt.Errorf("saascore: init google provider: %w", err)
	}
	return provider, nil
}

func (s *Stack) initBilling() *billing.Module {
	return billing.New(billing.Deps{
		Provider: stripeadapter.NewProvider(stripeadapter.Config{
			Enabled:        s.Config.Billing.Stripe.Enabled,
			SecretKey:      s.Config.Billing.Stripe.SecretKey,
			WebhookSecret:  s.Config.Billing.Stripe.WebhookSecret,
			PublishableKey: s.Config.Billing.Stripe.PublishableKey,
			SubscriptionPrices: map[billingdomain.PlanType]map[billingdomain.BillingInterval]string{
				billingdomain.PlanStarter: {
					billingdomain.IntervalMonthly: s.Config.Billing.Stripe.StarterMonthlyPriceID,
					billingdomain.IntervalYearly:  s.Config.Billing.Stripe.StarterYearlyPriceID,
				},
				billingdomain.PlanPro: {
					billingdomain.IntervalMonthly: s.Config.Billing.Stripe.ProMonthlyPriceID,
					billingdomain.IntervalYearly:  s.Config.Billing.Stripe.ProYearlyPriceID,
				},
				billingdomain.PlanPremium: {
					billingdomain.IntervalMonthly: s.Config.Billing.Stripe.PremiumMonthlyPriceID,
					billingdomain.IntervalYearly:  s.Config.Billing.Stripe.PremiumYearlyPriceID,
				},
			},
			LifetimePriceID: s.Config.Billing.Stripe.LifetimePriceID,
			CreditsPriceIDs: s.Config.Billing.Stripe.CreditsPriceIDs,
			CreditsPerUnit:  s.Config.Billing.Stripe.CreditsPerPackage,
			TrialDays:       s.Config.Billing.Stripe.TrialDays,
		}),
		Bus:          billingeventbus.NewInProc(),
		Customers:    billingstore.NewCustomerStore(s.Users),
		EventRepo:    billingrepo.NewBillingEventRepo(s.DB),
		UserResolver: billingstore.NewUserResolver(s.Users),
		GetUserID:    s.userIDFromGin,
	})
}

func (s *Stack) initReferral() *referral.Module {
	return referral.New(referral.Deps{
		Codes:            referralgormrepo.NewCodeRepo(s.DB),
		Referrals:        referralgormrepo.NewReferralRepo(s.DB),
		Generator:        codegen.NewDeterministic(s.Config.Referral.Prefix, 8),
		Bus:              referraleventbus.NewInProc(),
		GetUserID:        s.userIDFromGin,
		BaseLink:         s.Config.Referral.BaseLink,
		ActivationWindow: s.Config.Referral.ActivationWindow,
	})
}

func (s *Stack) subscribeStandardListeners() {
	s.Auth.Subscribe(authevent.KindUserSignedUp, s.onUserSignedUpBilling)
	s.Auth.Subscribe(authevent.KindUserSignedUp, s.onUserSignedUpReferral)
	s.Auth.Subscribe(authevent.KindUserSignedUp, s.onUserSignedUpHost)

	s.Billing.Subscribe(billingevent.KindSubscriptionActivated, s.onSubscriptionActivated)
	s.Billing.Subscribe(billingevent.KindSubscriptionRenewed, s.onSubscriptionRenewed)
	s.Billing.Subscribe(billingevent.KindSubscriptionUpdated, s.onSubscriptionUpdated)
	s.Billing.Subscribe(billingevent.KindSubscriptionReactivated, s.onSubscriptionReactivated)
	s.Billing.Subscribe(billingevent.KindSubscriptionCanceling, s.onSubscriptionCanceling)
	s.Billing.Subscribe(billingevent.KindSubscriptionCanceled, s.onSubscriptionCanceled)
	s.Billing.Subscribe(billingevent.KindPaymentFailed, s.onPaymentFailed)
	s.Billing.Subscribe(billingevent.KindCreditsPurchased, s.onCreditsPurchased)

	s.Referral.Subscribe(referralevent.KindReferralRegistered, s.onReferralRegisteredHost)
	s.Referral.Subscribe(referralevent.KindReferralActivated, s.onReferralActivated)
}

func (s *Stack) onUserSignedUpBilling(ctx context.Context, env authevent.Envelope) error {
	return billingstore.ApplyFreePlan(ctx, s.Users, env.UserID)
}

func (s *Stack) onUserSignedUpReferral(ctx context.Context, env authevent.Envelope) error {
	if env.UserID == "" {
		return nil
	}
	code := referralCode(ctx)
	if code == "" {
		return nil
	}
	_, err := s.Referral.Attribute.AttributeReferral(ctx, env.UserID, code)
	if err != nil {
		if isReferralSignupSkip(err) {
			slog.Warn("saascore: referral signup attribution skipped",
				"user_id", env.UserID,
				"referral_code", code,
				"error", err,
			)
			return nil
		}
		return err
	}
	slog.Info("saascore: referral signup attributed", "user_id", env.UserID, "referral_code", code)
	return nil
}

func (s *Stack) onUserSignedUpHost(ctx context.Context, env authevent.Envelope) error {
	if s.hostHooks.OnUserSignedUp == nil {
		return nil
	}
	payload, _ := env.Payload.(authevent.UserSignedUp)
	return s.hostHooks.OnUserSignedUp(ctx, UserSignedUpEvent{
		UserID:     env.UserID,
		OccurredAt: env.OccurredAt,
		Identity:   payload.Identity,
	})
}

func isReferralSignupSkip(err error) bool {
	return err == referraldomain.ErrInvalidCode ||
		err == referraldomain.ErrSelfReferral ||
		err == referraldomain.ErrAlreadyAttributed
}

func (s *Stack) onSubscriptionActivated(ctx context.Context, env billingevent.Envelope) error {
	p, _ := env.Payload.(billingevent.SubscriptionActivated)
	if err := billingstore.ApplySubscriptionSnapshot(ctx, s.Users, env.UserID, p.Snapshot); err != nil {
		return err
	}
	reward, err := s.resolveReferralReward(ctx, ReferralRewardPolicyInput{
		ReferrerID: "",
		RefereeID:  env.UserID,
	})
	if err != nil {
		return err
	}
	_, err = s.Referral.Attribute.ActivateReferral(ctx, env.UserID, reward)
	if err != nil {
		slog.Debug("saascore: referral activation skipped", "user_id", env.UserID, "error", err)
	}
	if s.hostHooks.OnSubscriptionActivated != nil {
		return s.hostHooks.OnSubscriptionActivated(ctx, subscriptionEventFromEnvelope(env, p.Snapshot))
	}
	return nil
}

func (s *Stack) onSubscriptionRenewed(ctx context.Context, env billingevent.Envelope) error {
	p, _ := env.Payload.(billingevent.SubscriptionRenewed)
	if err := billingstore.ApplySubscriptionSnapshot(ctx, s.Users, env.UserID, p.Snapshot); err != nil {
		return err
	}
	if s.hostHooks.OnSubscriptionRenewed != nil {
		return s.hostHooks.OnSubscriptionRenewed(ctx, subscriptionEventFromEnvelope(env, p.Snapshot))
	}
	return nil
}

func (s *Stack) onSubscriptionUpdated(ctx context.Context, env billingevent.Envelope) error {
	p, _ := env.Payload.(billingevent.SubscriptionUpdated)
	if err := billingstore.ApplySubscriptionSnapshot(ctx, s.Users, env.UserID, p.Snapshot); err != nil {
		return err
	}
	if s.hostHooks.OnSubscriptionUpdated != nil {
		return s.hostHooks.OnSubscriptionUpdated(ctx, subscriptionEventFromEnvelope(env, p.Snapshot))
	}
	return nil
}

func (s *Stack) onSubscriptionReactivated(ctx context.Context, env billingevent.Envelope) error {
	p, _ := env.Payload.(billingevent.SubscriptionReactivated)
	if err := billingstore.ApplySubscriptionSnapshot(ctx, s.Users, env.UserID, p.Snapshot); err != nil {
		return err
	}
	if s.hostHooks.OnSubscriptionReactivated != nil {
		return s.hostHooks.OnSubscriptionReactivated(ctx, subscriptionEventFromEnvelope(env, p.Snapshot))
	}
	return nil
}

func (s *Stack) onSubscriptionCanceling(ctx context.Context, env billingevent.Envelope) error {
	p, _ := env.Payload.(billingevent.SubscriptionCanceling)
	if err := billingstore.ApplySubscriptionCanceling(ctx, s.Users, env.UserID, p.EffectiveAt); err != nil {
		return err
	}
	if s.hostHooks.OnSubscriptionCanceling != nil {
		return s.hostHooks.OnSubscriptionCanceling(ctx, SubscriptionCancelingEvent{
			UserID:          env.UserID,
			OccurredAt:      env.OccurredAt,
			Provider:        env.Provider,
			ProviderEventID: env.ProviderEventID,
			Snapshot:        p.Snapshot,
			Mode:            p.Mode,
			EffectiveAt:     p.EffectiveAt,
		})
	}
	return nil
}

func (s *Stack) onSubscriptionCanceled(ctx context.Context, env billingevent.Envelope) error {
	if err := billingstore.ApplySubscriptionCanceled(ctx, s.Users, env.UserID); err != nil {
		return err
	}
	if s.hostHooks.OnSubscriptionCanceled != nil {
		p, _ := env.Payload.(billingevent.SubscriptionCanceled)
		return s.hostHooks.OnSubscriptionCanceled(ctx, SubscriptionCanceledEvent{
			UserID:                 env.UserID,
			OccurredAt:             env.OccurredAt,
			Provider:               env.Provider,
			ProviderEventID:        env.ProviderEventID,
			ProviderSubscriptionID: p.ProviderSubscriptionID,
			ProviderCustomerID:     p.ProviderCustomerID,
		})
	}
	return nil
}

func (s *Stack) onPaymentFailed(ctx context.Context, env billingevent.Envelope) error {
	if err := billingstore.ApplyPaymentFailed(ctx, s.Users, env.UserID); err != nil {
		return err
	}
	if s.hostHooks.OnPaymentFailed != nil {
		p, _ := env.Payload.(billingevent.PaymentFailed)
		return s.hostHooks.OnPaymentFailed(ctx, PaymentFailedEvent{
			UserID:                 env.UserID,
			OccurredAt:             env.OccurredAt,
			Provider:               env.Provider,
			ProviderEventID:        env.ProviderEventID,
			ProviderSubscriptionID: p.ProviderSubscriptionID,
			ProviderCustomerID:     p.ProviderCustomerID,
		})
	}
	return nil
}

func (s *Stack) onCreditsPurchased(ctx context.Context, env billingevent.Envelope) error {
	p, _ := env.Payload.(billingevent.CreditsPurchased)
	if err := s.Users.AddCredits(ctx, env.UserID, int(p.TotalCredits)); err != nil {
		return err
	}
	if s.hostHooks.OnCreditsPurchased != nil {
		return s.hostHooks.OnCreditsPurchased(ctx, CreditsPurchasedEvent{
			UserID:          env.UserID,
			OccurredAt:      env.OccurredAt,
			Provider:        env.Provider,
			ProviderEventID: env.ProviderEventID,
			Quantity:        p.Quantity,
			CreditsPerUnit:  p.CreditsPerUnit,
			TotalCredits:    p.TotalCredits,
			PriceID:         p.PriceID,
		})
	}
	return nil
}

func (s *Stack) onReferralRegisteredHost(ctx context.Context, env referralevent.Envelope) error {
	if s.hostHooks.OnReferralRegistered == nil {
		return nil
	}
	p, _ := env.Payload.(referralevent.ReferralRegistered)
	return s.hostHooks.OnReferralRegistered(ctx, ReferralRegisteredEvent{
		OccurredAt: env.OccurredAt,
		Referral:   p.Referral,
	})
}

func (s *Stack) onReferralActivated(ctx context.Context, env referralevent.Envelope) error {
	p, _ := env.Payload.(referralevent.ReferralActivated)
	if s.hostHooks.OnReferralActivated == nil {
		slog.Info("saascore: referral activated without host listener",
			"referrer_id", p.Referral.ReferrerID,
			"referee_id", p.Referral.RefereeID,
			"reward_credits", p.Referral.RewardCredits,
		)
		return nil
	}
	return s.hostHooks.OnReferralActivated(ctx, ReferralActivatedEvent{
		OccurredAt: env.OccurredAt,
		Referral:   p.Referral,
	})
}

func (s *Stack) resolveReferralReward(ctx context.Context, input ReferralRewardPolicyInput) (int, error) {
	if s.policyHooks.ResolveReferralReward == nil {
		return s.Config.Referral.ActivationReward, nil
	}
	reward, err := s.policyHooks.ResolveReferralReward(ctx, input)
	if err != nil {
		return 0, err
	}
	if reward < 0 {
		return 0, fmt.Errorf("saascore: referral reward cannot be negative")
	}
	return reward, nil
}

func subscriptionEventFromEnvelope(env billingevent.Envelope, snapshot billingdomain.SubscriptionSnapshot) SubscriptionEvent {
	return SubscriptionEvent{
		UserID:          env.UserID,
		OccurredAt:      env.OccurredAt,
		Provider:        env.Provider,
		ProviderEventID: env.ProviderEventID,
		Snapshot:        snapshot,
	}
}

func (s *Stack) userIDFromGin(c *gin.Context) (string, bool) {
	id := authhttp.Authenticated(c)
	if id == nil || strings.TrimSpace(id.UserID) == "" {
		return "", false
	}
	return id.UserID, true
}

type mailerWrapper struct {
	mod         *email.Module
	serviceName string
	emailCfg    EmailConfig
}

func (w mailerWrapper) SendProviderTemplate(ctx context.Context, ref string, to []emailcode.EmailAddress, vars map[string]any) error {
	addrs := make([]emaildomain.Address, len(to))
	for i, a := range to {
		addrs[i] = emaildomain.Address{Name: a.Name, Email: a.Email}
	}
	if w.mod == nil || w.mod.Sender == nil {
		return fmt.Errorf("saascore: email module not configured")
	}
	subject, htmlBody, textBody := buildEmailCodeMessage(w.serviceName, w.senderIdentity(), vars)
	_, err := w.mod.Send(ctx, &emaildomain.Message{
		To:       addrs,
		Subject:  subject,
		HTMLBody: htmlBody,
		TextBody: textBody,
	})
	return err
}

func (w mailerWrapper) senderIdentity() emailSenderIdentity {
	from := emaildomain.Address{}
	switch strings.ToLower(strings.TrimSpace(w.emailCfg.Provider)) {
	case "brevo":
		from = emaildomain.Address{
			Name:  strings.TrimSpace(w.emailCfg.Brevo.SenderName),
			Email: strings.TrimSpace(w.emailCfg.Brevo.SenderEmail),
		}
	case "resend":
		from = emaildomain.Address{
			Name:  strings.TrimSpace(w.emailCfg.Resend.SenderName),
			Email: strings.TrimSpace(w.emailCfg.Resend.SenderEmail),
		}
	}
	return newEmailSenderIdentity(w.serviceName, from)
}

func defaultEmailCodeTemplateRef(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return configured
	}
	return "auth_email_code"
}

type emailSenderIdentity struct {
	BrandName    string
	SupportEmail string
	WebsiteURL   string
	WebsiteHost  string
}

func newEmailSenderIdentity(serviceName string, from emaildomain.Address) emailSenderIdentity {
	brandName := strings.TrimSpace(from.Name)
	if brandName == "" {
		brandName = strings.TrimSpace(serviceName)
	}
	if brandName == "" {
		brandName = "Your Account"
	}

	supportEmail := strings.TrimSpace(from.Email)
	websiteURL := ""
	websiteHost := ""
	if i := strings.Index(supportEmail, "@"); i >= 0 && i+1 < len(supportEmail) {
		domain := strings.TrimSpace(supportEmail[i+1:])
		if domain != "" {
			websiteURL = "https://" + domain
			websiteHost = domain
		}
	}
	if websiteURL != "" {
		if parsed, err := url.Parse(websiteURL); err == nil {
			websiteHost = parsed.Host
		}
	}

	return emailSenderIdentity{
		BrandName:    brandName,
		SupportEmail: supportEmail,
		WebsiteURL:   websiteURL,
		WebsiteHost:  websiteHost,
	}
}

func buildEmailCodeMessage(serviceName string, sender emailSenderIdentity, vars map[string]any) (subject, htmlBody, textBody string) {
	code, _ := vars["code"].(string)
	brand := strings.TrimSpace(sender.BrandName)
	if brand == "" {
		brand = strings.TrimSpace(serviceName)
	}
	if brand == "" {
		brand = "Your Account"
	}
	supportEmail := strings.TrimSpace(sender.SupportEmail)
	if supportEmail == "" {
		supportEmail = "support@example.com"
	}
	websiteURL := strings.TrimSpace(sender.WebsiteURL)
	websiteHost := strings.TrimSpace(sender.WebsiteHost)
	if websiteURL == "" {
		websiteURL = "https://example.com"
	}
	if websiteHost == "" {
		websiteHost = "example.com"
	}

	subject = fmt.Sprintf("%s verification code", brand)
	textBody = fmt.Sprintf(
		"%s verification code: %s\n\nYour verification code is valid for 10 minutes. Do not share this code with anyone.\n\nIf you didn't request this code, you can ignore this email.\nNeed help? Contact %s\n%s",
		brand,
		code,
		supportEmail,
		websiteURL,
	)
	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Email Verification Code</title>
</head>
<body style="margin:0; padding:0; background:#f3f4f7; font-family:'Inter','Helvetica Neue',Arial,sans-serif;">
  <div style="max-width:600px; margin:24px auto; background:#ffffff; border-radius:24px; overflow:hidden; box-shadow:0 18px 45px rgba(99,102,241,0.12);">
    <div style="background:linear-gradient(90deg,#6366f1,#8b5cf6); padding:28px 32px;">
      <span style="font-size:22px; font-weight:600; letter-spacing:0.08em; color:#fff; text-transform:uppercase;">%s</span>
    </div>
    <div style="padding:42px 36px; text-align:center; color:#0b2545;">
      <p style="margin:0 0 12px; font-size:14px; letter-spacing:0.35em; text-transform:uppercase; color:#94a3b8;">Verification Code</p>
      <h2 style="margin:0 0 18px; font-size:28px; font-weight:700;">Email Verification</h2>
      <div style="margin:28px auto; max-width:320px; padding:18px; border-radius:18px; background:#e0e7ff; color:#4338ca; font-size:34px; font-weight:700; letter-spacing:6px; font-family:'Courier New',monospace;">%s</div>
      <p style="margin:18px 0; font-size:16px; line-height:1.7; color:#536079;">
        Your verification code is valid for <strong>10 minutes</strong>. Please use it promptly. For your account security, do not share this code with anyone.
      </p>
      <p style="margin:30px 0 0; font-size:14px; color:#94a3b8; line-height:1.6;">
        If you didn't request this code, please ignore this email.<br/>
        Need help? Contact <a href="mailto:%s" style="color:#6366f1; text-decoration:none;">%s</a>
      </p>
    </div>
    <div style="background:#f7f9fa; border-top:2px solid #e9eff4; border-bottom:2px solid #e9eff4; padding:18px 24px; text-align:center; font-size:12px; color:#94a3b8;">
      &copy; %s. All rights reserved.<br />
      <a href="%s" style="color:#6366f1; text-decoration:none;">%s</a>
    </div>
  </div>
  <style>
    @media (max-width:600px) {
      div[style*="padding:28px 32px"] {
        padding:22px !important;
      }
      span[style*="letter-spacing:0.08em"] {
        font-size:18px !important;
        letter-spacing:0.06em !important;
      }
      div[style*="padding:42px 36px"] {
        padding:32px 20px !important;
      }
      div[style*="letter-spacing:6px"] {
        font-size:26px !important;
      }
    }
  </style>
</body>
</html>`,
		brand,
		code,
		supportEmail,
		supportEmail,
		brand,
		websiteURL,
		websiteHost,
	)
	return subject, htmlBody, textBody
}
