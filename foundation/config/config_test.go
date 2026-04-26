package config

import (
	"os"
	"path/filepath"
	"testing"
)

type appCfg struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Name string `mapstructure:"name"`
	} `mapstructure:"server"`
	DB struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"db"`
}

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_YAML(t *testing.T) {
	p := writeFile(t, "c.yaml", `
server:
  port: 8080
  name: alpha
db:
  dsn: postgres://x
`)
	var c appCfg
	if err := Load(p, "", &c); err != nil {
		t.Fatal(err)
	}
	if c.Server.Port != 8080 || c.Server.Name != "alpha" || c.DB.DSN != "postgres://x" {
		t.Errorf("loaded = %+v", c)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	p := writeFile(t, "c.yaml", "server:\n  port: 8080\n")
	t.Setenv("APP_SERVER_PORT", "9090")
	var c appCfg
	if err := Load(p, "APP", &c); err != nil {
		t.Fatal(err)
	}
	if c.Server.Port != 9090 {
		t.Errorf("env override didn't take, got %d", c.Server.Port)
	}
}

func TestLoad_FileMissing(t *testing.T) {
	var c appCfg
	if err := Load("/no/such/file.yaml", "", &c); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_BadYAML(t *testing.T) {
	// Tab indentation + unbalanced key produces an unambiguous YAML error.
	p := writeFile(t, "c.yaml", "server:\n\tport: [unclosed")
	var c appCfg
	if err := Load(p, "", &c); err == nil {
		t.Error("expected parse error")
	}
}
