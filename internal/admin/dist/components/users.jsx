// Users page — table + detail slide-over

function Users() {
  const [selected, setSelected] = React.useState(null);
  const [query, setQuery] = React.useState('');
  const [checked, setChecked] = React.useState(new Set());

  const filtered = MOCK.users.filter(u =>
    !query || u.name.toLowerCase().includes(query.toLowerCase()) || u.email.toLowerCase().includes(query.toLowerCase())
  );

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
          <span className="faint" style={{ fontSize: 11 }}>{filtered.length.toLocaleString()} of {MOCK.users.length.toLocaleString()}</span>
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
                <th style={{ width: 28, paddingLeft: 16 }}>
                  <span className={"cb" + (checked.size === filtered.length ? ' on' : '')}
                    onClick={() => setChecked(checked.size === filtered.length ? new Set() : new Set(filtered.map(x => x.id)))}>
                    {checked.size === filtered.length && <Icon.Check width={10} height={10}/>}
                  </span>
                </th>
                <th>User</th>
                <th style={{ width: 90 }}>Verified</th>
                <th style={{ width: 70 }}>MFA</th>
                <th style={{ width: 120 }}>Auth</th>
                <th style={{ width: 140 }}>Orgs</th>
                <th style={{ width: 180 }}>Roles</th>
                <th style={{ width: 100 }}>Created <Icon.ArrowDown width={9} height={9} style={{opacity:0.5, verticalAlign:'middle'}}/></th>
                <th style={{ width: 110 }}>Last active</th>
                <th style={{ width: 36 }}></th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(u => (
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
                      <Avatar name={u.name} email={u.email}/>
                      <div style={{ minWidth: 0 }}>
                        <div style={{ fontWeight: 500, fontSize: 12.5 }}>{u.name}</div>
                        <div className="faint" style={{ fontSize: 11 }}>{u.email}</div>
                      </div>
                    </div>
                  </td>
                  <td>
                    {u.verified ? (
                      <span className="chip success" style={{ height: 18 }}><Icon.Check width={9} height={9}/>verified</span>
                    ) : (
                      <span className="chip warn" style={{ height: 18 }}>pending</span>
                    )}
                  </td>
                  <td>
                    {u.mfa ? (
                      <span className="row" style={{ gap: 4, color: 'var(--success)' }}>
                        <Icon.Shield width={12} height={12}/>
                        <span className="mono" style={{ fontSize: 10.5, color: 'var(--fg-muted)' }}>{u.mfa}</span>
                      </span>
                    ) : (
                      <span className="faint" style={{ fontSize: 11 }}>—</span>
                    )}
                  </td>
                  <td>
                    <span className="chip" style={{ height: 18 }}>{u.method}</span>
                  </td>
                  <td>
                    <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                      {u.orgs.slice(0, 2).map(o => <span key={o} className="chip" style={{ height: 18 }}>{o}</span>)}
                      {u.orgs.length > 2 && <span className="faint" style={{ fontSize: 11 }}>+{u.orgs.length - 2}</span>}
                    </div>
                  </td>
                  <td>
                    <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                      {u.roles.slice(0, 2).map(r => <span key={r} className="chip" style={{ height: 18 }}>{r}</span>)}
                      {u.roles.length > 2 && <span className="faint" style={{ fontSize: 11 }}>+{u.roles.length - 2}</span>}
                    </div>
                  </td>
                  <td className="mono faint" style={{ fontSize: 11 }}>{MOCK.relativeTime(u.created)}</td>
                  <td className="mono" style={{ fontSize: 11, color: NOW - u.lastActive < 60*60*1000 ? 'var(--success)' : 'var(--fg-muted)' }}>
                    {MOCK.relativeTime(u.lastActive)}
                  </td>
                  <td onClick={e => e.stopPropagation()}>
                    <button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {selected && <UserSlideover user={selected} onClose={() => setSelected(null)}/>}
    </div>
  );
}

function UserSlideover({ user, onClose }) {
  const [tab, setTab] = React.useState('profile');
  const tabs = [
    { id: 'profile', label: 'Profile' },
    { id: 'security', label: 'Security' },
    { id: 'rbac', label: 'Roles' },
    { id: 'orgs', label: 'Orgs' },
    { id: 'agents', label: 'Agents', chip: '3' },
    { id: 'activity', label: 'Activity' },
  ];

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
          <Avatar name={user.name} email={user.email} size={44}/>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 16, fontWeight: 500, letterSpacing: '-0.01em' }}>{user.name}</div>
            <div className="row" style={{ gap: 6, marginTop: 2 }}>
              <span className="faint" style={{ fontSize: 12 }}>{user.email}</span>
              <CopyField value={user.id}/>
            </div>
          </div>
        </div>
        <div className="row" style={{ gap: 6, marginTop: 12 }}>
          {user.verified ? <span className="chip success"><Icon.Check width={9} height={9}/>verified</span> : <span className="chip warn">pending</span>}
          {user.mfa && <span className="chip"><Icon.Shield width={10} height={10}/>mfa · {user.mfa}</span>}
          <span className="chip">{user.method}</span>
          <span className="chip">{user.orgs.length} org{user.orgs.length !== 1 && 's'}</span>
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
        {tab === 'profile' && <ProfileTab user={user}/>}
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

function ProfileTab({ user }) {
  return (
    <>
      <Field label="Name"><Input value={user.name}/></Field>
      <Field label="Email">
        <div className="row" style={{ gap: 6 }}>
          <Input value={user.email}/>
          {!user.verified && <button className="btn sm" style={{ whiteSpace: 'nowrap' }}>Send verification</button>}
        </div>
      </Field>
      <Field label="User ID"><CopyField value={user.id} truncate={0}/></Field>
      <Field label="Metadata (json)">
        <textarea
          defaultValue={'{\n  "plan": "pro",\n  "team": "nimbus-core",\n  "onboarded": true\n}'}
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
          <button className="btn sm danger">Delete user</button>
        </div>
        <div className="faint" style={{ fontSize: 10.5, marginTop: 6 }}>Deleting requires typing {user.email}. Revokes all sessions + agent consents in one transaction.</div>
      </div>
    </>
  );
}

function SecurityTab({ user }) {
  return (
    <>
      <Field label="Multi-factor">
        {user.mfa ? (
          <div style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
            <div className="row">
              <Icon.Shield width={14} height={14} style={{color:'var(--success)'}}/>
              <span style={{ fontSize: 12.5, fontWeight: 500 }}>{user.mfa === 'totp' ? 'TOTP authenticator' : 'WebAuthn'}</span>
              <span className="faint" style={{ fontSize: 11, marginLeft: 'auto' }}>Enrolled 4mo ago</span>
            </div>
          </div>
        ) : <span className="faint">Not enrolled</span>}
      </Field>

      <Field label="Passkeys" hint="FIDO2 credentials registered to this user">
        <div className="col" style={{ gap: 6 }}>
          {MOCK.passkeys.map(p => (
            <div key={p.id} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
              <div className="row">
                <Icon.Key width={13} height={13}/>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 12.5 }}>{p.name}</div>
                  <div className="faint mono" style={{ fontSize: 10.5 }}>created {MOCK.relativeTime(p.created)} · used {MOCK.relativeTime(p.lastUsed)}</div>
                </div>
                <button className="btn ghost sm">Rename</button>
                <button className="btn ghost sm danger">Delete</button>
              </div>
            </div>
          ))}
        </div>
      </Field>

      <Field label="OAuth accounts">
        <div className="col" style={{ gap: 6 }}>
          {MOCK.oauthAccounts.map((o, i) => (
            <div key={i} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
              <div className="row">
                <span className="mono" style={{ fontSize: 11, textTransform: 'uppercase' }}>{o.provider}</span>
                <span className="faint" style={{ fontSize: 11 }}>{o.email || o.username}</span>
                <span className="faint mono" style={{ fontSize: 10.5, marginLeft: 'auto' }}>linked {MOCK.relativeTime(o.linked)}</span>
                <button className="btn ghost sm">Unlink</button>
              </div>
            </div>
          ))}
        </div>
      </Field>

      <Field label="Active sessions" hint="3 sessions · revoke all on sign-out">
        <div className="col" style={{ gap: 6 }}>
          {MOCK.activeSessions.map(s => (
            <div key={s.id} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
              <div className="row">
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 12.5 }}>{s.device}</div>
                  <div className="faint mono" style={{ fontSize: 10.5 }}>{s.ip} · {s.location} · started {MOCK.relativeTime(s.created)}</div>
                </div>
                {s.mfa && <span className="chip success" style={{ height: 18 }}>mfa</span>}
                <button className="btn ghost sm danger">Revoke</button>
              </div>
            </div>
          ))}
          <button className="btn sm danger" style={{ alignSelf: 'flex-start', marginTop: 4 }}>Revoke all sessions</button>
        </div>
      </Field>
    </>
  );
}

function RolesTab({ user }) {
  const perms = ['users:read', 'users:write', 'agents:manage', 'billing:read', 'audit:export'];
  return (
    <>
      <Field label="Global roles">
        <div className="row" style={{ gap: 6, flexWrap: 'wrap' }}>
          {user.roles.map(r => <span key={r} className="chip solid">{r}<Icon.X width={10} height={10} style={{ opacity: 0.5 }}/></span>)}
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
  return (
    <div className="col" style={{ gap: 6 }}>
      {user.orgs.map(o => (
        <div key={o} style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
          <div className="row">
            <div className="avatar" style={{ width: 24, height: 24, fontSize: 10, background: hashColor(o) }}>{o[0]}</div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12.5, fontWeight: 500 }}>{o}</div>
              <div className="faint mono" style={{ fontSize: 10.5 }}>joined 7mo ago · invited by amelia@nimbus.sh</div>
            </div>
            <span className="chip">{o === 'Nimbus' ? 'owner' : 'member'}</span>
            <button className="btn ghost sm">Remove</button>
          </div>
        </div>
      ))}
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
  const events = [
    { t: mins(4), action: 'user.login', meta: 'passkey · iOS · 73.162.44.11' },
    { t: hrs(6), action: 'user.login', meta: 'password · macOS · 73.162.44.11' },
    { t: days(1), action: 'oauth.consent.granted', meta: 'agent: cursor · 4 scopes' },
    { t: days(1), action: 'session.created', meta: 'sess_a1' },
    { t: days(3), action: 'passkey.registered', meta: 'YubiKey 5C' },
    { t: days(11), action: 'mfa.enabled', meta: 'totp' },
    { t: days(14), action: 'user.created', meta: 'via invite' },
  ];
  return (
    <div className="col" style={{ gap: 0 }}>
      {events.map((e, i) => (
        <div key={i} className="row" style={{ padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 10 }}>
          <span className="mono faint" style={{ fontSize: 10.5, width: 70 }}>{MOCK.relativeTime(e.t)}</span>
          <span className="mono" style={{ fontSize: 11.5, width: 180 }}>{e.action}</span>
          <span className="mono faint" style={{ fontSize: 11 }}>{e.meta}</span>
        </div>
      ))}
    </div>
  );
}

// Helpers pulled in — declare locals so this file stands alone
const mins = (n) => Date.now() - n * 60 * 1000;
const hrs = (n) => Date.now() - n * 60 * 60 * 1000;
const days = (n) => Date.now() - n * 24 * 60 * 60 * 1000;
const NOW = Date.now();

window.Users = Users;
