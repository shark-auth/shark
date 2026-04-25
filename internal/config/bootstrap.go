package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BootstrapConfig holds the three knobs that must be resolved before the DB
// can be opened. Everything else is loaded from the system_config table via
// LoadRuntime after the store is available.
type BootstrapConfig struct {
	// DBPath is the path to the SQLite database file.
	// Defaults to ./shark.db (or ./dev.db in dev mode).
	DBPath string

	// Port is the HTTP listen port. Default 8080.
	Port string

	// Bind is the listen address. Default "0.0.0.0".
	Bind string

	// Mode is the runtime mode: "dev" or "prod". Read from ~/.shark/state.
	Mode string
}

// BootstrapOptions lets callers inject CLI-flag values (highest priority).
type BootstrapOptions struct {
	// FlagDBPath is the --db-path CLI flag value. Empty = not set.
	FlagDBPath string
	// FlagPort is the --port CLI flag value. Empty = not set.
	FlagPort string
	// FlagBind is the --bind CLI flag value. Empty = not set.
	FlagBind string
}

// Bootstrap resolves the three pre-DB knobs using the priority order:
//  1. CLI flag (highest)
//  2. Env var (SHARK_DB_PATH, SHARK_PORT, SHARK_BIND)
//  3. ~/.shark/state file (single line: key=value)
//  4. Defaults: db=./shark.db (./dev.db in dev mode), port=8080, bind=0.0.0.0
func Bootstrap(opts BootstrapOptions) (*BootstrapConfig, error) {
	state, err := readStateFile()
	if err != nil {
		// Non-fatal: missing state file is the normal case on first boot.
		state = map[string]string{}
	}

	mode := state["mode"]
	if mode == "" {
		mode = "prod"
	}

	defaultDB := "./shark.db"
	if mode == "dev" {
		defaultDB = "./dev.db"
	}

	cfg := &BootstrapConfig{
		DBPath: resolve(opts.FlagDBPath, "SHARK_DB_PATH", state["db_path"], defaultDB),
		Port:   resolve(opts.FlagPort, "SHARK_PORT", state["port"], "8080"),
		Bind:   resolve(opts.FlagBind, "SHARK_BIND", state["bind"], "0.0.0.0"),
		Mode:   mode,
	}
	return cfg, nil
}

// resolve returns the first non-empty value from: flag, env var, state file
// value, default.
func resolve(flagVal, envKey, stateVal, def string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if stateVal != "" {
		return stateVal
	}
	return def
}

// readStateFile reads ~/.shark/state and returns a key=value map.
// Lines beginning with '#' and blank lines are ignored.
func readStateFile() (map[string]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("config: state: home dir: %w", err)
	}
	path := filepath.Join(home, ".shark", "state")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("config: state: open %q: %w", path, err)
	}
	defer f.Close()

	m := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return m, scanner.Err()
}
