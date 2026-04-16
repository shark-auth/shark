package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/sharkauth/sharkauth/internal/server"
)

// applyDevMode mutates Options to run the server in developer mode:
// resettable dev.db beside the working dir, auto-generated 32-byte secret,
// dev inbox email capture (wired in server.Build), relaxed defaults.
func applyDevMode(opts *server.Options, reset bool) error {
	opts.DevMode = true

	if reset {
		for _, suffix := range []string{"", "-wal", "-shm"} {
			p := "./dev.db" + suffix
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("reset %s: %w", p, err)
			}
		}
	}

	secret, err := randomHex(32)
	if err != nil {
		return fmt.Errorf("generate dev secret: %w", err)
	}
	opts.SecretOverride = secret
	opts.StoragePathOverride = "./dev.db"

	return nil
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
