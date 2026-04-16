package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/sharkauth/sharkauth/internal/server"
)

// applyDevMode mutates Options to run the server in developer mode:
// ephemeral SQLite file, auto-generated 32-byte secret when missing,
// dev inbox email sender, dev routes enabled.
func applyDevMode(opts *server.Options, reset bool) error {
	opts.DevMode = true

	// Point at a resettable dev.db alongside the working directory.
	if reset {
		if err := os.Remove("./dev.db"); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reset dev.db: %w", err)
		}
	}

	// Auto-generate a secret so `shark serve --dev` works without any config.
	secret, err := randomHex(32)
	if err != nil {
		return fmt.Errorf("generate dev secret: %w", err)
	}
	opts.SecretOverride = secret

	// Email sender injection is wired in W8 once DevInboxSender exists.
	return nil
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
