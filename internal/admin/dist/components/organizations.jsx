// Organizations — split-pane browser (list left, persistent detail right)

function Organizations() {
  const [selectedId, setSelectedId] = React.useState(() => localStorage.getItem('org-selected') || MOCK.orgs[0].id);
  const [query, setQuery] = React.useState('');
  const [planFilter, setPlanFilter] = React.useState('all');

  React.useEffect(() => { localStorage.setItem('org-selected', selectedId); }, [selectedId]);

  const filtered = MOCK.orgs.filter(o => {
    if (query && !o.name.toLowerCase().includes(query.toLowerCase()) && !o.slug.includes(query.toLowerCase())) return false;
    if (planFilter !== 'all' && o.plan !== planFilter) return false;
    return true;
  });

  const selected = MOCK.orgs.find(o => o.id === selectedId) || MOCK.orgs[0];

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      {/* LEFT: Org list */}
      <div style={{
        width: 320, flexShrink: 0,
        borderRight: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        display: 'flex', flexDirection: 'column',
      }}>
        {/* Toolbar */}
        <div style={{ padding: '10px 12px', borderBottom: '1px solid var(--hairline)', display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="row" style={{ justifyContent: 'space-between' }}>
            <div className="row" style={{ gap: 6 }}>
              <span style={{ fontSize: 13, fontWeight: 500 }}>Organizations</span>
              <span className="chip" style={{ height: 17, fontSize: 10 }}>{MOCK.orgs.length}</span>
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
            <input placeholder="Search orgs…" value={query} onChange={(e) => setQuery(e.target.value)} style={{ flex: 1, fontSize: 11.5 }}/>
          </div>
          <div className="seg" style={{ display: 'inline-flex', border: '1px solid var(--hairline-strong)', borderRadius: 4, overflow: 'hidden', height: 24 }}>
            {['all', 'scale', 'team', 'starter'].map((p, i) => (
              <button key={p} onClick={() => setPlanFilter(p)}
                style={{ flex: 1, fontSize: 10.5, textTransform: 'capitalize',
                  background: planFilter === p ? 'var(--surface-3)' : 'var(--surface-1)',
                  color: planFilter === p ? 'var(--fg)' : 'var(--fg-muted)',
                  borderRight: i < 3 ? '1px solid var(--hairline)' : 'none',
                }}>{p}</button>
            ))}
          </div>
        </div>

        {/* List */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {filtered.map(o => (
            <OrgListItem key={o.id} org={o} active={selectedId === o.id} onClick={() => setSelectedId(o.id)}/>
          ))}
          {filtered.length === 0 && (
            <div className="faint" style={{ padding: 20, fontSize: 11.5, textAlign: 'center' }}>No orgs match.</div>
          )}
        </div>

        <div style={{ padding: '10px 12px', borderTop: '1px solid var(--hairline)', background: 'var(--surface-1)', fontSize: 10.5 }} className="row">
          <span className="faint">Total members</span>
          <span className="mono" style={{ marginLeft: 'auto' }}>{MOCK.orgs.reduce((a, o) => a + o.members, 0)}</span>
        </div>
      </div>

      {/* RIGHT: Org detail */}
      <div style={{ flex: 1, minWidth: 0, overflowY: 'auto' }}>
        <OrgDetail org={selected}/>
      </div>
    </div>
  );
}

function OrgListItem({ org, active, onClick }) {
  return (
    <div onClick={onClick}
      style={{
        padding: '10px 12px',
        borderBottom: '1px solid var(--hairline)',
        cursor: 'pointer',
        background: active ? 'var(--surface-2)' : 'transparent',
        borderLeft: active ? '2px solid var(--fg)' : '2px solid transparent',
        display: 'flex', gap: 10,
      }}>
      <div style={{
        width: 32, height: 32, borderRadius: 5,
        background: hashColor(org.name),
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 13, fontWeight: 600,
        flexShrink: 0,
      }}>{org.logo}</div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div className="row" style={{ gap: 6, marginBottom: 2 }}>
          <span style={{ fontSize: 12.5, fontWeight: 500 }}>{org.name}</span>
          {org.sso.enabled && <Icon.SSO width={10} height={10} style={{ opacity: 0.6 }}/>}
          {org.tags.includes('trial') && <span className="chip warn" style={{ height: 15, fontSize: 9, padding: '0 4px' }}>trial</span>}
        </div>
        <div className="faint mono" style={{ fontSize: 10.5 }}>{org.slug} · {org.plan}</div>
        <div className="row" style={{ gap: 10, marginTop: 4 }}>
          <span className="row" style={{ gap: 3, fontSize: 10.5 }}>
            <Icon.Users width={9} height={9} style={{ opacity: 0.5 }}/>
            <span className="mono">{org.members}</span>
          </span>
          <span className="row" style={{ gap: 3, fontSize: 10.5 }}>
            <span className="dot success" style={{ width: 4, height: 4 }}/>
            <span className="mono faint">{org.activeNow} now</span>
          </span>
          {org.pendingInvites > 0 && (
            <span className="row" style={{ gap: 3, fontSize: 10.5, color: 'var(--warn)' }}>
              <Icon.Mail width={9} height={9}/>
              <span className="mono">{org.pendingInvites}</span>
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

/* ---------------- DETAIL ---------------- */

function OrgDetail({ org }) {
  const [tab, setTab] = React.useState('overview');
  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'members', label: 'Members', chip: org.members },
    { id: 'invites', label: 'Invites', chip: org.pendingInvites || undefined },
    { id: 'sso', label: 'SSO & SCIM' },
    { id: 'domains', label: 'Domains' },
    { id: 'audit', label: 'Audit' },
  ];

  return (
    <div>
      {/* Header */}
      <div style={{ padding: '18px 20px 14px', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)' }}>
        <div className="row" style={{ gap: 14, alignItems: 'flex-start' }}>
          <div style={{
            width: 56, height: 56, borderRadius: 8,
            background: hashColor(org.name),
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 22, fontWeight: 600,
            flexShrink: 0,
            border: '1px solid var(--hairline-strong)',
          }}>{org.logo}</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="row" style={{ gap: 8, marginBottom: 4 }}>
              <h1 style={{ margin: 0, fontSize: 20, fontWeight: 500, letterSpacing: '-0.01em' }}>{org.name}</h1>
              <CopyField value={org.id}/>
              <span className="chip" style={{ textTransform: 'capitalize' }}>{org.plan}</span>
              {org.tags.map(t => (
                <span key={t} className={'chip' + (t === 'trial' ? ' warn' : t === 'strategic' ? ' success' : '')}>{t}</span>
              ))}
            </div>
            <div className="faint" style={{ fontSize: 12 }}>{org.description}</div>
            <div className="row" style={{ gap: 16, marginTop: 10, fontSize: 11 }}>
              <span className="faint">Owned by <span style={{ color: 'var(--fg)' }}>{org.ownerName}</span> · {org.ownerEmail}</span>
              <span className="faint mono">created {MOCK.relativeTime(org.created)}</span>
            </div>
          </div>
          <div className="row" style={{ gap: 6 }}>
            <button className="btn sm">Impersonate owner</button>
            <button className="btn sm"><Icon.Plus width={10} height={10}/>Invite</button>
            <button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button>
          </div>
        </div>

        {/* Stats strip */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 0, marginTop: 16, border: '1px solid var(--hairline)', borderRadius: 5, overflow: 'hidden', background: 'var(--surface-1)' }}>
          <OrgStat label="Members" value={`${org.members} / ${org.seats}`} sub={`${org.activeNow} active now`} progress={org.members / org.seats}/>
          <OrgStat label="SSO" value={org.sso.enabled ? org.sso.provider : 'off'} sub={org.sso.enabled ? `${org.sso.connection.toUpperCase()}${org.sso.enforced ? ' · enforced' : ' · optional'}` : 'not configured'} good={org.sso.enabled}/>
          <OrgStat label="SCIM" value={org.scim ? 'synced' : 'off'} sub={org.scim ? `last sync ${MOCK.relativeTime(org.sso.lastSync)}` : 'manual provisioning'} good={org.scim}/>
          <OrgStat label="MFA" value={org.mfaRequired ? 'required' : 'optional'} sub={org.mfaRequired ? 'enforced org-wide' : '—'} good={org.mfaRequired}/>
          <OrgStat label="Domains" value={`${org.verifiedDomains} / ${org.domains.length}`} sub="verified" good={org.verifiedDomains === org.domains.length} last/>
        </div>
      </div>

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 2, padding: '0 20px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        position: 'sticky', top: 0, zIndex: 2,
      }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '11px 10px', fontSize: 12.5,
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-muted)',
            fontWeight: tab === t.id ? 500 : 400,
            borderBottom: tab === t.id ? '1.5px solid var(--fg)' : '1.5px solid transparent',
            marginBottom: -1,
          }}>
            {t.label}
            {t.chip != null && <span className="chip" style={{ marginLeft: 6, height: 16, fontSize: 9.5, padding: '0 5px' }}>{t.chip}</span>}
          </button>
        ))}
      </div>

      {/* Content */}
      <div style={{ padding: 20 }}>
        {tab === 'overview' && <OrgOverviewTab org={org}/>}
        {tab === 'members' && <OrgMembersTab org={org}/>}
        {tab === 'invites' && <OrgInvitesTab org={org}/>}
        {tab === 'sso' && <OrgSsoTab org={org}/>}
        {tab === 'domains' && <OrgDomainsTab org={org}/>}
        {tab === 'audit' && <OrgAuditTab org={org}/>}
      </div>
    </div>
  );
}

function OrgStat({ label, value, sub, good, progress, last }) {
  return (
    <div style={{ padding: '10px 14px', borderRight: last ? 'none' : '1px solid var(--hairline)', minWidth: 0 }}>
      <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-dim)', marginBottom: 4 }}>{label}</div>
      <div className="row" style={{ gap: 6 }}>
        <span className="mono" style={{ fontSize: 15, fontWeight: 500, color: good ? 'var(--success)' : 'var(--fg)', textTransform: value === 'off' ? 'none' : undefined }}>{value}</span>
      </div>
      <div className="faint" style={{ fontSize: 10.5, marginTop: 2 }}>{sub}</div>
      {progress != null && (
        <div style={{ marginTop: 6, height: 3, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
          <div style={{ width: `${progress*100}%`, height: '100%', background: progress > 0.9 ? 'var(--warn)' : 'var(--fg)' }}/>
        </div>
      )}
    </div>
  );
}

/* --- Overview tab --- */

function OrgOverviewTab({ org }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 14 }}>
      <div className="col" style={{ gap: 14 }}>
        <Panel title="Membership">
          <MembershipBreakdown org={org}/>
        </Panel>
        <Panel title="Recent activity">
          <OrgAuditList org={org} limit={5}/>
        </Panel>
      </div>
      <div className="col" style={{ gap: 14 }}>
        <Panel title="Quick actions">
          <div className="col" style={{ gap: 6 }}>
            <button className="btn sm" style={{ justifyContent: 'flex-start' }}><Icon.Plus width={11} height={11}/>Invite members</button>
            <button className="btn sm" style={{ justifyContent: 'flex-start' }}><Icon.SSO width={11} height={11}/>Configure SSO</button>
            <button className="btn sm" style={{ justifyContent: 'flex-start' }}><Icon.Globe width={11} height={11}/>Add domain</button>
            <button className="btn sm" style={{ justifyContent: 'flex-start' }}><Icon.Shield width={11} height={11}/>Enforce MFA</button>
            <button className="btn sm danger" style={{ justifyContent: 'flex-start', marginTop: 4 }}>Suspend org</button>
          </div>
        </Panel>
        <Panel title="Billing">
          <div className="col" style={{ gap: 6, fontSize: 11.5 }}>
            <Row label="Plan" val={<span className="chip" style={{ textTransform: 'capitalize' }}>{org.plan}</span>}/>
            <Row label="Seats used" val={<span className="mono">{org.members} / {org.seats}</span>}/>
            <Row label="Renews" val={<span className="mono faint">in 41 days</span>}/>
            <Row label="MRR" val={<span className="mono">${org.plan === 'scale' ? '1,440' : org.plan === 'team' ? '320' : '0'}</span>}/>
          </div>
        </Panel>
      </div>
    </div>
  );
}

function Panel({ title, children, actions }) {
  return (
    <div className="card">
      <div className="card-header">
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
      <span className="faint">{label}</span>
      <span>{val}</span>
    </div>
  );
}

function MembershipBreakdown({ org }) {
  const members = MOCK.orgMembers[org.id] || [];
  const byRole = {};
  members.forEach(m => { byRole[m.role] = (byRole[m.role] || 0) + 1; });
  const mfaCount = members.filter(m => m.mfa).length;
  const ssoCount = members.filter(m => m.sso).length;
  const guestCount = members.filter(m => m.guest).length;

  return (
    <div className="col" style={{ gap: 12 }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
        <MiniStat label="MFA enrolled" value={`${mfaCount}/${members.length}`} pct={mfaCount/Math.max(1, members.length)} good={mfaCount === members.length}/>
        <MiniStat label="SSO-linked" value={`${ssoCount}/${members.length}`} pct={ssoCount/Math.max(1, members.length)}/>
        <MiniStat label="Guests" value={guestCount} pct={guestCount/Math.max(1, members.length)} warn={guestCount > 0}/>
      </div>
      <div className="col" style={{ gap: 4 }}>
        <div className="faint" style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 4 }}>By role</div>
        {Object.entries(byRole).map(([role, n]) => (
          <div key={role} className="row" style={{ gap: 8, fontSize: 11.5 }}>
            <span className="chip" style={{ width: 72, justifyContent: 'center', textTransform: 'capitalize' }}>{role}</span>
            <div style={{ flex: 1, height: 5, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
              <div style={{ width: `${(n/members.length)*100}%`, height: '100%', background: 'var(--fg)' }}/>
            </div>
            <span className="mono" style={{ width: 30, textAlign: 'right' }}>{n}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function MiniStat({ label, value, pct = 0, good, warn }) {
  const color = good ? 'var(--success)' : warn ? 'var(--warn)' : 'var(--fg)';
  return (
    <div style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)' }}>
      <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-dim)', marginBottom: 4 }}>{label}</div>
      <div className="mono" style={{ fontSize: 16, fontWeight: 500, color }}>{value}</div>
      <div style={{ marginTop: 6, height: 3, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
        <div style={{ width: `${pct*100}%`, height: '100%', background: color }}/>
      </div>
    </div>
  );
}

/* --- Members tab --- */

function OrgMembersTab({ org }) {
  const members = MOCK.orgMembers[org.id] || [];
  const [roleFilter, setRoleFilter] = React.useState('all');
  const filtered = roleFilter === 'all' ? members : members.filter(m => m.role === roleFilter);

  const roles = ['all', 'owner', 'admin', 'member', 'auditor'];

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
        <button className="btn sm">Export CSV</button>
        <button className="btn primary sm"><Icon.Plus width={11} height={11}/>Invite member</button>
      </div>

      <div className="card" style={{ overflow: 'hidden' }}>
        <table className="tbl">
          <thead>
            <tr>
              <th style={{ paddingLeft: 16 }}>Member</th>
              <th style={{ width: 100 }}>Role</th>
              <th style={{ width: 140 }}>Groups</th>
              <th style={{ width: 80 }}>MFA</th>
              <th style={{ width: 60 }}>SSO</th>
              <th style={{ width: 110 }}>Joined</th>
              <th style={{ width: 120 }}>Last active</th>
              <th style={{ width: 36 }}></th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(m => (
              <tr key={m.email}>
                <td style={{ paddingLeft: 16 }}>
                  <div className="row" style={{ gap: 8 }}>
                    <Avatar name={m.name} email={m.email}/>
                    <div style={{ minWidth: 0 }}>
                      <div className="row" style={{ gap: 6 }}>
                        <span style={{ fontSize: 12.5, fontWeight: 500 }}>{m.name}</span>
                        {m.guest && <span className="chip warn" style={{ height: 15, fontSize: 9, padding: '0 4px' }}>guest</span>}
                      </div>
                      <div className="faint" style={{ fontSize: 11 }}>{m.email}</div>
                    </div>
                  </div>
                </td>
                <td>
                  <span className={'chip' + (m.role === 'owner' ? ' solid' : '')} style={{ textTransform: 'capitalize' }}>{m.role}</span>
                </td>
                <td>
                  <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
                    {m.groups.length === 0 && <span className="faint" style={{ fontSize: 11 }}>—</span>}
                    {m.groups.slice(0, 2).map(g => <span key={g} className="chip" style={{ height: 17, fontSize: 10 }}>{g}</span>)}
                    {m.groups.length > 2 && <span className="faint" style={{ fontSize: 10.5 }}>+{m.groups.length - 2}</span>}
                  </div>
                </td>
                <td>
                  {m.mfa ? (
                    <span className="row" style={{ gap: 4, color: 'var(--success)' }}>
                      <Icon.Shield width={11} height={11}/>
                      <span className="mono" style={{ fontSize: 10.5, color: 'var(--fg-muted)' }}>{m.mfa}</span>
                    </span>
                  ) : <span className="chip warn" style={{ height: 17, fontSize: 10 }}>none</span>}
                </td>
                <td>{m.sso ? <Icon.Check width={12} height={12} style={{ color: 'var(--success)' }}/> : <span className="faint">—</span>}</td>
                <td className="mono faint" style={{ fontSize: 11 }}>{MOCK.relativeTime(m.joined)}</td>
                <td className="mono" style={{ fontSize: 11, color: Date.now() - m.lastActive < 60*60*1000 ? 'var(--success)' : 'var(--fg-muted)' }}>
                  {MOCK.relativeTime(m.lastActive)}
                </td>
                <td><button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}

/* --- Invites tab --- */

function OrgInvitesTab({ org }) {
  const invites = MOCK.orgInvites[org.id] || [];
  const pending = invites.filter(i => !i.expired);
  const expired = invites.filter(i => i.expired);

  if (invites.length === 0) {
    return (
      <div style={{ padding: 40, textAlign: 'center', border: '1px dashed var(--hairline-strong)', borderRadius: 6, background: 'var(--surface-1)' }}>
        <Icon.Mail width={24} height={24} style={{ opacity: 0.3, marginBottom: 8 }}/>
        <div style={{ fontSize: 13 }}>No pending invites</div>
        <div className="faint" style={{ fontSize: 11, marginTop: 2, marginBottom: 12 }}>Invite members to join {org.name}.</div>
        <button className="btn primary sm"><Icon.Plus width={11} height={11}/>Send invite</button>
      </div>
    );
  }

  return (
    <>
      <div className="row" style={{ marginBottom: 12 }}>
        <span style={{ fontSize: 12, fontWeight: 500 }}>{pending.length} pending</span>
        {expired.length > 0 && <span className="faint" style={{ fontSize: 11 }}>· {expired.length} expired</span>}
        <div style={{ flex: 1 }}/>
        <button className="btn sm">Resend all</button>
        <button className="btn primary sm"><Icon.Plus width={11} height={11}/>New invite</button>
      </div>
      <div className="card" style={{ overflow: 'hidden' }}>
        <table className="tbl">
          <thead>
            <tr>
              <th style={{ paddingLeft: 16 }}>Email</th>
              <th style={{ width: 100 }}>Role</th>
              <th style={{ width: 180 }}>Invited by</th>
              <th style={{ width: 100 }}>Sent</th>
              <th style={{ width: 120 }}>Expires</th>
              <th style={{ width: 140 }}></th>
            </tr>
          </thead>
          <tbody>
            {invites.map((i, idx) => {
              const expiresMs = Math.abs(i.expires) - (Date.now() - i.expires);
              const isExpired = i.expired;
              return (
                <tr key={idx}>
                  <td style={{ paddingLeft: 16 }}>
                    <div className="row" style={{ gap: 8 }}>
                      <Icon.Mail width={13} height={13} style={{ opacity: 0.5 }}/>
                      <span className="mono" style={{ fontSize: 11.5, color: isExpired ? 'var(--fg-dim)' : 'var(--fg)' }}>{i.email}</span>
                    </div>
                  </td>
                  <td><span className="chip" style={{ textTransform: 'capitalize' }}>{i.role}</span></td>
                  <td className="faint" style={{ fontSize: 11 }}>{i.sentBy}</td>
                  <td className="mono faint" style={{ fontSize: 11 }}>{MOCK.relativeTime(i.sent)}</td>
                  <td>
                    {isExpired
                      ? <span className="chip danger" style={{ height: 17, fontSize: 10 }}>expired</span>
                      : <span className="mono faint" style={{ fontSize: 11 }}>{MOCK.relativeTime(i.expires)}</span>}
                  </td>
                  <td>
                    <div className="row" style={{ gap: 4, justifyContent: 'flex-end', paddingRight: 12 }}>
                      {isExpired
                        ? <><button className="btn ghost sm">Resend</button><button className="btn ghost sm danger">Delete</button></>
                        : <><button className="btn ghost sm">Copy link</button><button className="btn ghost sm">Resend</button><button className="btn ghost sm danger">Revoke</button></>}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </>
  );
}

/* --- SSO tab --- */

function OrgSsoTab({ org }) {
  if (!org.sso.enabled) {
    return (
      <div style={{ padding: 40, textAlign: 'center', border: '1px dashed var(--hairline-strong)', borderRadius: 6, background: 'var(--surface-1)' }}>
        <Icon.SSO width={28} height={28} style={{ opacity: 0.3, marginBottom: 8 }}/>
        <div style={{ fontSize: 13 }}>SSO not configured</div>
        <div className="faint" style={{ fontSize: 11, marginTop: 2, marginBottom: 16 }}>Connect an identity provider to enable enterprise sign-in.</div>
        <div className="row" style={{ gap: 6, justifyContent: 'center' }}>
          <button className="btn">Connect Okta</button>
          <button className="btn">Azure AD</button>
          <button className="btn">Google Workspace</button>
          <button className="btn">SAML (generic)</button>
        </div>
      </div>
    );
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
      <Panel title="Identity Provider">
        <div className="col" style={{ gap: 10 }}>
          <div className="row" style={{ padding: 10, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', gap: 10 }}>
            <div style={{ width: 32, height: 32, borderRadius: 4, background: 'var(--surface-3)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 12, fontWeight: 600, textTransform: 'uppercase' }}>
              {org.sso.provider.slice(0, 2)}
            </div>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12.5, fontWeight: 500, textTransform: 'capitalize' }}>{org.sso.provider.replace('-', ' ')}</div>
              <div className="faint mono" style={{ fontSize: 10.5 }}>{org.sso.connection.toUpperCase()} · last sync {MOCK.relativeTime(org.sso.lastSync)}</div>
            </div>
            <span className="chip success">connected</span>
          </div>
          <div className="col" style={{ gap: 6, fontSize: 11.5 }}>
            <Row label="Entity ID" val={<CopyField value={`urn:shark:${org.slug}`} truncate={0}/>}/>
            <Row label="ACS URL" val={<CopyField value={`https://auth.shark.sh/sso/${org.slug}/acs`} truncate={0}/>}/>
            <Row label="Enforcement" val={<span className={org.sso.enforced ? 'chip success' : 'chip'}>{org.sso.enforced ? 'required' : 'optional'}</span>}/>
          </div>
          <div className="row" style={{ gap: 6, marginTop: 4 }}>
            <button className="btn sm">Test login</button>
            <button className="btn sm">Upload metadata</button>
            <button className="btn ghost sm danger" style={{ marginLeft: 'auto' }}>Disconnect</button>
          </div>
        </div>
      </Panel>

      <Panel title="SCIM Provisioning">
        {org.scim ? (
          <div className="col" style={{ gap: 10 }}>
            <div className="row" style={{ padding: 10, border: '1px solid color-mix(in oklch, var(--success) 30%, var(--hairline-strong))', borderRadius: 4, background: 'color-mix(in oklch, var(--success) 6%, var(--surface-1))', gap: 10 }}>
              <Icon.Check width={14} height={14} style={{ color: 'var(--success)' }}/>
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 12.5, fontWeight: 500 }}>SCIM 2.0 active</div>
                <div className="faint mono" style={{ fontSize: 10.5 }}>auto-provisioning · last sync {MOCK.relativeTime(org.sso.lastSync)}</div>
              </div>
              <span className="mono" style={{ fontSize: 11, color: 'var(--success)' }}>{org.members} synced</span>
            </div>
            <div className="col" style={{ gap: 6, fontSize: 11.5 }}>
              <Row label="SCIM endpoint" val={<CopyField value={`https://scim.shark.sh/${org.slug}/v2`} truncate={0}/>}/>
              <Row label="Bearer token" val={<CopyField value="scim_tok_8f4a2c9e1b7d" truncate={0}/>}/>
            </div>
            <button className="btn sm" style={{ alignSelf: 'flex-start' }}>Force sync now</button>
          </div>
        ) : (
          <div className="col" style={{ gap: 8 }}>
            <div className="faint" style={{ fontSize: 11.5 }}>SCIM is not enabled. Members are managed manually.</div>
            <button className="btn primary sm" style={{ alignSelf: 'flex-start' }}>Enable SCIM</button>
          </div>
        )}
      </Panel>

      <div style={{ gridColumn: '1 / -1' }}>
        <Panel title="Attribute mapping">
          <div className="col" style={{ gap: 0 }}>
            {[
              ['email', 'email', 'required'],
              ['given_name', 'first_name', 'required'],
              ['family_name', 'last_name', 'required'],
              ['groups', 'memberOf', 'optional'],
              ['department', 'department', 'optional'],
            ].map(([shark, idp, req], i) => (
              <div key={shark} className="row" style={{ padding: '8px 0', borderBottom: i < 4 ? '1px solid var(--hairline)' : 'none', gap: 12, fontSize: 11.5 }}>
                <span className="mono" style={{ width: 140 }}>{shark}</span>
                <Icon.ChevronRight width={11} height={11} style={{ opacity: 0.4 }}/>
                <span className="mono" style={{ flex: 1 }}>{idp}</span>
                <span className={'chip' + (req === 'required' ? ' warn' : '')} style={{ height: 17, fontSize: 10 }}>{req}</span>
              </div>
            ))}
          </div>
        </Panel>
      </div>
    </div>
  );
}

/* --- Domains tab --- */

function OrgDomainsTab({ org }) {
  return (
    <>
      <div className="row" style={{ marginBottom: 12 }}>
        <span style={{ fontSize: 12, fontWeight: 500 }}>Verified domains</span>
        <span className="faint" style={{ fontSize: 11 }}>· auto-join members with matching email</span>
        <div style={{ flex: 1 }}/>
        <button className="btn primary sm"><Icon.Plus width={11} height={11}/>Add domain</button>
      </div>
      <div className="col" style={{ gap: 8 }}>
        {org.domains.map((d, i) => {
          const verified = i < org.verifiedDomains;
          return (
            <div key={d} style={{ padding: 12, border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)' }}>
              <div className="row" style={{ gap: 10 }}>
                <Icon.Globe width={14} height={14} style={{ opacity: 0.6 }}/>
                <span className="mono" style={{ fontSize: 13, fontWeight: 500 }}>{d}</span>
                {verified
                  ? <span className="chip success"><Icon.Check width={9} height={9}/>verified</span>
                  : <span className="chip warn">pending verification</span>}
                <div style={{ flex: 1 }}/>
                {!verified && <button className="btn sm">Re-check</button>}
                <button className="btn ghost sm danger">Remove</button>
              </div>
              {!verified && (
                <div style={{ marginTop: 10, padding: 10, background: 'var(--surface-2)', borderRadius: 4, fontSize: 11 }}>
                  <div className="faint" style={{ marginBottom: 6 }}>Add this TXT record to {d}:</div>
                  <CopyField value={`shark-verify=k8nQ2xL9pR4mT7vB${i}`} truncate={0}/>
                </div>
              )}
              {verified && (
                <div className="row" style={{ gap: 16, marginTop: 8, fontSize: 11 }}>
                  <span className="faint">Auto-join: <span style={{ color: 'var(--fg)' }}>enabled</span></span>
                  <span className="faint">Default role: <span className="chip" style={{ height: 16, fontSize: 10 }}>member</span></span>
                  <span className="faint mono">verified {MOCK.relativeTime(orgDays(200))}</span>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </>
  );
}

/* --- Audit tab --- */

function OrgAuditTab({ org }) {
  return (
    <Panel title="Audit trail">
      <OrgAuditList org={org}/>
    </Panel>
  );
}

function OrgAuditList({ org, limit }) {
  const events = MOCK.orgAudit[org.id] || [
    { t: orgMins(30), who: 'system', action: 'sso.sync.scheduled', meta: 'next: in 4h' },
    { t: orgHrs(8), who: org.ownerEmail, action: 'invite.sent', meta: '1 member' },
    { t: orgDays(1), who: org.ownerEmail, action: 'org.settings.updated', meta: 'name' },
  ];
  const shown = limit ? events.slice(0, limit) : events;
  return (
    <div className="col" style={{ gap: 0 }}>
      {shown.map((e, i) => (
        <div key={i} className="row" style={{ padding: '8px 0', borderBottom: i < shown.length - 1 ? '1px solid var(--hairline)' : 'none', gap: 10 }}>
          <span className="mono faint" style={{ fontSize: 10.5, width: 80 }}>{MOCK.relativeTime(e.t)}</span>
          <Avatar name={e.who} email={e.who} size={18}/>
          <span className="faint" style={{ fontSize: 11, width: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{e.who}</span>
          <span className="mono" style={{ fontSize: 11.5, color: 'var(--fg)' }}>{e.action}</span>
          <span className="faint" style={{ fontSize: 11, marginLeft: 'auto' }}>{e.meta}</span>
        </div>
      ))}
    </div>
  );
}

// Helpers (locals — each Babel script has its own scope)
const orgMins = (n) => Date.now() - n * 60 * 1000;
const orgHrs  = (n) => Date.now() - n * 60 * 60 * 1000;
const orgDays = (n) => Date.now() - n * 24 * 60 * 60 * 1000;

window.Organizations = Organizations;
