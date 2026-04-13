package auth

import (
	"sync"
	"time"
)

// LockoutManager tracks failed login attempts per account and enforces lockout.
type LockoutManager struct {
	mu        sync.Mutex
	attempts  map[string]*lockoutEntry
	maxFails  int
	lockoutDur time.Duration
}

type lockoutEntry struct {
	failures    int
	lastFailure time.Time
	lockedUntil time.Time
}

// NewLockoutManager creates a LockoutManager.
// maxFailures is the number of failed attempts before lockout.
// lockoutDuration is how long the account stays locked.
func NewLockoutManager(maxFailures int, lockoutDuration time.Duration) *LockoutManager {
	lm := &LockoutManager{
		attempts:   make(map[string]*lockoutEntry),
		maxFails:   maxFailures,
		lockoutDur: lockoutDuration,
	}

	// Cleanup stale entries every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			lm.cleanup()
		}
	}()

	return lm
}

// IsLocked returns true if the account (by email) is currently locked out.
func (lm *LockoutManager) IsLocked(email string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, ok := lm.attempts[email]
	if !ok {
		return false
	}
	if time.Now().Before(entry.lockedUntil) {
		return true
	}
	return false
}

// RecordFailure records a failed login attempt. Returns true if the account is now locked.
func (lm *LockoutManager) RecordFailure(email string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	entry, ok := lm.attempts[email]
	if !ok {
		entry = &lockoutEntry{}
		lm.attempts[email] = entry
	}

	// Reset if lockout has expired
	if time.Now().After(entry.lockedUntil) && entry.failures >= lm.maxFails {
		entry.failures = 0
	}

	entry.failures++
	entry.lastFailure = time.Now()

	if entry.failures >= lm.maxFails {
		entry.lockedUntil = time.Now().Add(lm.lockoutDur)
		return true
	}

	return false
}

// RecordSuccess clears the failure counter for a successful login.
func (lm *LockoutManager) RecordSuccess(email string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	delete(lm.attempts, email)
}

func (lm *LockoutManager) cleanup() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for email, entry := range lm.attempts {
		if entry.lastFailure.Before(cutoff) && time.Now().After(entry.lockedUntil) {
			delete(lm.attempts, email)
		}
	}
}
