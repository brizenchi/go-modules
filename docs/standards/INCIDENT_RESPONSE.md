# Incident response and postmortems

How to use this project's monitoring, alerting and deployment platform is recorded in [PROJECT.md](./PROJECT.md).

## Severity

| Level | Definition | Examples | Response |
| --- | --- | --- | --- |
| **P1** | core functionality is down, or data, money or security is at stake | the whole site is down; paid but not provisioned; a leaked secret | work on it until resolved; update every 30 minutes |
| **P2** | part of the product is down or clearly degraded, with a workaround | one login method fails; P99 latency above 5 s | same day |
| **P3** | small impact | an admin page errors; a single user's problem | regular planning |

## Process

```text
detect → assess severity → mitigate → diagnose → fix → confirm recovery → postmortem
```

### 1. Detect and assess
- Sources: alerts, user reports, post-release checks.
- Establish the impact: which service, which endpoints, since when, how many users.
- Announce it in the team channel: "P1/P2: <symptom>, <owner> is on it".

### 2. Mitigate (before looking for the root cause)
| Situation | Action |
| --- | --- |
| Right after a release | **roll back immediately** |
| A third-party outage | turn off the dependent feature or switch providers |
| A defective feature | turn it off with a configuration flag |
| A leaked secret | revoke and rotate it now (see [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#leaked-secrets)) |
| Abnormal traffic | rate-limit or block the source at the edge |

### 3. Diagnose
1. **Metrics**: when did error rate and latency change? Which endpoint? Does it line up with a release or an external dependency?
2. **Traces**: find failing or slow requests and see which step went wrong.
3. **Logs**: read the detailed error for that request.
4. For user reports, ask for the request ID (`X-Request-ID`) to go straight to that request.

### 4. Fix and confirm
- Follow the [hotfix process](./GIT_WORKFLOW.md#hotfixes).
- Once metrics recover and alerts clear, announce the recovery in the team channel.
- If data was corrupted, correct it first and record how.

### 5. Communicate
- For P1, and P2 affecting paying users, tell affected users what happened, the impact, and the current state.
- If personal data was exposed, notify as required by applicable law.

## Postmortems

Write a postmortem **within three working days** of resolving a P1 or P2, in
`docs/incidents/YYYY-MM-DD-<summary>.md`. Blameless: the goal is better processes and systems.

```markdown
# <date> <one-line summary>

- Severity: P1 / P2
- Duration: <start> – <recovery> (X minutes)
- Owner:

## Impact
How many users, which features, any loss of data or money.

## Timeline (UTC)
| Time | Event |
| --- | --- |

## Root cause
Keep asking "why" until you reach a process or system cause.

## What went well / what to improve

## Action items
| Action | Owner | Due | Issue |
| --- | --- | --- | --- |
```

## On call

Once several people share the work, rotate on call weekly and configure the rotation in
the alerting system. Hand over this week's alerts, open problems and planned releases.
