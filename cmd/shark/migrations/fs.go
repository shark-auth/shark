// Package migrations exposes the canonical on-disk SQL migrations
// for `shark serve` as an embed.FS. Re-export so non-main packages
// (notably internal/testutil/cli) can reuse the same migration set
// instead of keeping a sibling copy that silently drifts.
//
// Before v1.5 Lane D, a separate `testmigrations/` directory lived
// under internal/testutil/cli and stopped at 00012 — far behind the
// canonical set — causing TestE2EServeFlow to fail with "no such
// column: proxy_public_domain". Removing that directory and routing
// the harness through this package eliminates the 4-dir drift class
// called out in the handoff gotchas.
package migrations

import "embed"

// FS is the canonical migrations directory, embedded so packages
// outside cmd/shark can depend on it without an os.ReadFile dance
// at test time. The embed pattern mirrors cmd/shark/main.go's
// //go:embed migrations/*.sql — both are kept intentionally in
// sync; main.go continues to use its own embed for clarity at the
// process entry point while the harness and any future non-main
// consumers import this package.
//
//go:embed *.sql
var FS embed.FS
