# foundation/jwt

> Opinionated JWT signer / verifier for service-to-service and user-session use cases.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/jwt.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/jwt)

Wraps `golang-jwt/v5` with sane defaults: HS256 by default, RS256 when
you need cross-org verification, mandatory `exp`+`iat`, leeway-aware
parsing, and an alg-confusion guard so verify can't be tricked into
treating an HS256 token as RS256.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/jwt
```

## Quick start

### HS256 (symmetric, single-org)

```go
import (
    "time"
    "github.com/brizenchi/go-modules/foundation/jwt"
)

signer, err := jwt.NewHS256("super-secret-32-bytes-of-entropy!", jwt.Options{
    Issuer:   "auth-svc",
    Audience: []string{"api"},
    Leeway:   30 * time.Second,
})
if err != nil { log.Fatal(err) }

token, err := signer.Sign(jwt.Claims{
    Subject: "user-42",
    TTL:     time.Hour,
    Extra:   map[string]any{"role": "admin"},
})

parsed, err := signer.Parse(token)
uid := parsed.Subject
role, _ := parsed.Extra["role"].(string)
```

### RS256 (asymmetric, cross-org)

```go
priv := loadRSAPrivateKey()  // *rsa.PrivateKey
pub  := loadRSAPublicKey()   // *rsa.PublicKey

signer, err := jwt.NewRS256(priv, pub, jwt.Options{Issuer: "auth-svc"})
```

## Errors

| Error              | When |
|--------------------|------|
| `ErrSecretRequired` | HS256 secret was empty |
| `ErrInvalidToken`   | Bad signature, malformed, or alg mismatch |
| `ErrExpired`        | Token past `exp` (after leeway) |
| `ErrTTLRequired`    | `Claims.TTL` was 0 on Sign |

Use `errors.Is` to discriminate.

## Security notes

- HS256 secrets must be ≥ 32 bytes of entropy. Don't reuse JWT secrets across services.
- RS256 verify checks the token alg against the configured key type — an
  attacker can't substitute HS256 with the public key as the secret.
- `exp` is enforced strictly with a configurable `Leeway` for clock skew.

## Testing

```bash
go test -race ./...
```

Coverage: 86%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
