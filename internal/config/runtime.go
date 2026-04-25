package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// LoadRuntime reads the system_config row from the DB, JSON-unmarshals the
// payload into Config, applies the same defaults as Load(), and returns a
// fully-resolved *Config. When the DB row is empty or contains '{}' the
// returned config is all-defaults (same as calling Load("")).
//
// The yaml loader is NOT called here — yaml is still the fallback handled by
// Build() in internal/server. This function is the DB-only path.
func LoadRuntime(ctx context.Context, store storage.Store) (*Config, error) {
	payload, err := store.GetSystemConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("config: load runtime: get system_config: %w", err)
	}

	// Start from defaults (same set as Load uses).
	cfg, err := Load("") // Load with no yaml path = pure defaults
	if err != nil {
		return nil, fmt.Errorf("config: load runtime: defaults: %w", err)
	}

	// Overlay the DB payload if it is non-trivial.
	if payload != "" && payload != "{}" {
		if err := json.Unmarshal([]byte(payload), cfg); err != nil {
			return nil, fmt.Errorf("config: load runtime: unmarshal payload: %w", err)
		}
		// Re-run the derived fields that Load() normally calls.
		cfg.Email.Resolve(&cfg.SMTP)
		cfg.Proxy.Resolve()
	}

	return cfg, nil
}

// SaveRuntime JSON-marshals cfg and writes it to the system_config table.
// This replaces cfg.Save(path) for the DB-backed config path.
func SaveRuntime(ctx context.Context, store storage.Store, cfg *Config) error {
	if err := store.SetSystemConfig(ctx, cfg); err != nil {
		return fmt.Errorf("config: save runtime: %w", err)
	}
	return nil
}
