# Direct plan confirmation

The pricing card and billing interval toggle select the target. Confirmation displays that target, the payment amount and effective date without editable selectors. The billing overview links to pricing for selection.

An already scheduled target shows its effective date and a Done action. A different target can replace the known schedule with one explicit confirmation. Before writing, read the latest subscription; after removing the reviewed schedule, verify state and price again. Changed account, pending payments, changed schedule, price or timing prevent submission and require review.

Implementation and validation:

1. Add failing regression tests for existing reservations, replacement confirmation, concurrent changes, failed removal and stale sessions.
2. Add the guarded orchestration helper, remove dialog selectors and update both pricing and billing callers.
3. Run frontend verification in an isolated copy. Check normal upgrade, downgrade, existing schedule, replacement and billing entry using an in-memory API fixture only.
