# Changelog — foundation/aix

## [Unreleased]

### Added

- Stateless `Generator` / `Client` for compatible text completions, with server-owned configuration, explicit inputs, context cancellation, bounded responses and sanitized errors.
- Injected HTTP transport and redirect refusal. No users, conversations, database, quotas, credit charging or HTTP routes.
- `Config.ReasoningEffort`, sent as `reasoning_effort` only when set, and `Config.Timeout` (default 45 seconds, at most 10 minutes).
- `Completion.FinishReason`, so callers can reject output cut off at the token limit.

### Changed

- `Config.MaxOutputTokens` may now be up to 65536 (previously 4096) for long-form output and providers that count reasoning tokens.
