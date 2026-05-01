# foundation/randx

> Crypto-grade random helpers: verification codes, URL-safe tokens, hex / base32 IDs.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/randx.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/randx)

Always reads from `crypto/rand`. Never use `math/rand` for anything a
user can see — predictable codes are an account-takeover vector.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/randx
```

## Quick start

```go
import "github.com/brizenchi/go-modules/foundation/randx"

// 6-digit verification code (SMS / email login).
code, _ := randx.NumericCode(6)            // "473829"

// Custom alphabet — strips look-alikes, good for short user-typed codes.
backup, _ := randx.Code(10, randx.Unambiguous)  // "kP7Ham3X9R"

// URL-safe token for password-reset / email-verify links.
tok, _ := randx.URLToken(32)               // 43-char base64-no-pad

// Hex token for internal handles / idempotency keys.
id, _ := randx.HexToken(16)                // 32 hex chars

// Raw bytes for signing keys / webhook secrets.
key, _ := randx.Bytes(32)
```

## API

| Func                    | Use for                                           |
|-------------------------|---------------------------------------------------|
| `Code(n, charset)`      | Codes drawn uniformly from a custom alphabet      |
| `MustCode(n, charset)`  | Same, panics on entropy failure (init / tests)    |
| `NumericCode(n)`        | Digit-only verification codes (SMS / email)       |
| `Bytes(n)`              | Raw cryptographic random bytes                    |
| `HexToken(n)`           | `2n`-char hex string                              |
| `URLToken(n)`           | URL-safe base64 (no padding)                      |
| `Base32Token(n)`        | Crockford-style base32 (no padding)               |

Predefined charsets: `Numeric`, `LowerAlphaNum`, `AlphaNum`, `Unambiguous`.

## Sizing tokens

Pick `n` (the byte count) by attack model, not character count:

| Use case                          | `Bytes(n)` |
|-----------------------------------|------------|
| Email-verify / password-reset link | 32         |
| Webhook signing secret             | 32         |
| Idempotency key                    | 16         |
| API key prefix (display)           | 8          |
| Short backup code (10–12 chars)    | use `Code` with `Unambiguous` |

32 bytes = 256 bits of entropy — comfortable margin against brute force.

## Errors

| Error                    | When                                  |
|--------------------------|---------------------------------------|
| `length must be > 0`     | `Code(0, …)` / `Bytes(0)`             |
| `charset must not be empty` | `Code(_, "")`                      |
| `read entropy: …`        | `crypto/rand` failed (extremely rare) |

## Testing

```bash
go test -race ./...
```

Coverage: 81%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
