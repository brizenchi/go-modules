// Package config is a thin viper wrapper that standardizes how every
// service loads configuration.
//
// Conventions enforced:
//   - YAML/TOML/JSON file at a known path (caller passes path).
//   - Environment variable override with a prefix (e.g. APP_DB_HOST).
//   - "." in keys maps to "_" in env vars (e.g. db.host → APP_DB_HOST).
//   - Unmarshal into a caller-supplied struct via mapstructure tags.
//
// Usage:
//
//	type AppConfig struct {
//	    Server struct{ Port int `mapstructure:"port"` } `mapstructure:"server"`
//	    DB     struct{ DSN  string `mapstructure:"dsn"` } `mapstructure:"db"`
//	}
//
//	var cfg AppConfig
//	if err := config.Load("./config.yaml", "APP", &cfg); err != nil { panic(err) }
//
// Stdlib + viper only — no project coupling.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load reads `path`, applies env overrides under `envPrefix`, and
// unmarshals into `out` (must be a pointer to a struct).
//
// envPrefix may be empty to disable env overrides.
//
// File extension determines the parser; supported: yaml/yml, json, toml.
func Load(path, envPrefix string, out any) error {
	v := viper.New()
	if err := configureFile(v, path); err != nil {
		return err
	}
	configureEnv(v, envPrefix)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := v.Unmarshal(out); err != nil {
		return fmt.Errorf("config: unmarshal: %w", err)
	}
	return nil
}

// LoadGlobal does Load AND installs the parsed viper as viper's global
// instance — useful for projects that still use viper.GetString(...) calls
// from legacy code paths during migration. Prefer Load() for new code.
func LoadGlobal(path, envPrefix string, out any) error {
	v := viper.GetViper()
	if err := configureFile(v, path); err != nil {
		return err
	}
	configureEnv(v, envPrefix)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if out != nil {
		if err := v.Unmarshal(out); err != nil {
			return fmt.Errorf("config: unmarshal: %w", err)
		}
	}
	return nil
}

func configureFile(v *viper.Viper, path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("config: stat %s: %w", path, err)
	}
	v.SetConfigFile(path)
	return nil
}

func configureEnv(v *viper.Viper, prefix string) {
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	if prefix != "" {
		v.SetEnvPrefix(prefix)
	}
}

// MustLoad is Load that panics on error — use only at boot.
func MustLoad(path, envPrefix string, out any) {
	if err := Load(path, envPrefix, out); err != nil {
		panic(err)
	}
}
