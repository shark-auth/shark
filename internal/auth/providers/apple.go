package providers

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
)

var appleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://appleid.apple.com/auth/authorize",
	TokenURL: "https://appleid.apple.com/auth/token",
}

// Apple implements the OAuthProvider interface for Sign in with Apple.
// Apple is more complex than other providers:
//   - client_secret is a JWT signed with an ES256 private key (.p8 file)
//   - user info comes from the id_token (JWT) rather than a userinfo endpoint
type Apple struct {
	cfg            oauth2.Config
	teamID         string
	keyID          string
	privateKeyPath string
	privateKeyPEM  []byte // cached for testing injection
}

// NewApple creates an Apple OAuth provider from config.
func NewApple(c config.AppleConfig, baseURL string) *Apple {
	return &Apple{
		cfg: oauth2.Config{
			ClientID: c.ClientID,
			Endpoint: appleEndpoint,
			RedirectURL: baseURL + "/api/v1/auth/oauth/apple/callback",
			Scopes:   []string{"name", "email"},
		},
		teamID:         c.TeamID,
		keyID:          c.KeyID,
		privateKeyPath: c.PrivateKeyPath,
	}
}

// NewAppleWithKey creates an Apple provider with an in-memory private key (for testing).
func NewAppleWithKey(clientID, teamID, keyID string, privateKeyPEM []byte, redirectURL string) *Apple {
	return &Apple{
		cfg: oauth2.Config{
			ClientID: clientID,
			Endpoint: appleEndpoint,
			RedirectURL: redirectURL,
			Scopes:   []string{"name", "email"},
		},
		teamID:        teamID,
		keyID:         keyID,
		privateKeyPEM: privateKeyPEM,
	}
}

func (a *Apple) Name() string { return "apple" }

func (a *Apple) AuthURL(state string) string {
	// Apple requires response_mode=form_post for web
	return a.cfg.AuthCodeURL(state, oauth2.SetAuthURLParam("response_mode", "form_post"))
}

func (a *Apple) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	// Generate the client_secret JWT
	secret, err := a.generateClientSecret()
	if err != nil {
		return nil, fmt.Errorf("generating apple client secret: %w", err)
	}
	a.cfg.ClientSecret = secret
	return a.cfg.Exchange(ctx, code)
}

func (a *Apple) GetUser(ctx context.Context, token *oauth2.Token) (*auth.OAuthUserInfo, error) {
	// Apple sends user info in the id_token (JWT).
	// We parse it without verification against Apple's JWKS for simplicity;
	// the token was received directly from Apple's token endpoint over TLS.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("no id_token in apple token response")
	}

	// Parse claims without verification (token is fresh from Apple's token endpoint)
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := jwt.MapClaims{}
	if _, _, err := parser.ParseUnverified(rawIDToken, claims); err != nil {
		return nil, fmt.Errorf("parsing apple id_token: %w", err)
	}

	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	if sub == "" {
		return nil, fmt.Errorf("apple id_token missing sub claim")
	}

	return &auth.OAuthUserInfo{
		ProviderID: sub,
		Email:      email,
		Name:       "", // Apple only sends name on first auth; handled by frontend
		AvatarURL:  "",
	}, nil
}

// generateClientSecret creates a signed JWT for Apple's token endpoint.
// See https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens
func (a *Apple) generateClientSecret() (string, error) {
	keyPEM := a.privateKeyPEM
	if keyPEM == nil {
		var err error
		keyPEM, err = os.ReadFile(a.privateKeyPath)
		if err != nil {
			return "", fmt.Errorf("reading apple private key: %w", err)
		}
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block from apple private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing apple private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss": a.teamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": "https://appleid.apple.com",
		"sub": a.cfg.ClientID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = a.keyID

	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("signing apple client secret JWT: %w", err)
	}

	return signed, nil
}

// appleIDTokenClaims is used only for JSON decoding of the id_token.
type appleIDTokenClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// unmarshalAppleIDToken decodes an id_token from the extras.
func unmarshalAppleIDToken(raw string) (*appleIDTokenClaims, error) {
	// Split JWT into parts and decode the payload (part 1)
	parts := splitJWT(raw)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload, err := jwt.NewParser().DecodeSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding jwt payload: %w", err)
	}

	var claims appleIDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshaling jwt payload: %w", err)
	}

	return &claims, nil
}

func splitJWT(raw string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == '.' {
			parts = append(parts, raw[start:i])
			start = i + 1
		}
	}
	parts = append(parts, raw[start:])
	return parts
}
