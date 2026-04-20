// @ts-nocheck
import React from 'react'
import { Icon, CopyField, hashColor } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'
import { TeachEmptyState } from './TeachEmptyState'
import { useTabParam, useURLParam } from './useURLParams'
import { usePageActions } from './useKeyboardShortcuts'
import { PermissionMatrix } from './rbac_matrix'

// Roles & Permissions — two-pane RBAC manager

export function RBAC() {
  const [view, setView] = useURLParam('view', 'roles');
  const { data: rolesRaw, loading, refresh } = useAPI('/roles');
  const roles = rolesRaw?.roles || (Array.isArray(rolesRaw) ? rolesRaw : []);

  usePageActions({ onNew: () => setCreating(true), onRefresh: refresh });

  const [selectedId, setSelectedId] = React.useState(null);
  const [creating, setCreating] = React.useState(false);
  const [newName, setNewName] = React.useState('');
  const [newDesc, setNewDesc] = React.useState('');
  const [saving, setSaving] = React.useState(false);
  const [err, setErr] = React.useState(null);

  // Auto-select first role once loaded
  React.useEffect(() => {
    if (roles.length > 0 && !selectedId) setSelectedId(roles[0].id);
  }, [roles.length]);

  const selected = roles.find(r => r.id === selectedId) || null;

  async function handleCreate(e) {
    e.preventDefault();
    if (!newName.trim()) return;
    setSaving(true);
    setErr(null);
    try {
      const created = await API.post('/roles', { name: newName.trim(), description: newDesc.trim() });
      setNewName('');
      setNewDesc('');
      setCreating(false);
      await refresh();
      if (created?.id) setSelectedId(created.id);
    } catch (ex) {
      setErr(ex.message);
    } finally {
      setSaving(false);
    }
  }

  if (loading && roles.length === 0) {
    return <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>Loading roles…</div>;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Top-level view switch: Roles list/detail vs Matrix grid */}
      <div style={{
        display: 'flex', gap: 2, padding: '0 20px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        flexShrink: 0,
      }}>
        {[
          { id: 'roles', label: 'Roles' },
          { id: 'matrix', label: 'Matrix' },
        ].map(v => (
          <button key={v.id} onClick={() => setView(v.id)} style={{
            padding: '10px 14px', fontSize: 12.5,
            color: view === v.id ? 'var(--fg)' : 'var(--fg-muted)',
            fontWeight: view === v.id ? 500 : 400,
            borderBottom: view === v.id ? '1.5px solid var(--fg)' : '1.5px solid transparent',
            marginBottom: -1,
            background: 'transparent',
            cursor: 'pointer',
          }}>
            {v.label}
          </button>
        ))}
      </div>

      {view === 'matrix' ? (
        <div style={{ flex: 1, overflow: 'auto' }}>
          <PermissionMatrix onSwitchToRoles={() => setView('roles')}/>
        </div>
      ) : (
    <div style={{ display: 'flex', flex: 1, minHeight: 0, overflow: 'hidden' }}>
      {/* LEFT: Role list */}
      <div style={{
        width: 300, flexShrink: 0,
        borderRight: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        display: 'flex', flexDirection: 'column',
      }}>
        {/* Toolbar */}
        <div style={{ padding: '10px 12px', borderBottom: '1px solid var(--hairline)', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="row" style={{ justifyContent: 'space-between' }}>
            <div className="row" style={{ gap: 6 }}>
              <span style={{ fontSize: 13, fontWeight: 500 }}>Roles</span>
              <span className="chip" style={{ height: 17, fontSize: 10 }}>{roles.length}</span>
            </div>
            <button className="btn primary sm" onClick={() => setCreating(c => !c)}>
              <Icon.Plus width={11} height={11}/>New Role
            </button>
          </div>
        </div>

        {/* New role inline form */}
        {creating && (
          <form onSubmit={handleCreate} style={{ padding: 10, borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)' }}>
            <div className="col" style={{ gap: 6 }}>
              <input
                autoFocus
                placeholder="Role name"
                value={newName}
                onChange={e => setNewName(e.target.value)}
                style={{ fontSize: 12, padding: '5px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
              />
              <input
                placeholder="Description (optional)"
                value={newDesc}
                onChange={e => setNewDesc(e.target.value)}
                style={{ fontSize: 12, padding: '5px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
              />
              {err && <div style={{ fontSize: 11, color: 'var(--danger)' }}>{err}</div>}
              <div className="row" style={{ gap: 6 }}>
                <button type="submit" className="btn primary sm" disabled={saving || !newName.trim()}>
                  {saving ? 'Creating…' : 'Create'}
                </button>
                <button type="button" className="btn ghost sm" onClick={() => { setCreating(false); setErr(null); }}>Cancel</button>
              </div>
            </div>
          </form>
        )}

        {/* List */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {roles.length === 0 && !loading && (
            <TeachEmptyState
              icon="Shield"
              title="No roles yet"
              description="Roles group permissions that can be assigned to users. Create one to start managing access."
              createLabel="New Role"
              cliSnippet="shark role create --name admin"
            />
          )}
          {roles.map(r => (
            <RoleListItem
              key={r.id}
              role={r}
              active={selectedId === r.id}
              onClick={() => setSelectedId(r.id)}
            />
          ))}
        </div>

        <div style={{ padding: '8px 12px', borderTop: '1px solid var(--hairline)', background: 'var(--surface-1)', fontSize: 10.5 }} className="row">
          <span className="faint">Total roles</span>
          <span className="mono" style={{ marginLeft: 'auto' }}>{roles.length}</span>
        </div>
      </div>

      {/* RIGHT: Role detail */}
      <div style={{ flex: 1, minWidth: 0, overflowY: 'auto' }}>
        {selected
          ? <RoleDetail key={selected.id} role={selected} onRefresh={refresh} onDelete={() => { setSelectedId(null); refresh(); }}/>
          : (
            <div style={{ padding: '60px 40px', textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
              {roles.length === 0
                ? 'Create your first role to get started.'
                : 'Select a role to view details.'}
            </div>
          )
        }
      </div>
    </div>
      )}
    </div>
  );
}

function RoleListItem({ role, active, onClick }) {
  return (
    <div
      onClick={onClick}
      style={{
        padding: '10px 12px',
        borderBottom: '1px solid var(--hairline)',
        cursor: 'pointer',
        background: active ? 'var(--surface-2)' : 'transparent',
        display: 'flex', gap: 10, alignItems: 'flex-start',
      }}
    >
      <div style={{
        width: 30, height: 30, borderRadius: 5,
        background: hashColor(role.name || '?'),
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0,
      }}>
        <Icon.Shield width={13} height={13} style={{ color: '#fff', opacity: 0.85 }}/>
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 12.5, fontWeight: 500, marginBottom: 2 }}>{role.name}</div>
        {role.description && (
          <div className="faint" style={{ fontSize: 11, overflow: 'hidden', whiteSpace: 'nowrap', textOverflow: 'ellipsis' }}>
            {role.description}
          </div>
        )}
      </div>
    </div>
  );
}

/* ---- Role Detail ---- */

function RoleDetail({ role, onRefresh, onDelete }) {
  const [tab, setTab] = useTabParam('permissions');

  return (
    <div>
      {/* Header */}
      <RoleDetailHeader role={role} onRefresh={onRefresh} onDelete={onDelete}/>

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 2, padding: '0 20px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        position: 'sticky', top: 0, zIndex: 2,
      }}>
        {[
          { id: 'permissions', label: 'Permissions' },
          { id: 'checker', label: 'Permission Checker' },
        ].map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '11px 10px', fontSize: 12.5,
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-muted)',
            fontWeight: tab === t.id ? 500 : 400,
            borderBottom: tab === t.id ? '1.5px solid var(--fg)' : '1.5px solid transparent',
            marginBottom: -1,
          }}>
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div style={{ padding: 20 }}>
        {tab === 'permissions' && <PermissionsTab role={role} onRefresh={onRefresh}/>}
        {tab === 'checker' && <PermissionChecker/>}
      </div>
    </div>
  );
}

function RoleDetailHeader({ role, onRefresh, onDelete }) {
  const toast = useToast();
  const [editName, setEditName] = React.useState(false);
  const [editDesc, setEditDesc] = React.useState(false);
  const [name, setName] = React.useState(role.name || '');
  const [desc, setDesc] = React.useState(role.description || '');
  const [saving, setSaving] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);

  // Sync when role changes
  React.useEffect(() => {
    setName(role.name || '');
    setDesc(role.description || '');
  }, [role.id]);

  async function saveName() {
    if (!name.trim() || name.trim() === role.name) { setEditName(false); return; }
    setSaving(true);
    try {
      await API.put('/roles/' + role.id, { name: name.trim(), description: desc });
      onRefresh();
    } catch (e) {
      toast.error('Save failed: ' + e.message);
      setName(role.name || '');
    } finally {
      setSaving(false);
      setEditName(false);
    }
  }

  async function saveDesc() {
    if (desc === role.description) { setEditDesc(false); return; }
    setSaving(true);
    try {
      await API.put('/roles/' + role.id, { name: role.name, description: desc });
      onRefresh();
    } catch (e) {
      toast.error('Save failed: ' + e.message);
      setDesc(role.description || '');
    } finally {
      setSaving(false);
      setEditDesc(false);
    }
  }
  function handleDelete() {
    toast.undo(`Deleted role "${role.name}"`, async () => {
      try {
        await API.del('/roles/' + role.id);
        onDelete();
      } catch (e) {
        toast.error('Delete failed: ' + e.message);
      }
    });
  }

  return (
    <div style={{ padding: '18px 20px 16px', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)' }}>
      <div className="row" style={{ gap: 12, alignItems: 'flex-start' }}>
        <div style={{
          width: 44, height: 44, borderRadius: 7,
          background: hashColor(role.name || '?'),
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          flexShrink: 0,
          border: '1px solid var(--hairline-strong)',
        }}>
          <Icon.Shield width={18} height={18} style={{ color: '#fff', opacity: 0.85 }}/>
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* Inline editable name */}
          {editName ? (
            <div className="row" style={{ gap: 6, marginBottom: 4 }}>
              <input
                autoFocus
                value={name}
                onChange={e => setName(e.target.value)}
                onBlur={saveName}
                onKeyDown={e => { if (e.key === 'Enter') saveName(); if (e.key === 'Escape') { setName(role.name || ''); setEditName(false); } }}
                style={{ fontSize: 18, fontWeight: 500, padding: '2px 6px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', letterSpacing: '-0.01em', width: 260 }}
              />
              {saving && <span className="faint" style={{ fontSize: 11 }}>Saving…</span>}
            </div>
          ) : (
            <div className="row" style={{ gap: 8, marginBottom: 4 }}>
              <h1
                style={{ margin: 0, fontSize: 18, fontWeight: 500, letterSpacing: '-0.01em', cursor: 'text' }}
                onClick={() => setEditName(true)}
                title="Click to edit"
              >{role.name}</h1>
              <CopyField value={role.id}/>
            </div>
          )}

          {/* Inline editable description */}
          {editDesc ? (
            <input
              autoFocus
              value={desc}
              onChange={e => setDesc(e.target.value)}
              onBlur={saveDesc}
              onKeyDown={e => { if (e.key === 'Enter') saveDesc(); if (e.key === 'Escape') { setDesc(role.description || ''); setEditDesc(false); } }}
              placeholder="Add a description…"
              style={{ fontSize: 12, color: 'var(--fg-muted)', padding: '2px 6px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', width: 380 }}
            />
          ) : (
            <div
              className="faint"
              style={{ fontSize: 12, cursor: 'text', minHeight: 18 }}
              onClick={() => setEditDesc(true)}
              title="Click to edit"
            >
              {role.description || <span style={{ opacity: 0.45 }}>No description — click to add</span>}
            </div>
          )}
        </div>
        <button
          className="btn sm danger"
          onClick={handleDelete}
          disabled={deleting}
          style={{ flexShrink: 0 }}
        >
          {deleting ? 'Deleting…' : 'Delete role'}
        </button>
      </div>
    </div>
  );
}

/* ---- Permissions Tab ---- */

function PermissionsTab({ role, onRefresh }) {
  const toast = useToast();
  const { data: roleData, loading: loadingRole, refresh: refreshRole } = useAPI('/roles/' + role.id);
  const { data: allPermsRaw, loading: loadingPerms } = useAPI('/permissions');

  const permissions = roleData?.permissions || roleData?.role?.permissions || [];
  const allPerms = allPermsRaw?.permissions || (Array.isArray(allPermsRaw) ? allPermsRaw : []);

  // Batch usage fetch — one call replaces N×2 per-row calls from PermissionRow.
  const permIds = permissions.map(p => p.id);
  const batchPath = permIds.length > 0
    ? '/admin/permissions/batch-usage?ids=' + permIds.join(',')
    : null;
  const { data: usageRaw, refresh: refreshUsage } = useAPI(batchPath, [permIds.join(',')]);
  const usageMap = usageRaw || {};

  const [detaching, setDetaching] = React.useState(null);
  const [attachId, setAttachId] = React.useState('');
  const [attaching, setAttaching] = React.useState(false);
  const [attachErr, setAttachErr] = React.useState(null);

  // Create & Attach form
  const [showCreate, setShowCreate] = React.useState(false);
  const [caAction, setCaAction] = React.useState('');
  const [caResource, setCaResource] = React.useState('');
  const [creating, setCreating] = React.useState(false);
  const [createErr, setCreateErr] = React.useState(null);

  const attachedIds = new Set(permissions.map(p => p.id));
  const availablePerms = allPerms.filter(p => !attachedIds.has(p.id));

  async function handleDetach(pid) {
    setDetaching(pid);
    try {
      await API.del('/roles/' + role.id + '/permissions/' + pid);
      refreshRole();
      onRefresh();
    } catch (e) {
      toast.error('Detach failed: ' + e.message);
    } finally {
      setDetaching(null);
    }
  }

  async function handleAttach(e) {
    e.preventDefault();
    if (!attachId) return;
    setAttaching(true);
    setAttachErr(null);
    try {
      await API.post('/roles/' + role.id + '/permissions', { permission_id: attachId });
      setAttachId('');
      refreshRole();
      onRefresh();
    } catch (ex) {
      setAttachErr(ex.message);
    } finally {
      setAttaching(false);
    }
  }

  async function handleCreateAttach(e) {
    e.preventDefault();
    if (!caAction.trim() || !caResource.trim()) return;
    setCreating(true);
    setCreateErr(null);
    let permId = null;
    try {
      const perm = await API.post('/permissions', { action: caAction.trim(), resource: caResource.trim() });
      permId = perm?.id || perm?.permission?.id;
      if (!permId) throw new Error('Permission created but no ID returned');

      try {
        await API.post('/roles/' + role.id + '/permissions', { permission_id: permId });
      } catch (attachErr) {
        // Attach failed — roll back the newly created permission to avoid an orphan.
        try { await API.del('/permissions/' + permId); } catch { /* best-effort */ }
        throw new Error('Attach failed: ' + (attachErr.message || 'unknown error'));
      }

      setCaAction('');
      setCaResource('');
      setShowCreate(false);
      refreshRole();
      refreshUsage();
      onRefresh();
    } catch (ex) {
      setCreateErr(ex.message);
    } finally {
      setCreating(false);
    }
  }

  if (loadingRole) {
    return <div className="faint" style={{ fontSize: 11.5 }}>Loading permissions…</div>;
  }

  return (
    <div className="col" style={{ gap: 16 }}>
      {/* Permissions table */}
      <div className="card" style={{ overflow: 'hidden' }}>
        <div className="card-header">
          <span>Attached permissions</span>
          <span className="chip" style={{ height: 16, fontSize: 10 }}>{permissions.length}</span>
        </div>
        <table className="tbl">
          <thead>
            <tr>
              <th style={{ paddingLeft: 14 }}>Action</th>
              <th>Resource</th>
              <th style={{ width: 60 }}></th>
            </tr>
          </thead>
          <tbody>
            {permissions.map(p => (
              <PermissionRow
                key={p.id}
                p={p}
                onDetach={handleDetach}
                detaching={detaching === p.id}
                roleCount={usageMap[p.id]?.roles ?? null}
                userCount={usageMap[p.id]?.users ?? null}
              />
            ))}
            {permissions.length === 0 && (
              <tr>
                <td colSpan={3} style={{ padding: '20px 14px', textAlign: 'center', color: 'var(--fg-dim)', fontSize: 11.5 }}>
                  No permissions attached. Attach one below.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Attach existing permission */}
      <div className="card">
        <div className="card-header"><span>Attach permission</span></div>
        <div className="card-body">
          <form onSubmit={handleAttach}>
            <div className="row" style={{ gap: 8 }}>
              <select
                value={attachId}
                onChange={e => setAttachId(e.target.value)}
                disabled={loadingPerms || attaching}
                style={{
                  flex: 1, fontSize: 12, padding: '5px 8px',
                  border: '1px solid var(--hairline-strong)',
                  borderRadius: 4, background: 'var(--surface-0)',
                  color: attachId ? 'var(--fg)' : 'var(--fg-dim)',
                }}
              >
                <option value="">
                  {loadingPerms ? 'Loading permissions…' : availablePerms.length === 0 ? 'No permissions available' : 'Select a permission…'}
                </option>
                {availablePerms.map(p => (
                  <option key={p.id} value={p.id}>{p.action} · {p.resource}</option>
                ))}
              </select>
              <button type="submit" className="btn primary sm" disabled={!attachId || attaching}>
                {attaching ? 'Attaching…' : 'Attach'}
              </button>
            </div>
            {attachErr && <div style={{ fontSize: 11, color: 'var(--danger)', marginTop: 6 }}>{attachErr}</div>}
          </form>

          <div style={{ marginTop: 12, paddingTop: 12, borderTop: '1px solid var(--hairline)' }}>
            <button
              className="btn ghost sm"
              onClick={() => setShowCreate(c => !c)}
              style={{ fontSize: 11.5 }}
            >
              <Icon.Plus width={10} height={10}/>
              Create &amp; Attach new permission
            </button>

            {showCreate && (
              <form onSubmit={handleCreateAttach} style={{ marginTop: 10 }}>
                <div className="col" style={{ gap: 8 }}>
                  <div className="row" style={{ gap: 8 }}>
                    <input
                      autoFocus
                      placeholder="Action (e.g. read)"
                      value={caAction}
                      onChange={e => setCaAction(e.target.value)}
                      style={{ flex: 1, fontSize: 12, padding: '5px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
                    />
                    <input
                      placeholder="Resource (e.g. documents)"
                      value={caResource}
                      onChange={e => setCaResource(e.target.value)}
                      style={{ flex: 1, fontSize: 12, padding: '5px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
                    />
                  </div>
                  {createErr && <div style={{ fontSize: 11, color: 'var(--danger)' }}>{createErr}</div>}
                  <div className="row" style={{ gap: 6 }}>
                    <button type="submit" className="btn primary sm" disabled={creating || !caAction.trim() || !caResource.trim()}>
                      {creating ? 'Creating…' : 'Create & Attach'}
                    </button>
                    <button type="button" className="btn ghost sm" onClick={() => { setShowCreate(false); setCreateErr(null); }}>
                      Cancel
                    </button>
                  </div>
                </div>
              </form>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ---- Permission Checker ---- */

function PermissionChecker() {
  const [userId, setUserId] = React.useState('');
  const [action, setAction] = React.useState('');
  const [resource, setResource] = React.useState('');
  const [checking, setChecking] = React.useState(false);
  const [result, setResult] = React.useState(null); // null | { allowed: bool, message?: string }
  const [err, setErr] = React.useState(null);

  async function handleCheck(e) {
    e.preventDefault();
    if (!userId.trim() || !action.trim() || !resource.trim()) return;
    setChecking(true);
    setResult(null);
    setErr(null);
    try {
      const res = await API.post('/auth/check', {
        user_id: userId.trim(),
        action: action.trim(),
        resource: resource.trim(),
      });
      setResult(res);
    } catch (ex) {
      setErr(ex.message);
    } finally {
      setChecking(false);
    }
  }

  const allowed = result?.allowed ?? result?.permitted ?? result?.result === 'allow';

  return (
    <div className="card" style={{ maxWidth: 520 }}>
      <div className="card-header"><span>Permission Checker</span></div>
      <div className="card-body">
        <form onSubmit={handleCheck}>
          <div className="col" style={{ gap: 10 }}>
            <div className="col" style={{ gap: 4 }}>
              <label style={{ fontSize: 10.5, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>User ID</label>
              <input
                value={userId}
                onChange={e => setUserId(e.target.value)}
                placeholder="usr_…"
                style={{ fontSize: 12, padding: '6px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)', fontFamily: 'var(--font-mono)' }}
              />
            </div>
            <div className="row" style={{ gap: 8 }}>
              <div className="col" style={{ gap: 4, flex: 1 }}>
                <label style={{ fontSize: 10.5, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Action</label>
                <input
                  value={action}
                  onChange={e => setAction(e.target.value)}
                  placeholder="read"
                  style={{ fontSize: 12, padding: '6px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
                />
              </div>
              <div className="col" style={{ gap: 4, flex: 1 }}>
                <label style={{ fontSize: 10.5, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>Resource</label>
                <input
                  value={resource}
                  onChange={e => setResource(e.target.value)}
                  placeholder="documents"
                  style={{ fontSize: 12, padding: '6px 8px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
                />
              </div>
            </div>

            {err && <div style={{ fontSize: 11.5, color: 'var(--danger)' }}>{err}</div>}

            {result !== null && !err && (
              <div style={{
                display: 'inline-flex', alignItems: 'center', gap: 8,
                padding: '8px 12px',
                borderRadius: 5,
                background: allowed ? 'rgba(34,197,94,0.08)' : 'rgba(239,68,68,0.08)',
                border: `1px solid ${allowed ? 'rgba(34,197,94,0.25)' : 'rgba(239,68,68,0.25)'}`,
              }}>
                <span style={{
                  display: 'inline-block', width: 8, height: 8, borderRadius: '50%',
                  background: allowed ? 'var(--success)' : 'var(--danger)',
                  flexShrink: 0,
                }}/>
                <span style={{
                  fontSize: 12.5, fontWeight: 500,
                  color: allowed ? 'var(--success)' : 'var(--danger)',
                }}>
                  {allowed ? 'Allowed' : 'Denied'}
                </span>
                {result.message && (
                  <span className="faint" style={{ fontSize: 11.5 }}>{result.message}</span>
                )}
              </div>
            )}

            <div>
              <button
                type="submit"
                className="btn primary sm"
                disabled={checking || !userId.trim() || !action.trim() || !resource.trim()}
              >
                {checking ? 'Checking…' : 'Check permission'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}

// PermissionRow — single row in the attached-permissions table. Counts are
// passed as props from PermissionsTab which fetches them in one batch call
// (GET /admin/permissions/batch-usage?ids=…) instead of 2 calls per row.
function PermissionRow({ p, onDetach, detaching, roleCount, userCount }) {
  // Use prop counts when available; fall back to '…' while the batch request
  // is in-flight (roleCount/userCount === null means not yet received).
  const roles = roleCount !== null ? roleCount : '…';
  const users = userCount !== null ? userCount : '…';

  return (
    <tr>
      <td style={{ paddingLeft: 14 }}>
        <span className="mono chip">{p.action}</span>
      </td>
      <td>
        <div>
          <span className="mono faint" style={{ fontSize: 12 }}>{p.resource}</span>
          <div className="faint" style={{ fontSize: 10.5, marginTop: 2 }}>
            Used by {roles} role{roles !== 1 ? 's' : ''} · {users} user{users !== 1 ? 's' : ''}
          </div>
        </div>
      </td>
      <td style={{ textAlign: 'right', paddingRight: 10 }}>
        <button
          className="btn ghost icon sm"
          title="Detach"
          disabled={detaching}
          onClick={() => onDetach(p.id)}
          style={{ color: 'var(--danger)' }}
        >
          <Icon.X width={11} height={11}/>
        </button>
      </td>
    </tr>
  );
}

