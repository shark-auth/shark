package oauth

import "context"

// resourceContextKey is the unexported context key used to thread the RFC 8707
// resource indicator through the fosite request lifecycle.
// Fosite's Sanitize() strips unrecognized form params before calling
// CreateAccessTokenSession, so we carry resource via context instead.
type resourceContextKey struct{}

// contextWithResource stores the resource indicator in ctx.
func contextWithResource(ctx context.Context, resource string) context.Context {
	return context.WithValue(ctx, resourceContextKey{}, resource)
}

// resourceFromContext retrieves the resource indicator from ctx.
// Returns an empty string if none was set.
func resourceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(resourceContextKey{}).(string)
	return v
}

// ValidateAudience checks whether expectedResource appears in the token's
// aud claim. Per RFC 8707 the aud claim may be a single string or a slice of
// strings — both forms are handled.
//
// Resource servers should call this when verifying incoming access tokens to
// ensure the token was issued specifically for them.
//
//	ok := ValidateAudience(parsedClaims["aud"], "https://api.example.com")
func ValidateAudience(tokenAud interface{}, expectedResource string) bool {
	if expectedResource == "" {
		return true // no restriction; accept any
	}
	switch v := tokenAud.(type) {
	case string:
		return v == expectedResource
	case []string:
		for _, a := range v {
			if a == expectedResource {
				return true
			}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expectedResource {
				return true
			}
		}
	}
	return false
}
