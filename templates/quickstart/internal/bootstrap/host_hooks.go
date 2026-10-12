package bootstrap

import (
	"context"
	"fmt"
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
	if event.TotalCredits == 0 {
		return nil
	}
	if err := deps.Users.GrantCredits(ctx, envelope.UserID, "stripe", envelope.ProviderEventID, event.TotalCredits); err != nil {
		return fmt.Errorf("apply purchased credits: %w", err)
	}
	return nil
}

// onReferralRegistered 在邀请关系绑定成功后调用。默认不发放任何奖励。
// 接入步骤与可复制示例见后端根目录 REFERRAL_HOOKS.md。
// 注册即送会员或积分时，在这里编写当前 SaaS 的规则。
// event.Referral.ReferrerID 是邀请人，RefereeID 是被邀请人。
// 回调可能重试；用邀请 ID + 奖励类型作为幂等键，并向下传递 ctx。
func onReferralRegistered(ctx context.Context, deps hostapi.Deps, envelope referralevent.Envelope, event referralevent.ReferralRegistered) error {
	return nil
}

// onReferralActivated 是邀请激活后的奖励扩展点，默认不发放积分、会员或其他奖励。
// 接入步骤与可复制示例见后端根目录 REFERRAL_HOOKS.md。
// 直接在函数内编写规则并调用自己的 service，无需奖励配置。
// event.Referral 包含邀请 ID、双方用户 ID 和激活时间；deps 提供宿主依赖。
// 回调可能因支付重试或管理员重放再次执行，发奖必须幂等；失败返回 error。
func onReferralActivated(ctx context.Context, deps hostapi.Deps, envelope referralevent.Envelope, event referralevent.ReferralActivated) error {
	return nil
}
