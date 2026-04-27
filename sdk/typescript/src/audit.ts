/**
 * Audit log access — admin API.
 *
 * Wraps `/api/v1/audit-logs` (list/get/export) and
 * `/api/v1/admin/audit-logs/purge` (see `internal/api/router.go`).
 */

import { httpRequest } from "./http.js";
import { SharkAPIError } from "./errors.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Audit log event. Open-ended — server shape may vary. */
export interface AuditEvent {
  id?: string;
  action?: string;
  actor_id?: string;
  actor_type?: string;
  target_id?: string;
  target_type?: string;
  status?: string;
  ip?: string;
  created_at?: string;
  metadata?: Record<string, unknown>;
  [key: string]: unknown;
}

/** Filters for {@link AuditClient.list}. */
export interface AuditListFilters {
  actor_id?: string;
  actor_type?: string;
  action?: string;
  target_id?: string;
  target_type?: string;
  /** ISO 8601 lower bound (server may also accept `from`). */
  since?: string;
  /** ISO 8601 upper bound. */
  until?: string;
  limit?: number;
  cursor?: string;
}

/** Result of {@link AuditClient.list}. */
export interface AuditListResult {
  events: AuditEvent[];
  next_cursor?: string | null;
  [key: string]: unknown;
}

/** Filters for {@link AuditClient.export}. */
export interface AuditExportFilters {
  /** ISO 8601 lower bound. Required by backend. */
  since?: string;
  /** ISO 8601 upper bound. Required by backend. */
  until?: string;
  from?: string;
  to?: string;
  action?: string;
  [key: string]: unknown;
}

/** Result of {@link AuditClient.purge}. */
export interface AuditPurgeResult {
  deleted?: number;
  [key: string]: unknown;
}

/** Options for {@link AuditClient}. */
export interface AuditClientOptions {
  baseUrl: string;
  adminKey: string;
}

const PREFIX = "/api/v1/audit-logs";
const PURGE = "/api/v1/admin/audit-logs/purge";
const EXPORT = "/api/v1/audit-logs/export";

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

/** Admin client for querying and exporting audit logs. */
export class AuditClient {
  private readonly _base: string;
  private readonly _key: string;

  constructor(opts: AuditClientOptions) {
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

  /** List audit log events. */
  async list(filters: AuditListFilters = {}): Promise<AuditListResult> {
    const qs = new URLSearchParams();
    qs.set("limit", String(filters.limit ?? 100));
    if (filters.actor_id) qs.set("actor_id", filters.actor_id);
    if (filters.actor_type) qs.set("actor_type", filters.actor_type);
    if (filters.action) qs.set("action", filters.action);
    if (filters.target_id) qs.set("target_id", filters.target_id);
    if (filters.target_type) qs.set("target_type", filters.target_type);
    if (filters.since) qs.set("since", filters.since);
    if (filters.until) qs.set("until", filters.until);
    if (filters.cursor) qs.set("cursor", filters.cursor);

    const url = `${this._base}${PREFIX}?${qs.toString()}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);

    const body = resp.json<unknown>();
    if (Array.isArray(body)) {
      return { events: body as AuditEvent[], next_cursor: null };
    }
    if (body && typeof body === "object") {
      const obj = body as Record<string, unknown>;
      if (Array.isArray(obj.events)) {
        return obj as AuditListResult;
      }
      if (Array.isArray(obj.data)) {
        return {
          events: obj.data as AuditEvent[],
          next_cursor: (obj.next_cursor as string | null | undefined) ?? null,
          ...obj,
        };
      }
    }
    return { events: [], next_cursor: null };
  }

  /** Fetch a single audit event by ID. */
  async get(eventId: string): Promise<AuditEvent> {
    const url = `${this._base}${PREFIX}/${encodeURIComponent(eventId)}`;
    const resp = await httpRequest(url, { headers: this._auth() });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    const body = resp.json<AuditEvent | { data: AuditEvent }>();
    if (
      body &&
      typeof body === "object" &&
      "data" in (body as Record<string, unknown>) &&
      Object.keys(body as Record<string, unknown>).length <= 2
    ) {
      return (body as { data: AuditEvent }).data;
    }
    return body as AuditEvent;
  }

  /**
   * Export audit logs.
   *
   * Backend route: `POST /api/v1/audit-logs/export`. Currently emits CSV;
   * `format` is forwarded for forward compatibility.
   *
   * Returns the raw export body text.
   */
  async export(format: string = "ndjson", filters: AuditExportFilters = {}): Promise<string> {
    const body: Record<string, unknown> = { format, ...filters };
    const url = `${this._base}${EXPORT}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200) await this._throw(resp.status, resp.text);
    return resp.text;
  }

  /**
   * Purge old audit log entries.
   *
   * Backend route: `POST /api/v1/admin/audit-logs/purge`. Deletes entries
   * older than `before` (ISO 8601). Use `dryRun` to count without deleting.
   */
  async purge(before?: string, dryRun: boolean = false): Promise<AuditPurgeResult> {
    const body: Record<string, unknown> = { dry_run: dryRun };
    if (before !== undefined) body.before = before;
    const url = `${this._base}${PURGE}`;
    const resp = await httpRequest(url, {
      method: "POST",
      headers: { ...this._auth(), "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (resp.status !== 200 && resp.status !== 202)
      await this._throw(resp.status, resp.text);
    try {
      const payload = resp.json<unknown>();
      if (
        payload &&
        typeof payload === "object" &&
        "data" in (payload as Record<string, unknown>) &&
        Object.keys(payload as Record<string, unknown>).length <= 2
      ) {
        return (payload as { data: AuditPurgeResult }).data;
      }
      return payload as AuditPurgeResult;
    } catch {
      return { raw: resp.text } as AuditPurgeResult;
    }
  }
}
