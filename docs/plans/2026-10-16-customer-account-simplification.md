# Customer account simplification

- Keep My account, Subscription & billing, Referral rewards as flat peer routes.
- My account displays read-only email, effective membership and sign-out only.
  Remove the duplicate service directory and pricing link.
- Billing displays current plan and remaining credits, followed by payment history
  and invoices. Retain existing subscription changes in an on-demand dialog;
  purchases stay on pricing. Remove credit-purchase and history accordions.
- Remove routine Refresh controls from customer pages. Update on entry, return
  to the page and after successful actions; retain contextual error retry.
- Use one compact public header spacing rule for comparison/demo directories,
  including pages with no introduction. Do not render empty description blocks.
- Verify EN/中文, 1440/768/390px, sign-out, effective membership, billing states,
  plan dialog, automatic updates, payment tables, invitation access and SSR SEO.

No production writes, provider changes, deployment or pushes are part of this task.

Follow-up: Credit history is a fourth peer in the shared sidebar/avatar menu.
Its immutable ledger displays the recorded source, signed change, historical
balance, timestamp and expiry. Distinguish inviter/invitee and signup/payment
rewards; preserve actual subscription explanations without inferring past plans
from the current membership. No grant rules or existing records are changed.

Follow-up: Combine purchases and invoices into one payment table. Match only
provider invoice IDs, retain invoice-only records and purchases without invoices,
and load each source's next page independently. A failed source remains retryable
without hiding the other history. Invoice downloads belong to their payment row.

Follow-up: Trim referrals to the invite-link copy control, four actual metrics
and invitation history. State the two reward stages once below the link; remove
duplicated rules, panel subtitles and the test walkthrough. Keep language options
in a fixed EN / 中文 order without changing the saved locale preference.
