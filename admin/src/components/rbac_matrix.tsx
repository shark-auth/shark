// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'
import { TeachEmptyState } from './TeachEmptyState'

// PermissionMatrix — grid view: rows = permissions, cols = roles.
// Cell = checkbox indicating whether a role has a permission.
// Optimistic UI with rollback on error. Column header click grants/revokes
// entire column (bulk op, confirmed).

export function PermissionMatrix({ onSwitchToRoles }) {
  const toast = useToast();
  const { data: rolesRaw, loading: loadingRoles, refresh: refreshRoles } = useAPI('/roles');
  const { data: permsRaw, loading: loadingPerms } = useAPI('/permissions');

  const roles = React.useMemo(
    () => rolesRaw?.roles || (Array.isArray(rolesRaw) ? rolesRaw : []),
    [rolesRaw]
  );
  const perms = React.useMemo(() => {
    const raw = permsRaw?.permissions || (Array.isArray(permsRaw) ? permsRaw : []);
    return [...raw].sort((a, b) => {
      const an = `${a.action}·${a.resource}`;
      const bn = `${b.action}·${b.resource}`;
      return an.localeCompare(bn);
    });
  }, [permsRaw]);

  // Fetch each role's permissions once and build a Set of pids per roleId.
  // Stored in state so we can apply optimistic updates locally.
  const [matrix, setMatrix] = React.useState({}); // { [roleId]: Set<pid> }
  const [matrixLoaded, setMatrixLoaded] = React.useState(false);

  React.useEffect(() => {
    if (roles.length === 0) { setMatrixLoaded(true); return; }
    let cancelled = false;
    (async () => {
      const next = {};
      await Promise.all(roles.map(async r => {
        try {
          const res = await API.get('/roles/' + r.id);
          const pList = res?.permissions || res?.role?.permissions || [];
          next[r.id] = new Set(pList.map(p => p.id));
        } catch {
          next[r.id] = new Set();
        }
      }));
      if (!cancelled) { setMatrix(next); setMatrixLoaded(true); }
    })();
    return () => { cancelled = true; };
  }, [roles.map(r => r.id).join(',')]);

  // Batch usage — "used by N users" per permission. Reuse existing batch endpoint.
  const permIds = perms.map(p => p.id);
  const batchPath = permIds.length > 0
    ? '/admin/permissions/batch-usage?ids=' + permIds.join(',')
    : null;
  const { data: usageRaw } = useAPI(batchPath, [permIds.join(',')]);
  const usageMap = usageRaw || {};

  // Filter
  const [filter, setFilter] = React.useState('');
  const filteredPerms = React.useMemo(() => {
    const q = filter.trim().toLowerCase();
    if (!q) return perms;
    return perms.filter(p => {
      const name = `${p.action} ${p.resource}`.toLowerCase();
      return name.includes(q);
    });
  }, [perms, filter]);

  // Pending cells — [roleId, pid] keys for in-flight requests (disable checkbox).
  const [pending, setPending] = React.useState(new Set());
  const cellKey = (rid, pid) => `${rid}:${pid}`;

  async function toggleCell(role, perm) {
    const k = cellKey(role.id, perm.id);
    if (pending.has(k)) return;
    const has = matrix[role.id]?.has(perm.id) ?? false;

    // Optimistic update
    setMatrix(prev => {
      const copy = { ...prev };
      const s = new Set(copy[role.id] || []);
      if (has) s.delete(perm.id); else s.add(perm.id);
      copy[role.id] = s;
      return copy;
    });
    setPending(prev => { const s = new Set(prev); s.add(k); return s; });

    try {
      if (has) {
        await API.del('/roles/' + role.id + '/permissions/' + perm.id);
      } else {
        await API.post('/roles/' + role.id + '/permissions', { permission_id: perm.id });
      }
    } catch (e) {
      // Rollback
      setMatrix(prev => {
        const copy = { ...prev };
        const s = new Set(copy[role.id] || []);
        if (has) s.add(perm.id); else s.delete(perm.id);
        copy[role.id] = s;
        return copy;
      });
      toast.error((has ? 'Revoke' : 'Grant') + ' failed: ' + e.message);
    } finally {
      setPending(prev => { const s = new Set(prev); s.delete(k); return s; });
    }
  }

  async function bulkToggleColumn(role) {
    const rolePerms = matrix[role.id] || new Set();
    // If most/all permissions are already granted, treat as revoke-all.
    const granted = filteredPerms.filter(p => rolePerms.has(p.id)).length;
    const allGranted = granted === filteredPerms.length && filteredPerms.length > 0;

    const action = allGranted ? 'revoke' : 'grant';
    const verb = allGranted ? 'Revoke ALL' : 'Grant ALL';
    const msg = allGranted
      ? `Revoke ALL ${filteredPerms.length} permissions from "${role.name}"? This cannot be undone from here.`
      : `Grant ALL ${filteredPerms.length} permissions to "${role.name}"? This is a dangerous bulk operation.`;

    if (!window.confirm(msg)) return;

    let successes = 0;
    let failures = 0;
    for (const p of filteredPerms) {
      const has = matrix[role.id]?.has(p.id) ?? false;
      if (action === 'grant' && has) continue;
      if (action === 'revoke' && !has) continue;
      try {
        if (action === 'grant') {
          await API.post('/roles/' + role.id + '/permissions', { permission_id: p.id });
        } else {
          await API.del('/roles/' + role.id + '/permissions/' + p.id);
        }
        successes++;
        setMatrix(prev => {
          const copy = { ...prev };
          const s = new Set(copy[role.id] || []);
          if (action === 'grant') s.add(p.id); else s.delete(p.id);
          copy[role.id] = s;
          return copy;
        });
      } catch {
        failures++;
      }
    }
    if (failures > 0) {
      toast.error(`${verb}: ${successes} ok, ${failures} failed`);
    } else {
      toast.success(`${verb} "${role.name}": ${successes} permissions updated`);
    }
  }

  // Empty state: zero roles
  if (!loadingRoles && roles.length === 0) {
    return (
      <div style={{ padding: 40 }}>
        <TeachEmptyState
          icon="Shield"
          title="No roles yet"
          description="The matrix needs at least one role. Create a role first, then come back to manage permissions across roles at a glance."
          createLabel="Create a role"
          cliSnippet="shark role create --name admin"
          onCreate={onSwitchToRoles}
        />
      </div>
    );
  }

  // Loading skeleton
  if (loadingRoles || loadingPerms || !matrixLoaded) {
    return (
      <div style={{ padding: 20 }}>
        <div className="faint" style={{ fontSize: 11.5, marginBottom: 12 }}>Loading permission matrix…</div>
        <div style={{ display: 'grid', gridTemplateColumns: `240px repeat(${Math.max(roles.length || 3, 1)}, 1fr)`, gap: 1, background: 'var(--hairline)' }}>
          {[...Array(6 * ((roles.length || 3) + 1))].map((_, i) => (
            <div key={i} style={{ height: 32, background: 'var(--surface-1)', opacity: 0.4 }}/>
          ))}
        </div>
      </div>
    );
  }

  const colCount = roles.length;

  return (
    <div style={{ padding: 20 }}>
      {/* Toolbar */}
      <div className="row" style={{ gap: 10, marginBottom: 14, justifyContent: 'space-between' }}>
        <div className="row" style={{ gap: 10 }}>
          <input
            placeholder="Filter permissions…"
            value={filter}
            onChange={e => setFilter(e.target.value)}
            style={{
              fontSize: 12, padding: '6px 10px',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4, background: 'var(--surface-0)',
              width: 260,
            }}
          />
          <span className="faint" style={{ fontSize: 11.5 }}>
            {filteredPerms.length} of {perms.length} permissions · {roles.length} role{roles.length !== 1 ? 's' : ''}
          </span>
        </div>
        <div className="faint" style={{ fontSize: 11 }}>
          Tip: click a column header to grant/revoke all.
        </div>
      </div>

      {/* Grid — sticky header row and first column */}
      <div style={{
        border: '1px solid var(--hairline)',
        borderRadius: 6,
        background: 'var(--surface-0)',
        overflow: 'auto',
        maxHeight: 'calc(100vh - 240px)',
      }}>
        <table style={{ borderCollapse: 'separate', borderSpacing: 0, width: '100%', fontSize: 12 }}>
          <thead>
            <tr>
              <th style={{
                position: 'sticky', top: 0, left: 0, zIndex: 3,
                background: 'var(--surface-1)',
                borderBottom: '1px solid var(--hairline)',
                borderRight: '1px solid var(--hairline)',
                textAlign: 'left', padding: '10px 12px',
                fontSize: 11, fontWeight: 500, color: 'var(--fg-muted)',
                minWidth: 280,
              }}>
                Permission
              </th>
              {roles.map(r => (
                <th
                  key={r.id}
                  onClick={() => bulkToggleColumn(r)}
                  title={`Click to grant/revoke ALL permissions for "${r.name}"`}
                  style={{
                    position: 'sticky', top: 0, zIndex: 2,
                    background: 'var(--surface-1)',
                    borderBottom: '1px solid var(--hairline)',
                    borderRight: '1px solid var(--hairline)',
                    textAlign: 'center', padding: '10px 8px',
                    fontSize: 11.5, fontWeight: 500,
                    cursor: 'pointer',
                    minWidth: 100,
                    whiteSpace: 'nowrap',
                  }}
                >
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
                    <Icon.Shield width={10} height={10} style={{ opacity: 0.6 }}/>
                    {r.name}
                  </span>
                </th>
              ))}
              <th style={{
                position: 'sticky', top: 0, zIndex: 2,
                background: 'var(--surface-1)',
                borderBottom: '1px solid var(--hairline)',
                padding: '10px 12px',
                fontSize: 11, fontWeight: 500, color: 'var(--fg-muted)',
                textAlign: 'right', minWidth: 120,
              }}>
                Usage
              </th>
            </tr>
          </thead>
          <tbody>
            {filteredPerms.length === 0 && (
              <tr>
                <td colSpan={colCount + 2} style={{ padding: 30, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
                  {perms.length === 0 ? 'No permissions defined yet.' : 'No permissions match the filter.'}
                </td>
              </tr>
            )}
            {filteredPerms.map(p => {
              const usage = usageMap[p.id] || {};
              const userCount = usage.users ?? null;
              return (
                <tr key={p.id}>
                  <td style={{
                    position: 'sticky', left: 0, zIndex: 1,
                    background: 'var(--surface-0)',
                    borderBottom: '1px solid var(--hairline)',
                    borderRight: '1px solid var(--hairline)',
                    padding: '8px 12px',
                  }}>
                    <div className="mono" style={{ fontSize: 11.5 }}>
                      <span className="chip" style={{ height: 16, fontSize: 10, marginRight: 6 }}>{p.action}</span>
                      <span className="faint">{p.resource}</span>
                    </div>
                  </td>
                  {roles.map(r => {
                    const has = matrix[r.id]?.has(p.id) ?? false;
                    const k = cellKey(r.id, p.id);
                    const busy = pending.has(k);
                    return (
                      <td
                        key={r.id}
                        style={{
                          borderBottom: '1px solid var(--hairline)',
                          borderRight: '1px solid var(--hairline)',
                          textAlign: 'center', padding: '6px 8px',
                          background: has ? 'rgba(34,197,94,0.05)' : 'transparent',
                        }}
                      >
                        <input
                          type="checkbox"
                          checked={has}
                          disabled={busy}
                          onChange={() => toggleCell(r, p)}
                          style={{ cursor: busy ? 'wait' : 'pointer', width: 14, height: 14 }}
                          title={`${has ? 'Revoke' : 'Grant'} ${p.action}·${p.resource} ${has ? 'from' : 'to'} ${r.name}`}
                        />
                      </td>
                    );
                  })}
                  <td style={{
                    borderBottom: '1px solid var(--hairline)',
                    padding: '8px 12px',
                    textAlign: 'right',
                    fontSize: 10.5,
                    color: 'var(--fg-dim)',
                    whiteSpace: 'nowrap',
                  }}>
                    {userCount !== null
                      ? `Used by ${userCount} user${userCount !== 1 ? 's' : ''}`
                      : '…'}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
