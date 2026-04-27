# foundation/httpresp

> Uniform `{code, msg, data}` JSON response envelope for Gin handlers.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/httpresp.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/httpresp)

Every helper sets both the HTTP status code AND the envelope `code` field
so frontend code can branch on a single field. Error helpers also call
`c.Abort()` so middleware downstream sees the chain as terminated.

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/httpresp
```

## Envelope shape

```json
{ "code": 200, "msg": "ok", "data": { ... } }
```

## Quick start

```go
import "github.com/brizenchi/go-modules/foundation/httpresp"

func handler(c *gin.Context) {
    httpresp.OK(c, gin.H{"id": 1})           // 200 + {code:200, msg:"ok", data}
    httpresp.BadRequest(c, "missing email")  // 400 + {code:400, msg:"...", data:null}
    httpresp.NotFound(c, "user not found")
    httpresp.Internal(c, "boom")
}

// Soft errors (HTTP 200 with a non-success envelope code — e.g. form validation):
httpresp.OKWith(c, 4001, "email already in use", nil)
```

## Helpers

| Helper                                        | HTTP status |
|-----------------------------------------------|-------------|
| `OK(c, data)`                                 | 200         |
| `OKWith(c, code, msg, data)`                  | 200         |
| `BadRequest(c, msg)`                          | 400         |
| `Unauthorized(c, msg)`                        | 401         |
| `Forbidden(c, msg)`                           | 403         |
| `NotFound(c, msg)`                            | 404         |
| `Conflict(c, msg)`                            | 409         |
| `TooManyRequests(c, msg)`                     | 429         |
| `Internal(c, msg)`                            | 500         |

All error helpers internally call `c.AbortWithStatusJSON`.

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
