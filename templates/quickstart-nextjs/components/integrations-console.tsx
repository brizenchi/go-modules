"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { readSession } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { buildIntegrationPatch, integrationDraft, integrationIssues, type IntegrationDraft, type IntegrationIssue } from "@/lib/integration-form";
import { getAdminIntegrations, saveAdminIntegration, IntegrationError, type IntegrationFailure, type IntegrationProvider, type IntegrationView, type IntegrationsView } from "@/lib/integrations-api";
import { newIntentKey, useConsoleAction, useConsoleSession } from "./console-kit";
import adminStyles from "./admin-shell.module.css";
import styles from "./integrations-console.module.css";

type Copy = { en: string; zh: string };
const failureCopy: Record<IntegrationFailure, Copy> = {
  auth: { en: "Your session expired. Sign in again to continue.", zh: "登录已过期，请重新登录后继续。" },
  forbidden: { en: "This account cannot manage service settings.", zh: "当前账号无权管理服务配置。" },
  conflict: { en: "The saved version changed. Reload saved settings before editing again. If your previous save timed out, retry the same save first.", zh: "已保存的版本发生变化，请重新读取配置后编辑。如果上次保存超时，可先重试相同的保存操作。" },
  invalid: { en: "Check the field formats, required keys, Stripe mode and deployment requirements. Existing keys must match the selected mode too.", zh: "请检查字段格式、必填密钥、Stripe 模式和部署要求，保留的密钥也必须与所选模式一致。" },
  storage: { en: "Unable to read or save configuration. Check the API database connection and service configuration tables, then try again.", zh: "暂时无法读取或保存配置。请检查后端数据库连接及配置表是否就绪，再重试。" },
  unavailable: { en: "Unable to complete the request. Check your connection and try again.", zh: "请求未能完成，请检查连接后重试。" }
};
const issueCopy: Record<IntegrationIssue, Copy> = {
  reason: { en: "Enter a change reason of 3–500 characters.", zh: "请填写 3–500 个字的修改原因。" },
  email: { en: "Enter a valid sender email address.", zh: "请填写有效的发件邮箱地址。" },
  name: { en: "The sender name can contain up to 100 characters.", zh: "发件人名称最多 100 个字。" },
  key: { en: "Check the key prefix and mode. Keys cannot contain spaces or line breaks.", zh: "请检查密钥前缀与模式，密钥不能包含空格或换行。" },
  missing_key: { en: "Add the required keys before enabling this service.", zh: "启用服务前，请先填写必需的密钥。" },
  mode: { en: "Select Test or Live mode for Stripe.", zh: "请选择 Stripe 测试或正式模式。" },
  price: { en: "Use price_ IDs. Enter up to 20 credit prices, separated by commas.", zh: "价格请使用 price_ 编号，积分包最多 20 个价格，以英文逗号分隔。" },
  duplicate_price: { en: "Each subscription, lifetime purchase and credit package needs a different price ID.", zh: "每个订阅、买断和积分包都需要独立的价格编号，不能重复。" },
  missing_price: { en: "Configure at least one price before enabling Stripe.", zh: "启用 Stripe 前，请至少配置一个价格。" },
  trial: { en: "Trial days must be a whole number from 0 to 730.", zh: "试用天数须为 0–730 的整数。" },
  credits: { en: "Credits per package must be a whole number from 1 to 1,000,000.", zh: "每个积分包须为 1–1,000,000 的整数。" }
};

function IntegrationFailureNotice({ error }: { error: unknown }) {
  const { t } = useI18n();
  if (!error) return null;
  const kind = error instanceof IntegrationError ? error.kind : "unavailable";
  return <p className={styles.error} role="alert">{t(failureCopy[kind])}</p>;
}

function ProviderForm({ view, storageReady, localPreview, onSaved, onBusy }: {
  view: IntegrationView;
  storageReady: boolean;
  localPreview: boolean;
  onSaved: (view: IntegrationView) => void;
  onBusy: (provider: IntegrationProvider, busy: boolean) => void;
}) {
  const { t } = useI18n();
  const { session } = useConsoleSession();
  const token = session?.user.role === "admin" ? session.token : "";
  const [draft, setDraft] = useState(() => integrationDraft(view));
  const [reason, setReason] = useState("");
  const [issues, setIssues] = useState<IntegrationIssue[]>([]);
  const action = useConsoleAction(token);
  const intent = useRef({ payload: "", key: "" });
  const controller = useRef<AbortController | null>(null);
  const resend = view.provider === "resend";
  const name = resend ? "Resend" : "Stripe";
  const database = draft.source === "database";
  const id = (field: string) => `integration-${view.provider}-${field}`;

  useEffect(() => { onBusy(view.provider, action.busy); }, [action.busy, onBusy, view.provider]);
  useEffect(() => {
    const requests = controller;
    const savedIntent = intent;
    return () => { requests.current?.abort(); savedIntent.current = { payload: "", key: "" }; onBusy(view.provider, false); };
  }, [onBusy, view.provider]);
  function update(change: Partial<IntegrationDraft>) { setDraft((current) => ({ ...current, ...change })); setIssues([]); }
  function field(key: string, value: string) { update({ fields: { ...draft.fields, [key]: value } }); }

  function submit(event: FormEvent) {
    event.preventDefault();
    if (!storageReady || action.busy) return;
    const nextIssues = integrationIssues(view, draft, reason);
    setIssues(nextIssues);
    if (nextIssues.length) return;
    const patch = buildIntegrationPatch(view, draft, reason);
    const payload = JSON.stringify(patch);
    if (intent.current.payload !== payload || !intent.current.key) intent.current = { payload, key: newIntentKey() };
    const key = intent.current.key;
    void action.run(async () => {
      const request = new AbortController();
      controller.current = request;
      const timer = setTimeout(() => request.abort(), 15000);
      try { return await saveAdminIntegration(token, view.provider, patch, key, request.signal); }
      finally { clearTimeout(timer); if (controller.current === request) controller.current = null; }
    }, (saved) => {
      intent.current = { payload: "", key: "" };
      setDraft(integrationDraft(saved));
      setReason("");
      action.setMessage(t(localPreview
        ? { en: "Saved to this local preview. External services stay inactive.", zh: "已保存到本地预览，外部服务保持停用。" }
        : saved.restart_required
          ? { en: "Settings saved. Restart all API instances to apply this version.", zh: "配置已保存，重启所有后端实例后应用此版本。" }
          : { en: "Settings saved. This API process already uses these values.", zh: "配置已保存，当前 API 进程已使用这些设置。" }));
      onSaved(saved);
    });
  }

  function textField(key: string, label: Copy, options: { type?: "text" | "email" | "number"; placeholder?: string; maxLength?: number; min?: number; max?: number; help?: Copy; required?: boolean } = {}) {
    return <div className="field" key={key}>
      <label htmlFor={id(key)}>{t(label)}</label>
      <input id={id(key)} value={draft.fields[key] || ""} type={options.type || "text"} placeholder={options.placeholder} maxLength={options.maxLength ?? 2048} min={options.min} max={options.max} step={options.type === "number" ? 1 : undefined} required={options.required} spellCheck={false} onChange={(event) => field(key, event.target.value)} aria-describedby={options.help ? `${id(key)}-hint` : undefined} />
      {options.help ? <small id={`${id(key)}-hint`} className={styles.hint}>{t(options.help)}</small> : null}
    </div>;
  }

  function secretField(key: string, label: Copy, placeholder: string) {
    const configured = view.secrets[key]?.configured;
    return <div className="field" key={key}>
      <label htmlFor={id(key)}>{t(label)} <span className={configured ? styles.configured : styles.unconfigured}>{t(configured ? { en: "Configured", zh: "已配置" } : { en: "Not configured", zh: "未配置" })}</span></label>
      <input id={id(key)} type="password" autoComplete="new-password" data-lpignore="true" spellCheck={false} maxLength={4096} placeholder={configured ? t({ en: "Leave blank to keep the saved key", zh: "留空保留已配置的密钥" }) : placeholder} value={draft.secrets[key] || ""} disabled={!draft.enabled && draft.clearSecrets[key]} onChange={(event) => update({ secrets: { ...draft.secrets, [key]: event.target.value } })} aria-describedby={`${id(key)}-hint`} />
      <small id={`${id(key)}-hint`} className={styles.hint}>{t({ en: "Only enter a value to replace this key. Saved keys are never displayed.", zh: "仅在更换时填写，已保存的密钥不会回显。" })}</small>
      {!draft.enabled && configured ? <label className={styles.check}><input type="checkbox" checked={Boolean(draft.clearSecrets[key])} onChange={(event) => update({ clearSecrets: { ...draft.clearSecrets, [key]: event.target.checked }, secrets: { ...draft.secrets, [key]: "" } })} />{t({ en: `Remove ${label.en} when saving`, zh: `保存时清除${label.zh}` })}</label> : null}
    </div>;
  }

  return <section className={styles.provider} aria-labelledby={`${id("title")}`}>
    <header className={styles.providerHeader}>
      <span className={`${styles.providerMark} ${resend ? styles.resend : styles.stripe}`} aria-hidden="true">{resend ? "R" : "S"}</span>
      <div><h2 id={id("title")}>{name}</h2><p>{t(resend ? { en: "Email delivery & sign-in codes", zh: "邮件发送与登录验证码" } : { en: "Subscriptions, payments & credit packages", zh: "订阅、支付与积分包" })}</p></div>
      <span className={view.restart_required || localPreview ? styles.pending : styles.status}>{t(localPreview ? { en: "Preview only", zh: "仅本地预览" } : view.restart_required ? { en: "Restart required", zh: "待重启生效" } : { en: "No pending changes", zh: "无待生效变更" })}</span>
    </header>
    <div className={styles.stateGrid}>
      <div><span>{t({ en: "Saved configuration", zh: "已保存配置" })}</span><strong>{t(view.source === "database" ? { en: "Database", zh: "后台数据库" } : { en: "Environment", zh: "环境变量" })} · {t(view.enabled ? { en: "Enabled", zh: "启用" } : { en: "Disabled", zh: "停用" })}</strong><small>{t({ en: `Version ${view.version}`, zh: `版本 ${view.version}` })}</small></div>
      <div><span>{t({ en: "Current API process", zh: "当前 API 进程" })}</span><strong>{localPreview ? t({ en: "External services inactive", zh: "外部服务未启用" }) : <>{t(view.active_source === "database" ? { en: "Database", zh: "后台数据库" } : { en: "Environment", zh: "环境变量" })} · {t(view.active_enabled ? { en: "Enabled", zh: "启用" } : { en: "Disabled", zh: "停用" })}</>}</strong><small>{t({ en: "Connection has not been tested", zh: "尚未验证服务商连接" })}</small></div>
    </div>
    <form onSubmit={submit} className={styles.form}>
      <fieldset disabled={action.busy || !storageReady} className={styles.fields}>
        <div className="field"><label htmlFor={id("source")}>{t({ en: "Configuration source", zh: "配置来源" })}</label><select id={id("source")} value={draft.source} onChange={(event) => update({ source: event.target.value as IntegrationDraft["source"], secrets: {}, clearSecrets: {} })}><option value="environment">{t({ en: "Deployment environment variables", zh: "使用部署环境变量" })}</option><option value="database">{t({ en: "Manage in this dashboard", zh: "在后台管理配置" })}</option></select><small className={styles.hint}>{t({ en: "Dashboard settings override environment settings for this service.", zh: "该服务的后台配置优先于环境变量。" })}</small></div>
        {database ? <>
          <label className={styles.enable}><input type="checkbox" checked={draft.enabled} onChange={(event) => update({ enabled: event.target.checked, clearSecrets: {} })} /><span><strong>{t({ en: `Enable ${name}`, zh: `启用 ${name}` })}</strong><small>{t({ en: "Applied after saving and restarting the API", zh: "保存并重启 API 后生效" })}</small></span></label>
          {resend ? <>
            <div className={styles.instructions}><p>{t({ en: "Create a Resend API key and verify your sending domain. Use an address on that domain as the sender.", zh: "在 Resend 创建 API Key 并验证发信域名，再填写该域名下的发件邮箱。" })}</p><div className={styles.links}><a href="https://resend.com/api-keys" target="_blank" rel="noreferrer">{t({ en: "API keys", zh: "获取 API Key" })} ↗</a><a href="https://resend.com/docs/dashboard/domains/introduction" target="_blank" rel="noreferrer">{t({ en: "Verify a domain", zh: "验证发信域名" })} ↗</a></div></div>
            {secretField("api_key", { en: "Resend API key", zh: "Resend API Key" }, "re_…")}
            <div className={styles.grid}>{textField("sender_email", { en: "Sender email", zh: "发件邮箱" }, { type: "email", placeholder: "hello@your-domain.com", maxLength: 254, required: draft.enabled })}{textField("sender_name", { en: "Sender name", zh: "发件人名称" }, { placeholder: t({ en: "Your product name", zh: "你的产品名称" }), maxLength: 200 })}</div>
            <label className={styles.check}><input type="checkbox" checked={draft.enabled && draft.fields.email_auth_enabled === "true"} disabled={!draft.enabled} onChange={(event) => field("email_auth_enabled", String(event.target.checked))} />{t({ en: "Use Resend for customer email-code sign-in", zh: "使用 Resend 发送用户登录验证码" })}</label>
          </> : <>
            <div className={styles.instructions}><p>{t({ en: "Choose a Stripe mode, then use keys and prices from that same mode. Configure your webhook endpoint in Stripe before accepting payments.", zh: "先选择 Stripe 模式，再填写同一模式下的密钥与价格编号，并在 Stripe 配置 Webhook 接收地址。" })}</p><div className={styles.links}><a href="https://docs.stripe.com/keys" target="_blank" rel="noreferrer">{t({ en: "API keys", zh: "API 密钥说明" })} ↗</a><a href="https://docs.stripe.com/webhooks" target="_blank" rel="noreferrer">Webhook ↗</a><a href="https://docs.stripe.com/products-prices/manage-prices" target="_blank" rel="noreferrer">{t({ en: "Products & prices", zh: "产品与价格" })} ↗</a></div></div>
            <div className="field"><label htmlFor={id("mode")}>{t({ en: "Stripe mode", zh: "Stripe 模式" })}</label><select id={id("mode")} value={draft.fields.mode} onChange={(event) => field("mode", event.target.value)}><option value="test">{t({ en: "Test — simulated payments", zh: "测试模式 · 模拟支付" })}</option><option value="live">{t({ en: "Live — real payments", zh: "正式模式 · 真实支付" })}</option></select><small className={styles.hint}>{t({ en: "When switching modes, replace the keys and choose prices from the matching Stripe dashboard.", zh: "切换模式时，请同步更换密钥，并从对应模式的 Stripe 控制台选择价格。" })}</small></div>
            <div className={styles.grid}>{secretField("secret_key", { en: "Secret API key", zh: "Secret API Key" }, `sk_${draft.fields.mode}_…`)}{secretField("webhook_secret", { en: "Webhook signing secret", zh: "Webhook 签名密钥" }, "whsec_…")}</div>
            {textField("publishable_key", { en: "Publishable key (optional)", zh: "Publishable Key（选填）" }, { placeholder: `pk_${draft.fields.mode}_…` })}
            <fieldset className={styles.prices}><legend>{t({ en: "Subscription prices", zh: "订阅价格" })}</legend><p className={styles.hint}>{t({ en: "Paste Stripe price_ IDs, not amounts. Leave unavailable offers blank. Monthly and yearly prices should use the matching recurring interval.", zh: "填写 Stripe 的 price_ 编号，留空表示不提供该选项。月付和年付价格须使用对应的周期。" })}</p>
              {([ ["starter", "Starter"], ["pro", "Pro"], ["premium", "Premium"] ] as const).map(([key, label]) => <div className={styles.priceRow} key={key}><strong>{label}</strong>{textField(`${key}_monthly`, { en: `${label} monthly`, zh: `${label} 月付` }, { placeholder: "price_…" })}{textField(`${key}_yearly`, { en: `${label} yearly`, zh: `${label} 年付` }, { placeholder: "price_…" })}</div>)}
            </fieldset>
            <div className={styles.grid}>{textField("lifetime", { en: "Lifetime purchase price (optional)", zh: "一次性买断价格（选填）" }, { placeholder: "price_…" })}{textField("credit_price_ids", { en: "Credit package prices (optional)", zh: "积分包价格（选填）" }, { placeholder: "price_…, price_…", help: { en: "Up to 20 unique IDs, separated by commas", zh: "最多 20 个独立编号，以英文逗号分隔" } })}</div>
            <div className={styles.grid}>{textField("trial_days", { en: "Subscription trial days", zh: "订阅试用天数" }, { type: "number", min: 0, max: 730, required: true, help: { en: "0 means no trial", zh: "0 表示不提供试用" } })}{textField("credits_per_package", { en: "Credits per package", zh: "每个积分包包含的积分" }, { type: "number", min: 1, max: 1000000, required: true, help: { en: "Applies to each configured credit package", zh: "适用于上方配置的每个积分包" } })}</div>
          </>}
        </> : <p className={styles.environment}>{t(view.source === "database" ? { en: "Saving removes this service's dashboard override. Restart the API to use its deployment environment configuration.", zh: "保存后将移除此服务的后台覆盖配置，重启 API 后使用部署环境变量。" } : { en: "This service follows the deployment environment. Choose “Manage in this dashboard” to edit it here.", zh: "此服务目前跟随部署环境变量。选择「在后台管理配置」即可在此编辑。" })}</p>}
        <div className={styles.saveArea}><div className="field"><label htmlFor={id("reason")}>{t({ en: "Change reason", zh: "修改原因" })}</label><input id={id("reason")} value={reason} onChange={(event) => { setReason(event.target.value); setIssues([]); }} placeholder={t({ en: "For example: set up test payments", zh: "例如：配置测试支付" })} minLength={3} maxLength={500} required /></div>
          {issues.length ? <ul className={styles.error} role="alert">{issues.map((issue) => <li key={issue}>{t(issueCopy[issue])}</li>)}</ul> : null}
          <button type="submit" className="button primary" disabled={action.busy || !token || !storageReady}>{t(action.busy ? { en: "Saving…", zh: "保存中…" } : { en: `Save ${name} settings`, zh: `保存 ${name} 配置` })}</button>
        </div>
      </fieldset>
      <IntegrationFailureNotice error={action.error} />
      {action.message ? <p className={styles.success} role="status">{action.message}</p> : null}
    </form>
  </section>;
}

function IntegrationsSession({ token }: { token: string }) {
  const { t } = useI18n();
  const [configuration, setConfiguration] = useState<IntegrationsView | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  const [busy, setBusy] = useState<Partial<Record<IntegrationProvider, boolean>>>({});
  const generation = useRef(0);
  const requestRef = useRef<AbortController | null>(null);
  const onBusy = useCallback((provider: IntegrationProvider, value: boolean) => { setBusy((current) => current[provider] === value ? current : { ...current, [provider]: value }); }, []);
  const load = useCallback(async () => {
    const current = ++generation.current;
    requestRef.current?.abort();
    const request = new AbortController();
    requestRef.current = request;
    const timer = setTimeout(() => request.abort(), 8000);
    setLoading(true); setError(null);
    try {
      const result = await getAdminIntegrations(token, request.signal);
      if (generation.current === current && readSession()?.token === token) { setConfiguration(result); setRevision((value) => value + 1); }
    } catch (failure) { if (generation.current === current && readSession()?.token === token) setError(failure); }
    finally { clearTimeout(timer); if (generation.current === current) setLoading(false); }
  }, [token]);
  useEffect(() => { const counter = generation; const request = requestRef; void load(); return () => { counter.current++; request.current?.abort(); }; }, [load]);
  function saved(view: IntegrationView) { setConfiguration((current) => current ? { ...current, providers: current.providers.map((provider) => provider.provider === view.provider ? view : provider) } : current); }
  return <>
    <header className={`${adminStyles.pageHeader} ${styles.pageHeader}`}><div><span className={adminStyles.eyebrow}>{t({ en: "EMAIL & PAYMENTS", zh: "邮件与支付" })}</span><h1>{t({ en: "Service configuration", zh: "服务配置" })}</h1><p>{t({ en: "Connect the services behind your SaaS. Save a configuration here, then restart all API instances to apply it.", zh: "配置 SaaS 使用的邮件与支付服务。保存配置后，重启所有后端实例使其生效。" })}</p></div><button className="button" type="button" disabled={loading || Object.values(busy).some(Boolean)} onClick={() => void load()}>{t({ en: "Reload saved settings", zh: "重新读取配置" })}</button></header>
    <div className={styles.stack}>
      <IntegrationFailureNotice error={error} />
      {loading ? <p role="status">{t({ en: "Loading saved and active configurations…", zh: "正在读取已保存配置和当前生效状态…" })}</p> : null}
      {configuration?.local_preview ? <aside className={styles.preview} role="status"><strong>{t({ en: "Local preview", zh: "本地预览" })}</strong><p>{t({ en: "Settings are saved only in this temporary preview database. Resend and Stripe stay inactive, including after a preview restart. Use test credentials here.", zh: "配置仅保存在此预览的临时数据库中。即使重启预览，Resend 和 Stripe 也不会连接外部服务。请使用测试凭据。" })}</p></aside> : null}
      {configuration && !configuration.storage_ready ? <IntegrationFailureNotice error={new IntegrationError("storage")} /> : null}
      {configuration ? <>{configuration.providers.map((view) => <ProviderForm key={`${revision}-${view.provider}`} view={view} storageReady={configuration.storage_ready && !loading} localPreview={configuration.local_preview} onSaved={saved} onBusy={onBusy} />)}<p className={styles.footnote}>{t({ en: "Service credentials are stored as plaintext in the database and are never returned to these fields. Database connection details, the administrator password and JWT secrets stay in deployment environment variables.", zh: "服务密钥以明文保存在数据库中，不会回显到表单。数据库连接、管理员密码和 JWT 密钥仍通过部署环境变量管理。" })}</p></> : null}
    </div>
  </>;
}

export function IntegrationsConsole() {
  const { session, ready } = useConsoleSession();
  if (!ready || session?.user.role !== "admin") return null;
  return <IntegrationsSession key={session.token} token={session.token} />;
}
