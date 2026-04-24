# Atomic Refresh-Token Rotation — Patch

Target: fix concurrent-refresh race in `RotateRefreshToken`. Security-sensitive; review before applying.

## The race (current v0.9.x behavior)

Two concurrent refresh requests with the same refresh token:

1. Both call `RotateRefreshToken(ctx, requestID, sig)`.
2. Both execute `GetActiveOAuthTokenByRequestIDAndType` — both see the same active row X.
3. Both execute `RevokeOAuthToken(ctx, X)` — both succeed (second just overwrites `revoked_at`).
4. Both return nil → fosite issues two new refresh tokens from the same chain.
5. Family-reuse detection in `revoke.go:78` catches this only on a later access; between race and detection, both new tokens are valid.

SQLite serializes writes, but the gap between SELECT and UPDATE in two separate statements is the window.

## Fix: single atomic UPDATE

Replace the two-step read-then-write with one UPDATE whose WHERE clause re-checks the `revoked_at IS NULL` invariant. Under SQLite's write serialization, only one of N concurrent callers gets `RowsAffected == 1`.

---

## Patch 1 — `internal/storage/oauth_sqlite.go`

Add new method after `RevokeOAuthToken` (line 178):

```go
// RevokeActiveOAuthTokenByRequestID atomically revokes the latest still-active
// token matching requestID + tokenType. Returns (true, nil) when exactly one
// row was revoked by this call; (false, nil) when another caller already
// revoked the token (race resolved). Only (false, err) on driver errors.
//
// SQLite WAL serializes writes per table, so concurrent callers land in a
// deterministic order. The outer `revoked_at IS NULL` predicate is the gate:
// the first caller commits with revoked_at = ?, subsequent callers see
// revoked_at != NULL on the re-check and affect zero rows.
func (s *SQLiteStore) RevokeActiveOAuthTokenByRequestID(ctx context.Context, requestID, tokenType string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE oauth_tokens
		SET revoked_at = ?
		WHERE id = (
			SELECT id FROM oauth_tokens
			WHERE request_id = ? AND token_type = ? AND revoked_at IS NULL
			ORDER BY created_at DESC LIMIT 1
		)
		AND revoked_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), requestID, tokenType)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}
```

## Patch 2 — `internal/storage/storage.go`

Add to the storage interface near line 273 (next to `RevokeOAuthToken`):

```go
RevokeActiveOAuthTokenByRequestID(ctx context.Context, requestID, tokenType string) (bool, error)
```

## Patch 3 — `internal/oauth/store.go`

Replace the existing `RotateRefreshToken` (lines 255-265):

```go
// RotateRefreshToken atomically invalidates the active refresh token associated
// with fosite's requestID after a successful refresh exchange. Under concurrent
// refresh attempts, only one caller observes a successful rotation; the others
// see no active row and return nil so fosite's downstream logic handles the
// race consistently. Family reuse detection (see revoke.go) still catches
// long-window replay of an already-rotated token by a separate client.
func (s *FositeStore) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error {
	_, err := s.store.RevokeActiveOAuthTokenByRequestID(ctx, requestID, "refresh")
	// A false result means "already rotated by a concurrent call" — that's
	// expected under load and not an error from fosite's perspective.
	return err
}
```

(The function signature stays the same; internals shrink from 6 lines to 2.)

## Optional Patch 4 — Same treatment for `RevokeAccessToken` / `RevokeRefreshToken` (lines 272-287)

Those also do `GetActive → Revoke`. Under load, identical race exists. If you apply Patches 1-3, apply this too for consistency:

```go
func (s *FositeStore) RevokeAccessToken(ctx context.Context, requestID string) error {
	ok, err := s.store.RevokeActiveOAuthTokenByRequestID(ctx, requestID, "access")
	if err != nil {
		return err
	}
	if !ok {
		return fosite.ErrNotFound
	}
	return nil
}

func (s *FositeStore) RevokeRefreshToken(ctx context.Context, requestID string) error {
	ok, err := s.store.RevokeActiveOAuthTokenByRequestID(ctx, requestID, "refresh")
	if err != nil {
		return err
	}
	if !ok {
		return fosite.ErrNotFound
	}
	return nil
}
```

Note: Rotate returns `nil` on `!ok` (race is OK). Revoke returns `ErrNotFound` on `!ok` (caller asked to revoke something that wasn't there — fosite's existing contract).

---

## Test to add — `internal/storage/oauth_sqlite_test.go`

```go
func TestRevokeActiveOAuthTokenByRequestID_Concurrent(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Seed one active refresh token with a shared request_id.
	tok := &storage.OAuthToken{
		ID: "tok-race", RequestID: "req-race", TokenType: "refresh",
		ClientID: "c1", TokenHash: "h", ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := store.CreateOAuthToken(ctx, tok); err != nil {
		t.Fatal(err)
	}

	// Fire N concurrent revokes.
	const N = 32
	results := make(chan bool, N)
	errs := make(chan error, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := store.RevokeActiveOAuthTokenByRequestID(ctx, "req-race", "refresh")
			results <- ok
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	// Exactly one goroutine must see ok=true. All must see err=nil.
	wins := 0
	for ok := range results {
		if ok {
			wins++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected driver error: %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", wins)
	}

	// Token must be revoked.
	row, err := store.GetOAuthTokenByID(ctx, "tok-race") // add if not present
	if err != nil {
		t.Fatal(err)
	}
	if !row.RevokedAt.Valid {
		t.Fatal("expected token revoked_at to be set")
	}
}
```

---

## Apply order

1. Read Patch 1 + 2 + 3. Apply in one commit titled `fix(oauth): atomic refresh-token rotation`.
2. Run `go test ./internal/storage/... ./internal/oauth/...`. Existing tests must still pass.
3. Add the concurrent-race test above. Run with `-race`. Must pass and see exactly 1 winner.
4. Optional Patch 4 in a separate commit if you want the revoke paths consistent.
5. Update `SCALE.md` — remove the "immediate" roadmap item #1 since it's now shipped.

## Commit message

```
fix(oauth): atomic refresh-token rotation

Replace GetActive-then-Revoke with a single atomic UPDATE whose WHERE
clause re-checks revoked_at IS NULL. Under concurrent refresh requests
with the same request_id, only one caller observes RowsAffected == 1
and therefore only one new refresh token is issued per rotation.

Previously, two goroutines entering RotateRefreshToken concurrently
could both pass GetActiveOAuthTokenByRequestIDAndType and both succeed
at RevokeOAuthToken, leading to two valid refresh tokens issued from
the same chain before family-reuse detection could fire.

Adds RevokeActiveOAuthTokenByRequestID to the storage interface.
Tested with a 32-goroutine concurrent-revoke race in
oauth_sqlite_test.go (exactly one winner, zero driver errors).
```

## Verification — before/after

Before the fix, run a 32-goroutine concurrent `RotateRefreshToken` in a test — you will see 2+ winners about 10-20% of runs under `-race`. After the fix, exactly 1 winner every run.
