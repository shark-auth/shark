package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
)

var githubEndpoint = oauth2.Endpoint{
	AuthURL:  "https://github.com/login/oauth/authorize",
	TokenURL: "https://github.com/login/oauth/access_token",
}

const githubUserURL = "https://api.github.com/user"

// GitHub implements the OAuthProvider interface for GitHub.
type GitHub struct {
	cfg oauth2.Config
}

// NewGitHub creates a GitHub OAuth provider from config.
func NewGitHub(c config.GitHubConfig, baseURL string) *GitHub {
	return &GitHub{
		cfg: oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     githubEndpoint,
			RedirectURL:  baseURL + "/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"user:email"},
		},
	}
}

// NewGitHubWithConfig creates a GitHub OAuth provider with a custom oauth2.Config.
// This is primarily used for testing where you need to override endpoints.
func NewGitHubWithConfig(cfg oauth2.Config) *GitHub {
	return &GitHub{cfg: cfg}
}

func (g *GitHub) Name() string { return "github" }

func (g *GitHub) AuthURL(state string) string {
	return g.cfg.AuthCodeURL(state)
}

func (g *GitHub) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.cfg.Exchange(ctx, code)
}

func (g *GitHub) GetUser(ctx context.Context, token *oauth2.Token) (*auth.OAuthUserInfo, error) {
	client := g.cfg.Client(ctx, token)

	resp, err := client.Get(githubUserURL)
	if err != nil {
		return nil, fmt.Errorf("fetching github user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user api returned status %d", resp.StatusCode)
	}

	var data struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding github user: %w", err)
	}

	name := data.Name
	if name == "" {
		name = data.Login
	}

	return &auth.OAuthUserInfo{
		ProviderID: strconv.Itoa(data.ID),
		Email:      data.Email,
		Name:       name,
		AvatarURL:  data.AvatarURL,
	}, nil
}
