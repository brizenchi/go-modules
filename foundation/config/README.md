# foundation/config

> Thin viper wrapper that standardizes file + env var loading into a struct.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/foundation/config.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/foundation/config)

Conventions enforced:

- YAML / TOML / JSON config file at a known path
- Environment variable override with a prefix (e.g. `APP_DB_HOST`)
- `.` in keys maps to `_` in env vars (`db.host` → `APP_DB_HOST`)
- Unmarshal into a caller-supplied struct via `mapstructure` tags

## Install

```bash
go get github.com/brizenchi/go-modules/foundation/config
```

## Quick start

```go
import "github.com/brizenchi/go-modules/foundation/config"

type AppConfig struct {
    Server struct {
        Port int `mapstructure:"port"`
    } `mapstructure:"server"`
    DB struct {
        DSN string `mapstructure:"dsn"`
    } `mapstructure:"db"`
}

var cfg AppConfig
if err := config.Load("./config.yaml", "APP", &cfg); err != nil {
    log.Fatal(err)
}
```

`config.yaml`:

```yaml
server:
  port: 8080
db:
  dsn: postgres://...
```

Env override at boot:

```bash
APP_SERVER_PORT=9090 ./myapp
```

## API

```go
func Load(path, envPrefix string, out any) error
```

- `path`: file path; extension picks the parser (yaml/yml/json/toml)
- `envPrefix`: empty disables env overrides
- `out`: pointer to a struct with `mapstructure` tags

## Testing

```bash
go test -race ./...
```

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
