package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/sharkauth/sharkauth/internal/proxy"
)

var (
	proxyUpstream  string
	proxyPort      int
	proxyAuthBase  string
	proxyRulesFile string
)

// proxyRulesFileSpec mirrors the rule YAML shape accepted by the shared
// config package, duplicated locally so the standalone CLI doesn't have
// to load a full sharkauth.yaml just to read a rules list.
type proxyRulesFileSpec struct {
	Rules []proxyRuleSpec `yaml:"rules"`
}

type proxyRuleSpec struct {
	Path    string   `yaml:"path"`
	Methods []string `yaml:"methods"`
	Require string   `yaml:"require"`
	Allow   string   `yaml:"allow"`
	Scopes  []string `yaml:"scopes"`
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run the standalone Shark reverse proxy",
	Long: `Starts a reverse proxy that connects to a Shark auth instance
for health monitoring and JWT verification. Useful when you want the
proxy in a separate process from the auth server (e.g. an edge
deployment).

Authorization: Bearer <jwt> is verified on every request against the
auth server's /.well-known/jwks.json, cached in-memory with a 15-minute
refresh (plus an eager refresh on an unknown kid). Requests without a
valid JWT are treated as anonymous; rules with require:authenticated
will therefore return 403.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if proxyUpstream == "" {
			return errors.New("--upstream is required")
		}
		if proxyAuthBase == "" {
			return errors.New("--auth is required (Shark auth base URL)")
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		logger := slog.Default()

		engine, err := loadProxyRulesFile(proxyRulesFile)
		if err != nil {
			return fmt.Errorf("load rules: %w", err)
		}

		breakerCfg := proxy.BreakerConfig{
			HealthURL:        proxyAuthBase + "/api/v1/admin/health",
			HealthInterval:   10 * time.Second,
			FailureThreshold: 3,
			CacheSize:        10000,
			CacheTTL:         5 * time.Minute,
			NegativeTTL:      30 * time.Second,
			MissBehavior:     proxy.MissReject,
		}
		breaker := proxy.NewBreaker(breakerCfg, logger)
		breaker.Start(ctx)
		defer breaker.Stop()

		proxyCfg := proxy.Config{
			Enabled:       true,
			Upstream:      proxyUpstream,
			Timeout:       30 * time.Second,
			StripIncoming: true,
		}
		h, err := proxy.New(proxyCfg, engine, logger)
		if err != nil {
			return fmt.Errorf("build proxy: %w", err)
		}

		// W15c: fetch JWKS from the Shark auth server and verify Bearer
		// tokens on every request. Falls back to anonymous identity when
		// no Authorization header is present; an invalid/expired/tampered
		// token short-circuits to 401 before the rules engine runs so
		// callers get a clear error rather than an opaque forbidden.
		jwks := newJWKSCache(proxyAuthBase, logger)
		if err := jwks.Start(ctx); err != nil {
			return fmt.Errorf("jwks: %w", err)
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				r = r.WithContext(proxy.WithIdentity(r.Context(), proxy.Identity{AuthMethod: "anonymous"}))
				h.ServeHTTP(w, r)
				return
			}
			id, verr := jwks.verifyBearer(r.Context(), auth)
			if verr != nil {
				logger.Debug("standalone proxy: JWT verify failed", "err", verr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(proxy.WithIdentity(r.Context(), id))
			h.ServeHTTP(w, r)
		})

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", proxyPort),
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}

		errCh := make(chan error, 1)
		go func() {
			logger.Info("shark proxy listening",
				"addr", srv.Addr,
				"upstream", proxyUpstream,
				"auth_base", proxyAuthBase,
			)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
			close(errCh)
		}()

		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		case err, ok := <-errCh:
			if ok && err != nil {
				return err
			}
			return nil
		}
	},
}

// loadProxyRulesFile compiles the rule list at path into a proxy.Engine.
// An empty path is permitted and yields an engine with no rules —
// effectively "deny everything via the default-deny fall-through", which
// matches the safe-by-default behavior for the MVP anonymous-only mode.
func loadProxyRulesFile(path string) (*proxy.Engine, error) {
	if path == "" {
		return proxy.NewEngine(nil)
	}
	raw, err := os.ReadFile(path) //#nosec G304 -- path comes from a CLI flag, not user input over the wire
	if err != nil {
		return nil, err
	}
	var spec proxyRulesFileSpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	specs := make([]proxy.RuleSpec, len(spec.Rules))
	for i, r := range spec.Rules {
		specs[i] = proxy.RuleSpec{
			Path:    r.Path,
			Methods: r.Methods,
			Require: r.Require,
			Allow:   r.Allow,
			Scopes:  r.Scopes,
		}
	}
	return proxy.NewEngine(specs)
}

func init() {
	proxyCmd.Flags().StringVar(&proxyUpstream, "upstream", "", "upstream URL to proxy to (required)")
	proxyCmd.Flags().IntVar(&proxyPort, "port", 8081, "port to listen on")
	proxyCmd.Flags().StringVar(&proxyAuthBase, "auth", "", "Shark auth base URL for health monitoring (required)")
	proxyCmd.Flags().StringVar(&proxyRulesFile, "rules", "", "path to a YAML file with a 'rules:' list (optional)")

	root.AddCommand(proxyCmd)
}
