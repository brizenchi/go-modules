# Billing UI scenarios

Shared labels and outcomes live in `lib/billing-presentation.ts`. Eligibility stays in
`lib/subscription-change.ts`; wording must not bypass pending payments or other restrictions.
Every label has English and Chinese text. Public pricing and private billing preserve the selected language.

| Scenario | Pricing | Billing / confirmation |
| --- | --- | --- |
| Signed out | Subscribe, then sign in with the selection preserved | Sign in |
| Free | Subscribe to the chosen tier | Upgrade |
| Current tier | Current plan, disabled, including when browsing the other interval | Change plan + Cancel subscription |
| Same tier, different interval | Show current interval badge; change the interval in the plan dialog | Review amount and timing |
| Higher/lower tier | Change plan, clickable | Higher tiers apply immediately after reviewed payment; lower tiers start at the next billing cycle |
| Amount due now | Confirm and pay the previewed amount; disclose collection from the saved payment method | Stripe attempts collection without a new Checkout. Pending payment shows a completion action; an applied change links to payment history and invoices |
| Change next period | Confirm scheduled change | Current plan remains until the effective date |
| Renewal canceled, period still open | Other plans remain selectable | Change plan + Renew; Renew restores renewal without a new purchase. Plan changes still require reviewed confirmation |
| Subscription ended | Resubscribe to the chosen tier | Subscribe again, through pricing |
| Pending or overdue payment | Targets remain selectable; confirmation stays restricted | Change plan opens payment recovery; do not create a second subscription |
| Scheduled change exists | Targets remain selectable; confirmation stays restricted | Change plan opens explicit schedule removal; the overview shows target/date only. Confirmed subscription cancellation clears the future schedule before stopping renewal |
| Lifetime active | Recurring plans and Lifetime disabled | Lifetime active; no upgrade or renewal action |
| Recurring subscription buying Lifetime | Buy Lifetime, clickable; full one-time price without subscription proration | Verified paid Lifetime Checkout retires the prior recurring subscription and starts permanent access immediately; failed/unpaid Checkout leaves recurring access untouched |
| Credit purchase | Independent of recurring tier or pending plan changes | Confirm the actual grant in credit history |
| Loading, errors, stale preview | Disable unknown actions; localized recovery message | No success claim, no raw provider messages |
| Returned from Checkout | Return URL is not proof of payment | Confirming / check actual records; no timer-based success |

Automated checks: `tests/billing-presentation.test.ts`, `tests/subscription-change.test.ts`,
`tests/pricing-checkout.test.ts`, and `tests/credit-checkout.test.ts`.
Run `npm run verify` in an isolated checkout when a local Next.js development server is already running.
No real payment or subscription mutation is needed for UI verification.
The recurring-plan overview has only Change plan and Cancel subscription/Renew controls.
Payment settings and scheduled-change recovery are handled inside the Change plan dialog.
