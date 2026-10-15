# Common customer account scope

The user confirmed that account, subscription/billing and referrals are sufficient. This supersedes the uncommitted notes-example hierarchy work. Keep credit history as an auxiliary account service.

Remove note/upload frontend pages and their public example articles. Redirect retired private addresses to account, and retired public articles to the localized examples directory. That directory directly opens the three existing common services rather than adding more article pages. Remove old calls to action and customer OAuth destinations. Keep backend implementation references, shared API wrappers, AI/storage/credit integration and existing data unchanged. Hide the irrelevant example-export setting from administrator UI while preserving its server-supplied value in the existing settings payload.

Regressions cover route recognition, OAuth return destinations, public links and sitemap entries. Run isolated frontend verify and local redirect checks. No production changes, commit or push requested.

Validation: final isolated frontend `npm run verify` passed (178 tests, ESLint, production build, 72 public pages). All six retired addresses returned 307 to their intended account or localized examples destination. Account, billing, referrals, credits and both examples-directory languages returned 200 without retired UI links. `git diff --check` passed. Earlier catalog-edit compile errors were corrected before the successful final run. No backend implementation or data was removed.
