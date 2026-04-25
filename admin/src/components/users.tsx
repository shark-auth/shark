// @ts-nocheck
import React from 'react'
import { Icon, Avatar, CopyField, Kbd, hashColor } from './shared'
import { API, useAPI } from './api'
import { MOCK } from './mock'
import { CLIFooter } from './CLIFooter'
import { useURLParam } from './useURLParams'
import { useToast } from './toast'
import { usePageActions } from './useKeyboardShortcuts'
import { BILLING_UI } from '../featureFlags'

// Users page — table + detail slide-over

// Helpers pulled in — declare locals so this file stands alone
const mins = (n) => Date.now() - n * 60 * 1000;
const hrs = (n) => Date.now() - n * 60 * 60 * 1000;
const days = (n) => Date.now() - n * 24 * 60 * 60 * 1000;
const NOW = Date.now();

export function Users() {
  const [selected, setSelected] = React.useState(null);
  const [creating, setCreating] = React.useState(false);
  const toast = useToast();
  const [query, setQuery] = useURLParam('search', '');
  const [filterVerified, setFilterVerified] = useURLParam('verified', '');
  const [filterMfa, setFilterMfa] = useURLParam('mfa', '');
  const [filterMethod, setFilterMethod] = useURLParam('method', '');
  const [debouncedQuery, setDebouncedQuery] = React.useState('');
  const [checked, setChecked] = React.useState(new Set());
  const [page, setPage] = React.useState(1);
  const [perPage] = React.useState(25);

  // Debounce search input — 300ms
  React.useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 300);
    return () => clearTimeout(t);
  }, [query]);

  // Reset page to 1 when search or any filter changes
  React.useEffect(() => { setPage(1); }, [debouncedQuery, filterVerified, filterMfa, filterMethod]);

  const params = new URLSearchParams();
  params.set('page', String(page));
  params.set('per_page', String(perPage));
  if (debouncedQuery) params.set('search', debouncedQuery);
  if (filterVerified === 'yes') params.set('email_verified', 'true');
  if (filterVerified === 'no') params.set('email_verified', 'false');
  if (filterMfa === 'on') params.set('mfa_enabled', 'true');
  if (filterMfa === 'off') params.set('mfa_enabled', 'false');
  if (filterMethod) params.set('auth_method', filterMethod);
  const { data: usersData, loading, refresh } = useAPI(`/users?${params.toString()}`);
  const rawUsers = usersData?.users || [];
  const total = usersData?.total || 0;
  const totalPages = Math.ceil(total / perPage) || 1;

  // Deep link: /admin/users/<id> selects that user once data loads.
  React.useEffect(() => {
    const m = window.location.pathname.match(/^\/admin\/users\/([^\/]+)/);
    if (m && rawUsers.length > 0 && !selected) {
      const u = rawUsers.find(x => x.id === m[1]);
      if (u) setSelected(u);
    }
  }, [rawUsers]);

  // r=refresh; n opens the create-user slide-over.
  usePageActions({
    onRefresh: refresh,
    onNew: () => setCreating(true),
  });

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

  const handleCreated = (newUser) => {
    setCreating(false);
    toast.success(`User ${newUser.email} created`);
    refresh();
    setSelected(newUser);
  };

  // Server now applies all filters. Pass-through.
  const users = rawUsers;

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
          {/* Search input — surface-1 bg, hairline-strong border, 28px height */}
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
              style={{ flex: 1, fontSize: 13, color: 'var(--fg)' }}
            />
            <Kbd keys="/"/>
          </div>
          <FilterSelect label="Verified" value={filterVerified} onChange={setFilterVerified} options={[
            { v: '', l: 'All' }, { v: 'yes', l: 'Verified' }, { v: 'no', l: 'Unverified' },
          ]}/>
          <FilterSelect label="MFA" value={filterMfa} onChange={setFilterMfa} options={[
            { v: '', l: 'All' }, { v: 'on', l: 'Enabled' }, { v: 'off', l: 'Disabled' },
          ]}/>
          <FilterSelect label="Auth method" value={filterMethod} onChange={setFilterMethod} options={[
            { v: '', l: 'All' }, { v: 'password', l: 'Password' }, { v: 'oauth', l: 'OAuth' }, { v: 'passkey', l: 'Passkey' }, { v: 'magic_link', l: 'Magic link' },
          ]}/>
          {(filterVerified || filterMfa || filterMethod) && (
            <button className="btn ghost sm" style={{ color: 'var(--fg-dim)' }} onClick={() => { setFilterVerified(''); setFilterMfa(''); setFilterMethod(''); }}>Clear filters</button>
          )}
          <div style={{ flex: 1 }}/>
          <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
            {loading ? '…' : `${total.toLocaleString()} total`}
          </span>
          <button className="btn sm">Export</button>
          <button className="btn primary sm" onClick={() => setCreating(true)}><Icon.Plus width={11} height={11}/>New user</button>
        </div>

        {/* Bulk action bar */}
        {checked.size > 0 && (
          <div style={{
            padding: '6px 16px',
            background: 'var(--surface-2)',
            borderBottom: '1px solid var(--hairline)',
            display: 'flex', alignItems: 'center', gap: 8, fontSize: 13,
          }}>
            <span style={{ fontSize: 13, color: 'var(--fg-muted)', lineHeight: 1.5 }}>{checked.size} selected</span>
            {/* v0.2: Assign role / Add to org require a picker modal — deferred */}
            <button className="btn sm" disabled title="v0.2 — use CLI: shark users assign-role">Assign role</button>
            <button className="btn sm" disabled title="v0.2 — use CLI: shark users add-to-org">Add to org</button>
            <button className="btn sm" onClick={async () => {
              // Export selected users as CSV (client-side from already-loaded list)
              const selectedUsers = users.filter(u => checked.has(u.id));
              const header = 'id,email,name,email_verified,mfa_enabled,created_at';
              const rows = selectedUsers.map(u =>
                [u.id, u.email, u.name || '', u.email_verified ? 'true' : 'false', (u.mfa_enabled || u.mfa) ? 'true' : 'false', u.created_at || ''].join(',')
              );
              const csv = [header, ...rows].join('\n');
              const blob = new Blob([csv], { type: 'text/csv' });
              const url = URL.createObjectURL(blob);
              const a = document.createElement('a');
              a.href = url; a.download = 'users-export.csv'; a.click();
              URL.revokeObjectURL(url);
            }}>Export CSV</button>
            {/* Delete-bulk: CLI-only for safety */}
            <span style={{ width: 1, height: 16, background: 'var(--hairline-strong)', marginLeft: 4 }}/>
            <button className="btn sm danger" disabled title="Bulk delete is CLI-only for safety: shark users delete --ids …">Delete</button>
            <div style={{ flex: 1 }}/>
            <button className="btn ghost sm" onClick={() => setChecked(new Set())}>Clear</button>
          </div>
        )}

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table className="tbl">
            <thead style={{ background: 'var(--surface-0)' }}>
              <tr>
                <th style={{ width: 28, paddingLeft: 16, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>
                  <span className={"cb" + (users.length > 0 && checked.size === users.length ? ' on' : '')}
                    onClick={() => setChecked(checked.size === users.length ? new Set() : new Set(users.map(x => x.id)))}>
                    {users.length > 0 && checked.size === users.length && <Icon.Check width={10} height={10}/>}
                  </span>
                </th>
                <th style={{ position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>User</th>
                <th style={{ width: 90, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Verified</th>
                <th style={{ width: 70, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>MFA</th>
                <th style={{ width: 120, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Auth</th>
                <th style={{ width: 140, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Orgs</th>
                <th style={{ width: 180, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Roles</th>
                <th style={{ width: 100, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Created <Icon.ArrowDown width={9} height={9} style={{opacity:0.5, verticalAlign:'middle'}}/></th>
                <th style={{ width: 110, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Last active</th>
                <th style={{ width: 36, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}></th>
              </tr>
            </thead>
            <tbody>
              {loading && users.length === 0 && (
                <tr><td colSpan={10} style={{ padding: 32, textAlign: 'center', color: 'var(--fg-muted)', fontSize: 13 }}>Loading…</td></tr>
              )}
              {!loading && users.length === 0 && (
                <tr><td colSpan={10} style={{ padding: 32, textAlign: 'center', color: 'var(--fg-muted)', fontSize: 13 }}>No users found</td></tr>
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
                          <div style={{ fontWeight: 500, fontSize: 13 }}>{u.name || '—'}</div>
                          <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{u.email}</div>
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
                          <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{u.mfa_method || u.mfa || 'on'}</span>
                        </span>
                      ) : (
                        <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>—</span>
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
                        {orgs.length > 2 && <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>+{orgs.length - 2}</span>}
                      </div>
                    </td>
                    <td>
                      <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                        {roles.slice(0, 2).map((r, i) => {
                          const name = typeof r === 'string' ? r : (r.name || r.id);
                          return <span key={i} className="chip" style={{ height: 18 }}>{name}</span>;
                        })}
                        {roles.length > 2 && <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>+{roles.length - 2}</span>}
                      </div>
                    </td>
                    <td className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
                      {u.created_at ? MOCK.relativeTime(new Date(u.created_at).getTime()) : (u.created ? MOCK.relativeTime(u.created) : '—')}
                    </td>
                    <td className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: (() => {
                      const ts = u.last_login_at ? new Date(u.last_login_at).getTime() : null;
                      return ts && (Date.now() - ts < 60*60*1000) ? 'var(--success)' : 'var(--fg-muted)';
                    })() }}>
                      {(() => {
                        const ts = u.last_login_at ? new Date(u.last_login_at).getTime() : null;
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
          display: 'flex', alignItems: 'center', gap: 6,
          background: 'var(--surface-0)',
        }}>
          <button
            className="btn ghost sm"
            disabled={page <= 1}
            onClick={() => setPage(p => Math.max(1, p - 1))}
          >← Prev</button>
          <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)', padding: '0 6px' }}>Page {page} of {totalPages}</span>
          <button
            className="btn ghost sm"
            disabled={page >= totalPages}
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
          >Next →</button>
          <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-dim)', marginLeft: 8 }}>{total.toLocaleString()} total</span>
        </div>
      </div>

      {selected && (
        <UserSlideover
          user={selected}
          onClose={() => {
            setSelected(null);
            const search = window.location.search;
            window.history.replaceState(null, '', '/admin/users' + search);
          }}
          onDelete={handleDelete}
          onRefreshList={refresh}
        />
      )}

      {creating && (
        <CreateUserSlideover
          onClose={() => setCreating(false)}
          onCreated={handleCreated}
        />
      )}
    </div>
  );
}

// CreateUserSlideover — T05 — right-side panel to create a new user via
// POST /admin/users. Matches UserSlideover styling (540px wide, surface-0,
// hairline border, 140ms slideIn). ESC + backdrop-click close with a dirty
// check. Submits: email (required, HTML5 validated), password (optional, min
// 12 chars if provided, reveal toggle), name, email_verified, role_ids,
// org_ids (multiselect). 409 highlights email; 400 shows inline errors.
function CreateUserSlideover({ onClose, onCreated }) {
  const toast = useToast();
  const [email, setEmail] = React.useState('');
  const [password, setPassword] = React.useState('');
  const [showPassword, setShowPassword] = React.useState(false);
  const [name, setName] = React.useState('');
  const [emailVerified, setEmailVerified] = React.useState(false);
  const [roleIds, setRoleIds] = React.useState([]);
  const [orgIds, setOrgIds] = React.useState([]);
  const [submitting, setSubmitting] = React.useState(false);
  const [fieldErrors, setFieldErrors] = React.useState({}); // {email, password, name, form}
  const emailRef = React.useRef(null);

  const { data: rolesData } = useAPI('/roles');
  const { data: orgsData } = useAPI('/admin/organizations');
  const roles = rolesData?.roles || rolesData?.data || (Array.isArray(rolesData) ? rolesData : []);
  const orgs = orgsData?.organizations || orgsData?.data || (Array.isArray(orgsData) ? orgsData : []);

  const dirty = email !== '' || password !== '' || name !== '' || emailVerified || roleIds.length > 0 || orgIds.length > 0;

  const tryClose = () => {
    if (dirty) {
      if (!window.confirm('Discard this new user?')) return;
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

  // Focus email on mount.
  React.useEffect(() => {
    if (emailRef.current) emailRef.current.focus();
  }, []);

  const toggleIn = (list, id) => list.includes(id) ? list.filter(x => x !== id) : [...list, id];

  const validate = () => {
    const errs = {};
    if (!email.trim()) errs.email = 'Email is required';
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) errs.email = 'Enter a valid email address';
    if (password !== '' && password.length < 12) errs.password = 'Password must be at least 12 characters';
    setFieldErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (submitting) return;
    if (!validate()) return;
    setSubmitting(true);

    const body: any = {
      email: email.trim(),
      email_verified: emailVerified,
    };
    if (password) body.password = password;
    if (name.trim()) body.name = name.trim();
    if (roleIds.length > 0) body.role_ids = roleIds;
    if (orgIds.length > 0) body.org_ids = orgIds;

    // Direct fetch so we can read status code for 409 vs 400 branching.
    try {
      const key = localStorage.getItem('shark_admin_key');
      const res = await fetch('/api/v1/admin/users', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${key}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      });

      if (res.status === 201) {
        const newUser = await res.json();
        onCreated(newUser);
        return;
      }

      const errBody = await res.json().catch(() => ({}));
      const msg = errBody.message || errBody.error || `HTTP ${res.status}`;
      const code = errBody.code || errBody.error;

      if (res.status === 409) {
        setFieldErrors({ email: msg });
        toast.error(msg);
      } else if (res.status === 400) {
        // Heuristic: route message to the most likely offending field.
        const lower = (code || msg).toLowerCase();
        if (lower.includes('password') || lower === 'weak_password') {
          setFieldErrors({ password: msg });
        } else if (lower.includes('email')) {
          setFieldErrors({ email: msg });
        } else {
          setFieldErrors({ form: msg });
        }
        toast.error('Could not create user. Check the form.');
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
      {/* Panel — right-side slide-over, matches UserSlideover styling */}
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
            <span className="faint" style={{ fontSize: 11 }}>POST /admin/users</span>
          </div>
          <div style={{ fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em', fontFamily: 'var(--font-display)', lineHeight: 1.2 }}>
            Create user
          </div>
          <div className="faint" style={{ fontSize: 13, marginTop: 4 }}>
            New account. Password is optional — leave blank to invite via email later.
          </div>
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <Field label="Email">
              <input
                ref={emailRef}
                type="email"
                required
                value={email}
                onChange={e => { setEmail(e.target.value); if (fieldErrors.email) setFieldErrors({ ...fieldErrors, email: undefined }); }}
                placeholder="user@example.com"
                style={{
                  width: '100%', height: 28, padding: '0 8px',
                  background: 'var(--surface-1)',
                  border: '1px solid ' + (fieldErrors.email ? 'var(--danger)' : 'var(--hairline-strong)'),
                  borderRadius: 4, fontSize: 13, color: 'var(--fg)',
                }}
              />
              {fieldErrors.email && (
                <div style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--danger)', marginTop: 3 }}>{fieldErrors.email}</div>
              )}
            </Field>

            <Field label="Password" hint="Optional · min 12 chars if provided">
              <div style={{ position: 'relative' }}>
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={e => { setPassword(e.target.value); if (fieldErrors.password) setFieldErrors({ ...fieldErrors, password: undefined }); }}
                  placeholder="leave blank to invite via email"
                  autoComplete="new-password"
                  style={{
                    width: '100%', height: 28, padding: '0 32px 0 8px',
                    background: 'var(--surface-1)',
                    border: '1px solid ' + (fieldErrors.password ? 'var(--danger)' : 'var(--hairline-strong)'),
                    borderRadius: 4, fontSize: 13, color: 'var(--fg)',
                  }}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(s => !s)}
                  style={{
                    position: 'absolute', right: 4, top: '50%', transform: 'translateY(-50%)',
                    height: 22, padding: '0 6px',
                    background: 'transparent', border: 'none',
                    color: 'var(--fg-muted)', fontSize: 11, cursor: 'pointer',
                  }}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? 'Hide' : 'Show'}
                </button>
              </div>
              {fieldErrors.password && (
                <div style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--danger)', marginTop: 3 }}>{fieldErrors.password}</div>
              )}
            </Field>

            <Field label="Name" hint="Optional">
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                placeholder="Full name"
                style={{
                  width: '100%', height: 28, padding: '0 8px',
                  background: 'var(--surface-1)',
                  border: '1px solid var(--hairline-strong)',
                  borderRadius: 4, fontSize: 13, color: 'var(--fg)',
                }}
              />
            </Field>

            <Field label="Email verified">
              <label className="row" style={{ gap: 8, cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={emailVerified}
                  onChange={e => setEmailVerified(e.target.checked)}
                />
                <span style={{ fontSize: 13, color: 'var(--fg)' }}>
                  Mark email as already verified (skip confirmation email)
                </span>
              </label>
            </Field>

            <Field label="Roles" hint={`${roleIds.length} selected`}>
              <MultiSelect
                items={(Array.isArray(roles) ? roles : []).map(r => ({ id: r.id || r, label: r.name || r.id || String(r) }))}
                selected={roleIds}
                onToggle={(id) => setRoleIds(s => toggleIn(s, id))}
                empty="No roles defined"
              />
            </Field>

            <Field label="Organizations" hint={`${orgIds.length} selected`}>
              <MultiSelect
                items={(Array.isArray(orgs) ? orgs : []).map(o => ({ id: o.id, label: o.name || o.slug || o.id }))}
                selected={orgIds}
                onToggle={(id) => setOrgIds(s => toggleIn(s, id))}
                empty="No organizations yet"
              />
            </Field>

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
            {submitting ? 'Creating…' : 'Create user'}
          </button>
        </div>
      </form>
    </>
  );
}

// MultiSelect — lightweight chip toggle list used by CreateUserSlideover.
function MultiSelect({ items, selected, onToggle, empty }) {
  if (!items || items.length === 0) {
    return <span className="faint" style={{ fontSize: 12 }}>{empty}</span>;
  }
  return (
    <div className="row" style={{ gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
      {items.map(it => {
        const on = selected.includes(it.id);
        return (
          <button
            key={it.id}
            type="button"
            onClick={() => onToggle(it.id)}
            className={'chip' + (on ? ' solid' : '')}
            style={{
              cursor: 'pointer',
              border: '1px solid ' + (on ? 'transparent' : 'var(--hairline-strong)'),
            }}
          >
            {on && <Icon.Check width={9} height={9}/>}
            {it.label}
          </button>
        );
      })}
    </div>
  );
}

function UserSlideover({ user, onClose, onDelete, onRefreshList }) {
  // Initialize tab from URL sub-path /admin/users/<id>/<tab> when present.
  const initialTab = (() => {
    const m = window.location.pathname.match(/^\/admin\/users\/[^\/]+\/([^\/?]+)/);
    return m ? m[1] : 'profile';
  })();
  const [tab, setTabState] = React.useState(initialTab);
  // Keep URL in sync as user clicks tabs (replaceState — no history pollution).
  const setTab = (t) => {
    setTabState(t);
    const search = window.location.search;
    const url = `/admin/users/${user.id}` + (t && t !== 'profile' ? `/${t}` : '') + search;
    window.history.replaceState(null, '', url);
  };
  // When opened (or switching user), sync URL to /admin/users/<id>.
  React.useEffect(() => {
    const search = window.location.search;
    const url = `/admin/users/${user.id}` + (tab && tab !== 'profile' ? `/${tab}` : '') + search;
    if (window.location.pathname !== url.split('?')[0]) {
      window.history.replaceState(null, '', url);
    }
  }, [user.id]);
  const tabs = [
    { id: 'profile', label: 'Profile' },
    { id: 'security', label: 'Security' },
    { id: 'rbac', label: 'Roles' },
    { id: 'orgs', label: 'Orgs' },
    { id: 'agents', label: 'Agents' },
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
            {/* 20px for user name in slide-over header */}
            <div style={{ fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em', fontFamily: 'var(--font-display)', lineHeight: 1.2 }}>{user.name || '—'}</div>
            <div className="row" style={{ gap: 6, marginTop: 4 }}>
              <span className="faint" style={{ fontSize: 13 }}>{user.email}</span>
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

      {/* Tabs — 11px uppercase, active: --fg + bottom border, inactive: --fg-dim */}
      <div style={{
        display: 'flex', gap: 0,
        padding: '0 16px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
      }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '9px 10px',
            fontSize: 11,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-dim)',
            fontWeight: tab === t.id ? 600 : 400,
            borderBottom: tab === t.id ? '1px solid var(--fg)' : '1px solid transparent',
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

      <CLIFooter command={`shark user show ${user.id}`}/>
    </div>
  );
}

function Field({ label, children, hint }) {
  return (
    <div style={{ marginBottom: 0 }}>
      <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 4, lineHeight: 1.5 }}>{label}</div>
      {children}
      {hint && <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>{hint}</div>}
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
        fontSize: 13, color: 'var(--fg)',
      }}
      {...rest}
    />
  );
}

function TierSection({ user, onUpdated }) {
  const TIERS = ['free', 'pro'];
  const current = user.metadata?.tier || user.tier || 'free';
  const [selected, setSelected] = React.useState(current);
  const [saving, setSaving] = React.useState(false);
  const toast = useToast();

  const save = async () => {
    if (selected === current) return;
    setSaving(true);
    try {
      await API.patch('/admin/users/' + user.id + '/tier', { tier: selected });
      toast.success('Tier updated to ' + selected);
      if (onUpdated) onUpdated();
    } catch (e) {
      toast.error(e.message || 'Failed to update tier');
      setSelected(current);
    } finally {
      setSaving(false);
    }
  };

  const upgradeAt = user.metadata?.tier_upgraded_at || user.tier_upgraded_at;

  return (
    <div style={{ padding: 12, background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 5 }}>
      <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 10, fontWeight: 600 }}>Billing tier</div>
      <div className="row" style={{ gap: 8, alignItems: 'center' }}>
        <div className="seg" style={{ display: 'flex' }}>
          {TIERS.map(t => (
            <button key={t} type="button"
              className={'seg-btn' + (selected === t ? ' on' : '')}
              style={{
                padding: '4px 14px', fontSize: 12,
                background: selected === t ? 'var(--fg)' : 'transparent',
                color: selected === t ? 'var(--bg)' : 'var(--fg-muted)',
                border: '1px solid var(--hairline-strong)',
                borderRadius: 4, marginRight: -1, cursor: 'pointer',
                fontWeight: selected === t ? 600 : 400,
              }}
              onClick={() => setSelected(t)}
            >
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
        </div>
        {selected !== current && (
          <button className="btn primary sm" disabled={saving} onClick={save}>
            {saving ? 'Saving…' : 'Apply'}
          </button>
        )}
        {upgradeAt && (
          <span className="faint mono" style={{ fontSize: 10 }}>
            Last changed {new Date(upgradeAt).toLocaleDateString()}
          </span>
        )}
      </div>
      <div className="faint" style={{ fontSize: 11, marginTop: 6 }}>
        Tokens carry tier at issue time — force a session logout for immediate cutover.
      </div>
    </div>
  );
}

function ProfileTab({ user, onDelete, onRefreshList }) {
  const [nameVal, setNameVal] = React.useState(user.name || '');
  const [emailVal, setEmailVal] = React.useState(user.email || '');
  const [saving, setSaving] = React.useState(false);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = React.useState(false);
  const [deleteInput, setDeleteInput] = React.useState('');
  const toast = useToast();

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

  const handleSendVerification = async () => {
    try {
      await API.post(`/users/${user.id}/verify/send`);
      toast.success('Verification email sent');
    } catch (e) {
      toast.error(e.message || 'Failed to send verification email');
    }
  };

  const isVerified = user.email_verified !== undefined ? user.email_verified : user.verified;
  const metadata = user.metadata ? JSON.stringify(user.metadata, null, 2) : '{\n  \n}';

  return (
    <>
      {/* Identity fields — tight 8px gap */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <Field label="Name">
          <Input value={nameVal} onChange={e => setNameVal(e.target.value)}/>
        </Field>
        <Field label="Email">
          <div className="row" style={{ gap: 6 }}>
            <Input value={emailVal} onChange={e => setEmailVal(e.target.value)}/>
            {!isVerified && (
              <button
                className="btn sm"
                style={{ whiteSpace: 'nowrap' }}
                onClick={handleSendVerification}
              >
                Send verification
              </button>
            )}
          </div>
        </Field>
        <Field label="User ID"><CopyField value={user.id} truncate={0}/></Field>
      </div>

      {/* Tier — billing tier management. Hidden pre-launch (BILLING_UI flag).
          Backend tier endpoints remain wired; UI returns post-Stripe. */}
      {BILLING_UI && (
        <div style={{ marginTop: 20 }}>
          <TierSection user={user} onUpdated={onRefreshList}/>
        </div>
      )}

      {/* Metadata — separated section */}
      <div style={{ marginTop: 24 }}>
        <Field label="Metadata (json)">
          <textarea
            defaultValue={metadata}
            style={{
              width: '100%', minHeight: 90,
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4, padding: 8,
              fontFamily: 'var(--font-mono)', fontSize: 11,
              lineHeight: 1.5,
              color: 'var(--fg)', resize: 'vertical',
            }}
          />
        </Field>
      </div>

      <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16, marginBottom: 0 }}>
        <button className="btn primary sm" onClick={handleSave} disabled={saving}>
          {saving ? 'Saving…' : 'Save changes'}
        </button>
      </div>

      {/* Danger zone — separated with 24px */}
      <div style={{
        marginTop: 24, padding: 12,
        border: '1px solid color-mix(in oklch, var(--danger) 30%, var(--hairline))',
        borderRadius: 5,
        background: 'color-mix(in oklch, var(--danger) 5%, var(--surface-1))',
      }}>
        <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--danger)', marginBottom: 8, fontWeight: 600, lineHeight: 1.5 }}>Danger zone</div>
        <div className="row" style={{ gap: 8 }}>
          <button className="btn sm" onClick={async () => {
            try { await API.post('/auth/password/send-reset-link', { email: user.email }); toast.success('Password reset email sent'); }
            catch (e) { toast.error(e.message || 'Failed to send reset'); }
          }}>Reset password</button>
          <button className="btn sm" onClick={async () => {
            try { await API.del('/users/' + user.id + '/mfa'); toast.success('MFA disabled'); onRefreshList && onRefreshList(); }
            catch (e) { toast.error(e.message || 'Failed to disable MFA'); }
          }} disabled={!user.mfa_enabled && !user.mfa_method && !user.mfa}>Disable MFA</button>
          <button className="btn sm danger" onClick={() => setDeleteConfirmOpen(true)}>Delete user</button>
        </div>
        <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 6 }}>Deleting requires typing user email. Revokes all sessions + agent consents in one transaction.</div>
      </div>

      {deleteConfirmOpen && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 50,
          background: 'rgba(0,0,0,0.7)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={() => { setDeleteConfirmOpen(false); setDeleteInput(''); }}>
          <div className="card" style={{ width: 400, padding: 20, animation: 'slideIn 120ms ease-out' }}
            onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
              <div style={{ width: 28, height: 28, borderRadius: 6, background: 'var(--danger-bg)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Icon.Warn width={14} height={14} style={{ color: 'var(--danger)' }}/>
              </div>
              <div>
                <div style={{ fontWeight: 600, fontSize: 13 }}>Delete user</div>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 1 }}>This action is permanent and cannot be undone.</div>
              </div>
            </div>
            <div style={{ fontSize: 12, color: 'var(--fg-muted)', marginBottom: 12, lineHeight: 1.5 }}>
              Type <span className="mono" style={{ color: 'var(--danger)', background: 'var(--danger-bg)', padding: '1px 5px', borderRadius: 3, fontSize: 11 }}>{user.email}</span> to confirm.
            </div>
            <input autoFocus type="text" value={deleteInput} onChange={e => setDeleteInput(e.target.value)}
              placeholder={user.email}
              style={{
                width: '100%', height: 32, padding: '0 10px',
                background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)',
                borderRadius: 5, color: 'var(--fg)', fontSize: 12, marginBottom: 14,
              }}/>
            <div className="row" style={{ justifyContent: 'flex-end', gap: 8 }}>
              <button className="btn sm ghost" onClick={() => { setDeleteConfirmOpen(false); setDeleteInput(''); }}>Cancel</button>
              <button className="btn sm danger"
                disabled={deleteInput !== user.email}
                style={{ opacity: deleteInput !== user.email ? 0.45 : 1 }}
                onClick={() => { setDeleteConfirmOpen(false); setDeleteInput(''); onDelete && onDelete(user.id); }}>
                Delete user
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

function SecurityTab({ user }) {
  const { data: sessionsData, refresh: refreshSessions } = useAPI(
    user ? `/users/${user.id}/sessions` : null
  );
  const sessions = sessionsData?.data || [];

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
              <span style={{ fontSize: 13, fontWeight: 500 }}>{mfaLabel === 'totp' ? 'TOTP authenticator' : mfaLabel === 'webauthn' ? 'WebAuthn' : mfaLabel}</span>
              <span className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginLeft: 'auto' }}>Enrolled</span>
            </div>
          </div>
        ) : <span className="faint" style={{ fontSize: 13 }}>Not enrolled</span>}
      </Field>

      <div style={{ marginTop: 24 }}>
        <Field label="Active sessions" hint={`${Array.isArray(sessions) ? sessions.length : 0} sessions · revoke all on sign-out`}>
          <div className="col" style={{ gap: 6, marginTop: 4 }}>
            {Array.isArray(sessions) && sessions.length === 0 && (
              <span className="faint" style={{ fontSize: 13 }}>No active sessions</span>
            )}
            {Array.isArray(sessions) && sessions.map(s => (
              <div key={s.id} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
                <div className="row">
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 13 }}>{s.device || s.user_agent || s.id}</div>
                    <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>
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

  // Fetch effective permissions — flat deduped list from all assigned roles.
  const { data: permsData } = useAPI(user ? `/users/${user.id}/permissions` : null);
  const perms = Array.isArray(permsData) ? permsData : (permsData?.data || permsData?.permissions || []);

  const handleRemoveRole = async (roleId) => {
    await API.del('/users/' + user.id + '/roles/' + roleId);
    refreshRoles();
  };

  return (
    <>
      <Field label="Global roles">
        <div className="row" style={{ gap: 6, flexWrap: 'wrap', marginTop: 4 }}>
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
      <div style={{ marginTop: 24 }}>
        <Field label="Effective permissions" hint="Flattened from all assigned roles">
          <div className="col" style={{ gap: 4, marginTop: 4 }}>
            {perms.length === 0 && (
              <span className="faint" style={{ fontSize: 12 }}>No permissions via any role</span>
            )}
            {perms.map((p, i) => {
              const label = typeof p === 'string' ? p : (p.action && p.resource ? `${p.action}:${p.resource}` : (p.action || p.id || String(i)));
              return (
                <div key={p.id || i} className="row" style={{ padding: '5px 8px', background: 'var(--surface-1)', borderRadius: 3 }}>
                  <Icon.Check width={11} height={11} style={{ color: 'var(--success)' }}/>
                  <span className="mono" style={{ fontSize: 11, lineHeight: 1.5 }}>{label}</span>
                </div>
              );
            })}
          </div>
        </Field>
      </div>
      <div style={{ marginTop: 24 }}>
        <Field label="Check permission" hint="Simulate an authorization check for this user">
          <div className="row" style={{ gap: 6, marginTop: 4 }}>
            <Input value="agents:manage" />
            <button className="btn">Check</button>
          </div>
        </Field>
      </div>
    </>
  );
}

function OrgsTab({ user }) {
  const toast = useToast();
  const orgs = user.orgs || [];
  const [removing, setRemoving] = React.useState(null);

  const handleRemove = async (orgId, orgName) => {
    if (!confirm(`Remove ${user.email} from "${orgName}"?`)) return;
    setRemoving(orgId);
    try {
      await API.del(`/admin/organizations/${orgId}/members/${user.id}`);
      toast.success(`Removed from ${orgName}`);
      // The user object came from the list — refresh the page-level list so the
      // orgs chip count updates. Since we don't own onRefreshList here, a
      // full page reload is the safest option for now.
      window.location.reload();
    } catch (e) {
      toast.error('Remove failed: ' + (e?.message || 'unknown error'));
    } finally {
      setRemoving(null);
    }
  };

  return (
    <div className="col" style={{ gap: 6 }}>
      {orgs.map((o, i) => {
        const name = typeof o === 'string' ? o : (o.name || o.slug || o.id);
        const orgId = typeof o === 'object' ? (o.id || o.organization_id) : null;
        const role = typeof o === 'object' ? (o.role || 'member') : 'member';
        return (
          <div key={i} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
            <div className="row">
              <div className="avatar" style={{ width: 24, height: 24, fontSize: 11, background: hashColor(name) }}>{name[0]}</div>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 13, fontWeight: 500 }}>{name}</div>
                <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>{role}</div>
              </div>
              <span className="chip">{role}</span>
              {orgId && (
                <button
                  className="btn ghost sm danger"
                  disabled={removing === orgId}
                  onClick={() => handleRemove(orgId, name)}
                >
                  {removing === orgId ? '…' : 'Remove'}
                </button>
              )}
            </div>
          </div>
        );
      })}
      {orgs.length === 0 && <span className="faint" style={{ fontSize: 13 }}>Not a member of any orgs</span>}
      <button className="btn sm" style={{ alignSelf: 'flex-start', marginTop: 4 }}><Icon.Plus width={10} height={10}/>Invite to org</button>
    </div>
  );
}

function AgentsConsentsTab({ user }) {
  const toast = useToast();
  const { data: consentsData, loading, refresh } = useAPI(
    user ? `/admin/oauth/consents?user_id=${user.id}` : null
  );
  const consents = consentsData?.data || consentsData?.consents || [];
  const [revoking, setRevoking] = React.useState(null);

  const handleRevoke = async (consentId, agentName) => {
    if (!confirm(`Revoke access for "${agentName}"? All tied tokens will be invalidated.`)) return;
    setRevoking(consentId);
    try {
      await API.del(`/admin/oauth/consents/${consentId}`);
      toast.success(`Revoked access for ${agentName}`);
      refresh();
    } catch (e) {
      toast.error('Revoke failed: ' + (e?.message || 'unknown error'));
    } finally {
      setRevoking(null);
    }
  };

  if (loading) {
    return <div className="faint" style={{ fontSize: 13, padding: '16px 0' }}>Loading consents…</div>;
  }

  return (
    <>
      <div style={{
        padding: 10, marginBottom: 12,
        border: '1px solid var(--hairline)',
        borderRadius: 5,
        background: 'var(--surface-1)',
        fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)',
      }}>
        <Icon.Info width={12} height={12} style={{ verticalAlign: 'middle', marginRight: 6, opacity: 0.7 }}/>
        User has granted {consents.length} agent{consents.length !== 1 ? 's' : ''} access to their account. Revoking a consent invalidates all tied tokens.
      </div>
      {consents.length === 0 && (
        <div className="faint" style={{ fontSize: 13, padding: '16px 0' }}>No active agent consents.</div>
      )}
      <div className="col" style={{ gap: 6 }}>
        {consents.map((c, i) => {
          const agentName = c.agent_name || c.client_id;
          const scopes = typeof c.scope === 'string' ? c.scope.split(' ') : (c.scopes || []);
          const grantedAt = c.granted_at ? MOCK.relativeTime(new Date(c.granted_at).getTime()) : '—';
          return (
            <div key={c.id || i} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
              <div className="row" style={{ marginBottom: 6 }}>
                <Avatar name={agentName} agent size={22}/>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 500 }}>{agentName}</div>
                  <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>granted {grantedAt}</div>
                </div>
                <button
                  className="btn ghost sm danger"
                  disabled={revoking === c.id}
                  onClick={() => handleRevoke(c.id, agentName)}
                >
                  {revoking === c.id ? '…' : 'Revoke'}
                </button>
              </div>
              <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                {scopes.filter(Boolean).map(s => <span key={s} className="chip mono" style={{ height: 17, fontSize: 11 }}>{s}</span>)}
              </div>
            </div>
          );
        })}
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
    return <div className="faint" style={{ fontSize: 13, padding: '16px 0' }}>Loading activity…</div>;
  }

  if (!Array.isArray(events) || events.length === 0) {
    return <div className="faint" style={{ fontSize: 13, padding: '16px 0' }}>No activity recorded</div>;
  }

  return (
    <div className="col" style={{ gap: 0 }}>
      {events.map((e, i) => {
        const ts = e.created_at ? new Date(e.created_at).getTime() : e.t;
        return (
          <div key={i} className="row" style={{ padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 10 }}>
            <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5, width: 70 }}>{ts ? MOCK.relativeTime(ts) : '—'}</span>
            <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, width: 180 }}>{e.action || e.event_type || '—'}</span>
            <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{e.meta || e.metadata || e.details || ''}</span>
          </div>
        );
      })}
    </div>
  );
}

function FilterSelect({ label, value, onChange, options }) {
  return (
    <div style={{ position: 'relative', display: 'inline-flex' }}>
      <select
        value={value}
        onChange={e => onChange(e.target.value)}
        style={{
          appearance: 'none', WebkitAppearance: 'none',
          height: 28, padding: '0 24px 0 8px',
          background: value ? 'var(--surface-3)' : 'var(--surface-2)',
          border: '1px solid ' + (value ? 'var(--fg-dim)' : 'var(--hairline-strong)'),
          borderRadius: 5, color: 'var(--fg)',
          fontSize: 12, fontWeight: 500, cursor: 'pointer',
          colorScheme: 'dark',
        }}
      >
        {options.map(o => <option key={o.v} value={o.v}>{o.v ? o.l : label + ': All'}</option>)}
      </select>
      <Icon.ChevronDown width={10} height={10} style={{
        position: 'absolute', right: 6, top: '50%', transform: 'translateY(-50%)',
        pointerEvents: 'none', opacity: 0.5,
      }}/>
    </div>
  );
}
