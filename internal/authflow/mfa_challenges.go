package authflow

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// challengeTTL is how long a flow-step MFA challenge lives before expiry.
// Short enough to be useless if captured, long enough for a human to act.
const challengeTTL = 5 * time.Minute

// MFAChallenge is a single pending challenge entry.
type MFAChallenge struct {
	UserID    string
	FlowRunID string // informational; not validated on verify
	IssuedAt  time.Time
}

// ChallengeStore holds in-flight MFA challenges for the require_mfa_challenge
// step. It is safe for concurrent use.
//
// Design note: in-memory is acceptable for single-instance self-host and
// matches the spec (CLOUD.md §1 calls out that this must migrate to Redis for
// the multi-tenant Cloud fork before GA). Challenges that survive a restart
// are simply lost — the user must attempt auth again, which is a known and
// documented failure mode.
type ChallengeStore struct {
	mu         sync.Mutex
	challenges map[string]MFAChallenge
}

// GlobalChallengeStore is the package-level singleton used by Engine steps.
// Tests that need isolation should call NewChallengeStore and inject it.
var GlobalChallengeStore = NewChallengeStore()

// NewChallengeStore returns an empty ChallengeStore.
func NewChallengeStore() *ChallengeStore {
	return &ChallengeStore{
		challenges: make(map[string]MFAChallenge),
	}
}

// Issue mints a new challenge ID, stores it, and returns it. Old, expired
// entries are swept lazily on each Issue call.
func (cs *ChallengeStore) Issue(userID, flowRunID string) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.sweepLocked()

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand can't fail in practice; if it does use timestamp fallback
		buf = []byte(time.Now().UTC().Format("20060102150405.000000000"))
	}
	id := "mfac_" + hex.EncodeToString(buf)
	cs.challenges[id] = MFAChallenge{
		UserID:    userID,
		FlowRunID: flowRunID,
		IssuedAt:  time.Now(),
	}
	return id
}

// Consume looks up challengeID, validates it belongs to userID and hasn't
// expired, then deletes it. Returns true on success.
func (cs *ChallengeStore) Consume(challengeID, userID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, ok := cs.challenges[challengeID]
	if !ok {
		return false
	}
	if c.UserID != userID {
		return false
	}
	if time.Since(c.IssuedAt) > challengeTTL {
		delete(cs.challenges, challengeID)
		return false
	}
	delete(cs.challenges, challengeID)
	return true
}

// Peek returns the challenge without consuming it (for test assertions).
func (cs *ChallengeStore) Peek(challengeID string) (MFAChallenge, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.challenges[challengeID]
	return c, ok
}

// sweepLocked removes all expired entries. Must be called with cs.mu held.
func (cs *ChallengeStore) sweepLocked() {
	now := time.Now()
	for id, c := range cs.challenges {
		if now.Sub(c.IssuedAt) > challengeTTL {
			delete(cs.challenges, id)
		}
	}
}
