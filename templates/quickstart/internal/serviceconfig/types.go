// Package serviceconfig stores the administrator's provider overrides. Provider
// credentials deliberately remain plaintext in the private database; they are
// write-only at the HTTP boundary and excluded from SQL logging and audit data.
package serviceconfig

import (
	"errors"
	"time"
)

var (
	ErrProvider = errors.New("unsupported integration")
	ErrVersion  = errors.New("integration settings changed; reload before saving")
	ErrStorage  = errors.New("integration settings storage unavailable")
	ErrSchema   = errors.New("integration settings table is missing; apply the service configuration migration")
)

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// Record is private persistence. Payload contains plaintext credentials and must
// never be returned, logged, or copied into operations audit details.
type Record struct {
	Provider  string    `gorm:"type:varchar(16);primaryKey" json:"-"`
	Source    string    `gorm:"type:varchar(16);not null" json:"-"`
	Version   int64     `gorm:"not null" json:"-"`
	Payload   string    `gorm:"type:text;not null" json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (Record) TableName() string { return "service_provider_settings" }
func Models() []any              { return []any{&Record{}} }

type SecretStatus struct {
	Configured bool `json:"configured"`
}

// Provider is the only representation safe to return from the admin API.
type Provider struct {
	Provider        string                  `json:"provider"`
	Source          string                  `json:"source"`
	Version         int64                   `json:"version"`
	Enabled         bool                    `json:"enabled"`
	Fields          map[string]string       `json:"fields"`
	Secrets         map[string]SecretStatus `json:"secrets"`
	ActiveEnabled   bool                    `json:"active_enabled"`
	ActiveSource    string                  `json:"active_source"`
	RestartRequired bool                    `json:"restart_required"`
}

type Snapshot struct {
	StorageReady bool       `json:"storage_ready"`
	LocalPreview bool       `json:"local_preview"`
	Providers    []Provider `json:"providers"`
}

type Patch struct {
	Version int64             `json:"version"`
	Source  string            `json:"source"`
	Enabled *bool             `json:"enabled,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
	Secrets map[string]string `json:"secrets,omitempty"`
	Reason  string            `json:"reason"`
}

type configuration struct {
	Enabled bool              `json:"enabled"`
	Fields  map[string]string `json:"fields"`
	Secrets map[string]string `json:"secrets"`
}

type state struct {
	source  string
	version int64
	config  configuration
}
