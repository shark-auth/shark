// @ts-nocheck
import React from 'react'
import { Icon, Avatar, CopyField, Kbd, hashColor } from './shared'
import { API, useAPI } from './api'
import { MOCK } from './mock'

// Users page — table + detail slide-over

// Helpers pulled in — declare locals so this file stands alone
const mins = (n) => Date.now() - n * 60 * 1000;
const hrs = (n) => Date.now() - n * 60 * 60 * 1000;
const days = (n) => Date.now() - n * 24 * 60 * 60 * 1000;
const NOW = Date.now();

export function Users() {
  const [selected, setSelected] = React.useState(null);
  const [query, setQuery] = React.useState('');
  const [debouncedQuery, setDebouncedQuery] = React.useState('');
  const [checked, setChecked] = React.useState(new Set());
  const [page, setPage] = React.useState(1);
  const [perPage] = React.useState(25);

  // Debounce search input — 300ms
  React.useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(t);
  }, [query]);

  // Reset page to 1 when search changes
  React.useEffect(() => { setPage(1); }, [debouncedQuery]);

  const searchParam = debouncedQuery ? `&search=${encodeURIComponent(debouncedQuery)}` : '';
  const { data: usersData, loading, refresh } = useAPI(
    `/users?page=${page}&per_page=${perPage}${searchParam}`
  );
  const users = usersData?.users || [];
  const total = usersData?.total || 0;
  const totalPages = Math.ceil(total / perPage) || 1;

  const handleDelete = async (userId) => {
    await API.del('/users/' + userId);
    refresh();
    setSelected(null);
  };

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Toolbar */}
        <div style={{
          padding: '10px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 8, alignItems: 'center',
          background: 'var(--surface-0)',
        }}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 6,
            border: '1px solid var(--hairline-strong)',
            background: 'var(--surface-1)',
            borderRadius: 5,
            padding: '0 8px',
            height: 28, width: 280,
          }}>
            <Icon.Search width={12} height={12} style={{ opacity: 0.5 }}/>
            <input
              placeholder="Search by name, email, or id…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              style={{ flex: 1, fontSize: 12, color: 'var(--fg)' }}
            />
            <Kbd keys="/"/>
          </div>
          <button className="btn"><Icon.Filter width={11} height={11}/>Verified<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <button className="btn"><Icon.Filter width={11} height={11}/>MFA<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <button className="btn"><Icon.Filter width={11} height={11}/>Auth method<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <button className="btn"><Icon.Filter width={11} height={11}/>Org<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <button className="btn ghost sm" style={{ color: 'var(--fg-dim)' }}>+ Add filter</button>
          <div style={{ flex: 1 }}/>
          <span className="faint" style={{ fontSize: 11 }}>
            {loading ? '…' : `${total.toLocaleString()} total`}
          </span>
          <button className="btn sm">Export</button>
          <button className="btn primary sm"><Icon.Plus width={11} height={11}/>New user</button>
        </div>

        {checked.size > 0 && (
          <div style={{
            padding: '6px 16px',
            background: 'var(--surface-2)',
            borderBottom: '1px solid var(--hairline)',
            display: 'flex', alignItems: 'center', gap: 10, fontSize: 12,
          }}>
            <span>{checked.size} selected</span>
            <button className="btn sm">Assign role</button>
            <button className="btn sm">Add to org</button>
            <button className="btn sm">Export CSV</button>
            <button className="btn sm danger">Delete</button>
            <div style={{ flex: 1 }}/>
            <button className="btn ghost sm" onClick={() => setChecked(new Set())}>Clear</button>
          </div>
        )}

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ width: 28, paddingLeft: 16, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>
                  <span className={"cb" + (users.length > 0 && checked.size === users.length ? ' on' : '')}
                    onClick={() => setChecked(checked.size === users.length ? new Set() : new Set(users.map(x => x.id)))}>
                    {users.length > 0 && checked.size === users.length && <Icon.Check width={10} height={10}/>}
                  </span>
                </th>
                <th style={{ position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>User</th>
                <th style={{ width: 90, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>Verified</th>
                <th style={{ width: 70, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>MFA</th>
                <th style={{ width: 120, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>Auth</th>
                <th style={{ width: 140, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>Orgs</th>
                <th style={{ width: 180, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>Roles</th>
                <th style={{ width: 100, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>Created <Icon.ArrowDown width={9} height={9} style={{opacity:0.5, verticalAlign:'middle'}}/></th>
                <th style={{ width: 110, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}>Last active</th>
                <th style={{ width: 36, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}></th>
              </tr>
            </thead>
            <tbody>
              {loading && users.length === 0 && (
                <tr><td colSpan={10} style={{ padding: 32, textAlign: 'center', color: 'var(--fg-muted)', fontSize: 12 }}>Loading…</td></tr>
              )}
              {!loading && users.length === 0 && (
                <tr><td colSpan={10} style={{ padding: 32, textAlign: 'center', color: 'var(--fg-muted)', fontSize: 12 }}>No users found</td></tr>
              )}
              {users.map(u => {
                const orgs = u.orgs || [];
                const roles = u.roles || [];
                return (
                  <tr key={u.id}
                      className={selected?.id === u.id ? 'active' : ''}
                      onClick={() => setSelected(u)}
                      style={{ cursor: 'pointer' }}>
                    <td style={{ paddingLeft: 16 }} onClick={e => e.stopPropagation()}>
                      <span className={"cb" + (checked.has(u.id) ? ' on' : '')}
                        onClick={() => {
                          const next = new Set(checked);
                          next.has(u.id) ? next.delete(u.id) : next.add(u.id);
                          setChecked(next);
                        }}>
                        {checked.has(u.id) && <Icon.Check width={10} height={10}/>}
                      </span>
                    </td>
                    <td>
                      <div className="row" style={{ gap: 8 }}>
                        <Avatar name={u.name || u.email} email={u.email}/>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ fontWeight: 500, fontSize: 12.5 }}>{u.name || '—'}</div>
                          <div className="faint" style={{ fontSize: 11 }}>{u.email}</div>
                        </div>
                      </div>
                    </td>
                    <td>
                      {u.email_verified ? (
                        <span className="chip success" style={{ height: 18 }}><Icon.Check width={9} height={9}/>verified</span>
                      ) : (
                        <span className="chip warn" style={{ height: 18 }}>pending</span>
                      )}
                    </td>
                    <td>
                      {u.mfa_enabled || u.mfa ? (
                        <span className="row" style={{ gap: 4, color: 'var(--success)' }}>
                          <Icon.Shield width={12} height={12}/>
                          <span className="mono" style={{ fontSize: 10.5, color: 'var(--fg-muted)' }}>{u.mfa_method || u.mfa || 'on'}</span>
                        </span>
                      ) : (
                        <span className="faint" style={{ fontSize: 11 }}>—</span>
                      )}
                    </td>
                    <td>
                      <span className="chip" style={{ height: 18 }}>{u.auth_method || u.method || '—'}</span>
                    </td>
                    <td>
                      <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                        {orgs.slice(0, 2).map((o, i) => {
                          const name = typeof o === 'string' ? o : (o.name || o.slug || o.id);
                          return <span key={i} className="chip" style={{ height: 18 }}>{name}</span>;
                        })}
                        {orgs.length > 2 && <span className="faint" style={{ fontSize: 11 }}>+{orgs.length - 2}</span>}
                      </div>
                    </td>
                    <td>
                      <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                        {roles.slice(0, 2).map((r, i) => {
                          const name = typeof r === 'string' ? r : (r.name || r.id);
                          return <span key={i} className="chip" style={{ height: 18 }}>{name}</span>;
                        })}
                        {roles.length > 2 && <span className="faint" style={{ fontSize: 11 }}>+{roles.length - 2}</span>}
                      </div>
                    </td>
                    <td className="mono faint" style={{ fontSize: 11 }}>
                      {u.created_at ? MOCK.relativeTime(new Date(u.created_at).getTime()) : (u.created ? MOCK.relativeTime(u.created) : '—')}
                    </td>
                    <td className="mono" style={{ fontSize: 11, color: (() => {
                      const ts = u.last_active_at ? new Date(u.last_active_at).getTime() : u.lastActive;
                      return ts && (Date.now() - ts < 60*60*1000) ? 'var(--success)' : 'var(--fg-muted)';
                    })() }}>
                      {(() => {
                        const ts = u.last_active_at ? new Date(u.last_active_at).getTime() : u.lastActive;
                        return ts ? MOCK.relativeTime(ts) : '—';
                      })()}
                    </td>
                    <td onClick={e => e.stopPropagation()}>
                      <button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div style={{
          padding: '8px 16px',
          borderTop: '1px solid var(--hairline)',
          display: 'flex', alignItems: 'center', gap: 8,
          background: 'var(--surface-0)',
        }}>
          <button
            className="btn sm"
            disabled={page <= 1}
            onClick={() => setPage(p => Math.max(1, p - 1))}
          >Prev</button>
          <span className="faint" style={{ fontSize: 12 }}>Page {page} of {totalPages}</span>
          <button
            className="btn sm"
            disabled={page >= totalPages}
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
          >Next</button>
          <span className="faint" style={{ fontSize: 11, marginLeft: 8 }}>{total.toLocaleString()} total</span>
        </div>
      </div>

      {selected && (
        <UserSlideover
          user={selected}
          onClose={() => setSelected(null)}
          onDelete={handleDelete}
          onRefreshList={refresh}
        />
      )}
    </div>
  );
}

function UserSlideover({ user, onClose, onDelete, onRefreshList }) {
  const [tab, setTab] = React.useState('profile');
  const tabs = [
    { id: 'profile', label: 'Profile' },
    { id: 'security', label: 'Security' },
    { id: 'rbac', label: 'Roles' },
    { id: 'orgs', label: 'Orgs' },
    { id: 'agents', label: 'Agents', chip: '3' },
    { id: 'activity', label: 'Activity' },
  ];

  const orgs = user.orgs || [];
  const mfaLabel = user.mfa_method || user.mfa;
  const method = user.auth_method || user.method;
  const isVerified = user.email_verified !== undefined ? user.email_verified : user.verified;

  return (
    <div style={{
      width: 540, borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column',
      flexShrink: 0,
      animation: 'slideIn 140ms ease-out',
    }}>
      {/* Header */}
      <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
          <button className="btn ghost sm" onClick={onClose}><Icon.X width={12} height={12}/>Close</button>
          <div className="row" style={{ gap: 4 }}>
            <button className="btn ghost sm">Impersonate</button>
            <button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button>
          </div>
        </div>
        <div className="row" style={{ gap: 12 }}>
          <Avatar name={user.name || user.email} email={user.email} size={44}/>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 16, fontWeight: 500, letterSpacing: '-0.01em' }}>{user.name || '—'}</div>
            <div className="row" style={{ gap: 6, marginTop: 2 }}>
              <span className="faint" style={{ fontSize: 12 }}>{user.email}</span>
              <CopyField value={user.id}/>
            </div>
          </div>
        </div>
        <div className="row" style={{ gap: 6, marginTop: 12 }}>
          {isVerified ? <span className="chip success"><Icon.Check width={9} height={9}/>verified</span> : <span className="chip warn">pending</span>}
          {mfaLabel && <span className="chip"><Icon.Shield width={10} height={10}/>mfa · {mfaLabel}</span>}
          {method && <span className="chip">{method}</span>}
          <span className="chip">{orgs.length} org{orgs.length !== 1 && 's'}</span>
        </div>
      </div>

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 2,
        padding: '0 16px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
      }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '10px 10px',
            fontSize: 12,
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-muted)',
            fontWeight: tab === t.id ? 500 : 400,
            borderBottom: tab === t.id ? '1.5px solid var(--fg)' : '1.5px solid transparent',
            marginBottom: -1,
          }}>
            {t.label}
            {t.chip && <span className="chip" style={{ marginLeft: 5, height: 15, fontSize: 9, padding: '0 4px' }}>{t.chip}</span>}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
        {tab === 'profile' && <ProfileTab user={user} onDelete={onDelete} onRefreshList={onRefreshList}/>}
        {tab === 'security' && <SecurityTab user={user}/>}
        {tab === 'rbac' && <RolesTab user={user}/>}
        {tab === 'orgs' && <OrgsTab user={user}/>}
        {tab === 'agents' && <AgentsConsentsTab user={user}/>}
        {tab === 'activity' && <ActivityTab user={user}/>}
      </div>

      {/* CLI footer */}
      <div style={{
        borderTop: '1px solid var(--hairline)',
        padding: '10px 16px',
        background: 'var(--surface-1)',
        fontSize: 11,
      }}>
        <div className="row" style={{ gap: 8 }}>
          <Icon.Terminal width={12} height={12} style={{ opacity: 0.5 }}/>
          <span className="faint" style={{ fontSize: 10.5 }}>cli</span>
          <code className="mono" style={{ flex: 1, color: 'var(--fg-muted)' }}>shark user show {user.id}</code>
          <button className="btn ghost icon sm" title="Copy"><Icon.Copy width={11} height={11}/></button>
        </div>
      </div>
    </div>
  );
}

function Field({ label, children, hint }) {
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 4 }}>{label}</div>
      {children}
      {hint && <div className="faint" style={{ fontSize: 11, marginTop: 3 }}>{hint}</div>}
    </div>
  );
}

function Input({ value, ...rest }) {
  return (
    <input
      defaultValue={value}
      style={{
        width: '100%',
        height: 28, padding: '0 8px',
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4,
        fontSize: 12.5, color: 'var(--fg)',
      }}
      {...rest}
    />
  );
}

function ProfileTab({ user, onDelete, onRefreshList }) {
  const [nameVal, setNameVal] = React.useState(user.name || '');
  const [emailVal, setEmailVal] = React.useState(user.email || '');
  const [saving, setSaving] = React.useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      const patch = {};
      if (nameVal !== (user.name || '')) patch.name = nameVal;
      if (emailVal !== (user.email || '')) patch.email = emailVal;
      if (Object.keys(patch).length > 0) {
        await API.patch('/users/' + user.id, patch);
        if (onRefreshList) onRefreshList();
      }
    } catch (e) {
      console.error('Save failed:', e);
    } finally {
      setSaving(false);
    }
  };

  const isVerified = user.email_verified !== undefined ? user.email_verified : user.verified;
  const metadata = user.metadata ? JSON.stringify(user.metadata, null, 2) : '{\n  \n}';

  return (
    <>
      <Field label="Name">
        <Input value={nameVal} onChange={e => setNameVal(e.target.value)}/>
      </Field>
      <Field label="Email">
        <div className="row" style={{ gap: 6 }}>
          <Input value={emailVal} onChange={e => setEmailVal(e.target.value)}/>
          {!isVerified && <button className="btn sm" style={{ whiteSpace: 'nowrap' }}>Send verification</button>}
        </div>
      </Field>
      <Field label="User ID"><CopyField value={user.id} truncate={0}/></Field>
      <Field label="Metadata (json)">
        <textarea
          defaultValue={metadata}
          style={{
            width: '100%', minHeight: 90,
            background: 'var(--surface-1)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 4, padding: 8,
            fontFamily: 'var(--font-mono)', fontSize: 11.5,
            color: 'var(--fg)', resize: 'vertical',
          }}
        />
      </Field>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <button className="btn primary sm" onClick={handleSave} disabled={saving}>
          {saving ? 'Saving…' : 'Save changes'}
        </button>
      </div>
      <div style={{
        marginTop: 16, padding: 12,
        border: '1px solid color-mix(in oklch, var(--danger) 30%, var(--hairline))',
        borderRadius: 5,
        background: 'color-mix(in oklch, var(--danger) 5%, var(--surface-1))',
      }}>
        <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--danger)', marginBottom: 8, fontWeight: 500 }}>Danger zone</div>
        <div className="row" style={{ gap: 8 }}>
          <button className="btn sm">Reset password</button>
          <button className="btn sm">Disable MFA</button>
          <button className="btn sm danger" onClick={() => {
            if (confirm(`Delete ${user.email}? This revokes all sessions and agent consents.`)) {
              onDelete && onDelete(user.id);
            }
          }}>Delete user</button>
        </div>
        <div className="faint" style={{ fontSize: 10.5, marginTop: 6 }}>Deleting requires confirmation. Revokes all sessions + agent consents in one transaction.</div>
      </div>
    </>
  );
}

function SecurityTab({ user }) {
  const { data: sessionsData, refresh: refreshSessions } = useAPI(
    user ? `/users/${user.id}/sessions` : null
  );
  const sessions = sessionsData?.sessions || sessionsData || [];

  const handleRevokeSession = async (sessionId) => {
    await API.del('/admin/sessions/' + sessionId);
    refreshSessions();
  };

  const handleRevokeAll = async () => {
    await API.del('/users/' + user.id + '/sessions');
    refreshSessions();
  };

  const mfaLabel = user.mfa_method || user.mfa;

  return (
    <>
      <Field label="Multi-factor">
        {mfaLabel ? (
          <div style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
            <div className="row">
              <Icon.Shield width={14} height={14} style={{color:'var(--success)'}}/>
              <span style={{ fontSize: 12.5, fontWeight: 500 }}>{mfaLabel === 'totp' ? 'TOTP authenticator' : mfaLabel === 'webauthn' ? 'WebAuthn' : mfaLabel}</span>
              <span className="faint" style={{ fontSize: 11, marginLeft: 'auto' }}>Enrolled</span>
            </div>
          </div>
        ) : <span className="faint">Not enrolled</span>}
      </Field>

      <div style={{ marginTop: 20 }}>
        <Field label="Active sessions" hint={`${Array.isArray(sessions) ? sessions.length : 0} sessions · revoke all on sign-out`}>
          <div className="col" style={{ gap: 6 }}>
            {Array.isArray(sessions) && sessions.length === 0 && (
              <span className="faint" style={{ fontSize: 12 }}>No active sessions</span>
            )}
            {Array.isArray(sessions) && sessions.map(s => (
              <div key={s.id} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
                <div className="row">
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 12.5 }}>{s.device || s.user_agent || s.id}</div>
                    <div className="faint mono" style={{ fontSize: 10.5 }}>
                      {[s.ip, s.location || s.city, s.created_at ? 'started ' + MOCK.relativeTime(new Date(s.created_at).getTime()) : ''].filter(Boolean).join(' · ')}
                    </div>
                  </div>
                  {(s.mfa || s.mfa_method) && <span className="chip success" style={{ height: 18 }}>mfa</span>}
                  <button className="btn ghost sm danger" onClick={() => handleRevokeSession(s.id)}>Revoke</button>
                </div>
              </div>
            ))}
            {Array.isArray(sessions) && sessions.length > 0 && (
              <button className="btn sm danger" style={{ alignSelf: 'flex-start', marginTop: 4 }} onClick={handleRevokeAll}>
                Revoke all sessions
              </button>
            )}
          </div>
        </Field>
      </div>
    </>
  );
}

function RolesTab({ user }) {
  const { data: rolesData, refresh: refreshRoles } = useAPI(
    user ? `/users/${user.id}/roles` : null
  );
  const roles = rolesData?.roles || rolesData || user.roles || [];

  const handleRemoveRole = async (roleId) => {
    await API.del('/users/' + user.id + '/roles/' + roleId);
    refreshRoles();
  };

  const perms = ['users:read', 'users:write', 'agents:manage', 'billing:read', 'audit:export'];

  return (
    <>
      <Field label="Global roles">
        <div className="row" style={{ gap: 6, flexWrap: 'wrap' }}>
          {(Array.isArray(roles) ? roles : []).map((r, i) => {
            const name = typeof r === 'string' ? r : (r.name || r.id);
            const id = typeof r === 'string' ? r : r.id;
            return (
              <span key={i} className="chip solid">{name}
                <Icon.X width={10} height={10} style={{ opacity: 0.5, cursor: 'pointer' }}
                  onClick={() => handleRemoveRole(id)}/>
              </span>
            );
          })}
          <button className="btn sm"><Icon.Plus width={10} height={10}/>Assign role</button>
        </div>
      </Field>
      <Field label="Effective permissions" hint="Flattened from all roles">
        <div className="col" style={{ gap: 4 }}>
          {perms.map(p => (
            <div key={p} className="row" style={{ padding: '5px 8px', background: 'var(--surface-1)', borderRadius: 3 }}>
              <Icon.Check width={11} height={11} style={{ color: 'var(--success)' }}/>
              <span className="mono" style={{ fontSize: 11.5 }}>{p}</span>
              <span className="faint" style={{ marginLeft: 'auto', fontSize: 10.5 }}>from <span className="mono">admin</span></span>
            </div>
          ))}
        </div>
      </Field>
      <Field label="Check permission" hint="Simulate an authorization check for this user">
        <div className="row" style={{ gap: 6 }}>
          <Input value="agents:manage" />
          <button className="btn">Check</button>
        </div>
      </Field>
    </>
  );
}

function OrgsTab({ user }) {
  const orgs = user.orgs || [];
  return (
    <div className="col" style={{ gap: 6 }}>
      {orgs.map((o, i) => {
        const name = typeof o === 'string' ? o : (o.name || o.slug || o.id);
        const role = typeof o === 'object' ? (o.role || 'member') : 'member';
        return (
          <div key={i} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
            <div className="row">
              <div className="avatar" style={{ width: 24, height: 24, fontSize: 10, background: hashColor(name) }}>{name[0]}</div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 12.5, fontWeight: 500 }}>{name}</div>
                <div className="faint mono" style={{ fontSize: 10.5 }}>member</div>
              </div>
              <span className="chip">{role}</span>
              <button className="btn ghost sm">Remove</button>
            </div>
          </div>
        );
      })}
      {orgs.length === 0 && <span className="faint" style={{ fontSize: 12 }}>Not a member of any orgs</span>}
      <button className="btn sm" style={{ alignSelf: 'flex-start', marginTop: 4 }}><Icon.Plus width={10} height={10}/>Invite to org</button>
    </div>
  );
}

function AgentsConsentsTab({ user }) {
  const consents = [
    { agent: 'Cursor', scopes: ['openid','profile','email','repos:read'], granted: '12d ago', tokens: 3 },
    { agent: 'Claude CLI', scopes: ['openid','profile','workspace:read','workspace:write'], granted: '4d ago', tokens: 2 },
    { agent: 'Zapier MCP', scopes: ['workspace:read','webhooks:rw'], granted: '2mo ago', tokens: 1 },
  ];
  return (
    <>
      <div style={{
        padding: 10, marginBottom: 12,
        border: '1px solid var(--hairline)',
        borderRadius: 5,
        background: 'var(--surface-1)',
        fontSize: 11.5, color: 'var(--fg-muted)',
      }}>
        <Icon.Info width={12} height={12} style={{ verticalAlign: 'middle', marginRight: 6, opacity: 0.7 }}/>
        User has granted {consents.length} agents access to their account. Revoking a consent invalidates all tied tokens.
      </div>
      <div className="col" style={{ gap: 6 }}>
        {consents.map((c, i) => (
          <div key={i} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
            <div className="row" style={{ marginBottom: 6 }}>
              <Avatar name={c.agent} agent size={22}/>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 12.5, fontWeight: 500 }}>{c.agent}</div>
                <div className="faint mono" style={{ fontSize: 10.5 }}>granted {c.granted} · {c.tokens} live tokens</div>
              </div>
              <button className="btn ghost sm danger">Revoke</button>
            </div>
            <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
              {c.scopes.map(s => <span key={s} className="chip mono" style={{ height: 17, fontSize: 10 }}>{s}</span>)}
            </div>
          </div>
        ))}
      </div>
    </>
  );
}

function ActivityTab({ user }) {
  const { data: activityData, loading } = useAPI(
    user ? `/users/${user.id}/audit-logs` : null
  );
  const events = activityData?.events || activityData?.logs || activityData || [];

  if (loading) {
    return <div className="faint" style={{ fontSize: 12, padding: '16px 0' }}>Loading activity…</div>;
  }

  if (!Array.isArray(events) || events.length === 0) {
    return <div className="faint" style={{ fontSize: 12, padding: '16px 0' }}>No activity recorded</div>;
  }

  return (
    <div className="col" style={{ gap: 0 }}>
      {events.map((e, i) => {
        const ts = e.created_at ? new Date(e.created_at).getTime() : e.t;
        return (
          <div key={i} className="row" style={{ padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 10 }}>
            <span className="mono faint" style={{ fontSize: 10.5, width: 70 }}>{ts ? MOCK.relativeTime(ts) : '—'}</span>
            <span className="mono" style={{ fontSize: 11.5, width: 180 }}>{e.action || e.event_type || '—'}</span>
            <span className="mono faint" style={{ fontSize: 11 }}>{e.meta || e.metadata || e.details || ''}</span>
          </div>
        );
      })}
    </div>
  );
}

