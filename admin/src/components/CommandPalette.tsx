// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API } from './api'

const PAGES = [
  { id: 'overview', label: 'Overview', icon: 'Home', hint: 'g h' },
  { id: 'users', label: 'Users', icon: 'Users', hint: 'g u' },
  { id: 'sessions', label: 'Sessions', icon: 'Session' },
  { id: 'orgs', label: 'Organizations', icon: 'Org', hint: 'g o' },
  { id: 'agents', label: 'Agents', icon: 'Agent', hint: 'g g' },
  { id: 'audit', label: 'Audit Log', icon: 'Audit', hint: 'g a' },
  { id: 'apps', label: 'Applications', icon: 'App' },
  { id: 'keys', label: 'API Keys', icon: 'Key', hint: 'g k' },
  { id: 'webhooks', label: 'Webhooks', icon: 'Webhook', hint: 'g w' },
  { id: 'rbac', label: 'Roles & Permissions', icon: 'Shield' },
  { id: 'sso', label: 'SSO Connections', icon: 'SSO' },
  { id: 'auth', label: 'Authentication', icon: 'Lock' },
  { id: 'signing', label: 'Signing Keys', icon: 'Signing' },
  { id: 'settings', label: 'Settings', icon: 'Settings', hint: 'g s' },
  { id: 'dev-inbox', label: 'Dev Inbox', icon: 'Mail', hint: 'g d' },
  { id: 'device-flow', label: 'Device Flow', icon: 'Device' },
  { id: 'tokens', label: 'Tokens', icon: 'Token', hint: 'g t' },
  { id: 'vault', label: 'Vault', icon: 'Vault', hint: 'g v' },
  { id: 'consents', label: 'Consents', icon: 'Consent' },
  { id: 'explorer', label: 'API Explorer', icon: 'Explorer' },
  { id: 'debug', label: 'Session Debugger', icon: 'Debug' },
  { id: 'schemas', label: 'Event Schemas', icon: 'Schema' },
  { id: 'proxy', label: 'Proxy', icon: 'Proxy' },
  { id: 'oidc', label: 'OIDC Provider', icon: 'Globe' },
  { id: 'impersonation', label: 'Impersonation', icon: 'Impersonate' },
  { id: 'compliance', label: 'Compliance', icon: 'Compliance' },
  { id: 'migrations', label: 'Migrations', icon: 'Migration' },
  { id: 'branding', label: 'Branding', icon: 'Brand' },
  { id: 'flow', label: 'Flow Builder', icon: 'Flow' },
];

const ACTIONS = [
  { label: 'Create User',          icon: 'Plus', target: 'users' },
  { label: 'Create API Key',       icon: 'Plus', target: 'keys' },
  { label: 'Create Webhook',       icon: 'Plus', target: 'webhooks' },
  { label: 'Create Application',   icon: 'Plus', target: 'apps' },
  { label: 'Create Organization',  icon: 'Plus', target: 'orgs' },
  { label: 'Create Agent',         icon: 'Plus', target: 'agents' },
  { label: 'Create SSO Connection', icon: 'Plus', target: 'sso' },
  { label: 'Create Role',          icon: 'Plus', target: 'rbac' },
];

// Hook: debounced entity search across users / agents / apps / orgs
function useEntitySearch(query) {
  const [results, setResults] = React.useState({ users: [], agents: [], apps: [], orgs: [] });
  const [loading, setLoading] = React.useState(false);

  React.useEffect(() => {
    const q = query.trim();
    if (q.length < 2) { setResults({ users: [], agents: [], apps: [], orgs: [] }); setLoading(false); return; }
    setLoading(true);
    const ctrl = { cancelled: false };
    const t = setTimeout(async () => {
      const safe = encodeURIComponent(q);
      const [usersR, agentsR, appsR, orgsR] = await Promise.allSettled([
        API.get(`/users?search=${safe}&per_page=5`),
        API.get(`/agents?limit=20`),
        API.get(`/admin/apps`),
        API.get(`/admin/organizations`),
      ]);
      if (ctrl.cancelled) return;

      const lower = q.toLowerCase();
      const pickArr = (r, ...keys) => {
        if (r.status !== 'fulfilled' || !r.value) return [];
        const v = r.value;
        for (const k of keys) {
          if (Array.isArray(v?.[k])) return v[k];
        }
        if (Array.isArray(v)) return v;
        return [];
      };

      const userList = pickArr(usersR, 'users', 'data');
      const agentList = pickArr(agentsR, 'data', 'agents');
      const appList = pickArr(appsR, 'apps', 'data', 'applications');
      const orgList = pickArr(orgsR, 'organizations', 'data', 'orgs');

      const matchAgent = (a) =>
        (a.name || '').toLowerCase().includes(lower) ||
        (a.client_id || '').toLowerCase().includes(lower);
      const matchApp = (a) =>
        (a.name || '').toLowerCase().includes(lower) ||
        (a.client_id || '').toLowerCase().includes(lower) ||
        (a.id || '').toLowerCase().includes(lower);
      const matchOrg = (o) =>
        (o.name || '').toLowerCase().includes(lower) ||
        (o.slug || '').toLowerCase().includes(lower);

      setResults({
        users: userList.slice(0, 5),
        agents: agentList.filter(matchAgent).slice(0, 5),
        apps: appList.filter(matchApp).slice(0, 5),
        orgs: orgList.filter(matchOrg).slice(0, 5),
      });
      setLoading(false);
    }, 200);
    return () => { ctrl.cancelled = true; clearTimeout(t); };
  }, [query]);

  return { results, loading };
}

export function CommandPalette({ open, onClose, setPage }) {
  const [query, setQuery] = React.useState('');
  const [selected, setSelected] = React.useState(0);
  const inputRef = React.useRef(null);

  React.useEffect(() => {
    if (open) { setQuery(''); setSelected(0); setTimeout(() => inputRef.current?.focus(), 10); }
  }, [open]);

  const { results, loading } = useEntitySearch(query);

  if (!open) return null;

  const q = query.toLowerCase();
  const filteredPages = q ? PAGES.filter(p => p.label.toLowerCase().includes(q)) : PAGES;
  const filteredActions = q ? ACTIONS.filter(a => a.label.toLowerCase().includes(q)) : ACTIONS;

  const entityItems = [
    ...results.users.map(u => ({
      type: 'entity', kind: 'User',
      label: u.name || u.email || u.id,
      sub: u.email || u.id,
      target: 'users', params: { search: u.email || u.id },
    })),
    ...results.agents.map(a => ({
      type: 'entity', kind: 'Agent',
      label: a.name || a.client_id || a.id,
      sub: a.client_id || a.id,
      target: 'agents', params: { search: a.name || a.client_id || '' },
    })),
    ...results.apps.map(a => ({
      type: 'entity', kind: 'App',
      label: a.name || a.client_id || a.id,
      sub: a.client_id || a.id,
      target: 'apps', params: { search: a.name || '' },
    })),
    ...results.orgs.map(o => ({
      type: 'entity', kind: 'Org',
      label: o.name || o.slug || o.id,
      sub: o.slug || o.id,
      target: 'orgs', params: { search: o.name || o.slug || '' },
    })),
  ];

  const allResults = [
    ...entityItems,
    ...filteredPages.map(p => ({ ...p, type: 'page' })),
    ...filteredActions.map(a => ({ ...a, type: 'action' })),
  ];

  function execute(item) {
    if (item.type === 'page') setPage(item.id);
    else if (item.type === 'action') setPage(item.target, { new: '1' });
    else if (item.type === 'entity') setPage(item.target, item.params);
    onClose();
  }

  function handleKeyDown(e) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelected(s => Math.min(s + 1, allResults.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSelected(s => Math.max(s - 1, 0)); }
    else if (e.key === 'Enter' && allResults[selected]) { execute(allResults[selected]); }
    else if (e.key === 'Escape') { onClose(); }
  }

  let cursor = 0;
  const entityStart = cursor; cursor += entityItems.length;
  const pageStart = cursor; cursor += filteredPages.length;
  const actionStart = cursor;

  const ICON_BY_KIND = { User: 'Users', Agent: 'Agent', App: 'App', Org: 'Org' };

  return (
    <div onClick={onClose} style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'flex-start', justifyContent: 'center',
      paddingTop: '15vh', zIndex: 55, animation: 'fadeIn 80ms ease-out',
    }}>
      <div onClick={e => e.stopPropagation()} style={{
        width: 560, maxWidth: '90vw',
        background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)',
        borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-lg)',
        overflow: 'hidden',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '0 16px', height: 44, borderBottom: '1px solid var(--hairline)' }}>
          <Icon.Search width={14} height={14} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
          <input
            ref={inputRef}
            value={query}
            onChange={e => { setQuery(e.target.value); setSelected(0); }}
            onKeyDown={handleKeyDown}
            placeholder="Search users, agents, apps, orgs… or jump to a page"
            style={{ flex: 1, fontSize: 14, color: 'var(--fg)', background: 'transparent' }}
          />
          {loading && <span className="faint mono" style={{ fontSize: 10 }}>searching…</span>}
        </div>

        <div style={{ maxHeight: 420, overflowY: 'auto' }}>
          {allResults.length === 0 && !loading && (
            <div style={{ padding: '24px 16px', textAlign: 'center', fontSize: 13, color: 'var(--fg-dim)' }}>
              No results for "{query}"
            </div>
          )}

          {entityItems.length > 0 && (
            <div style={{ padding: '8px 16px 4px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-faint)', fontWeight: 500 }}>
              Matches
            </div>
          )}
          {entityItems.map((it, i) => {
            const idx = entityStart + i;
            const IconEl = Icon[ICON_BY_KIND[it.kind] || 'Search'] || Icon.Search;
            return (
              <button key={'ent-' + i}
                onClick={() => execute(it)}
                onMouseEnter={() => setSelected(idx)}
                style={{
                  width: '100%', display: 'flex', alignItems: 'center', gap: 10,
                  height: 38, padding: '0 16px',
                  background: selected === idx ? 'var(--surface-3)' : 'transparent',
                  color: 'var(--fg)', fontSize: 13, textAlign: 'left',
                }}
              >
                <IconEl width={14} height={14} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
                <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
                  <span style={{ fontSize: 12.5, lineHeight: 1.2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{it.label}</span>
                  {it.sub && (
                    <span className="mono faint" style={{ fontSize: 10.5, lineHeight: 1.3, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {it.sub}
                    </span>
                  )}
                </div>
                <span className="chip" style={{ height: 16, fontSize: 9, padding: '0 5px' }}>{it.kind}</span>
              </button>
            );
          })}

          {filteredPages.length > 0 && (
            <div style={{ padding: '8px 16px 4px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-faint)', fontWeight: 500, borderTop: entityItems.length > 0 ? '1px solid var(--hairline)' : 'none' }}>
              Pages
            </div>
          )}
          {filteredPages.map((p, i) => {
            const idx = pageStart + i;
            const IconEl = Icon[p.icon] || Icon.Home;
            return (
              <button key={p.id}
                onClick={() => execute({ ...p, type: 'page' })}
                onMouseEnter={() => setSelected(idx)}
                style={{
                  width: '100%', display: 'flex', alignItems: 'center', gap: 10,
                  height: 36, padding: '0 16px',
                  background: selected === idx ? 'var(--surface-3)' : 'transparent',
                  color: 'var(--fg)', fontSize: 13, textAlign: 'left',
                }}
              >
                <IconEl width={14} height={14} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
                <span style={{ flex: 1 }}>{p.label}</span>
                {p.hint && <span style={{ fontSize: 11, color: 'var(--fg-faint)', fontFamily: 'var(--font-mono)' }}>{p.hint}</span>}
              </button>
            );
          })}

          {filteredActions.length > 0 && (
            <div style={{ padding: '8px 16px 4px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-faint)', fontWeight: 500, borderTop: filteredPages.length > 0 ? '1px solid var(--hairline)' : 'none' }}>
              Actions
            </div>
          )}
          {filteredActions.map((a, i) => {
            const idx = actionStart + i;
            const IconEl = Icon[a.icon] || Icon.Plus;
            return (
              <button key={a.label}
                onClick={() => execute({ ...a, type: 'action' })}
                onMouseEnter={() => setSelected(idx)}
                style={{
                  width: '100%', display: 'flex', alignItems: 'center', gap: 10,
                  height: 36, padding: '0 16px',
                  background: selected === idx ? 'var(--surface-3)' : 'transparent',
                  color: 'var(--fg)', fontSize: 13, textAlign: 'left',
                }}
              >
                <IconEl width={14} height={14} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
                <span style={{ flex: 1 }}>{a.label}</span>
              </button>
            );
          })}
        </div>

        <div style={{
          height: 28, padding: '0 16px', borderTop: '1px solid var(--hairline)',
          display: 'flex', alignItems: 'center', gap: 16,
          fontSize: 11, color: 'var(--fg-dim)',
        }}>
          <span><span style={{ fontFamily: 'var(--font-mono)' }}>↑↓</span> navigate</span>
          <span><span style={{ fontFamily: 'var(--font-mono)' }}>↵</span> select</span>
          <span><span style={{ fontFamily: 'var(--font-mono)' }}>esc</span> close</span>
        </div>
      </div>
    </div>
  );
}
