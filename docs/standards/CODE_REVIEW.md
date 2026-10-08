# Code review

## Purpose

Review looks for **correctness problems, security problems, violations of the
architecture, and code that will be hard to maintain**. Formatting and style are left
to formatters and linters and are not discussed in review.

## Turnaround

| Who | Expectation |
| --- | --- |
| Reviewer | first round of comments within one working day; hotfixes as soon as possible |
| Author | answers every comment: change the code, or explain why not |
| Both | after three rounds of disagreement, talk it through live and record the outcome in the pull request |

## Comment levels

Start each comment with its level:

| Prefix | Meaning |
| --- | --- |
| `blocking:` | must change before merging: a defect, a security problem, an architecture violation |
| `suggestion:` | recommended; the author decides |
| `nit:` | minor; fine to leave |
| `question:` | the reviewer does not understand something and needs an explanation |

## Checklist

### Correctness
- [ ] Edge cases: nulls, empty lists, repeated requests, concurrency, timeouts
- [ ] Every error is handled and logged once, per [error handling](./CODE_STYLE.md#error-handling)
- [ ] The request context (cancellation, trace) is passed through; outbound calls have timeouts
- [ ] Money, quota and stock logic is idempotent (replays, double submits)

### Architecture
- [ ] Follows the layering and layout recorded in [PROJECT.md](./PROJECT.md)
- [ ] Public interfaces and shared libraries change additively; breaking changes come with a migration note

### APIs and data
- [ ] Follows [API_STANDARD.md](./API_STANDARD.md): paths, status codes, error format, pagination, time format
- [ ] Callers' types or clients (frontend, other services) are updated in the same pull request
- [ ] Database changes follow [DATABASE.md](./DATABASE.md): applied migrations are untouched, queries are indexed

### Security
- [ ] Follows [SECURITY_STANDARD.md](./SECURITY_STANDARD.md): authentication and ownership checks, input validation, no sensitive data in logs
- [ ] No real secrets committed; fake secrets in tests are obviously fake

### Observability
- [ ] Logs are structured, at the right level, carry the request context and contain nothing sensitive
- [ ] New external dependencies can be monitored (timeouts, error logs or metrics)

### Tests
- [ ] Follows [TESTING.md](./TESTING.md): new logic is tested, fixed bugs have regression tests
- [ ] Tests do not use the network and do not depend on execution order

## Before asking for review

The author goes through the checklist above and runs the checks for the parts they changed.

## AI-generated code

AI-generated code meets the same bar as hand-written code. The author must be able to
explain every line; AI authorship is no reason to skip tests or review.
