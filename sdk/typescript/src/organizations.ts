/**
 * Organization management — admin API.
 *
 * Wraps `/api/v1/admin/organizations` plus the user-facing
 * `/api/v1/organizations/invitations/{token}/accept` route.
 *
 * Backend reference: `internal/api/router.go`.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Organization row as returned by the server. */
export interface Organization {
  id: string;
  name: string;
  slug?: string;
  /** Server stores metadata as a JSON-encoded string. */
  metadata?: string | Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

/** Organization member row. */
export interface OrgMember {
  id?: string;
  user_id: string;
  org_id?: string;
  role: string;
  email?: string;
  created_at?: string;
  [key: string]: unknown;
}

/** Organization invitation row. */
export interface OrgInvitation {
  id: string;
  org_id?: string;
  email: string;
  role: string;
  token?: string;
  expires_at?: string;
  created_at?: string;
  [key: string]: unknown;
}

/** Input for {@link OrganizationsClient.create}. */
export interface CreateOrganizationInput {
  name: string;
  slug?: string;
  /** dict will be auto-`JSON.stringify()`-ed; string passed through. */
  metadata?: string | Record<string, unknown>;
  [key: string]: unknown;
}

/** Input for {@link OrganizationsClient.update}. */
export interface UpdateOrganizationInput {
  name?: string;
  slug?: string;
  metadata?: string | Record<string, unknown>;
  [key: string]: unknown;
}

/** Options for {@link OrganizationsClient}. */
export interface OrganizationsClientOptions {
  baseUrl: string;
  adminKey: string;
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Admin client for managing organizations, members, and invitations. */
export class OrganizationsClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: OrganizationsClientOptions) {
    this._base = opts.baseUrl.replace(/\/+$/, "");
    this._key = opts.adminKey;
  }

  private _auth(): Record<string, string> {
    return { Authorization: `Bearer ${this._key}` };
  }

  private async _throw(status: number, text: string): Promise<never> {
    let code = "api_error";
    let message = text.slice(0, 300);
    try {
      const body = JSON.parse(text) as {
        error?: { code?: string; message?: string } | string;
        message?: string;
      };
      if (typeof body.error === "object" && body.error) {
        if (body.error.code) code = body.error.code;
        if (body.error.message) message = body.error.message;
      } else if (typeof body.error === "string") {
        message = body.error;
      } else if (body.message) {
        message = body.message;
      }
    } catch {
      // keep defaults
    }
    throw new SharkAPIError(message, code, status);
  }

  private static _unwrap<T>(body: unknown): T {
    if (
      body &&
      typeof body === "object" &&
      "data" in (body as Record<string, unknown>) &&
      Object.keys(body as Record<string, unknown>).length <= 2
    ) {
      return (body as { data: T }).data;
    }
    return body as T;
  }

  /** dict input → JSON-encoded string; string passed through. */
  private static _coerceMetadata(value: unknown): string | undefined {
    if (value === undefined || value === null) return undefined;
    if (typeof value === "string") return value;
    return JSON.stringify(value);
  }

  // --------------------------------------------------------------- CRUD

  /** Create an organization (POST /api/v1/admin/organizations). */
  async create(
    name: string,
    slug?: string,
    extra: Record<string, unknown> = {},
  ): Promise<Organization> {
    const body: Record<string, unknown> = { name, ...extra };
    if (slug !== undefined) body.slug = slug;
    if ("metadata" in body) {
      body.metadata = OrganizationsClient._coerceMetadata(body.metadata);
    }
    const url = `${this._base}/api/v1/admin/organizations`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<Organization>(resp.json());
  }

  /** List organizations. */
  async list(): Promise<Organization[]> {
    const url = `${this._base}/api/v1/admin/organizations`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<Organization[] | { data?: Organization[] }>();
    if (Array.isArray(body)) return body;
    return body?.data ?? [];
  }

  /** Get a single organization. */
  async get(orgId: string): Promise<Organization> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(orgId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<Organization>(resp.json());
  }

  /** Update an organization (PATCH). */
  async update(orgId: string, fields: UpdateOrganizationInput): Promise<Organization> {
    const body: Record<string, unknown> = { ...fields };
    if ("metadata" in body) {
      body.metadata = OrganizationsClient._coerceMetadata(body.metadata);
    }
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(orgId)}`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<Organization>(resp.json());
  }

  /** Delete an organization. */
  async delete(orgId: string): Promise<void> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(orgId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ------------------------------------------------------------- Members

  /** List members of an organization. */
  async listMembers(orgId: string): Promise<OrgMember[]> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/members`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<OrgMember[] | { data?: OrgMember[]; members?: OrgMember[] }>();
    if (Array.isArray(body)) return body;
    return body?.data ?? body?.members ?? [];
  }

  /** Add a member directly (best-effort — backend may require invitation flow). */
  async addMember(orgId: string, userId: string, role: string): Promise<OrgMember> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/members`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ user_id: userId, role }),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<OrgMember>(resp.json());
  }

  /** Update a member's role (PATCH). */
  async updateMemberRole(
    orgId: string,
    memberId: string,
    role: string,
  ): Promise<OrgMember> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/members/${encodeURIComponent(memberId)}`;
    const resp = await httpRequest(url, {
      method: "PATCH",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ role }),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<OrgMember>(resp.json());
  }

  /** Remove a member from the organization. */
  async removeMember(orgId: string, memberId: string): Promise<void> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/members/${encodeURIComponent(memberId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // --------------------------------------------------------- Invitations

  /** List invitations for an organization. */
  async listInvitations(orgId: string): Promise<OrgInvitation[]> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/invitations`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<
      OrgInvitation[] | { data?: OrgInvitation[]; invitations?: OrgInvitation[] }
    >();
    if (Array.isArray(body)) return body;
    return body?.data ?? body?.invitations ?? [];
  }

  /** Create a new invitation. */
  async createInvitation(
    orgId: string,
    email: string,
    role: string,
  ): Promise<OrgInvitation> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/invitations`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ email, role }),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<OrgInvitation>(resp.json());
  }

  /** Delete (cancel) an invitation. */
  async deleteInvitation(orgId: string, invitationId: string): Promise<void> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/invitations/${encodeURIComponent(invitationId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  /** Resend an invitation email. */
  async resendInvitation(orgId: string, invitationId: string): Promise<void> {
    const url = `${this._base}/api/v1/admin/organizations/${encodeURIComponent(
      orgId,
    )}/invitations/${encodeURIComponent(invitationId)}/resend`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (![200, 202, 204].includes(resp.status))
      await this._throw(resp.status, resp.text);
  }

  /**
   * Accept an invitation via token.
   *
   * Calls `POST /api/v1/organizations/invitations/{token}/accept` (the
   * user-facing prefix, NOT `/admin/`). Normally session-cookie auth'd;
   * the admin key header is sent for parity.
   */
  async acceptInvitation(token: string): Promise<Record<string, unknown>> {
    const url = `${this._base}/api/v1/organizations/invitations/${encodeURIComponent(
      token,
    )}/accept`;
    const resp = await httpRequest(url, { method: "POST", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return OrganizationsClient._unwrap<Record<string, unknown>>(resp.json());
  }
}
