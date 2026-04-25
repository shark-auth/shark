// @ts-nocheck
import React from 'react'
import { Icon, Avatar, CopyField, hashColor } from './shared'
import { API, useAPI } from './api'
import { MOCK } from './mock'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { TeachEmptyState } from './TeachEmptyState'
import { useTabParam } from './useURLParams'
import { usePageActions } from './useKeyboardShortcuts'

// Organizations — split-pane browser (list left, persistent detail right)

export function Organizations() {
  const { data: orgsRaw, loading, refresh } = useAPI('/admin/organizations');
  const orgs = orgsRaw?.organizations || [];
  const toast = useToast();

  usePageActions({ onRefresh: refresh, onNew: () => setCreating(true) });

  const [selectedId, setSelectedId] = React.useState(() => localStorage.getItem('org-selected') || '');
  const [query, setQuery] = React.useState('');
  const [creating, setCreating] = React.useState(false);

  // `?new=1` query auto-opens the slide-over on mount; strip the param after.
  React.useEffect(() => {
    const sp = new URLSearchParams(window.location.search);
    if (sp.get('new') === '1') {
      setCreating(true);
      sp.delete('new');
      const qs = sp.toString();
      const url = window.location.pathname + (qs ? '?' + qs : '');
      window.history.replaceState(null, '', url);
    }
  }, []);

  const handleCreated = (newOrg) => {
    setCreating(false);
    toast.success(`Organization "${newOrg.name}" created`);
    refresh();
    setSelectedId(newOrg.id);
  };

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
            <button className="btn primary sm" onClick={() => setCreating(true)}><Icon.Plus width={11} height={11}/>New</button>
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
            <TeachEmptyState
              icon="Org"
              title="No organizations yet"
              description="Organizations let you group users into tenants with separate roles and permissions."
              createLabel="New Organization"
              cliSnippet="shark org create --name 'Acme Inc'"
            />
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

      {creating && (
        <CreateOrgSlideover
          onClose={() => setCreating(false)}
          onCreated={handleCreated}
        />
      )}
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
  const [tab, setTab] = useTabParam('overview');

  const { data: membersRaw, loading: membersLoading, refresh: membersRefresh } = useAPI('/admin/organizations/' + org.id + '/members');
  const { data: rolesRaw, loading: rolesLoading, refresh: rolesRefresh } = useAPI('/admin/organizations/' + org.id + '/roles');

  const members = membersRaw?.members || [];
  const roles = rolesRaw?.roles || [];

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'members', label: 'Members', chip: members.length || undefined },
    { id: 'invitations', label: 'Invitations' },
    { id: 'roles', label: 'Roles', chip: roles.length || undefined },
    { id: 'sso', label: 'SSO' },
    { id: 'audit', label: 'Audit' },
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
        {tab === 'invitations' && <OrgInvitationsTab org={org}/>}
        {tab === 'roles' && <OrgRolesTab org={org} roles={roles} loading={rolesLoading} onRefresh={rolesRefresh}/>}
        {tab === 'sso' && <OrgSSOTab org={org} onRefresh={onRefresh}/>}
        {tab === 'audit' && <OrgAuditTab org={org}/>}
        {tab === 'settings' && <OrgSettingsTab org={org} onRefresh={onRefresh}/>}
      </div>
      <CLIFooter command={`shark org show ${org.id}`}/>
    </div>
  );
}

function OrgActions({ org, onRefresh }) {
  const [open, setOpen] = React.useState(false);
  const toast = useToast();

  function handleDelete() {
    toast.undo(`Deleted "${org.name}"`, async () => {
      try {
        await API.del('/admin/organizations/' + org.id);
        onRefresh();
      } catch (e) {
        toast.error('Delete failed: ' + e.message);
        onRefresh();
      }
    });
  }

  return (
    <div style={{ position: 'relative' }}>
      <button className="btn ghost icon sm" onClick={() => setOpen(o => !o)}>
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
        {org.metadata && (() => { const meta = typeof org.metadata === 'string' ? (JSON.parse(org.metadata || '{}')) : (org.metadata || {}); return Object.keys(meta).length > 0 ? (
          <Panel title="Metadata">
            <div className="col" style={{ gap: 8, fontSize: 13 }}>
              {Object.entries(meta).map(([k, v]) => (
                <Row key={k} label={k} val={<span style={{ fontFamily: 'var(--font-mono)', color: 'var(--fg-dim)', fontSize: 11, lineHeight: 1.5 }}>{String(v)}</span>}/>
              ))}
            </div>
          </Panel>
        ) : null; })()}
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
                    <Avatar name={m.user_name || m.name || m.user_email || m.email} email={m.user_email || m.email}/>
                    <div style={{ minWidth: 0 }}>
                      <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)', lineHeight: 1.4 }}>{m.user_name || m.name || '—'}</div>
                      <div style={{ fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5 }}>{m.user_email || m.email}</div>
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

function OrgInvitationsTab({ org }) {
  const { data, loading, refresh } = useAPI('/admin/organizations/' + org.id + '/invitations');
  const invites = data?.invitations || data || [];
  const toast = useToast();

  const handleRevoke = async (id) => {
    try {
      await API.del('/admin/organizations/' + org.id + '/invitations/' + id);
      refresh();
      toast.success('Invitation revoked');
    } catch (e) {
      toast.error(`Revoke failed: ${e.message}`);
    }
  };

  const handleResend = async (id) => {
    try {
      await API.post('/admin/organizations/' + org.id + '/invitations/' + id + '/resend');
      refresh();
      toast.success('Invitation resent');
    } catch (e) {
      toast.error(`Resend failed: ${e.message}`);
    }
  };

  if (loading) return <div className="faint" style={{ fontSize: 11, padding: 8 }}>Loading…</div>;
  if (invites.length === 0) return (
    <div style={{ textAlign: 'center', padding: 32 }}>
      <div className="faint" style={{ fontSize: 13 }}>No pending invitations</div>
      <div className="faint" style={{ fontSize: 11, marginTop: 4 }}>Invite members from the Members tab or CLI: <span className="mono">shark org invite --org {org.slug || org.id} --email user@example.com</span></div>
    </div>
  );

  return (
    <table className="tbl" style={{ fontSize: 12 }}>
      <thead><tr>
        <th>Email</th><th>Role</th><th>Expires</th><th>Status</th><th style={{width:120}}></th>
      </tr></thead>
      <tbody>
        {invites.map(inv => (
          <tr key={inv.id}>
            <td className="mono">{inv.email}</td>
            <td><span className="chip" style={{ height: 18, fontSize: 10 }}>{inv.role || 'member'}</span></td>
            <td className="mono faint" style={{ fontSize: 11 }}>{relTime(inv.expires_at || inv.expires)}</td>
            <td>
              <span className={'chip ' + (inv.status === 'accepted' ? 'success' : inv.status === 'expired' ? 'warn' : '')} style={{ height: 18, fontSize: 10 }}>
                {inv.status || 'pending'}
              </span>
            </td>
            <td>
              <div className="row" style={{ gap: 4 }}>
                <button className="btn ghost sm" onClick={() => handleResend(inv.id)}>Resend</button>
                <button className="btn ghost sm" style={{ color: 'var(--danger)' }} onClick={() => handleRevoke(inv.id)}>Revoke</button>
              </div>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function OrgSSOTab({ org, onRefresh }) {
  const [saving, setSaving] = React.useState(false);
  const [err, setErr] = React.useState<string | null>(null);

  async function toggleSSO() {
    setSaving(true);
    setErr(null);
    try {
      await API.patch('/admin/organizations/' + org.id, { sso_required: !org.sso_required });
      onRefresh();
    } catch (e: any) {
      setErr(e.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div style={{ padding: 4 }}>
      <div style={{ marginBottom: 12 }}>
        <div className="row" style={{ gap: 8, marginBottom: 8, justifyContent: 'space-between' }}>
          <div>
            <div style={{ fontSize: 12, fontWeight: 500, marginBottom: 3 }}>Require SSO</div>
            <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
              When enforced, members must authenticate via SSO. Password/magic-link login disabled for this org.
            </div>
          </div>
          <button
            className="btn sm"
            style={{ flexShrink: 0, fontSize: 11, opacity: saving ? 0.5 : 1 }}
            onClick={toggleSSO}
            disabled={saving}
          >
            {saving ? 'Saving…' : org.sso_required ? 'Disable SSO' : 'Enforce SSO'}
          </button>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span className="chip" style={{ height: 18, fontSize: 10 }}>{org.sso_required ? 'enforced' : 'optional'}</span>
          {err && <span style={{ fontSize: 11, color: 'var(--danger)' }}>{err}</span>}
        </div>
      </div>
      {org.sso_domain && (
        <div style={{ padding: '8px 0', borderTop: '1px solid var(--hairline)' }}>
          <span style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Domain routing</span>
          <div className="mono" style={{ fontSize: 12, marginTop: 4 }}>{org.sso_domain}</div>
          <div className="faint" style={{ fontSize: 11, marginTop: 2 }}>Emails matching @{org.sso_domain} auto-route to this org's SSO connection.</div>
        </div>
      )}
      {!org.sso_domain && (
        <div className="faint" style={{ fontSize: 11, padding: '8px 0', borderTop: '1px solid var(--hairline)' }}>
          No SSO domain bound. Bind one via the API: <span className="mono">PATCH /admin/organizations/{'{id}'}</span> with <span className="mono">sso_domain</span>.
        </div>
      )}
    </div>
  );
}

function OrgAuditTab({ org }) {
  const { data, loading } = useAPI('/audit-logs?limit=50&org_id=' + org.id);
  const events = data?.items || data || [];

  if (loading) return <div className="faint" style={{ fontSize: 11, padding: 8 }}>Loading…</div>;
  if (events.length === 0) return <div className="faint" style={{ fontSize: 12, padding: 16 }}>No audit events for this organization.</div>;

  return (
    <div style={{ maxHeight: 400, overflow: 'auto' }}>
      {events.slice(0, 30).map((e, i) => (
        <div key={e.id || i} className="row" style={{ padding: '6px 0', borderBottom: '1px solid var(--hairline)', gap: 8, fontSize: 11 }}>
          <span className="mono faint" style={{ width: 60, flexShrink: 0 }}>{relTime(e.created_at || e.t)}</span>
          <span className="mono" style={{ flex: 1 }}>{e.event_type || e.action || '—'}</span>
          <span className="faint">{e.actor_email || e.actor_id || '—'}</span>
        </div>
      ))}
    </div>
  );
}

// --- CreateOrgSlideover — W01 ---
// Right-side slide-over to create a new organization via POST /admin/organizations.
// Mirrors CreateUserSlideover (T05) pattern: ESC + backdrop close with dirty
// check, ?new=1 auto-open, inline field errors, 201/409/400 response handling.
// Fields: name (required), slug (auto-derived, editable), description (optional),
// metadata (optional JSON object).
function CreateOrgSlideover({ onClose, onCreated }) {
  const toast = useToast();
  const [name, setName] = React.useState('');
  const [slug, setSlug] = React.useState('');
  const [slugManual, setSlugManual] = React.useState(false); // user has edited slug
  const [description, setDescription] = React.useState('');
  const [metadata, setMetadata] = React.useState('');
  const [submitting, setSubmitting] = React.useState(false);
  const [fieldErrors, setFieldErrors] = React.useState({});
  const nameRef = React.useRef(null);

  const dirty = name !== '' || slug !== '' || description !== '' || metadata !== '';

  // Auto-derive slug from name (only when user hasn't manually edited).
  React.useEffect(() => {
    if (slugManual) return;
    const derived = name
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '')
      .slice(0, 63);
    setSlug(derived);
  }, [name, slugManual]);

  const tryClose = () => {
    if (dirty) {
      if (!window.confirm('Discard this new organization?')) return;
    }
    onClose();
  };

  // ESC closes; guard while submitting.
  React.useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape' && !submitting) tryClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [dirty, submitting]);

  // Focus name on mount.
  React.useEffect(() => {
    if (nameRef.current) nameRef.current.focus();
  }, []);

  const slugRE = /^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$/;

  const validate = () => {
    const errs = {};
    if (!name.trim()) errs.name = 'Name is required';
    else if (name.trim().length > 64) errs.name = 'Name must be 64 characters or fewer';
    if (!slugRE.test(slug)) errs.slug = 'Slug must be 3-64 chars, lowercase a-z, 0-9, hyphens, no leading/trailing hyphen';
    if (metadata.trim() !== '') {
      try {
        const parsed = JSON.parse(metadata.trim());
        if (typeof parsed !== 'object' || Array.isArray(parsed) || parsed === null) {
          errs.metadata = 'Metadata must be a JSON object, e.g. {"key": "value"}';
        }
      } catch {
        errs.metadata = 'Metadata is not valid JSON';
      }
    }
    setFieldErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (submitting) return;
    if (!validate()) return;
    setSubmitting(true);

    const body = {
      name: name.trim(),
      slug: slug.trim(),
    };
    if (description.trim()) body.description = description.trim();
    if (metadata.trim()) body.metadata = metadata.trim();

    try {
      const key = localStorage.getItem('shark_admin_key');
      const res = await fetch('/api/v1/admin/organizations', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${key}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      if (res.status === 201) {
        const newOrg = await res.json();
        onCreated(newOrg);
        return;
      }

      const errBody = await res.json().catch(() => ({}));
      const msg = errBody.message || errBody.error || `HTTP ${res.status}`;
      const code = errBody.code || errBody.error || '';

      if (res.status === 409) {
        setFieldErrors({ slug: msg });
        toast.error(msg);
      } else if (res.status === 400) {
        const lower = (code || msg).toLowerCase();
        if (lower.includes('slug')) {
          setFieldErrors({ slug: msg });
        } else if (lower.includes('name')) {
          setFieldErrors({ name: msg });
        } else if (lower.includes('metadata')) {
          setFieldErrors({ metadata: msg });
        } else {
          setFieldErrors({ form: msg });
        }
        toast.error('Could not create organization. Check the form.');
      } else {
        setFieldErrors({ form: msg });
        toast.error(msg);
      }
    } catch (err) {
      const msg = err?.message || 'Network error';
      setFieldErrors({ form: msg });
      toast.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const inputStyle = (hasError) => ({
    width: '100%', height: 28, padding: '0 8px',
    background: 'var(--surface-1)',
    border: '1px solid ' + (hasError ? 'var(--danger)' : 'var(--hairline-strong)'),
    borderRadius: 4, fontSize: 13, color: 'var(--fg)',
  });

  return (
    <>
      {/* Backdrop */}
      <div
        onClick={() => !submitting && tryClose()}
        style={{
          position: 'fixed', inset: 0, zIndex: 40,
          background: 'rgba(0,0,0,0.35)',
        }}
      />
      {/* Panel */}
      <form
        onSubmit={handleSubmit}
        style={{
          position: 'fixed', top: 0, right: 0, bottom: 0,
          width: 520, maxWidth: '100vw',
          zIndex: 41,
          borderLeft: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          display: 'flex', flexDirection: 'column',
          animation: 'slideIn 140ms ease-out',
        }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
            <button type="button" className="btn ghost sm" onClick={tryClose} disabled={submitting}>
              <Icon.X width={12} height={12}/>Close
            </button>
            <span className="faint" style={{ fontSize: 11 }}>POST /admin/organizations</span>
          </div>
          <div style={{ fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em', fontFamily: 'var(--font-display)', lineHeight: 1.2 }}>
            Create organization
          </div>
          <div className="faint" style={{ fontSize: 13, marginTop: 4 }}>
            New tenant. Slug is auto-derived from name and can be edited below.
          </div>
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <OrgSlideoverField label="Name">
              <input
                ref={nameRef}
                type="text"
                required
                value={name}
                onChange={e => {
                  setName(e.target.value);
                  if (fieldErrors.name) setFieldErrors({ ...fieldErrors, name: undefined });
                }}
                placeholder="Acme Inc"
                style={inputStyle(fieldErrors.name)}
              />
              {fieldErrors.name && (
                <div style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--danger)', marginTop: 3 }}>{fieldErrors.name}</div>
              )}
            </OrgSlideoverField>

            <OrgSlideoverField label="Slug" hint="Unique identifier · lowercase, hyphens · 3-64 chars">
              <input
                type="text"
                value={slug}
                onChange={e => {
                  setSlug(e.target.value);
                  setSlugManual(true);
                  if (fieldErrors.slug) setFieldErrors({ ...fieldErrors, slug: undefined });
                }}
                placeholder="acme-inc"
                style={inputStyle(fieldErrors.slug)}
              />
              {fieldErrors.slug && (
                <div style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--danger)', marginTop: 3 }}>{fieldErrors.slug}</div>
              )}
            </OrgSlideoverField>

            <OrgSlideoverField label="Description" hint="Optional">
              <textarea
                value={description}
                onChange={e => setDescription(e.target.value)}
                placeholder="What is this organization for?"
                rows={3}
                style={{
                  width: '100%', padding: '6px 8px',
                  background: 'var(--surface-1)',
                  border: '1px solid var(--hairline-strong)',
                  borderRadius: 4, fontSize: 13, color: 'var(--fg)',
                  resize: 'vertical', fontFamily: 'inherit',
                }}
              />
            </OrgSlideoverField>

            <OrgSlideoverField label="Metadata" hint={'Optional · JSON object, e.g. {"plan": "pro"}'}>
              <textarea
                value={metadata}
                onChange={e => {
                  setMetadata(e.target.value);
                  if (fieldErrors.metadata) setFieldErrors({ ...fieldErrors, metadata: undefined });
                }}
                placeholder={'{"plan": "pro", "region": "us-east-1"}'}
                rows={4}
                style={{
                  width: '100%', padding: '6px 8px',
                  background: 'var(--surface-1)',
                  border: '1px solid ' + (fieldErrors.metadata ? 'var(--danger)' : 'var(--hairline-strong)'),
                  borderRadius: 4, fontSize: 12, color: 'var(--fg)',
                  resize: 'vertical', fontFamily: 'var(--font-mono)',
                }}
              />
              {fieldErrors.metadata && (
                <div style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--danger)', marginTop: 3 }}>{fieldErrors.metadata}</div>
              )}
            </OrgSlideoverField>

            {fieldErrors.form && (
              <div style={{
                padding: 8,
                border: '1px solid var(--danger)',
                borderRadius: 4,
                background: 'color-mix(in oklch, var(--danger) 8%, var(--surface-1))',
                fontSize: 12, color: 'var(--danger)', lineHeight: 1.5,
              }}>
                {fieldErrors.form}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div style={{
          padding: 12, borderTop: '1px solid var(--hairline)',
          display: 'flex', justifyContent: 'flex-end', gap: 8,
          background: 'var(--surface-0)',
        }}>
          <button type="button" className="btn sm ghost" onClick={tryClose} disabled={submitting}>
            Cancel
          </button>
          <button type="submit" className="btn primary sm" disabled={submitting}>
            {submitting ? 'Creating…' : 'Create organization'}
          </button>
        </div>
      </form>
    </>
  );
}

// OrgSlideoverField — label + optional hint wrapper used inside CreateOrgSlideover.
function OrgSlideoverField({ label, children, hint }) {
  return (
    <div style={{ marginBottom: 0 }}>
      <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 4, lineHeight: 1.5 }}>{label}</div>
      {children}
      {hint && <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>{hint}</div>}
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
