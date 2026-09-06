"use client";

import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { SiteShell } from "@/components/site-shell";
import { SignInDialog } from "@/components/sign-in-dialog";
import { Notice, Panel, DetailRows } from "@/components/ui";
import { readReferralCode, readSession, writeReferralCode, REFERRAL_EVENT, SESSION_EVENT } from "@/lib/auth";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";

function InvitePageInner() {
  const params = useSearchParams();
  const { t } = useI18n();
  const [savedCode, setSavedCode] = useState("");
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [ready, setReady] = useState(false);
  const [signInOpen, setSignInOpen] = useState(false);
  const inboundCode = params.get("ref");

  useEffect(() => {
    // Existing accounts cannot accept another invitation. Do not save an
    // inbound code that could leak into a later signup on a shared browser.
    if (!readSession() && inboundCode !== null) {
      writeReferralCode(inboundCode);
    }
    const sync = () => {
      setSavedCode(readReferralCode());
      setSession(readSession());
      setReady(true);
    };
    sync();
    window.addEventListener(REFERRAL_EVENT, sync);
    window.addEventListener(SESSION_EVENT, sync);
    window.addEventListener("storage", sync);
    return () => {
      window.removeEventListener(REFERRAL_EVENT, sync);
      window.removeEventListener(SESSION_EVENT, sync);
      window.removeEventListener("storage", sync);
    };
  }, [inboundCode]);

  return (
    <SiteShell
      eyebrow={t({ en: "You're invited", zh: "好友邀请" })}
      title={t({ en: `Get started with ${appEnv.appName}.`, zh: `从好友的邀请开始使用 ${appEnv.appName}。` })}
      description={t({ en: "Create your account to explore the product. A valid invitation is linked automatically at signup, whichever sign-in method you choose.", zh: "注册账号，开始体验产品。选择任一登录方式首次注册时，系统会自动校验邀请码并建立邀请关系。" })}
      sideTitle={t({ en: "How the invitation works", zh: "邀请如何生效" })}
      showEnvironment={false}
      sideBody={<DetailRows rows={[
        { label: t({ en: "1 · Join", zh: "1 · 注册" }), value: t({ en: "Create a new account through this invitation.", zh: "通过邀请首次创建账号。" }) },
        { label: t({ en: "2 · Explore", zh: "2 · 体验" }), value: t({ en: "Try the product and choose a plan when you need it.", zh: "体验产品，按需要选择订阅套餐。" }) },
        { label: t({ en: "3 · Reward", zh: "3 · 奖励" }), value: t({ en: "Your inviter earns credits after your first qualifying paid subscription, within the reward window.", zh: "在奖励期限内首次完成符合条件的付费订阅，邀请人获得积分。" }) }
      ]} />}
      toc={[{ id: "accept-invite", label: t({ en: "Accept invitation", zh: "接受邀请" }) }]}
    >
      <div className="page-grid">
        <Panel title={t({ en: "Your invitation", zh: "你的邀请" })} subtitle={t({ en: "One account, one invitation. Joining does not require a payment.", zh: "每个新账号只能绑定一次邀请，注册无需付款。" })}>
          <div id="accept-invite" />
          {!ready ? (
            <p role="status">{t({ en: "Loading your invitation…", zh: "正在读取邀请…" })}</p>
          ) : session ? (
            <>
              <Notice>{t({ en: "You're signed in. Existing accounts cannot accept a new invitation. If you just registered, a valid code was submitted during signup; your inviter can check their referral history.", zh: "你已登录。已有账号不能重新绑定邀请；如果你刚刚完成注册，有效邀请码已随注册提交，邀请人可在邀请记录中查看结果。" })}</Notice>
              <div className="button-row">
                <Link className="button primary" href="/account">{t({ en: "Go to my account", zh: "进入我的账号" })}</Link>
                <Link className="button" href="/referrals">{t({ en: "Invite my friends", zh: "邀请我的好友" })}</Link>
              </div>
            </>
          ) : savedCode ? (
            <>
              <Notice>
                {t({ en: "Invite code saved", zh: "已记录邀请码" })}: <span className="inline-code">{savedCode}</span>
                <p>{t({ en: "Keep using this browser to finish signup. The code is checked when your new account is created.", zh: "请在当前浏览器完成注册。邀请码将在新账号创建时校验，当前尚未完成绑定。" })}</p>
              </Notice>
              <div className="button-row">
                <button className="button primary" type="button" onClick={() => setSignInOpen(true)}>{t({ en: "Create my account", zh: "注册并开始使用" })}</button>
                <Link className="button" href="/">{t({ en: "Explore first", zh: "先了解产品" })}</Link>
              </div>
            </>
          ) : (
            <>
              <Notice>{t({ en: "This link has no invite code. Ask your friend to copy their complete invitation link. You can also create an account without an invitation.", zh: "此链接没有邀请码，请好友重新复制完整的邀请链接。你也可以直接注册使用。" })}</Notice>
              <div className="button-row">
                <button className="button primary" type="button" onClick={() => setSignInOpen(true)}>{t({ en: "Create an account", zh: "直接注册" })}</button>
              </div>
            </>
          )}
        </Panel>
        {appEnv.demoMode ? (
          <Panel title={t({ en: "Try the complete referral flow", zh: "体验完整邀请流程" })}>
            <p>{t({ en: "Account creation and referral records are real in this demo. Payments use Stripe's test environment. To test an invitation, open it in a separate browser or private window and register a new account, then complete a qualifying test subscription. The inviter can refresh their referral center to see the result.", zh: "演示站的注册和邀请记录都是真实的，支付使用 Stripe 测试环境。测试时，请在另一个浏览器或无痕窗口打开邀请链接，用新账号注册，再完成符合条件的测试订阅。邀请人刷新邀请中心即可查看结果。" })}</p>
            <Link className="button" href="/#test-payment">{t({ en: "View test payment instructions", zh: "查看测试支付说明" })}</Link>
          </Panel>
        ) : null}
      </div>
      <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} />
    </SiteShell>
  );
}

export default function InvitePage() {
  return <Suspense fallback={null}><InvitePageInner /></Suspense>;
}
