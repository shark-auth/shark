// Package config provides helpers for reading and writing the ~/.shark/state file.
// This file owns mode-state I/O (Phase C). Phase B may add internal/config/state.go
// with the same helpers; if that lands first, this file becomes a thin shim.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ModeState is persisted to ~/.shark/state between server restarts.
type ModeState struct {
	Mode string `json:"mode"` // "prod" or "dev"
}

// sharkStateDir returns the ~/.shark directory path.
func sharkStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".shark"), nil
}

// sharkStatePath returns the full path to ~/.shark/state.
func sharkStatePath() (string, error) {
	dir, err := sharkStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state"), nil
}

// ReadModeState reads the current mode from ~/.shark/state.
// Returns "prod" as the default when the file is absent or unreadable.
func ReadModeState() (string, error) {
	path, err := sharkStatePath()
	if err != nil {
		return "prod", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "prod", nil
		}
		return "prod", err
	}
	var s ModeState
	if err := json.Unmarshal(data, &s); err != nil {
		return "prod", nil
	}
	if s.Mode != "dev" && s.Mode != "prod" {
		return "prod", nil
	}
	return s.Mode, nil
}

// WriteModeState persists the given mode to ~/.shark/state.
func WriteModeState(mode string) error {
	dir, err := sharkStateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "state")
	data, err := json.Marshal(ModeState{Mode: mode})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
