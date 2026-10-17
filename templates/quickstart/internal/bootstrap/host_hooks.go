package bootstrap

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	authevent "github.com/brizenchi/go-modules/modules/auth/event"
	billingevent "github.com/brizenchi/go-modules/modules/billing/event"
	emaildomain "github.com/brizenchi/go-modules/modules/email/domain"
	referraldomain "github.com/brizenchi/go-modules/modules/referral/domain"
	referralevent "github.com/brizenchi/go-modules/modules/referral/event"
	"github.com/brizenchi/quickstart-template/internal/credit"
	"github.com/brizenchi/quickstart-template/internal/creditpurchase"
	"github.com/brizenchi/quickstart-template/internal/hostapi"
)

// 这些函数就是当前 SaaS 的业务回调。共享模块只发布事件；
// 要发邮件、送积分、创建工作区或调用其他 service，都在这里决定。

func onUserSignedUp(ctx context.Context, deps hostapi.Deps, _ authevent.Envelope, event authevent.UserSignedUp) error {
	if deps.Config.SignupCredits > 0 {
		if err := deps.Credits.GrantCredits(ctx, event.Identity.UserID, "signup", event.Identity.UserID, deps.Config.SignupCredits); err != nil {
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

func onTrialConverted(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.TrialConverted) error {
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

func onRefundUpdated(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.RefundUpdated) error {
	return nil
}

func onChargeRefunded(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.ChargeRefunded) error {
	return nil
}

func onDisputeUpdated(_ context.Context, _ hostapi.Deps, _ billingevent.Envelope, _ billingevent.DisputeUpdated) error {
	return nil
}

func onCreditsPurchased(ctx context.Context, deps hostapi.Deps, envelope billingevent.Envelope, event billingevent.CreditsPurchased) error {
	if err := creditpurchase.New(deps.DB, deps.Credits).Grant(ctx, envelope, event); err != nil {
		return fmt.Errorf("apply purchased credits: %w", err)
	}
	return nil
}

// onReferralRegistered 注册绑定成功立即奖励，与购买无关。
func onReferralRegistered(ctx context.Context, deps hostapi.Deps, _ referralevent.Envelope, event referralevent.ReferralRegistered) error {
	return grantReferralCredits(ctx, deps, event.Referral, "referral_signup", "邀请注册奖励", "受邀注册奖励")
}

// Fixed referral rewards belong to the host, not provider configuration.
const referralInviterCredits = 20
const referralInviteeCredits = 10

// onReferralActivated 在符合条件的首次付费后给双方发积分。
// 同一邀请的每种奖励只发一次；第二笔失败时，重试只补齐缺失奖励。
// 修改奖励规则后，需同步邀请页面文案；接入说明见 REFERRAL_HOOKS.md。
func onReferralActivated(ctx context.Context, deps hostapi.Deps, _ referralevent.Envelope, event referralevent.ReferralActivated) error {
	// 可选的首次付费额外奖励：不需要时注释以下 return，改为 return nil。
	// 同时把 subscriptions.go 三处激活调用的奖励统计值改为 0，并更新页面文案。
	return grantReferralCredits(ctx, deps, event.Referral, "referral", "邀请付费奖励", "受邀付费奖励")
}

func grantReferralCredits(ctx context.Context, deps hostapi.Deps, ref referraldomain.Referral, source, inviterReason, inviteeReason string) error {
	if ref.ID == 0 || ref.ReferrerID == "" || ref.RefereeID == "" || ref.ReferrerID == ref.RefereeID {
		return fmt.Errorf("grant referral credits: invalid referral")
	}
	sourceID := strconv.FormatUint(ref.ID, 10)
	if _, err := deps.Credits.GrantCreditsWithExpiry(ctx, credit.Grant{
		UserID: ref.ReferrerID, Source: source + "_inviter", SourceID: sourceID,
		Amount: referralInviterCredits, Reason: inviterReason,
	}); err != nil {
		return fmt.Errorf("grant inviter credits: %w", err)
	}
	if _, err := deps.Credits.GrantCreditsWithExpiry(ctx, credit.Grant{
		UserID: ref.RefereeID, Source: source + "_invitee", SourceID: sourceID,
		Amount: referralInviteeCredits, Reason: inviteeReason,
	}); err != nil {
		return fmt.Errorf("grant invitee credits: %w", err)
	}
	return nil
}
