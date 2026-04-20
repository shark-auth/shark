package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var slugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

// --- Create organization ---

type createOrgRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Metadata string `json:"metadata,omitempty"`
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Metadata  string `json:"metadata"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func orgToResponse(o *storage.Organization) organizationResponse {
	return organizationResponse{
		ID: o.ID, Name: o.Name, Slug: o.Slug, Metadata: o.Metadata,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

func (s *Server) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, errPayload("unauthorized", "No valid session"))
		return
	}

	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_name", "Name is required"))
		return
	}
	if !slugRE.MatchString(req.Slug) {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_slug", "Slug must be 3-64 chars, lowercase a-z, 0-9, hyphens, no leading/trailing hyphen"))
		return
	}
	if _, err := s.Store.GetOrganizationBySlug(r.Context(), req.Slug); err == nil {
		writeJSON(w, http.StatusConflict, errPayload("slug_taken", "An organization with this slug already exists"))
		return
	}

	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	org := &storage.Organization{
		ID: "org_" + id, Name: req.Name, Slug: req.Slug,
		Metadata: normalizeMetadata(req.Metadata),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Store.CreateOrganization(r.Context(), org); err != nil {
		internal(w, err)
		return
	}
	// Creator becomes owner.
	if err := s.Store.CreateOrganizationMember(r.Context(), &storage.OrganizationMember{
		OrganizationID: org.ID, UserID: userID,
		Role: storage.OrgRoleOwner, JoinedAt: now,
	}); err != nil {
		internal(w, err)
		return
	}

	// Seed 3 builtin org roles and grant the owner role to the creator.
	// On failure: compensate by deleting the org (user can retry).
	if s.RBAC != nil {
		if err := s.RBAC.SeedOrgRoles(r.Context(), org.ID); err != nil {
			_ = s.Store.DeleteOrganization(r.Context(), org.ID)
			internal(w, err)
			return
		}
		ownerRole, err := s.Store.GetOrgRoleByName(r.Context(), org.ID, "owner")
		if err == nil {
			if grantErr := s.RBAC.GrantOrgRole(r.Context(), org.ID, userID, ownerRole.ID, userID); grantErr != nil {
				slog.Warn("org: grant owner role failed", "org_id", org.ID, "user_id", userID, "err", grantErr)
			}
		}
	}

	s.auditOrg(r.Context(), userID, "organization.create", org.ID, ipOf(r), uaOf(r))
	s.emit(r.Context(), storage.WebhookEventOrgCreated, map[string]any{
		"id": org.ID, "name": org.Name, "slug": org.Slug, "created_by": userID,
	})

	writeJSON(w, http.StatusCreated, orgToResponse(org))
}

// --- List user's orgs ---

func (s *Server) handleListMyOrganizations(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, errPayload("unauthorized", "No valid session"))
		return
	}
	orgs, err := s.Store.ListOrganizationsByUserID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]organizationResponse, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, orgToResponse(o))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

// --- Get / Update / Delete ---

func (s *Server) handleGetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	userID := mw.GetUserID(r.Context())
	if !s.isOrgMember(r.Context(), orgID, userID) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	}
	org, err := s.Store.GetOrganizationByID(r.Context(), orgID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orgToResponse(org))
}

type updateOrgRequest struct {
	Name     *string `json:"name,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

func (s *Server) handleUpdateOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	org, err := s.Store.GetOrganizationByID(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name != "" {
			org.Name = name
		}
	}
	if req.Metadata != nil {
		org.Metadata = normalizeMetadata(*req.Metadata)
	}
	org.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.Store.UpdateOrganization(r.Context(), org); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orgToResponse(org))
}

func (s *Server) handleDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	userID := mw.GetUserID(r.Context())
	if err := s.Store.DeleteOrganization(r.Context(), orgID); err != nil {
		internal(w, err)
		return
	}
	s.auditOrg(r.Context(), userID, "organization.delete", orgID, ipOf(r), uaOf(r))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Organization deleted"})
}

// --- Members ---

type memberResponse struct {
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name,omitempty"`
	Role      string `json:"role"`
	JoinedAt  string `json:"joined_at"`
}

func (s *Server) handleListOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	userID := mw.GetUserID(r.Context())
	if !s.isOrgMember(r.Context(), orgID, userID) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	}
	members, err := s.Store.ListOrganizationMembers(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]memberResponse, 0, len(members))
	for _, m := range members {
		out = append(out, memberResponse{
			UserID: m.UserID, UserEmail: m.UserEmail, UserName: m.UserName,
			Role: m.Role, JoinedAt: m.JoinedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) handleUpdateOrganizationMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "uid")
	actor := mw.GetUserID(r.Context())

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if !isValidOrgRole(req.Role) {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_role", "Role must be owner, admin, or member"))
		return
	}

	target, err := s.Store.GetOrganizationMember(r.Context(), orgID, targetUserID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Member not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	// Demoting the last owner is forbidden — use transfer-ownership flow instead.
	if target.Role == storage.OrgRoleOwner && req.Role != storage.OrgRoleOwner {
		if err := s.refuseIfLastOwner(r.Context(), orgID); err != nil {
			writeJSON(w, http.StatusConflict, errPayload("last_owner", err.Error()))
			return
		}
	}
	oldRole := target.Role
	if err := s.Store.UpdateOrganizationMemberRole(r.Context(), orgID, targetUserID, req.Role); err != nil {
		internal(w, err)
		return
	}

	// Sync org RBAC role assignments (best-effort — do not revert the membership change on failure).
	if s.RBAC != nil && oldRole != req.Role {
		if oldOrgRole, err := s.Store.GetOrgRoleByName(r.Context(), orgID, oldRole); err == nil {
			if revokeErr := s.RBAC.RevokeOrgRole(r.Context(), orgID, targetUserID, oldOrgRole.ID); revokeErr != nil {
				slog.Warn("org: revoke old org role failed", "org_id", orgID, "user_id", targetUserID, "err", revokeErr)
			}
		}
		if newOrgRole, err := s.Store.GetOrgRoleByName(r.Context(), orgID, req.Role); err == nil {
			if grantErr := s.RBAC.GrantOrgRole(r.Context(), orgID, targetUserID, newOrgRole.ID, actor); grantErr != nil {
				slog.Warn("org: grant new org role failed", "org_id", orgID, "user_id", targetUserID, "err", grantErr)
			}
		}
	}

	// Audit: role change within organization member.
	s.auditOrg(r.Context(), actor, "org.member.role_update", orgID, ipOf(r), uaOf(r))

	writeJSON(w, http.StatusOK, map[string]string{"message": "Role updated"})
}

func (s *Server) handleRemoveOrganizationMember(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	targetUserID := chi.URLParam(r, "uid")

	target, err := s.Store.GetOrganizationMember(r.Context(), orgID, targetUserID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Member not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if target.Role == storage.OrgRoleOwner {
		if err := s.refuseIfLastOwner(r.Context(), orgID); err != nil {
			writeJSON(w, http.StatusConflict, errPayload("last_owner", err.Error()))
			return
		}
	}
	if err := s.Store.DeleteOrganizationMember(r.Context(), orgID, targetUserID); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Member removed"})
}

// --- Invitations ---

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type inviteResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// handleCreateOrgInvitation persists an invitation and sends the accept email.
// Token plaintext only lives in memory during this request — DB stores the
// SHA-256 hash so leaks can't reveal valid invite links.
func (s *Server) handleCreateOrgInvitation(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	actor := mw.GetUserID(r.Context())

	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !emailRegex.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_email", "Invalid email address"))
		return
	}
	if req.Role == "" {
		req.Role = storage.OrgRoleMember
	}
	if !isValidOrgRole(req.Role) {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_role", "Role must be owner, admin, or member"))
		return
	}

	org, err := s.Store.GetOrganizationByID(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}

	rawToken, tokenHash, err := newInvitationToken()
	if err != nil {
		internal(w, err)
		return
	}

	id, _ := gonanoid.New()
	now := time.Now().UTC()
	expires := now.Add(72 * time.Hour)
	invitedBy := actor
	inv := &storage.OrganizationInvitation{
		ID: "inv_" + id, OrganizationID: orgID,
		Email: req.Email, Role: req.Role, TokenHash: tokenHash,
		InvitedBy: &invitedBy,
		ExpiresAt: expires.Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}
	if err := s.Store.CreateOrganizationInvitation(r.Context(), inv); err != nil {
		internal(w, err)
		return
	}

	// Fire-and-forget: best effort send. If email fails we still return 201
	// because the invitation exists and the admin can resend via a future
	// endpoint. Log it.
	if s.MagicLinkManager != nil {
		go s.sendOrgInvitationEmail(inv, org, actor, rawToken) //#nosec G118 -- fire-and-forget invitation email; bounded via WithTimeout inside the callee
	}

	s.auditOrg(r.Context(), actor, "organization.invitation.create", orgID, ipOf(r), uaOf(r))

	writeJSON(w, http.StatusCreated, inviteResponse{
		ID: inv.ID, Email: inv.Email, Role: inv.Role,
		ExpiresAt: inv.ExpiresAt, CreatedAt: inv.CreatedAt,
	})
}

// handleAcceptOrgInvitation consumes a raw token, links the caller's user to
// the org with the invited role. Requires a session so there's a user to link.
func (s *Server) handleAcceptOrgInvitation(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, errPayload("unauthorized", "Sign in before accepting invitation"))
		return
	}
	rawToken := chi.URLParam(r, "token")
	if rawToken == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Token is required"))
		return
	}

	hash := hashInvitationToken(rawToken)
	inv, err := s.Store.GetOrganizationInvitationByTokenHash(r.Context(), hash)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("invitation_not_found", "Invitation not found or already used"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if inv.AcceptedAt != nil {
		writeJSON(w, http.StatusConflict, errPayload("invitation_used", "Invitation already accepted"))
		return
	}
	expires, _ := time.Parse(time.RFC3339, inv.ExpiresAt)
	if time.Now().UTC().After(expires) {
		writeJSON(w, http.StatusGone, errPayload("invitation_expired", "Invitation has expired"))
		return
	}

	// Match is on email to stop token sharing across accounts.
	me, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}
	if !strings.EqualFold(me.Email, inv.Email) {
		writeJSON(w, http.StatusForbidden, errPayload("email_mismatch", "This invitation was sent to a different email"))
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	// Ignore duplicate membership error — idempotent accept.
	_ = s.Store.CreateOrganizationMember(r.Context(), &storage.OrganizationMember{
		OrganizationID: inv.OrganizationID, UserID: userID,
		Role: inv.Role, JoinedAt: now,
	})
	if err := s.Store.MarkOrganizationInvitationAccepted(r.Context(), inv.ID, now); err != nil {
		internal(w, err)
		return
	}
	s.auditOrg(r.Context(), userID, "organization.invitation.accept", inv.OrganizationID, ipOf(r), uaOf(r))
	s.emit(r.Context(), storage.WebhookEventOrgMemberAdded, map[string]any{
		"organization_id": inv.OrganizationID,
		"user_id":         userID,
		"role":            inv.Role,
		"via":             "invitation",
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation accepted", "organization_id": inv.OrganizationID})
}

// --- Helpers ---

func (s *Server) isOrgMember(ctx context.Context, orgID, userID string) bool {
	if userID == "" {
		return false
	}
	_, err := s.Store.GetOrganizationMember(ctx, orgID, userID)
	return err == nil
}

func (s *Server) refuseIfLastOwner(ctx context.Context, orgID string) error {
	members, err := s.Store.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return err
	}
	owners := 0
	for _, m := range members {
		if m.Role == storage.OrgRoleOwner {
			owners++
		}
	}
	if owners <= 1 {
		return fmt.Errorf("organization must have at least one owner")
	}
	return nil
}

func (s *Server) sendOrgInvitationEmail(inv *storage.OrganizationInvitation, org *storage.Organization, inviterID, rawToken string) {
	// Detached from the request context (fire-and-forget email send), bounded
	// to 15s so the goroutine can't leak on shutdown or slow SMTP.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	inviter, _ := s.Store.GetUserByID(ctx, inviterID)
	acceptURL := fmt.Sprintf("%s/organizations/invitations/%s/accept",
		strings.TrimRight(s.Config.Server.BaseURL, "/"), rawToken)
	html, err := email.RenderOrganizationInvitation(email.OrganizationInvitationData{
		AppName:      s.Config.MFA.Issuer,
		OrgName:      org.Name,
		Role:         inv.Role,
		AcceptURL:    acceptURL,
		InviterEmail: inviterEmailOrEmpty(inviter),
		ExpiryHours:  72,
	})
	if err != nil {
		return
	}
	if sender := s.emailSender(); sender != nil {
		_ = sender.Send(&email.Message{
			To:      inv.Email,
			Subject: fmt.Sprintf("You're invited to %s", org.Name),
			HTML:    html,
		})
	}
}

func inviterEmailOrEmpty(u *storage.User) string {
	if u == nil {
		return ""
	}
	return u.Email
}

func (s *Server) auditOrg(ctx context.Context, actor, action, orgID, ip, ua string) {
	if s.AuditLogger == nil {
		return
	}
	_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
		ActorID: actor, ActorType: "user",
		Action: action, TargetType: "organization", TargetID: orgID,
		IP: ip, UserAgent: ua, Status: "success",
	})
}

// emailSender reaches into the MagicLinkManager which already holds the wired
// Sender. Returns nil if email is not configured — callers treat as no-op.
func (s *Server) emailSender() email.Sender {
	if s.MagicLinkManager == nil {
		return nil
	}
	return s.MagicLinkManager.Sender()
}

func isValidOrgRole(r string) bool {
	return r == storage.OrgRoleOwner || r == storage.OrgRoleAdmin || r == storage.OrgRoleMember
}

func newInvitationToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	hash = hashInvitationToken(raw)
	return raw, hash, nil
}

func hashInvitationToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func normalizeMetadata(m string) string {
	m = strings.TrimSpace(m)
	if m == "" {
		return "{}"
	}
	return m
}

func errPayload(code, msg string) map[string]string {
	return map[string]string{"error": code, "message": msg}
}
