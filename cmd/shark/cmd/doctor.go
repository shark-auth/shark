package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// checkResult holds the outcome of a single doctor check.
type checkResult struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Warn   bool   `json:"warn,omitempty"` // pass but advisory
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run self-diagnostic checks against a configured shark deployment",
	Long:  "Verifies config, DB, migrations, JWT keys, port, reachability, admin key, SMTP, and vault.",
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().Bool("json", false, "emit one JSON object per check to stdout (machine-readable)")
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	useJSON, _ := cmd.Flags().GetBool("json")
	ctx := context.Background()

	// --- load config (exit 2 on failure) ---
	cfg, err := config.Load("")
	if err != nil {
		msg := fmt.Sprintf("cannot load config: %v", err)
		if useJSON {
			r := checkResult{Name: "config", OK: false, Detail: msg}
			_ = json.NewEncoder(cmd.OutOrStdout()).Encode(r)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "✗ config              %s\n", msg)
		}
		return fmt.Errorf("%w", errExit2)
	}

	// --- open DB (needed by multiple checks) ---
	dbPath := resolveDBPath(cfg)
	store, dbErr := storage.NewSQLiteStore(dbPath)
	if dbErr != nil {
		// DB open fail: mark db checks as failed but continue others
	}
	if store != nil {
		defer store.Close()
	}

	checks := []func() checkResult{
		func() checkResult { return checkConfig(ctx, cfg) },
		func() checkResult { return checkDBWritability(ctx, cfg, store, dbErr) },
		func() checkResult { return checkMigrations(ctx, cfg, store, dbErr) },
		func() checkResult { return checkJWTKeys(ctx, store, dbErr) },
		func() checkResult { return checkPortBind(cfg) },
		func() checkResult { return checkBaseURLReachability(cfg) },
		func() checkResult { return checkAdminKey(ctx, cfg, store, dbErr) },
		func() checkResult { return checkSMTP(cfg) },
		func() checkResult { return checkVault(cfg) },
	}

	results := make([]checkResult, 0, len(checks))
	passed := 0

	for _, fn := range checks {
		r := fn()
		results = append(results, r)

		if useJSON {
			_ = json.NewEncoder(cmd.OutOrStdout()).Encode(r)
		} else {
			sym := "✓"
			if !r.OK {
				sym = "✗"
			} else if r.Warn {
				sym = "⚠"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %-20s %s\n", sym, r.Name, r.Detail)
		}

		if r.OK {
			passed++
		}
	}

	total := len(results)
	summary := fmt.Sprintf("%d/%d checks passed", passed, total)
	if useJSON {
		_ = json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"summary": summary,
			"passed":  passed,
			"total":   total,
		})
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", summary)
	}

	if passed < total {
		return fmt.Errorf("%w", errExit1)
	}
	return nil
}

// errExit1 / errExit2 are sentinel errors that main.go maps to exit codes.
// cobra's RunE returns these; the caller in main.go checks os.Exit(1/2).
// We rely on the existing pattern: non-nil error → exit 1 from cobra.
// For exit 2 we use a special wrapper that main.go can detect.
var (
	errExit1 = errCode{code: 1, msg: "one or more checks failed"}
	errExit2 = errCode{code: 2, msg: "config not loadable"}
)

type errCode struct {
	code int
	msg  string
}

func (e errCode) Error() string    { return e.msg }
func (e errCode) ExitCode() int    { return e.code }

// AsExitCoder checks whether err (or any wrapped error) carries an ExitCode()
// method and returns the code. Used by main.go to select the correct os.Exit value.
func AsExitCoder(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	type exitCoder interface{ ExitCode() int }
	// Walk the error chain manually (errors.As requires a concrete target type).
	for e := err; e != nil; {
		if ec, ok := e.(exitCoder); ok {
			return ec.ExitCode(), true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := e.(unwrapper); ok {
			e = u.Unwrap()
		} else {
			break
		}
	}
	return 0, false
}

// resolveDBPath returns the effective DB file path, honouring env var override.
func resolveDBPath(cfg *config.Config) string {
	if p := os.Getenv("SHARK_DB_PATH"); p != "" {
		return p
	}
	return cfg.Storage.Path
}

// --- individual checks ---

func checkConfig(_ context.Context, cfg *config.Config) checkResult {
	detail := fmt.Sprintf("base_url=%s port=%d db=%s", cfg.Server.BaseURL, cfg.Server.Port, cfg.Storage.Path)
	return checkResult{Name: "config", OK: true, Detail: detail}
}

func checkDBWritability(ctx context.Context, cfg *config.Config, store *storage.SQLiteStore, openErr error) checkResult {
	dbPath := resolveDBPath(cfg)
	if openErr != nil {
		return checkResult{Name: "db_writability", OK: false,
			Detail: fmt.Sprintf("cannot open DB at %s: %v", dbPath, openErr)}
	}

	// Attempt a write + rollback using a sentinel table (doctor_sentinel).
	// We use the raw DB handle via a direct SQL operation on a known safe table.
	// We INSERT into a temp table that we create and drop within a transaction.
	db := store.DB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return checkResult{Name: "db_writability", OK: false,
			Detail: fmt.Sprintf("begin tx: %v", err)}
	}
	_, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _doctor_sentinel (v TEXT)`)
	if err != nil {
		_ = tx.Rollback()
		return checkResult{Name: "db_writability", OK: false,
			Detail: fmt.Sprintf("create sentinel: %v", err)}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO _doctor_sentinel VALUES ('ok')`)
	if err != nil {
		_ = tx.Rollback()
		return checkResult{Name: "db_writability", OK: false,
			Detail: fmt.Sprintf("write sentinel: %v", err)}
	}
	_ = tx.Rollback() // intentional rollback — we never want to persist this

	// Report file size
	info, _ := os.Stat(dbPath)
	sizeStr := "size=unknown"
	if info != nil {
		sizeStr = fmt.Sprintf("size=%.1fMB", float64(info.Size())/1e6)
	}
	return checkResult{Name: "db_writability", OK: true,
		Detail: fmt.Sprintf("path=%s %s", dbPath, sizeStr)}
}

func checkMigrations(ctx context.Context, _ *config.Config, store *storage.SQLiteStore, openErr error) checkResult {
	if openErr != nil {
		return checkResult{Name: "migrations", OK: false, Detail: fmt.Sprintf("DB unavailable: %v", openErr)}
	}
	db := store.DB()

	// goose tracks applied migrations in the `goose_db_version` table.
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goose_db_version WHERE is_applied = 1`).Scan(&count)
	if err != nil {
		// fallback: try user_version pragma
		var ver int
		if e2 := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&ver); e2 == nil {
			return checkResult{Name: "migrations", OK: true,
				Detail: fmt.Sprintf("user_version=%d (goose table not found)", ver)}
		}
		return checkResult{Name: "migrations", OK: false,
			Detail: fmt.Sprintf("cannot query migrations: %v", err)}
	}

	// Check for pending (unapplied) rows — goose marks these as is_applied=0
	var pending int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goose_db_version WHERE is_applied = 0`).Scan(&pending)

	detail := fmt.Sprintf("applied=%d", count)
	if pending > 0 {
		return checkResult{Name: "migrations", OK: false,
			Detail: fmt.Sprintf("%s pending=%d — run migrations before starting server", detail, pending)}
	}
	return checkResult{Name: "migrations", OK: true, Detail: detail}
}

func checkJWTKeys(ctx context.Context, store *storage.SQLiteStore, openErr error) checkResult {
	if openErr != nil {
		return checkResult{Name: "jwt_keys", OK: false, Detail: fmt.Sprintf("DB unavailable: %v", openErr)}
	}
	db := store.DB()

	rows, err := db.QueryContext(ctx,
		`SELECT algorithm, created_at FROM jwt_signing_keys WHERE status = 'active'`)
	if err != nil {
		return checkResult{Name: "jwt_keys", OK: false,
			Detail: fmt.Sprintf("query active keys: %v", err)}
	}
	defer rows.Close()

	type keyInfo struct {
		alg       string
		createdAt time.Time
	}
	var keys []keyInfo
	for rows.Next() {
		var alg, createdAtStr string
		if e := rows.Scan(&alg, &createdAtStr); e != nil {
			continue
		}
		t, _ := time.Parse(time.RFC3339, createdAtStr)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		keys = append(keys, keyInfo{alg: alg, createdAt: t})
	}
	if err := rows.Err(); err != nil {
		return checkResult{Name: "jwt_keys", OK: false, Detail: fmt.Sprintf("scan: %v", err)}
	}

	if len(keys) == 0 {
		return checkResult{Name: "jwt_keys", OK: false,
			Detail: "no active signing keys — run `shark serve` to auto-generate"}
	}

	// build detail string; warn if any key >90 days old
	oldCount := 0
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		age := time.Since(k.createdAt)
		days := int(age.Hours() / 24)
		parts = append(parts, fmt.Sprintf("%s(age=%dd)", k.alg, days))
		if days > 90 {
			oldCount++
		}
	}
	detail := fmt.Sprintf("count=%d %s", len(keys), strings.Join(parts, " "))
	if oldCount > 0 {
		return checkResult{Name: "jwt_keys", OK: true, Warn: true,
			Detail: detail + " — consider rotating (>90 days)"}
	}
	return checkResult{Name: "jwt_keys", OK: true, Detail: detail}
}

func checkPortBind(cfg *config.Config) checkResult {
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		pid := findPIDOnPort(cfg.Server.Port)
		detail := fmt.Sprintf("port %d already in use", cfg.Server.Port)
		if pid != "" {
			detail += fmt.Sprintf(" — PID %s", pid)
		}
		return checkResult{Name: "port_bind", OK: false, Detail: detail}
	}
	_ = ln.Close()
	return checkResult{Name: "port_bind", OK: true,
		Detail: fmt.Sprintf("port %d is free", cfg.Server.Port)}
}

// findPIDOnPort uses lsof (Unix) or netstat (Windows) to find the PID holding a port.
func findPIDOnPort(port int) string {
	portStr := fmt.Sprintf("%d", port)
	if runtime.GOOS == "windows" {
		out, err := exec.Command("netstat", "-ano").Output()
		if err != nil {
			return ""
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, ":"+portStr) && strings.Contains(line, "LISTENING") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					return fields[len(fields)-1]
				}
			}
		}
		return ""
	}
	// Unix: lsof
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%s", portStr)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func checkBaseURLReachability(cfg *config.Config) checkResult {
	url := strings.TrimRight(cfg.Server.BaseURL, "/") + "/api/v1/health"
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return checkResult{Name: "base_url", OK: false,
			Detail: fmt.Sprintf("GET %s: %v", url, err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return checkResult{Name: "base_url", OK: true,
			Detail: fmt.Sprintf("GET %s → %d", url, resp.StatusCode)}
	}
	return checkResult{Name: "base_url", OK: false,
		Detail: fmt.Sprintf("GET %s → %d (expected 200)", url, resp.StatusCode)}
}

func checkAdminKey(ctx context.Context, cfg *config.Config, store *storage.SQLiteStore, openErr error) checkResult {
	// Key file locations to probe
	dbPath := resolveDBPath(cfg)
	keyPaths := []string{
		filepath.Join(filepath.Dir(dbPath), "admin.key.firstboot"),
		"data/admin.key.firstboot",
		"admin.key.firstboot",
	}
	for _, p := range keyPaths {
		if _, err := os.Stat(p); err == nil {
			return checkResult{Name: "admin_key", OK: true,
				Detail: fmt.Sprintf("key file found at %s", p)}
		}
	}

	// No key file — check if at least one API key row (admin key) is present in DB
	if openErr == nil && store != nil {
		var count int
		err := store.DB().QueryRowContext(ctx, `SELECT COUNT(*) FROM api_keys`).Scan(&count)
		if err == nil && count > 0 {
			return checkResult{Name: "admin_key", OK: true,
				Detail: fmt.Sprintf("no key file but %d api_key(s) configured in DB", count)}
		}
	}

	return checkResult{Name: "admin_key", OK: false,
		Detail: "admin.key.firstboot not found AND no api_keys in DB — run `shark serve` to initialise"}
}

func checkSMTP(cfg *config.Config) checkResult {
	host := cfg.Email.Host
	port := cfg.Email.Port
	if host == "" {
		host = cfg.SMTP.Host
		port = cfg.SMTP.Port
	}
	if host == "" {
		return checkResult{Name: "smtp", OK: true, Detail: "not configured (skipped)"}
	}
	if port == 0 {
		port = 587
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Attempt TCP dial first
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return checkResult{Name: "smtp", OK: false,
			Detail: fmt.Sprintf("cannot reach %s: %v", addr, err)}
	}
	_ = conn.Close()

	// Attempt TLS handshake (SMTPS port 465) — skip for STARTTLS ports
	if port == 465 {
		tlsConn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 5 * time.Second},
			"tcp", addr,
			&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12},
		)
		if err != nil {
			return checkResult{Name: "smtp", OK: false,
				Detail: fmt.Sprintf("TLS handshake failed for %s: %v", addr, err)}
		}
		_ = tlsConn.Close()
		return checkResult{Name: "smtp", OK: true, Detail: fmt.Sprintf("%s TLS ok", addr)}
	}
	return checkResult{Name: "smtp", OK: true, Detail: fmt.Sprintf("%s TCP ok (STARTTLS)", addr)}
}

func checkVault(cfg *config.Config) checkResult {
	secret := cfg.Server.Secret
	if secret == "" {
		return checkResult{Name: "vault", OK: false,
			Detail: "server.secret is empty — vault key derivation impossible"}
	}
	enc, err := auth.NewFieldEncryptor(secret)
	if err != nil {
		return checkResult{Name: "vault", OK: false,
			Detail: fmt.Sprintf("NewFieldEncryptor: %v", err)}
	}
	const testPlain = "shark-doctor-vault-test"
	ciphertext, err := enc.Encrypt(testPlain)
	if err != nil {
		return checkResult{Name: "vault", OK: false,
			Detail: fmt.Sprintf("encrypt: %v", err)}
	}
	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		return checkResult{Name: "vault", OK: false,
			Detail: fmt.Sprintf("decrypt: %v", err)}
	}
	if decrypted != testPlain {
		return checkResult{Name: "vault", OK: false,
			Detail: "round-trip mismatch — key derivation is broken"}
	}
	return checkResult{Name: "vault", OK: true, Detail: "AES-256-GCM round-trip ok"}
}

