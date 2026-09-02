package bootstrap

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	emaildomain "github.com/brizenchi/go-modules/modules/email/domain"
	referralevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
)

// 这些函数就是当前 SaaS 的业务回调。共享模块只发布事件；
// 要发邮件、送积分、创建工作区或调用其他 service，都在这里决定。

func onUserSignedUp(ctx context.Context, deps hostapi.Deps, _ authevent.Envelope, event authevent.UserSignedUp) error {
	if deps.Config.SignupCredits > 0 {
		if err := deps.Users.GrantCredits(ctx, event.Identity.UserID, "signup", event.Identity.UserID, deps.Config.SignupCredits); err != nil {
			return fmt.Errorf("grant signup credits: %w", err)
		}
	}

	welcome := deps.Config.WelcomeEmail
	if !welcome.Enabled || deps.Modules.Email == nil {
		return nil
	}
	if only := strings.TrimSpace(welcome.OnlyProvider); only != "" && !strings.EqualFold(only, string(event.Identity.Provider)) {
		return nil
	}
	subject := strings.TrimSpace(welcome.Subject)
	if subject == "" {
		subject = "欢迎加入"
	}
	textBody := welcome.TextBody
	htmlBody := welcome.HTMLBody
	if strings.TrimSpace(textBody) == "" && strings.TrimSpace(htmlBody) == "" {
		textBody = "你的账号已经创建成功。"
	}
	_, err := deps.Modules.Email.Send(ctx, &emaildomain.Message{
		To:       []emaildomain.Address{{Name: event.Identity.Username, Email: event.Identity.Email}},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	})
	return err
}

func onUserLoggedIn(_ context.Context, _ hostapi.Deps, _ authevent.Envelope, _ authevent.UserLoggedIn) error {
	return nil
}

func onSubscriptionActivated(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.SubscriptionActivated) error {
	return nil
}

func onSubscriptionRenewed(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.SubscriptionRenewed) error {
	return nil
}

func onSubscriptionUpdated(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.SubscriptionUpdated) error {
	return nil
}

func onSubscriptionReactivated(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.SubscriptionReactivated) error {
	return nil
}

func onSubscriptionCanceling(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.SubscriptionCanceling) error {
	return nil
}

func onSubscriptionCanceled(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.SubscriptionCanceled) error {
	return nil
}

func onPaymentFailed(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.PaymentFailed) error {
	return nil
}

func onCreditsPurchased(ctx context.Context, deps hostapi.Deps, envelope billingevent.Envelope, event billingevent.CreditsPurchased) error {
	if event.TotalCredits == 0 {
		return nil
	}
	if err := deps.Users.GrantCredits(ctx, envelope.UserID, "stripe", envelope.ProviderEventID, event.TotalCredits); err != nil {
		return fmt.Errorf("apply purchased credits: %w", err)
	}
	return nil
}

func onReferralRegistered(_ context.Context, _ hostapi.Deps, _ referralevent.Envelope, _ referralevent.ReferralRegistered) error {
	return nil
}

func onReferralActivated(ctx context.Context, deps hostapi.Deps, _ referralevent.Envelope, event referralevent.ReferralActivated) error {
	reward := int64(event.Referral.RewardCredits)
	if reward == 0 {
		return nil
	}
	sourceID := strconv.FormatUint(event.Referral.ID, 10)
	if event.Referral.ID == 0 {
		sourceID = event.Referral.ReferrerID + ":" + event.Referral.RefereeID
	}
	if err := deps.Users.GrantCredits(ctx, event.Referral.ReferrerID, "referral", sourceID, reward); err != nil {
		return fmt.Errorf("apply referral reward: %w", err)
	}
	return nil
}
