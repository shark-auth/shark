package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
)

//#nosec G101 -- public OAuth 2.0 endpoint URLs, not credentials
var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/api/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",
}

const discordUserURL = "https://discord.com/api/users/@me"

// Discord implements the OAuthProvider interface for Discord.
type Discord struct {
	cfg oauth2.Config
}

// NewDiscord creates a Discord OAuth provider from config.
func NewDiscord(c config.DiscordConfig, baseURL string) *Discord {
	scopes := []string{"identify", "email"}
	if len(c.Scopes) > 0 {
		scopes = c.Scopes
	}
	return &Discord{
		cfg: oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Endpoint:     discordEndpoint,
			RedirectURL:  baseURL + "/api/v1/auth/oauth/discord/callback",
			Scopes:       scopes,
		},
	}
}

func (d *Discord) Name() string { return "discord" }

func (d *Discord) AuthURL(state string) string {
	return d.cfg.AuthCodeURL(state)
}

func (d *Discord) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return d.cfg.Exchange(ctx, code)
}

func (d *Discord) GetUser(ctx context.Context, token *oauth2.Token) (*auth.OAuthUserInfo, error) {
	client := d.cfg.Client(ctx, token)

	resp, err := client.Get(discordUserURL)
	if err != nil {
		return nil, fmt.Errorf("fetching discord user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord user api returned status %d", resp.StatusCode)
	}

	var data struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Email         string `json:"email"`
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding discord user: %w", err)
	}

	name := data.GlobalName
	if name == "" {
		name = data.Username
	}

	var avatarURL string
	if data.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", data.ID, data.Avatar)
	}

	return &auth.OAuthUserInfo{
		ProviderID: data.ID,
		Email:      data.Email,
		Name:       name,
		AvatarURL:  avatarURL,
	}, nil
}
