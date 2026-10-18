# Billing UI scenarios

Shared labels and outcomes live in `lib/billing-presentation.ts`. Eligibility stays in
`lib/subscription-change.ts`; wording must not bypass pending payments or other restrictions.
Every label has English and Chinese text. Public pricing and private billing preserve the selected language.

## Copy references

Checked against official product help on 2026-10-18:

- [SendPulse: View your subscriptions](https://web.sendpulse.com/knowledge-base/account-settings/billing/service-subscriptions) uses **Cancel plan change** followed by confirmation to remove a future change and continue the current plan. It also documents replacing a scheduled downgrade by selecting a different plan.
- [Notion: Change your plan](https://www.notion.com/help/upgrade-or-downgrade-your-plan) distinguishes current access from downgrades that begin at the end of the billing cycle. Pricing shows the current plan and upcoming plan separately, using the API-provided effective date. **Keep {plan}** is this project's outcome-oriented wording, not a quoted Notion label.
- [ElevenLabs: How can I downgrade my subscription?](https://elevenlabs.io/docs/help-center/account/cancellation-refunds/how-can-i-downgrade-my-subscription) uses **Cancel downgrade** to undo a pending downgrade before the current billing cycle ends.

The pricing card uses **Keep {plan} / 保留 {plan}**, with **Keep {plan}?** in confirmation, to describe the outcome. The operation remains canceling the scheduled plan change; the result is **Plan change canceled / 已取消套餐变更**. Future changes can affect either the tier or the billing interval. The secondary confirmation action is **Go back / 返回**, avoiding two different cancellation meanings in one dialog. This operation is separate from **Cancel subscription / 取消订阅**, which stops renewal.

English action labels follow the references; Chinese translations and explanatory text describe this project's actual behavior. Do not copy another product's credit expiry, refund or proration policy. Confirmation must preserve the selected plan, show the effective date or amount due, and disclose any charge from a saved payment method.

| Scenario | Pricing | Billing / confirmation |
| --- | --- | --- |
| Signed out | Subscribe, then sign in with the selection preserved | Sign in |
| Free | Subscribe to the chosen tier | Upgrade |
| Current tier | Current plan, disabled, including when browsing the other interval; when a future change exists, Keep {plan} is enabled and the current card shows its access end date | Keep {plan}? shows the scheduled target/date and confirms removal without a purchase, charge or interval change; Plan change canceled confirms the result |
| Selected target | The card and monthly/yearly toggle choose the exact target | Read-only confirmation of plan, amount and timing; no second selection |
| Higher/lower tier | Change plan, clickable | Higher tiers apply immediately after reviewed payment; lower tiers start at the next billing cycle |
| Amount due now | Confirm and pay the previewed amount; disclose collection from the saved payment method | Stripe attempts collection without a new Checkout. Pending payment shows a completion action; an applied change links to payment history and invoices |
| Change next period | Confirm scheduled change | Current plan remains until the effective date |
| Renewal canceled, period still open | Other plans remain selectable | Change plan + Renew; Renew restores renewal without a new purchase. Plan changes still require reviewed confirmation |
| Subscription ended | Resubscribe to the chosen tier | Subscribe again, through pricing |
| Pending or overdue payment | Targets remain selectable; confirmation stays restricted | Change plan opens payment recovery; do not create a second subscription |
| Scheduled change exists | The exact scheduled plan/interval shows Upcoming plan and its effective date, with Scheduled disabled. Other targets remain selectable. Browsing another interval names the actual scheduled cycle and allows a different selection. The current tier offers Keep {plan}; marketing recommendation highlights are hidden for existing subscribers | The existing target shows its scheduled date and Done. Another target previews the replacement; one confirmation clears the reviewed schedule and submits that exact target. Changed prices or subscription state require fresh review |
| Lifetime active | Recurring plans and Lifetime disabled | Lifetime active; no upgrade or renewal action |
| Recurring subscription buying Lifetime | Buy Lifetime, clickable; full one-time price without subscription proration | Verified paid Lifetime Checkout retires the prior recurring subscription and starts permanent access immediately; failed/unpaid Checkout leaves recurring access untouched |
| Credit purchase | Independent of recurring tier or pending plan changes | Confirm the actual grant in credit history |
| Loading, errors, stale preview | Disable unknown actions; localized recovery message | No success claim, no raw provider messages |
| Returned from Checkout | Return URL is not proof of payment | Confirming / check actual records; no timer-based success |

Automated checks: `tests/billing-presentation.test.ts`, `tests/subscription-change.test.ts`,
`tests/pricing-checkout.test.ts`, and `tests/credit-checkout.test.ts`.
Run `npm run verify` in an isolated checkout when a local Next.js development server is already running.
No real payment or subscription mutation is needed for UI verification.
Pricing retains one login's subscription snapshot in browser memory for one minute across route visits, merges concurrent reads, and keeps existing content during background updates. Session changes discard the snapshot; successful subscription changes invalidate it immediately. Purchase and change confirmations still re-read the API directly. Covered by `tests/pricing-subscription-cache.test.ts`.
The recurring-plan overview has only Change plan and Cancel subscription/Renew controls.
Change plan in the overview opens pricing for selection. Its confirmation dialog never asks users to select a target again.
Pending payments are resolved inside the confirmation dialog. Confirmed subscription cancellation clears any future plan before stopping renewal.

The membership API derives effective access from the canonical `billing_subscriptions` status and expiry on every request; it does not rewrite the historical plan to Free. Canceling retains access until its end date; expired or past-due subscriptions receive no paid access by default. Signed subscription lifecycle webhooks update that snapshot, and the host's minute-based recovery job refreshes subscriptions in bounded batches and replays missed events. Manual membership overrides and Lifetime use their separate entitlement rules.

Future plan changes are authoritative Stripe Subscription Schedule phases. The subscription API returns `scheduled_change` (plan, billing cycle, effective date) from Stripe. Local `billing_subscriptions.plan` remains the currently active plan; synchronized raw snapshots can include the future change, with no dedicated `next_plan` column. Removing a schedule keeps the current Stripe subscription running; it does not cancel renewal or create another subscription. The frontend re-reads state before and after the removal and refuses to silently remove a different schedule.

Admin user lists and customer details show effective membership separately from the payment subscription. The latter includes the current plan/cycle, cancellation or payment status, and the next scheduled plan with its effective date. Admin reads use the synchronized database snapshot without per-user Stripe requests; unavailable or obsolete schedules are not invented. Raw snapshots and payment links are never exposed. This requires deploying both the backend response additions and the frontend; no schema migration is needed. Covered by `tests/admin-subscription.test.ts` and the operations subscription presentation tests.
