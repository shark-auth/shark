/**
 * Role and permission management — admin API.
 *
 * Wraps:
 *   - `/api/v1/roles` — CRUD + permission attach/detach
 *   - `/api/v1/permissions` — list/create/delete
 *   - `/api/v1/users/{user_id}/roles` and
 *     `/api/v1/users/{user_id}/roles/{rid}` — role assign/revoke
 *
 * See `internal/api/router.go` for the canonical route inventory.
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Role row. */
export interface Role {
  id: string;
  name: string;
  description?: string;
  created_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

/** Permission row. */
export interface Permission {
  id: string;
  action: string;
  resource: string;
  created_at?: string;
  [key: string]: unknown;
}

/** Input for {@link RbacClient.updateRole}. */
export interface UpdateRoleInput {
  name?: string;
  description?: string;
  [key: string]: unknown;
}

/** Options for {@link RbacClient}. */
export interface RbacClientOptions {
  baseUrl: string;
  adminKey: string;
}

const ROLES = "/api/v1/roles";
const PERMS = "/api/v1/permissions";
const USERS = "/api/v1/users";

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Admin client for the global RBAC surface. */
export class RbacClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: RbacClientOptions) {
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

  private static _list<T>(body: unknown, ...keys: string[]): T[] {
    if (Array.isArray(body)) return body as T[];
    if (body && typeof body === "object") {
      const obj = body as Record<string, unknown>;
      for (const k of [...keys, "data"]) {
        const v = obj[k];
        if (Array.isArray(v)) return v as T[];
      }
    }
    return [];
  }

  // ------------------------------------------------------------- Roles

  /** GET /api/v1/roles. */
  async listRoles(): Promise<Role[]> {
    const url = `${this._base}${ROLES}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return RbacClient._list<Role>(resp.json(), "roles");
  }

  /** POST /api/v1/roles. */
  async createRole(name: string, description?: string): Promise<Role> {
    const body: Record<string, unknown> = { name };
    if (description !== undefined) body.description = description;
    const url = `${this._base}${ROLES}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return RbacClient._unwrap<Role>(resp.json());
  }

  /** GET /api/v1/roles/{id}. */
  async getRole(roleId: string): Promise<Role> {
    const url = `${this._base}${ROLES}/${encodeURIComponent(roleId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return RbacClient._unwrap<Role>(resp.json());
  }

  /** PUT /api/v1/roles/{id} (backend exposes update as PUT, not PATCH). */
  async updateRole(roleId: string, fields: UpdateRoleInput): Promise<Role> {
    const url = `${this._base}${ROLES}/${encodeURIComponent(roleId)}`;
    const resp = await httpRequest(url, {
      method: "PUT",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(fields),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return RbacClient._unwrap<Role>(resp.json());
  }

  /** DELETE /api/v1/roles/{id}. */
  async deleteRole(roleId: string): Promise<void> {
    const url = `${this._base}${ROLES}/${encodeURIComponent(roleId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // ----------------------------------------------------- Permissions

  /** GET /api/v1/permissions. */
  async listPermissions(): Promise<Permission[]> {
    const url = `${this._base}${PERMS}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return RbacClient._list<Permission>(resp.json(), "permissions");
  }

  /** POST /api/v1/permissions. */
  async createPermission(action: string, resource: string): Promise<Permission> {
    const url = `${this._base}${PERMS}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ action, resource }),
    });
    if (resp.status !== 200 && resp.status !== 201)
      await this._throw(resp.status, resp.text);
    return RbacClient._unwrap<Permission>(resp.json());
  }

  /** POST /api/v1/roles/{role_id}/permissions. */
  async attachPermission(roleId: string, permissionId: string): Promise<void> {
    const url = `${this._base}${ROLES}/${encodeURIComponent(roleId)}/permissions`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ permission_id: permissionId }),
    });
    if (![200, 201, 204].includes(resp.status))
      await this._throw(resp.status, resp.text);
  }

  /** DELETE /api/v1/roles/{role_id}/permissions/{permission_id}. */
  async detachPermission(roleId: string, permissionId: string): Promise<void> {
    const url = `${this._base}${ROLES}/${encodeURIComponent(
      roleId,
    )}/permissions/${encodeURIComponent(permissionId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // --------------------------------------------- User <-> role assignment

  /** POST /api/v1/users/{user_id}/roles. */
  async assignRole(userId: string, roleId: string): Promise<void> {
    const url = `${this._base}${USERS}/${encodeURIComponent(userId)}/roles`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify({ role_id: roleId }),
    });
    if (![200, 201, 204].includes(resp.status))
      await this._throw(resp.status, resp.text);
  }

  /** DELETE /api/v1/users/{user_id}/roles/{role_id}. */
  async revokeRole(userId: string, roleId: string): Promise<void> {
    const url = `${this._base}${USERS}/${encodeURIComponent(
      userId,
    )}/roles/${encodeURIComponent(roleId)}`;
    const resp = await httpRequest(url, { method: "DELETE", headers: this._auth() });
    if (resp.status !== 200 && resp.status !== 204)
      await this._throw(resp.status, resp.text);
  }

  // --------------------------------------------- Read-only helpers

  /** GET /api/v1/users/{user_id}/roles. */
  async listUserRoles(userId: string): Promise<Role[]> {
    const url = `${this._base}${USERS}/${encodeURIComponent(userId)}/roles`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return RbacClient._list<Role>(resp.json(), "roles");
  }

  /** GET /api/v1/users/{user_id}/permissions. */
  async listUserPermissions(userId: string): Promise<Permission[]> {
    const url = `${this._base}${USERS}/${encodeURIComponent(userId)}/permissions`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return RbacClient._list<Permission>(resp.json(), "permissions");
  }
}
