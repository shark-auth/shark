package sso

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// GenerateSPMetadata returns SAML SP metadata XML for the given connection.
// This is used by IdP administrators to configure their side of the connection.
func (s *SSOManager) GenerateSPMetadata(ctx context.Context, connectionID string) ([]byte, error) {
	conn, err := s.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	if conn.Type != "saml" {
		return nil, fmt.Errorf("connection %q is not SAML (type=%s)", connectionID, conn.Type)
	}

	sp, err := s.buildSAMLSP(conn)
	if err != nil {
		return nil, fmt.Errorf("build saml sp: %w", err)
	}

	metadata := sp.ServiceProvider.Metadata()
	data, err := xml.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}
	return data, nil
}

// HandleSAMLACS processes a SAML assertion from the IdP, creates/links
// the user, and creates a session.
func (s *SSOManager) HandleSAMLACS(ctx context.Context, connectionID string, r *http.Request) (*storage.User, *storage.Session, error) {
	conn, err := s.GetConnection(ctx, connectionID)
	if err != nil {
		return nil, nil, fmt.Errorf("get connection: %w", err)
	}
	if conn.Type != "saml" {
		return nil, nil, fmt.Errorf("connection %q is not SAML", connectionID)
	}
	if !conn.Enabled {
		return nil, nil, fmt.Errorf("connection %q is disabled", connectionID)
	}

	sp, err := s.buildSAMLSP(conn)
	if err != nil {
		return nil, nil, fmt.Errorf("build saml sp: %w", err)
	}

	// Parse the SAML response
	if err := r.ParseForm(); err != nil {
		return nil, nil, fmt.Errorf("parse form: %w", err)
	}

	assertion, err := sp.ServiceProvider.ParseResponse(r, []string{""})
	if err != nil {
		return nil, nil, fmt.Errorf("parse saml response: %w", err)
	}

	// Extract attributes from assertion
	attrs := extractSAMLAttributes(assertion)
	sub := attrs["sub"]
	if sub == "" {
		// Fall back to NameID
		if assertion.Subject != nil && assertion.Subject.NameID != nil {
			sub = assertion.Subject.NameID.Value
		}
	}
	if sub == "" {
		return nil, nil, fmt.Errorf("SAML assertion missing subject identifier")
	}

	email := attrs["email"]
	if email == "" {
		// Try common attribute names
		for _, key := range []string{
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"urn:oid:0.9.2342.19200300.100.1.3",
			"mail",
		} {
			if v := attrs[key]; v != "" {
				email = v
				break
			}
		}
	}
	if email == "" {
		return nil, nil, fmt.Errorf("SAML assertion missing email attribute")
	}

	name := attrs["name"]
	if name == "" {
		for _, key := range []string{
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"urn:oid:2.16.840.1.113730.3.1.241",
			"displayName",
		} {
			if v := attrs[key]; v != "" {
				name = v
				break
			}
		}
	}

	user, err := s.findOrCreateUser(ctx, connectionID, sub, email, name)
	if err != nil {
		return nil, nil, fmt.Errorf("find or create user: %w", err)
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()
	session, err := s.sessions.CreateSession(ctx, user.ID, ip, ua, "sso")
	if err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

// buildSAMLSP creates a crewjam/saml service provider from connection config.
func (s *SSOManager) buildSAMLSP(conn *storage.SSOConnection) (*samlsp.Middleware, error) {
	entityID := derefStr(conn.SAMLSPEntityID)
	if entityID == "" {
		entityID = s.cfg.SSO.SAML.SPEntityID
	}
	if entityID == "" {
		entityID = s.cfg.Server.BaseURL
	}

	acsURL := derefStr(conn.SAMLSPAcsURL)
	if acsURL == "" {
		acsURL = fmt.Sprintf("%s/api/v1/sso/saml/%s/acs", s.cfg.Server.BaseURL, conn.ID)
	}

	parsedACS, err := url.Parse(acsURL)
	if err != nil {
		return nil, fmt.Errorf("parse acs url: %w", err)
	}

	parsedRoot, err := url.Parse(s.cfg.Server.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}

	idpURL := derefStr(conn.SAMLIdPURL)
	idpMetadataURL, err := url.Parse(idpURL)
	if err != nil {
		return nil, fmt.Errorf("parse idp url: %w", err)
	}

	sp := saml.ServiceProvider{
		EntityID:          entityID,
		AcsURL:            *parsedACS,
		MetadataURL:       *parsedRoot,
		IDPMetadata:       &saml.EntityDescriptor{},
		AllowIDPInitiated: true,
	}

	// Parse IdP certificate if provided
	idpCert := derefStr(conn.SAMLIdPCert)
	if idpCert != "" {
		block, _ := pem.Decode([]byte(idpCert))
		if block != nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse idp certificate: %w", err)
			}

			sp.IDPMetadata = &saml.EntityDescriptor{
				EntityID: idpMetadataURL.String(),
				IDPSSODescriptors: []saml.IDPSSODescriptor{
					{
						SSODescriptor: saml.SSODescriptor{
							RoleDescriptor: saml.RoleDescriptor{
								KeyDescriptors: []saml.KeyDescriptor{
									{
										Use: "signing",
										KeyInfo: saml.KeyInfo{
											X509Data: saml.X509Data{
												X509Certificates: []saml.X509Certificate{
													{Data: base64.StdEncoding.EncodeToString(cert.Raw)},
												},
											},
										},
									},
								},
							},
						},
						SingleSignOnServices: []saml.Endpoint{
							{
								Binding:  saml.HTTPRedirectBinding,
								Location: idpURL,
							},
						},
					},
				},
			}
		}
	}

	middleware := &samlsp.Middleware{
		ServiceProvider: sp,
	}

	return middleware, nil
}

// extractSAMLAttributes flattens SAML assertion attributes into a map.
func extractSAMLAttributes(assertion *saml.Assertion) map[string]string {
	attrs := make(map[string]string)
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) > 0 {
				attrs[attr.Name] = attr.Values[0].Value
				// Also store FriendlyName mapping
				if attr.FriendlyName != "" {
					attrs[attr.FriendlyName] = attr.Values[0].Value
				}
			}
		}
	}
	return attrs
}
