package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Write a minimal config with just the required secret
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")
	os.WriteFile(cfgPath, []byte(`server:
  secret: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`), 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Auth.PasswordMinLength != 8 {
		t.Fatalf("expected default password_min_length 8, got %d", cfg.Auth.PasswordMinLength)
	}
	if cfg.Auth.Argon2id.Memory != 65536 {
		t.Fatalf("expected default argon2id memory 65536, got %d", cfg.Auth.Argon2id.Memory)
	}
	if cfg.MFA.RecoveryCodes != 10 {
		t.Fatalf("expected default recovery_codes 10, got %d", cfg.MFA.RecoveryCodes)
	}
}

func TestLoad_YAMLOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")
	os.WriteFile(cfgPath, []byte(`server:
  port: 9090
  secret: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
auth:
  session_lifetime: "7d"
  password_min_length: 12
`), 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Auth.PasswordMinLength != 12 {
		t.Fatalf("expected password_min_length 12, got %d", cfg.Auth.PasswordMinLength)
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")
	os.WriteFile(cfgPath, []byte(`server:
  secret: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`), 0644)

	t.Setenv("SHARKAUTH_SERVER__PORT", "3000")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Fatalf("expected port 3000 from env var, got %d", cfg.Server.Port)
	}
}

func TestLoad_EnvVarInterpolation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test.yaml")
	os.WriteFile(cfgPath, []byte(`server:
  secret: "${TEST_SHARK_SECRET}"
`), 0644)

	t.Setenv("TEST_SHARK_SECRET", "abcdefghijklmnopqrstuvwxyz123456")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Secret != "abcdefghijklmnopqrstuvwxyz123456" {
		t.Fatalf("expected interpolated secret, got %q", cfg.Server.Secret)
	}
}

func TestParseDuration_Days(t *testing.T) {
	d := parseDuration("30d", 0)
	if d != 30*24*time.Hour {
		t.Fatalf("expected 30 days, got %v", d)
	}
}

func TestParseDuration_Standard(t *testing.T) {
	d := parseDuration("2h30m", 0)
	if d != 2*time.Hour+30*time.Minute {
		t.Fatalf("expected 2h30m, got %v", d)
	}
}

func TestParseDuration_Fallback(t *testing.T) {
	d := parseDuration("", 5*time.Minute)
	if d != 5*time.Minute {
		t.Fatalf("expected fallback 5m, got %v", d)
	}
}

func TestSessionLifetimeDuration(t *testing.T) {
	a := &AuthConfig{SessionLifetime: "7d"}
	d := a.SessionLifetimeDuration()
	if d != 7*24*time.Hour {
		t.Fatalf("expected 7 days, got %v", d)
	}
}
