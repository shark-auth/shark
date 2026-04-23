// Package oauth — session.go defines the fosite session used for OAuth
// requests. It embeds openid.DefaultSession (so OpenID Connect grants keep
// working) and adds JWT access-token support by implementing the
// oauth2.JWTSessionContainer interface. This lets fosite's DefaultJWTStrategy
// generate RFC 7519 JWT access tokens signed by the same ES256 key advertised
// at /.well-known/jwks.json.
package oauth

import (
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"
)

// SharkSession is fosite's per-request session object. It satisfies:
//
//   - fosite.Session (via embedded openid.DefaultSession)
//   - openid.Session (via embedded openid.DefaultSession)
//   - oauth2.JWTSessionContainer (via JWTClaims + JWTHeader accessors)
//
// JWTClaims here populate JWT access-token claims. IDTokenClaims on the
// embedded DefaultSession still populate ID token claims for OIDC flows.
type SharkSession struct {
	*openid.DefaultSession

	// JWTClaims holds the claims for the JWT access token. Populated per
	// request before the token is signed. Extra carries shark-specific
	// claims like cnf.jkt (DPoP) and act (token-exchange delegation).
	JWTClaims *jwt.JWTClaims

	// JWTHeader holds JWT header values (kid is set on these).
	JWTHeader *jwt.Headers
}

// GetJWTClaims satisfies oauth2.JWTSessionContainer. fosite's DefaultJWTStrategy
// mutates this container (scope, audience, expiry) before signing.
func (s *SharkSession) GetJWTClaims() jwt.JWTClaimsContainer {
	if s.JWTClaims == nil {
		s.JWTClaims = &jwt.JWTClaims{}
	}
	return s.JWTClaims
}

// GetJWTHeader satisfies oauth2.JWTSessionContainer.
func (s *SharkSession) GetJWTHeader() *jwt.Headers {
	if s.JWTHeader == nil {
		s.JWTHeader = &jwt.Headers{}
	}
	return s.JWTHeader
}

// Clone returns a deep copy — required because fosite clones sessions when
// stashing them in its stores. We defer to openid.DefaultSession.Clone() and
// then rebuild the JWT fields shallow-copied from the original.
func (s *SharkSession) Clone() fosite.Session {
	if s == nil {
		return nil
	}
	clone := &SharkSession{}
	if s.DefaultSession != nil {
		if c, ok := s.DefaultSession.Clone().(*openid.DefaultSession); ok {
			clone.DefaultSession = c
		}
	}
	if s.JWTClaims != nil {
		jc := *s.JWTClaims
		// deep-copy Extra so downstream mutations don't leak across clones.
		if s.JWTClaims.Extra != nil {
			jc.Extra = make(map[string]interface{}, len(s.JWTClaims.Extra))
			for k, v := range s.JWTClaims.Extra {
				jc.Extra[k] = v
			}
		}
		if len(s.JWTClaims.Audience) > 0 {
			jc.Audience = append([]string(nil), s.JWTClaims.Audience...)
		}
		if len(s.JWTClaims.Scope) > 0 {
			jc.Scope = append([]string(nil), s.JWTClaims.Scope...)
		}
		clone.JWTClaims = &jc
	}
	if s.JWTHeader != nil {
		h := *s.JWTHeader
		if s.JWTHeader.Extra != nil {
			h.Extra = make(map[string]interface{}, len(s.JWTHeader.Extra))
			for k, v := range s.JWTHeader.Extra {
				h.Extra[k] = v
			}
		}
		clone.JWTHeader = &h
	}
	return clone
}
