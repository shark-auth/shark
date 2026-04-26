// Package api — admin-key-authenticated organization management handlers.
//
// The user-facing handlers in `organization_handlers.go` rely on session auth
// + RBAC permission middleware. Dashboard pages send the admin Bearer key, so
// those routes 401 from the admin UI. Rather than add an admin-key bypass to
// the existing routes (which would muddle the permission model), we mount a
// parallel `/admin/organizations/*` group authenticated only by the admin
// API key — same shape as `/admin/sessions`, `/admin/apps`, `/admin/flows`.
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/email"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// --- Create org (admin) ---

type adminCreateOrgRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"` // accepted for forward-compat, not stored
	Metadata    *string `json:"metadata,omitempty"`
}

// handleAdminCreateOrganization handles POST /api/v1/admin/organizations.
// Creates a new org without requiring a session user — the admin key is the
// only credential. No owner membership row is created (admin-managed orgs
// are bootstrapped; owners can be added via POST /admin/organizations/{id}/members
// or by having users join the org later). RBAC builtin roles are seeded if
// the RBAC subsystem is present.
func (s *Server) handleAdminCreateOrganization(w http.ResponseWriter, r *http.Request) {
	var req adminCreateOrgRequest
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
	meta := ""
	if req.Metadata != nil {
		meta = normalizeMetadata(*req.Metadata)
	}
	org := &storage.Organization{
		ID:        "org_" + id,
		Name:      req.Name,
		Slug:      req.Slug,
		Metadata:  meta,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Store.CreateOrganization(r.Context(), org); err != nil {
		internal(w, err)
		return
	}

	// Seed builtin RBAC roles (owner/admin/member). Non-fatal — log on error.
	if s.RBAC != nil {
		if err := s.RBAC.SeedOrgRoles(r.Context(), org.ID); err != nil {
			slog.Warn("admin create org: seed roles failed", "org_id", org.ID, "err", err)
		}
	}

	s.auditAdminOrg(r, "admin.organization.create", org.ID, map[string]any{
		"org_name": org.Name,
		"org_slug": org.Slug,
	})
	s.emit(r.Context(), storage.WebhookEventOrgCreated, map[string]any{
		"id": org.ID, "name": org.Name, "slug": org.Slug,
	})

	writeJSON(w, http.StatusCreated, orgToResponse(org))
}

// --- Update org (admin) ---

type adminUpdateOrgRequest struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"` // not stored; accepted for forward-compat
	Metadata    *string `json:"metadata,omitempty"`
}

// handleAdminUpdateOrganization handles PATCH /api/v1/admin/organizations/{id}.
// Body fields are optional; omitted fields keep their existing values.
func (s *Server) handleAdminUpdateOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")

	var req adminUpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
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

	// Track which fields the request actually mutated so the audit row can
	// surface a `changed_fields` array instead of a generic update event.
	changedFields := []string{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name != "" && name != org.Name {
			org.Name = name
			changedFields = append(changedFields, "name")
		}
	}
	if req.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*req.Slug))
		if slug != "" {
			if !slugRE.MatchString(slug) {
				writeJSON(w, http.StatusBadRequest, errPayload("invalid_slug", "Slug must be 3-64 chars, lowercase a-z, 0-9, hyphens, no leading/trailing hyphen"))
				return
			}
			// Reject duplicate slug unless it belongs to this org already.
			if other, err := s.Store.GetOrganizationBySlug(r.Context(), slug); err == nil && other.ID != org.ID {
				writeJSON(w, http.StatusConflict, errPayload("slug_taken", "An organization with this slug already exists"))
				return
			}
			if slug != org.Slug {
				org.Slug = slug
				changedFields = append(changedFields, "slug")
			}
		}
	}
	if req.Metadata != nil {
		newMeta := normalizeMetadata(*req.Metadata)
		if newMeta != org.Metadata {
			org.Metadata = newMeta
			changedFields = append(changedFields, "metadata")
		}
	}
	org.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := s.Store.UpdateOrganization(r.Context(), org); err != nil {
		internal(w, err)
		return
	}
	s.auditAdminOrg(r, "admin.organization.update", orgID, map[string]any{
		"changed_fields": changedFields,
	})
	writeJSON(w, http.StatusOK, orgToResponse(org))
}

// handleAdminDeleteOrganization handles DELETE /api/v1/admin/organizations/{id}.
// Cascade is the storage layer's responsibility (FK ON DELETE CASCADE).
func (s *Server) handleAdminDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")

	// Verify exists for a clean 404 (DeleteOrganization is a no-op for missing rows).
	org, err := s.Store.GetOrganizationByID(r.Context(), orgID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	} else if err != nil {
		internal(w, err)
		return
	}

	// Snapshot member count BEFORE delete so the audit row records the
	// blast radius. Best-effort: a failure here is non-fatal.
	memberCount := 0
	if members, mErr := s.Store.ListOrganizationMembers(r.Context(), orgID); mErr == nil {
		memberCount = len(members)
	}

	if err := s.Store.DeleteOrganization(r.Context(), orgID); err != nil {
		internal(w, err)
		return
	}
	s.auditAdminOrg(r, "admin.organization.delete", orgID, map[string]any{
		"org_name":     org.Name,
		"member_count": memberCount,
	})
	s.emit(r.Context(), storage.WebhookEventOrgDeleted, map[string]any{"id": orgID})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Organization deleted"})
}

// --- Org RBAC (admin) ---

// handleAdminCreateOrgRole handles POST /api/v1/admin/organizations/{id}/roles.
// Mirrors the user-facing handleCreateOrgRole but skips RBAC permission
// middleware — admin key is full-access by design.
func (s *Server) handleAdminCreateOrgRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")

	// Sanity-check the org exists so callers don't end up with a role row
	// pointing at a missing parent (FK would catch it, but explicit 404 reads
	// better than the generic internal_error fallback).
	if _, err := s.Store.GetOrganizationByID(r.Context(), orgID); errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	} else if err != nil {
		internal(w, err)
		return
	}

	var req createOrgRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Name is required"))
		return
	}

	role, err := s.RBAC.CreateOrgRole(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		internal(w, err)
		return
	}
	// Permission attachment is a separate API call, so on creation the
	// permissions count is always 0. Surface it explicitly so the audit row
	// is self-describing rather than implying we just couldn't be bothered.
	s.auditAdminOrg(r, "admin.organization.role.create", orgID, map[string]any{
		"role_name":         role.Name,
		"role_id":           role.ID,
		"permissions_count": 0,
	})
	writeJSON(w, http.StatusCreated, orgRoleToResponse(role))
}

// handleAdminListOrgRoles handles GET /api/v1/admin/organizations/{id}/roles.
// Returns all org-scoped RBAC roles (builtin and custom) for the given org.
func (s *Server) handleAdminListOrgRoles(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")

	if _, err := s.Store.GetOrganizationByID(r.Context(), orgID); errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	} else if err != nil {
		internal(w, err)
		return
	}

	roles, err := s.Store.GetOrgRolesByOrgID(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]orgRoleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, orgRoleToResponse(role))
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": out})
}

// --- Invitations (admin) ---

// handleAdminListOrgInvitations handles GET /api/v1/admin/organizations/{id}/invitations.
// Returns pending (non-expired, non-accepted) invitations for the given org.
func (s *Server) handleAdminListOrgInvitations(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")

	if _, err := s.Store.GetOrganizationByID(r.Context(), orgID); errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Organization not found"))
		return
	} else if err != nil {
		internal(w, err)
		return
	}

	all, err := s.Store.ListOrganizationInvitationsByOrgID(r.Context(), orgID)
	if err != nil {
		internal(w, err)
		return
	}

	now := time.Now().UTC()
	pending := make([]*storage.OrganizationInvitation, 0, len(all))
	for _, inv := range all {
		if inv.AcceptedAt != nil {
			continue
		}
		exp, err := time.Parse(time.RFC3339, inv.ExpiresAt)
		if err == nil && exp.Before(now) {
			continue
		}
		pending = append(pending, inv)
	}
	writeJSON(w, http.StatusOK, map[string]any{"invitations": pending})
}

// handleAdminDeleteOrgInvitation handles
// DELETE /api/v1/admin/organizations/{id}/invitations/{invitationId}.
// Removes the row entirely; outstanding email links become 404 on accept.
func (s *Server) handleAdminDeleteOrgInvitation(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	invID := chi.URLParam(r, "invitationId")

	inv, err := s.Store.GetOrganizationInvitationByID(r.Context(), invID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Invitation not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if inv.OrganizationID != orgID {
		// URL mismatch — don't let an admin nuke another org's invitation
		// via a crafted URL.
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Invitation not found"))
		return
	}

	if err := s.Store.DeleteOrganizationInvitation(r.Context(), invID); err != nil {
		internal(w, err)
		return
	}
	s.auditAdminOrg(r, "admin.organization.invitation.delete", orgID, map[string]any{
		"invitation_id": invID,
		"email":         inv.Email,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation deleted"})
}

// handleAdminResendOrgInvitation handles
// POST /api/v1/admin/organizations/{id}/invitations/{invitationId}/resend.
// Rotates the invitation token (so any old link stops working), bumps expiry,
// and re-emails the same address.
func (s *Server) handleAdminResendOrgInvitation(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "id")
	invID := chi.URLParam(r, "invitationId")

	inv, err := s.Store.GetOrganizationInvitationByID(r.Context(), invID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Invitation not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if inv.OrganizationID != orgID {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Invitation not found"))
		return
	}
	if inv.AcceptedAt != nil {
		writeJSON(w, http.StatusConflict, errPayload("invitation_used", "Invitation already accepted"))
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
	expires := time.Now().UTC().Add(72 * time.Hour).Format(time.RFC3339)
	if err := s.Store.UpdateOrganizationInvitationToken(r.Context(), invID, tokenHash, expires); err != nil {
		internal(w, err)
		return
	}
	inv.TokenHash = tokenHash
	inv.ExpiresAt = expires

	emailSent := false
	if s.MagicLinkManager != nil && s.emailSender() != nil {
		// Synchronous send: dev provider writes to the inbox so smoke can
		// observe it, and admin gets immediate feedback when SMTP is
		// misconfigured rather than a silent fire-and-forget swallow.
		s.sendAdminInvitationResend(r.Context(), inv, org, rawToken)
		emailSent = true
	}

	s.auditAdminOrg(r, "admin.organization.invitation.resend", orgID, map[string]any{
		"invitation_id": invID,
		"email":         inv.Email,
	})
	resp := map[string]any{
		"message":    "Invitation resent",
		"id":         inv.ID,
		"email":      inv.Email,
		"expires_at": inv.ExpiresAt,
		"email_sent": emailSent,
	}
	writeJSON(w, http.StatusOK, resp)
}

// sendAdminInvitationResend mirrors sendOrgInvitationEmail but runs in-line so
// the admin endpoint can report whether the send succeeded. We still cap the
// SMTP attempt at 15s to keep the request responsive.
func (s *Server) sendAdminInvitationResend(parent context.Context, inv *storage.OrganizationInvitation, org *storage.Organization, rawToken string) {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()

	var inviter *storage.User
	if inv.InvitedBy != nil {
		inviter, _ = s.Store.GetUserByID(ctx, *inv.InvitedBy)
	}
	acceptURL := fmt.Sprintf("%s/organizations/invitations/%s/accept",
		strings.TrimRight(s.Config.Server.BaseURL, "/"), rawToken)

	branding, _ := s.Store.ResolveBranding(ctx, "")
	rendered, err := email.RenderOrganizationInvitation(ctx, s.Store, branding, email.OrganizationInvitationData{
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
			Subject: rendered.Subject,
			HTML:    rendered.HTML,
		})
	}
}

// handleAdminRemoveOrgMember handles
// DELETE /api/v1/admin/organizations/{id}/members/{uid}.
// Admin override of the user-facing remove-member flow — no RBAC permission
// check required (admin key is full-access). Prevents removing the last owner.
func (s *Server) handleAdminRemoveOrgMember(w http.ResponseWriter, r *http.Request) {
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
	// Best-effort lookup of the member's email for the audit row — the
	// membership record only holds user_id, so a user delete after this
	// point would erase the email forever otherwise.
	memberEmail := ""
	if u, uErr := s.Store.GetUserByID(r.Context(), targetUserID); uErr == nil && u != nil {
		memberEmail = u.Email
	}
	if err := s.Store.DeleteOrganizationMember(r.Context(), orgID, targetUserID); err != nil {
		internal(w, err)
		return
	}
	s.auditAdminOrg(r, "admin.organization.member.remove", orgID, map[string]any{
		"member_id":    targetUserID,
		"member_email": memberEmail,
	})
	writeJSON(w, http.StatusOK, map[string]string{"message": "Member removed"})
}

// auditAdminOrg writes an admin-actor audit log row with structured metadata.
// ActorID is hardcoded to "admin_key" — admin-key auth doesn't carry a per-user
// identity, but tagging it explicitly lets dashboard filters distinguish
// admin-driven events from user/session/agent actor types. The metadata map
// is per-action (e.g. {org_name, org_slug} for create, {changed_fields: [...]}
// for update) so the audit table can render diff fields without re-parsing
// the action string.
func (s *Server) auditAdminOrg(r *http.Request, action, orgID string, meta map[string]any) {
	if s.AuditLogger == nil {
		return
	}
	metaJSON := []byte("{}")
	if meta != nil {
		if b, err := json.Marshal(meta); err == nil {
			metaJSON = b
		}
	}
	_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
		ActorType:  "admin",
		ActorID:    "admin_key",
		Action:     action,
		TargetType: "organization",
		TargetID:   orgID,
		IP:         ipOf(r),
		UserAgent:  uaOf(r),
		Status:     "success",
		Metadata:   string(metaJSON),
	})
}

