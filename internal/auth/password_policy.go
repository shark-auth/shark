package auth

import (
	"strings"
	"unicode"
)

// commonPasswords is a small set of extremely common passwords to reject outright.
// Kept intentionally small to avoid bloating the binary — this catches the worst offenders.
var commonPasswords = map[string]bool{
	"password": true, "12345678": true, "123456789": true, "1234567890": true,
	"qwerty123": true, "password1": true, "iloveyou": true, "admin123": true,
	"welcome1": true, "abc12345": true, "password123": true, "letmein12": true,
	"monkey123": true, "dragon12": true, "master123": true, "qwerty12": true,
	"baseball1": true, "shadow12": true, "michael1": true, "football1": true,
	"trustno1": true, "jordan23": true, "harley12": true, "ranger12": true,
}

// ValidatePasswordComplexity checks that a password meets minimum complexity requirements:
//   - At least minLength characters
//   - Contains at least 1 uppercase letter
//   - Contains at least 1 lowercase letter
//   - Contains at least 1 digit
//   - Not in the common passwords list
//
// Returns an empty string if valid, or a human-readable reason if invalid.
func ValidatePasswordComplexity(password string, minLength int) string {
	if len(password) < minLength {
		return "Password must be at least 8 characters"
	}

	if commonPasswords[strings.ToLower(password)] {
		return "This password is too common, please choose a stronger one"
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return "Password must contain at least one uppercase letter, one lowercase letter, and one digit"
	}

	return ""
}
