package serviceconfig

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/brizenchi/quickstart-template/internal/platform"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Manager struct {
	db           *gorm.DB
	environment  platform.Config
	localPreview bool
	mu           sync.RWMutex
	active       map[string]state
	validate     func(platform.Config) error
}

// SetValidator installs the host's deployment validation before routes start.
// Its error is never forwarded because host errors may include input values.
func (m *Manager) SetValidator(validate func(platform.Config) error) {
	m.mu.Lock()
	m.validate = validate
	m.mu.Unlock()
}

func NewManager(db *gorm.DB, environment platform.Config, localPreview bool) *Manager {
	m := &Manager{db: quiet(db), environment: environment, localPreview: localPreview, active: map[string]state{}}
	for _, provider := range providerNames {
		m.active[provider] = state{source: "environment", config: environmentConfiguration(environment, provider)}
	}
	return m
}

// QuietDB prevents credential-bearing SQL from reaching a deployment's expanded
// SQL logger, including when a caller passes an existing transaction.
func QuietDB(db *gorm.DB) *gorm.DB { return quiet(db) }

func quiet(db *gorm.DB) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.Session(&gorm.Session{Logger: logger.Discard})
}

// Load is called once before constructing any provider or registering routes.
// It never migrates, changes runtime modules, or contacts an external provider.
func (m *Manager) Load(ctx context.Context) (platform.Config, error) {
	cfg := m.environment
	states, _, err := m.readStates(ctx)
	if err != nil {
		return platform.Config{}, err
	}
	if !m.localPreview {
		for _, provider := range providerNames {
			current := states[provider]
			if current.source == "database" {
				applyConfiguration(&cfg, provider, current.config)
			}
		}
	} else {
		// Even a restarted isolated fixture cannot activate a saved real service.
		disabled := false
		cfg.Billing.Enabled = &disabled
		cfg.Email.Provider = "log"
		for _, provider := range providerNames {
			states[provider] = state{source: "environment", config: environmentConfiguration(cfg, provider)}
		}
	}
	m.mu.Lock()
	m.active = states
	m.mu.Unlock()
	return cfg, nil
}

func (m *Manager) Snapshot(ctx context.Context) (Snapshot, error) {
	states, ready, err := m.readStates(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{StorageReady: ready, LocalPreview: m.localPreview, Providers: make([]Provider, 0, len(providerNames))}
	for _, provider := range providerNames {
		result.Providers = append(result.Providers, m.safeProvider(provider, states[provider]))
	}
	return result, nil
}

func (m *Manager) readStates(ctx context.Context) (map[string]state, bool, error) {
	states := make(map[string]state, len(providerNames))
	for _, provider := range providerNames {
		states[provider] = state{source: "environment", config: environmentConfiguration(m.environment, provider)}
	}
	if m.db == nil {
		return nil, false, ErrStorage
	}
	var rows []Record
	if err := m.db.WithContext(ctx).Where("provider IN ?", providerNames).Find(&rows).Error; err != nil {
		if missingTable(err) {
			return states, false, nil
		}
		return nil, true, ErrStorage
	}
	for _, row := range rows {
		current, err := m.decode(row)
		if err != nil {
			return nil, true, err
		}
		states[row.Provider] = current
	}
	return states, true, nil
}

func (m *Manager) decode(row Record) (state, error) {
	current := state{source: row.Source, version: row.Version, config: environmentConfiguration(m.environment, row.Provider)}
	if row.Source == "environment" {
		return current, nil
	}
	if row.Source != "database" || row.Version < 1 {
		return state{}, invalid("stored integration settings are invalid")
	}
	var stored configuration
	if err := json.Unmarshal([]byte(row.Payload), &stored); err != nil || len(stored.Fields) != len(current.config.Fields) || len(stored.Secrets) != len(current.config.Secrets) {
		return state{}, invalid("stored integration settings are invalid")
	}
	for key := range stored.Fields {
		if _, allowed := current.config.Fields[key]; !allowed {
			return state{}, invalid("stored integration contains unsupported fields")
		}
	}
	for key := range stored.Secrets {
		if _, allowed := current.config.Secrets[key]; !allowed {
			return state{}, invalid("stored integration contains unsupported credentials")
		}
	}
	if err := validateConfiguration(row.Provider, &stored); err != nil {
		return state{}, err
	}
	current.config = stored
	return current, nil
}

func (m *Manager) safeProvider(provider string, current state) Provider {
	m.mu.RLock()
	active := m.active[provider]
	m.mu.RUnlock()
	result := Provider{
		Provider: provider, Source: current.source, Version: current.version, Enabled: current.config.Enabled,
		Fields: cloneConfiguration(current.config).Fields, Secrets: make(map[string]SecretStatus, len(current.config.Secrets)),
		ActiveEnabled: active.config.Enabled, ActiveSource: active.source,
		RestartRequired: current.source != active.source || !reflect.DeepEqual(current.config, active.config),
	}
	for key, value := range current.config.Secrets {
		result.Secrets[key] = SecretStatus{Configured: value != ""}
	}
	return result
}

// Update participates in the caller's audit transaction. It never modifies the
// active snapshot, so saving credentials cannot change in-flight requests.
func (m *Manager) Update(ctx context.Context, tx *gorm.DB, provider string, patch Patch) (Provider, error) {
	if !knownProvider(provider) {
		return Provider{}, ErrProvider
	}
	if tx == nil {
		return Provider{}, ErrStorage
	}
	db := quiet(tx).WithContext(ctx)
	var row Record
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("provider = ?", provider).Take(&row).Error
	exists := err == nil
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		if missingTable(err) {
			return Provider{}, ErrSchema
		}
		return Provider{}, ErrStorage
	}
	current := state{source: "environment", config: environmentConfiguration(m.environment, provider)}
	if exists {
		current, err = m.decode(row)
		if err != nil {
			return Provider{}, err
		}
	}
	if current.version != patch.Version {
		return Provider{}, ErrVersion
	}
	next := cloneConfiguration(current.config)
	if err := validatePatch(provider, patch, &next); err != nil {
		return Provider{}, err
	}
	payload := ""
	if patch.Source == "database" {
		raw, err := json.Marshal(next)
		if err != nil {
			return Provider{}, ErrStorage
		}
		payload = string(raw)
	} else {
		next = environmentConfiguration(m.environment, provider)
	}
	if err := m.validateCandidate(db, provider, patch.Source, next); err != nil {
		return Provider{}, err
	}
	row = Record{Provider: provider, Source: patch.Source, Version: current.version + 1, Payload: payload, UpdatedAt: time.Now().UTC()}
	var result *gorm.DB
	if exists {
		result = db.Model(&Record{}).Where("provider = ? AND version = ?", provider, current.version).Updates(map[string]any{"source": row.Source, "version": row.Version, "payload": row.Payload, "updated_at": row.UpdatedAt})
	} else {
		result = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	}
	if result.Error != nil {
		return Provider{}, ErrStorage
	}
	if result.RowsAffected != 1 {
		return Provider{}, ErrVersion
	}
	return m.safeProvider(provider, state{source: row.Source, version: row.Version, config: next}), nil
}

func missingTable(err error) bool {
	// PostgreSQL's undefined_table and SQLite's exact missing-table error are the
	// only allowed fallback cases. Connection and permission failures fail closed.
	var state interface{ SQLState() string }
	if errors.As(err, &state) {
		return state.SQLState() == "42P01"
	}
	return err.Error() == "no such table: service_provider_settings"
}

func (m *Manager) validateCandidate(db *gorm.DB, provider, source string, next configuration) error {
	m.mu.RLock()
	validate := m.validate
	m.mu.RUnlock()
	if validate == nil {
		return nil
	}
	candidate := m.environment
	var rows []Record
	if err := db.Where("provider IN ?", providerNames).Find(&rows).Error; err != nil {
		return ErrStorage
	}
	for _, row := range rows {
		if row.Provider == provider || row.Source == "environment" {
			continue
		}
		current, err := m.decode(row)
		if err != nil {
			return err
		}
		applyConfiguration(&candidate, row.Provider, current.config)
	}
	if source == "database" {
		applyConfiguration(&candidate, provider, next)
	}
	if err := validate(candidate); err != nil {
		return invalid("integration settings are incompatible with deployment configuration; check provider credentials, frontend origin, and authentication settings")
	}
	return nil
}
