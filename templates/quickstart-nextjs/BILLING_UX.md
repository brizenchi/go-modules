# Billing UI scenarios

Shared labels and outcomes live in `lib/billing-presentation.ts`. Eligibility stays in
`lib/subscription-change.ts`; wording must not bypass pending payments or other restrictions.
Every label has English and Chinese text. Public pricing and private billing preserve the selected language.

| Scenario | Pricing | Billing / confirmation |
| --- | --- | --- |
| Signed out | Subscribe, then sign in with the selection preserved | Sign in |
| Free | Subscribe to the chosen tier | Upgrade |
| Same tier and interval | Current plan, disabled | Change plan |
| Same tier, different interval | Switch to monthly/yearly; show current interval badge | Review amount and timing |
| Higher/lower tier | Upgrade/Downgrade to the named tier | Preview first; never submit on selection |
| Amount due now | Confirm and pay the previewed amount | Payment required if payment is still pending |
| Change next period | Confirm scheduled change | Current plan remains until the effective date |
| Renewal canceled, period still open | One resume-renewal notice; changes disabled | Resume renewal; keep current access and existing credits |
| Subscription ended | Resubscribe to the chosen tier | Subscribe again, through pricing |
| Pending or overdue payment | One payment notice; changes disabled | Complete or review payment |
| Scheduled change exists | One scheduled-change notice; changes disabled | Show target/date and cancel scheduled change |
| Lifetime active | Recurring plans and Lifetime disabled | Lifetime active; no upgrade or renewal action |
| Recurring subscription buying Lifetime | Disabled; explain it must end first | Existing billing remains available |
| Credit purchase | Independent of recurring tier or pending plan changes | Confirm the actual grant in credit history |
| Loading, errors, stale preview | Disable unknown actions; localized recovery message | No success claim, no raw provider messages |
| Returned from Checkout | Return URL is not proof of payment | Confirming / check actual records; no timer-based success |

Automated checks: `tests/billing-presentation.test.ts`, `tests/subscription-change.test.ts`,
`tests/pricing-checkout.test.ts`, and `tests/credit-checkout.test.ts`.
Run `npm run verify` in an isolated checkout when a local Next.js development server is already running.
No real payment or subscription mutation is needed for UI verification.
