// @ts-nocheck
import React from 'react'
import { Icon, Avatar, CopyField, hashColor } from './shared'
import { API, useAPI } from './api'
import { MOCK } from './mock'

// Organizations — split-pane browser (list left, persistent detail right)

export function Organizations() {
  const { data: orgsRaw, loading, refresh } = useAPI('/admin/organizations');
  const orgs = orgsRaw?.organizations || [];

  const [selectedId, setSelectedId] = React.useState(() => localStorage.getItem('org-selected') || '');
  const [query, setQuery] = React.useState('');

  // Auto-select first org once loaded
  React.useEffect(() => {
    if (orgs.length > 0 && !selectedId) setSelectedId(orgs[0].id);
  }, [orgs.length]);

  React.useEffect(() => { if (selectedId) localStorage.setItem('org-selected', selectedId); }, [selectedId]);

  const filtered = orgs.filter(o => {
    if (query && !o.name.toLowerCase().includes(query.toLowerCase()) && !(o.slug || '').includes(query.toLowerCase())) return false;
    return true;
  });

  const selected = orgs.find(o => o.id === selectedId) || orgs[0] || null;

  if (loading && orgs.length === 0) {
    return <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12, lineHeight: 1.5 }}>Loading organizations…</div>;
  }

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      {/* LEFT: Org list */}
      <div style={{
        width: 280, flexShrink: 0,
        borderRight: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        display: 'flex', flexDirection: 'column',
      }}>
        {/* Toolbar */}
        <div style={{ padding: '10px 12px', borderBottom: '1px solid var(--hairline)', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="row" style={{ justifyContent: 'space-between' }}>
            <div className="row" style={{ gap: 6 }}>
              <span style={{ fontSize: 13, fontWeight: 500, fontFamily: 'var(--font-display)' }}>Organizations</span>
              <span className="chip" style={{ height: 17, fontSize: 10, lineHeight: 1.5 }}>{orgs.length}</span>
            </div>
            <button className="btn primary sm"><Icon.Plus width={11} height={11}/>New</button>
          </div>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 6,
            border: '1px solid var(--hairline-strong)',
            background: 'var(--surface-1)',
            borderRadius: 5, padding: '0 8px', height: 26,
          }}>
            <Icon.Search width={11} height={11} style={{ opacity: 0.5 }}/>
            <input placeholder="Search orgs…" value={query} onChange={(e) => setQuery(e.target.value)} style={{ flex: 1, fontSize: 11 }}/>
          </div>
        </div>

        {/* List */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {filtered.length === 0 && !loading && orgs.length === 0 ? (
            <div style={{ padding: '32px 20px', textAlign: 'center' }}>
              <div style={{ fontSize: 13, color: 'var(--fg-dim)', lineHeight: 1.6 }}>No organizations yet.</div>
              <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5, marginTop: 6 }}>Create one to manage multi-tenant access.</div>
            </div>
          ) : filtered.length === 0 && !loading ? (
            <div style={{ padding: 20, fontSize: 11, textAlign: 'center', color: 'var(--fg-muted)', lineHeight: 1.5 }}>No orgs match.</div>
          ) : (
            filtered.map(o => (
              <OrgListItem key={o.id} org={o} active={selectedId === o.id} onClick={() => setSelectedId(o.id)}/>
            ))
          )}
        </div>

        <div style={{ padding: '8px 12px', borderTop: '1px solid var(--hairline)', background: 'var(--surface-0)', fontSize: 11, lineHeight: 1.5 }} className="row">
          <span style={{ color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.08em', fontSize: 10 }}>Total</span>
          <span className="mono" style={{ marginLeft: 'auto', color: 'var(--fg-dim)' }}>{orgs.length}</span>
        </div>
      </div>

      {/* RIGHT: Org detail */}
      <div style={{ flex: 1, minWidth: 0, overflowY: 'auto' }}>
        {selected
          ? <OrgDetail org={selected} onRefresh={refresh}/>
          : <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 13, lineHeight: 1.5 }}>Select an organization</div>
        }
      </div>
    </div>
  );
}

function OrgListItem({ org, active, onClick }) {
  const initials = (org.name || '?').charAt(0).toUpperCase();
  return (
    <div onClick={onClick}
      style={{
        padding: '9px 12px',
        borderBottom: '1px solid var(--hairline)',
        cursor: 'pointer',
        background: active ? 'var(--surface-2)' : 'transparent',
        display: 'flex', gap: 10,
      }}>
      <div style={{
        width: 28, height: 28, borderRadius: 5,
        background: hashColor(org.name),
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 12, fontWeight: 600,
        flexShrink: 0,
      }}>{initials}</div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)', marginBottom: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{org.name}</div>
        <div style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg-dim)', lineHeight: 1.5 }}>{org.slug}</div>
      </div>
    </div>
  );
}

/* ---------------- DETAIL ---------------- */

function OrgDetail({ org, onRefresh }) {
  const [tab, setTab] = React.useState('overview');

  const { data: membersRaw, loading: membersLoading, refresh: membersRefresh } = useAPI('/admin/organizations/' + org.id + '/members');
  const { data: rolesRaw, loading: rolesLoading, refresh: rolesRefresh } = useAPI('/admin/organizations/' + org.id + '/roles');

  const members = membersRaw?.members || [];
  const roles = rolesRaw?.roles || [];

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'members', label: 'Members', chip: members.length || undefined },
    { id: 'roles', label: 'Roles', chip: roles.length || undefined },
    { id: 'settings', label: 'Settings' },
  ];

  return (
    <div>
      {/* Header */}
      <div style={{ padding: '16px 20px 14px', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)' }}>
        <div className="row" style={{ gap: 12, alignItems: 'flex-start' }}>
          <div style={{
            width: 48, height: 48, borderRadius: 7,
            background: hashColor(org.name),
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 20, fontWeight: 600,
            flexShrink: 0,
            border: '1px solid var(--hairline-strong)',
          }}>{(org.name || '?').charAt(0).toUpperCase()}</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="row" style={{ gap: 8, marginBottom: 3 }}>
              <h1 style={{
                margin: 0, fontSize: 20, fontWeight: 500,
                letterSpacing: '-0.02em',
                fontFamily: 'var(--font-display)',
                lineHeight: 1.2,
              }}>{org.name}</h1>
              <CopyField value={org.id}/>
            </div>
            <div style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg-dim)', lineHeight: 1.5 }}>{org.slug}</div>
            {org.created_at && (
              <div className="row" style={{ gap: 16, marginTop: 6, fontSize: 11, lineHeight: 1.5 }}>
                <span style={{ color: 'var(--fg-muted)', fontFamily: 'var(--font-mono)' }}>created {relTime(org.created_at)}</span>
                {org.created_by && <span style={{ color: 'var(--fg-muted)' }}>by <span style={{ color: 'var(--fg)' }}>{org.created_by}</span></span>}
              </div>
            )}
          </div>
          <div className="row" style={{ gap: 6 }}>
            <button className="btn sm"><Icon.Plus width={10} height={10}/>Invite</button>
            <OrgActions org={org} onRefresh={onRefresh}/>
          </div>
        </div>

        {/* Stats strip */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0, marginTop: 14, border: '1px solid var(--hairline)', borderRadius: 5, overflow: 'hidden', background: 'var(--surface-1)' }}>
          <OrgStat label="Members" value={String(members.length)} sub="in this org"/>
          <OrgStat label="Roles" value={String(roles.length)} sub="defined"/>
          <OrgStat label="Updated" value={org.updated_at ? relTime(org.updated_at) : '—'} sub="last change" last/>
        </div>
      </div>

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 0, padding: '0 20px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        position: 'sticky', top: 0, zIndex: 2,
      }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '10px 10px', fontSize: 11,
            textTransform: 'uppercase', letterSpacing: '0.08em',
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-dim)',
            fontWeight: tab === t.id ? 500 : 400,
            borderBottom: tab === t.id ? '1px solid var(--fg)' : '1px solid transparent',
            marginBottom: -1,
          }}>
            {t.label}
            {t.chip != null && <span className="chip" style={{ marginLeft: 6, height: 16, fontSize: 9.5, lineHeight: 1.5, padding: '0 5px' }}>{t.chip}</span>}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={{ padding: 20 }}>
        {tab === 'overview' && <OrgOverviewTab org={org} members={members}/>}
        {tab === 'members' && <OrgMembersTab org={org} members={members} loading={membersLoading} onRefresh={membersRefresh}/>}
        {tab === 'roles' && <OrgRolesTab org={org} roles={roles} loading={rolesLoading} onRefresh={rolesRefresh}/>}
        {tab === 'settings' && <OrgSettingsTab org={org} onRefresh={onRefresh}/>}
      </div>
    </div>
  );
}

function OrgActions({ org, onRefresh }) {
  const [open, setOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);

  async function handleDelete() {
    if (!confirm(`Delete organization "${org.name}"? This cannot be undone.`)) return;
    setDeleting(true);
    try {
      await API.del('/admin/organizations/' + org.id);
      onRefresh();
    } catch (e) {
      alert('Delete failed: ' + e.message);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div style={{ position: 'relative' }}>
      <button className="btn ghost icon sm" onClick={() => setOpen(o => !o)} disabled={deleting}>
        <Icon.More width={12} height={12}/>
      </button>
      {open && (
        <div style={{
          position: 'absolute', right: 0, top: '100%', marginTop: 4,
          background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)',
          borderRadius: 5, padding: 4, zIndex: 10, minWidth: 140,
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
        }} onMouseLeave={() => setOpen(false)}>
          <button className="btn ghost sm danger" style={{ width: '100%', justifyContent: 'flex-start' }} onClick={handleDelete}>
            Delete org
          </button>
        </div>
      )}
    </div>
  );
}

function OrgStat({ label, value, sub, good, progress, last }) {
  return (
    <div style={{ padding: '10px 14px', borderRight: last ? 'none' : '1px solid var(--hairline)', minWidth: 0 }}>
      <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-muted)', marginBottom: 4, lineHeight: 1.5 }}>{label}</div>
      <div className="row" style={{ gap: 6 }}>
        <span className="mono" style={{ fontSize: 15, fontWeight: 500, color: good ? 'var(--success)' : 'var(--fg)', lineHeight: 1.3 }}>{value}</span>
      </div>
      <div style={{ fontSize: 10, color: 'var(--fg-muted)', marginTop: 2, lineHeight: 1.5 }}>{sub}</div>
      {progress != null && (
        <div style={{ marginTop: 6, height: 3, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
          <div style={{ width: `${progress*100}%`, height: '100%', background: progress > 0.9 ? 'var(--warn)' : 'var(--fg)' }}/>
        </div>
      )}
    </div>
  );
}

/* --- Overview tab --- */

function OrgOverviewTab({ org, members }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 24 }}>
      <div className="col" style={{ gap: 24 }}>
        <Panel title="Membership">
          <MembershipBreakdown members={members}/>
        </Panel>
      </div>
      <div className="col" style={{ gap: 24 }}>
        <Panel title="Quick actions">
          <div className="col" style={{ gap: 6 }}>
            <button className="btn sm" style={{ justifyContent: 'flex-start' }}><Icon.Plus width={11} height={11}/>Invite members</button>
            <button className="btn sm danger" style={{ justifyContent: 'flex-start', marginTop: 2 }}>Suspend org</button>
          </div>
        </Panel>
        {org.metadata && Object.keys(org.metadata).length > 0 && (
          <Panel title="Metadata">
            <div className="col" style={{ gap: 8, fontSize: 13 }}>
              {Object.entries(org.metadata).map(([k, v]) => (
                <Row key={k} label={k} val={<span style={{ fontFamily: 'var(--font-mono)', color: 'var(--fg-dim)', fontSize: 11, lineHeight: 1.5 }}>{String(v)}</span>}/>
              ))}
            </div>
          </Panel>
        )}
      </div>
    </div>
  );
}

function Panel({ title, children, actions }) {
  return (
    <div className="card">
      <div className="card-header" style={{ fontFamily: 'var(--font-display)', fontSize: 13, fontWeight: 500 }}>
        <span>{title}</span>
        {actions}
      </div>
      <div className="card-body">{children}</div>
    </div>
  );
}

function Row({ label, val }) {
  return (
    <div className="row" style={{ justifyContent: 'space-between' }}>
      <span style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5 }}>{label}</span>
      <span>{val}</span>
    </div>
  );
}

function MembershipBreakdown({ members }) {
  if (!members || members.length === 0) {
    return <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5 }}>No members yet.</div>;
  }

  const byRole = {};
  members.forEach(m => { byRole[m.role] = (byRole[m.role] || 0) + 1; });
  const total = members.length;

  return (
    <div className="col" style={{ gap: 8 }}>
      <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', lineHeight: 1.5, marginBottom: 2 }}>By role</div>
      {Object.entries(byRole).map(([role, n]) => (
        <div key={role} className="row" style={{ gap: 8, fontSize: 13 }}>
          <span className="chip" style={{ width: 72, justifyContent: 'center', textTransform: 'capitalize', fontSize: 11, fontFamily: 'var(--font-mono)' }}>{role}</span>
          <div style={{ flex: 1, height: 4, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
            <div style={{ width: `${(n/total)*100}%`, height: '100%', background: 'var(--fg)' }}/>
          </div>
          <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, width: 30, textAlign: 'right', lineHeight: 1.5 }}>{n}</span>
        </div>
      ))}
      <div style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg-muted)', lineHeight: 1.5, marginTop: 4 }}>{total} total members</div>
    </div>
  );
}

/* --- Members tab --- */

function OrgMembersTab({ org, members, loading, onRefresh }) {
  const [roleFilter, setRoleFilter] = React.useState('all');

  const roles = ['all', ...Array.from(new Set(members.map(m => m.role).filter(Boolean)))];
  const filtered = roleFilter === 'all' ? members : members.filter(m => m.role === roleFilter);

  if (loading) return <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5, padding: 20 }}>Loading members…</div>;

  return (
    <>
      <div className="row" style={{ marginBottom: 12, gap: 8 }}>
        <div className="seg" style={{ display: 'inline-flex', border: '1px solid var(--hairline-strong)', borderRadius: 5, overflow: 'hidden', height: 26 }}>
          {roles.map((r, i) => (
            <button key={r} onClick={() => setRoleFilter(r)}
              style={{ padding: '0 10px', fontSize: 11, textTransform: 'capitalize',
                background: roleFilter === r ? 'var(--surface-3)' : 'var(--surface-1)',
                color: roleFilter === r ? 'var(--fg)' : 'var(--fg-muted)',
                borderRight: i < roles.length - 1 ? '1px solid var(--hairline)' : 'none',
              }}>{r}</button>
          ))}
        </div>
        <div style={{ flex: 1 }}/>
        <button className="btn primary sm"><Icon.Plus width={11} height={11}/>Invite member</button>
      </div>

      <div className="card" style={{ overflow: 'hidden' }}>
        <table className="tbl">
          <thead>
            <tr>
              <th style={{
                paddingLeft: 16,
                fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
                color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-1)',
              }}>Member</th>
              <th style={{
                width: 100,
                fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
                color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-1)',
              }}>Role</th>
              <th style={{
                width: 120,
                fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
                color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-1)',
              }}>Joined</th>
              <th style={{
                width: 36,
                position: 'sticky', top: 0, background: 'var(--surface-1)',
              }}></th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(m => (
              <tr key={m.user_id || m.email}>
                <td style={{ paddingLeft: 16 }}>
                  <div className="row" style={{ gap: 8 }}>
                    <Avatar name={m.name || m.email} email={m.email}/>
                    <div style={{ minWidth: 0 }}>
                      <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)', lineHeight: 1.4 }}>{m.name || '—'}</div>
                      <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5 }}>{m.email}</div>
                    </div>
                  </div>
                </td>
                <td>
                  <span className={'chip' + (m.role === 'owner' ? ' solid' : '')} style={{ textTransform: 'capitalize', fontSize: 11, fontFamily: 'var(--font-mono)' }}>{m.role || '—'}</span>
                </td>
                <td style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg-dim)', lineHeight: 1.5 }}>{m.joined_at ? relTime(m.joined_at) : '—'}</td>
                <td><button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button></td>
              </tr>
            ))}
            {filtered.length === 0 && (
              <tr><td colSpan={4} style={{ padding: '20px 16px', textAlign: 'center', color: 'var(--fg-muted)', fontSize: 11, lineHeight: 1.5 }}>No members.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </>
  );
}

/* --- Roles tab --- */

function OrgRolesTab({ org, roles, loading, onRefresh }) {
  const [creating, setCreating] = React.useState(false);
  const [newName, setNewName] = React.useState('');
  const [newDesc, setNewDesc] = React.useState('');
  const [saving, setSaving] = React.useState(false);
  const [err, setErr] = React.useState(null);

  async function handleCreate(e) {
    e.preventDefault();
    if (!newName.trim()) return;
    setSaving(true);
    setErr(null);
    try {
      await API.post('/admin/organizations/' + org.id + '/roles', { name: newName.trim(), description: newDesc.trim() });
      setNewName('');
      setNewDesc('');
      setCreating(false);
      onRefresh();
    } catch (e) {
      setErr(e.message);
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5, padding: 20 }}>Loading roles…</div>;

  return (
    <>
      <div className="row" style={{ marginBottom: 12 }}>
        <span style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)' }}>{roles.length} role{roles.length !== 1 ? 's' : ''}</span>
        <div style={{ flex: 1 }}/>
        <button className="btn primary sm" onClick={() => setCreating(c => !c)}><Icon.Plus width={11} height={11}/>New role</button>
      </div>

      {/* Inline create form — compact, not modal */}
      {creating && (
        <form onSubmit={handleCreate} style={{ marginBottom: 12 }}>
          <div className="row" style={{ gap: 6, alignItems: 'flex-start', flexWrap: 'wrap' }}>
            <input
              autoFocus
              placeholder="Role name"
              value={newName}
              onChange={e => setNewName(e.target.value)}
              style={{
                fontSize: 13, padding: '5px 8px', height: 28,
                border: '1px solid var(--hairline-strong)', borderRadius: 4,
                background: 'var(--surface-0)', width: 160,
              }}
            />
            <input
              placeholder="Description (optional)"
              value={newDesc}
              onChange={e => setNewDesc(e.target.value)}
              style={{
                fontSize: 13, padding: '5px 8px', height: 28,
                border: '1px solid var(--hairline-strong)', borderRadius: 4,
                background: 'var(--surface-0)', flex: 1, minWidth: 120,
              }}
            />
            <button type="submit" className="btn primary sm" disabled={saving || !newName.trim()} style={{ height: 28 }}>{saving ? 'Creating…' : 'Create'}</button>
            <button type="button" className="btn ghost sm" onClick={() => { setCreating(false); setErr(null); }} style={{ height: 28 }}>Cancel</button>
          </div>
          {err && <div style={{ fontSize: 11, color: 'var(--danger)', marginTop: 6, lineHeight: 1.5 }}>{err}</div>}
        </form>
      )}

      <div className="col" style={{ gap: 8 }}>
        {roles.map(r => (
          <div key={r.id || r.name} style={{
            padding: '10px 12px',
            border: '1px solid var(--hairline)',
            borderRadius: 5,
            background: 'var(--surface-1)',
          }}>
            <div className="row" style={{ gap: 8 }}>
              <span style={{ fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg)', lineHeight: 1.5 }}>{r.name}</span>
              {r.description && <span style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5 }}>{r.description}</span>}
              <div style={{ flex: 1 }}/>
              <CopyField value={r.id || r.name}/>
            </div>
          </div>
        ))}
        {roles.length === 0 && (
          <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5, padding: '16px 0' }}>No roles defined.</div>
        )}
      </div>
    </>
  );
}

/* --- Settings tab --- */

function OrgSettingsTab({ org, onRefresh }) {
  const [name, setName] = React.useState(org.name || '');
  const [saving, setSaving] = React.useState(false);
  const [err, setErr] = React.useState(null);
  const [ok, setOk] = React.useState(false);

  // Reset form when org changes
  React.useEffect(() => {
    setName(org.name || '');
    setErr(null);
    setOk(false);
  }, [org.id]);

  async function handleSave(e) {
    e.preventDefault();
    setSaving(true);
    setErr(null);
    setOk(false);
    try {
      await API.patch('/admin/organizations/' + org.id, { name: name.trim() });
      setOk(true);
      onRefresh();
    } catch (e) {
      setErr(e.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div style={{ maxWidth: 480 }}>
      <Panel title="General">
        <form onSubmit={handleSave}>
          <div className="col" style={{ gap: 16 }}>
            <div className="col" style={{ gap: 4 }}>
              <label style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.08em', lineHeight: 1.5 }}>Display name</label>
              <input
                value={name}
                onChange={e => setName(e.target.value)}
                style={{ fontSize: 13, padding: '6px 10px', border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)' }}
              />
            </div>
            <div className="col" style={{ gap: 4 }}>
              <label style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.08em', lineHeight: 1.5 }}>Slug</label>
              <input
                value={org.slug || ''}
                disabled
                style={{ fontSize: 13, padding: '6px 10px', border: '1px solid var(--hairline)', borderRadius: 4, background: 'var(--surface-1)', color: 'var(--fg-dim)' }}
              />
            </div>
            {err && <div style={{ fontSize: 11, color: 'var(--danger)', lineHeight: 1.5 }}>{err}</div>}
            {ok && <div style={{ fontSize: 11, color: 'var(--success)', lineHeight: 1.5 }}>Saved.</div>}
            <div>
              <button type="submit" className="btn primary sm" disabled={saving || !name.trim() || name.trim() === org.name}>
                {saving ? 'Saving…' : 'Save changes'}
              </button>
            </div>
          </div>
        </form>
      </Panel>
    </div>
  );
}

// Relative time helper (ISO string or ms timestamp)
function relTime(val) {
  if (!val) return '—';
  const ms = typeof val === 'number' ? val : Date.parse(val);
  if (isNaN(ms)) return String(val);
  const diff = Date.now() - ms;
  if (diff < 0) return 'just now';
  if (diff < 60000) return Math.floor(diff/1000) + 's ago';
  if (diff < 3600000) return Math.floor(diff/60000) + 'm ago';
  if (diff < 86400000) return Math.floor(diff/3600000) + 'h ago';
  return Math.floor(diff/86400000) + 'd ago';
}
