package oauth

import (
	"encoding/json"
	"net/http"
)

// authServerMetadata holds the RFC 8414 Authorization Server Metadata fields.
type authServerMetadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	JwksURI                                    string   `json:"jwks_uri"`
	RegistrationEndpoint                       string   `json:"registration_endpoint"`
	RevocationEndpoint                         string   `json:"revocation_endpoint"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint"`
	DeviceAuthorizationEndpoint                string   `json:"device_authorization_endpoint"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported"`
	ScopesSupported                            []string `json:"scopes_supported"`
	DPoPSigningAlgValuesSupported              []string `json:"dpop_signing_alg_values_supported"`
	ServiceDocumentation                       string   `json:"service_documentation"`
}

// MetadataHandler returns an http.HandlerFunc that serves RFC 8414
// Authorization Server Metadata at /.well-known/oauth-authorization-server.
// This is the MCP discovery entrypoint — MCP clients use this to find
// authorization, token, and registration endpoints.
//
// The JSON payload is marshaled once inside the closure and reused on every
// subsequent request, so there is zero per-request allocation for encoding.
func MetadataHandler(issuer string) http.HandlerFunc {
	meta := authServerMetadata{
		Issuer:                        issuer,
		AuthorizationEndpoint:         issuer + "/oauth/authorize",
		TokenEndpoint:                 issuer + "/oauth/token",
		JwksURI:                       issuer + "/.well-known/jwks.json",
		RegistrationEndpoint:          issuer + "/oauth/register",
		RevocationEndpoint:            issuer + "/oauth/revoke",
		IntrospectionEndpoint:         issuer + "/oauth/introspect",
		DeviceAuthorizationEndpoint:   issuer + "/oauth/device",
		ResponseTypesSupported:        []string{"code"},
		ResponseModesSupported:        []string{"query"},
		GrantTypesSupported: []string{
			"authorization_code",
			"client_credentials",
			"refresh_token",
			"urn:ietf:params:oauth:grant-type:device_code",
			"urn:ietf:params:oauth:grant-type:token-exchange",
		},
		CodeChallengeMethodsSupported: []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_basic",
			"client_secret_post",
			"private_key_jwt",
			"none",
		},
		TokenEndpointAuthSigningAlgValuesSupported: []string{"ES256", "RS256"},
		ScopesSupported: []string{"openid", "profile", "email"},
		DPoPSigningAlgValuesSupported: []string{"ES256", "RS256"},
		ServiceDocumentation:          "https://sharkauth.com/docs",
	}

	payload, err := json.Marshal(meta)
	if err != nil {
		// This can only happen if the struct contains non-serialisable types,
		// which it does not. Panic at startup rather than serve a broken handler.
		panic("oauth: failed to marshal server metadata: " + err.Error())
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}
}
