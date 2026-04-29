package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/shark-auth/shark/internal/config"
)

var testArgon2Config = config.Argon2idConfig{
	Memory:      16384,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func TestHashPassword(t *testing.T) {
	password := "correcthorsebatterystaple"

	hash, err := HashPassword(password, testArgon2Config)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !match {
		t.Fatal("VerifyPassword returned false for correct password")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	password := "correcthorsebatterystaple"
	wrongPassword := "wrongpassword"

	hash, err := HashPassword(password, testArgon2Config)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	match, err := VerifyPassword(wrongPassword, hash)
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if match {
		t.Fatal("VerifyPassword returned true for wrong password")
	}
}

func TestHashUniqueness(t *testing.T) {
	password := "samepassword"

	hash1, err := HashPassword(password, testArgon2Config)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	hash2, err := HashPassword(password, testArgon2Config)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash1 == hash2 {
		t.Fatal("Two hashes of the same password should be different (random salt)")
	}
}

func TestVerifyBcryptHash(t *testing.T) {
	password := "auth0password"

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generating bcrypt hash: %v", err)
	}

	match, err := VerifyPassword(password, string(bcryptHash))
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if !match {
		t.Fatal("VerifyPassword returned false for correct bcrypt password")
	}

	// Wrong password against bcrypt hash
	match, err = VerifyPassword("wrongpassword", string(bcryptHash))
	if err != nil {
		t.Fatalf("VerifyPassword returned error: %v", err)
	}
	if match {
		t.Fatal("VerifyPassword returned true for wrong bcrypt password")
	}
}

func TestNeedsRehash(t *testing.T) {
	argon2Hash, err := HashPassword("password", testArgon2Config)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if NeedsRehash(argon2Hash) {
		t.Fatal("NeedsRehash returned true for argon2id hash")
	}

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generating bcrypt hash: %v", err)
	}

	if !NeedsRehash(string(bcryptHash)) {
		t.Fatal("NeedsRehash returned false for bcrypt hash")
	}
}
