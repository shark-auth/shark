// @ts-nocheck
import React from 'react'
import { Icon } from './shared'

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
  { label: 'Create User', icon: 'Plus', target: 'users' },
  { label: 'Create API Key', icon: 'Plus', target: 'keys' },
  { label: 'Create Webhook', icon: 'Plus', target: 'webhooks' },
  { label: 'Create Application', icon: 'Plus', target: 'apps' },
  { label: 'Create Organization', icon: 'Plus', target: 'orgs' },
  { label: 'Create SSO Connection', icon: 'Plus', target: 'sso' },
  { label: 'Create Role', icon: 'Plus', target: 'rbac' },
];

export function CommandPalette({ open, onClose, setPage }) {
  const [query, setQuery] = React.useState('');
  const [selected, setSelected] = React.useState(0);
  const inputRef = React.useRef(null);

  React.useEffect(() => {
    if (open) { setQuery(''); setSelected(0); setTimeout(() => inputRef.current?.focus(), 10); }
  }, [open]);

  if (!open) return null;

  const q = query.toLowerCase();
  const filteredPages = q ? PAGES.filter(p => p.label.toLowerCase().includes(q)) : PAGES;
  const filteredActions = q ? ACTIONS.filter(a => a.label.toLowerCase().includes(q)) : ACTIONS;
  const allResults = [
    ...filteredPages.map(p => ({ ...p, type: 'page' })),
    ...filteredActions.map(a => ({ ...a, type: 'action' })),
  ];

  function execute(item) {
    if (item.type === 'page') setPage(item.id);
    else if (item.type === 'action') setPage(item.target);
    onClose();
  }

  function handleKeyDown(e) {
    if (e.key === 'ArrowDown') { e.preventDefault(); setSelected(s => Math.min(s + 1, allResults.length - 1)); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); setSelected(s => Math.max(s - 1, 0)); }
    else if (e.key === 'Enter' && allResults[selected]) { execute(allResults[selected]); }
    else if (e.key === 'Escape') { onClose(); }
  }

  // Find where actions start for group headers
  const pageCount = filteredPages.length;

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
        {/* Search input */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '0 16px', height: 44, borderBottom: '1px solid var(--hairline)' }}>
          <Icon.Search width={14} height={14} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
          <input
            ref={inputRef}
            value={query}
            onChange={e => { setQuery(e.target.value); setSelected(0); }}
            onKeyDown={handleKeyDown}
            placeholder="Type a command or search..."
            style={{ flex: 1, fontSize: 14, color: 'var(--fg)', background: 'transparent' }}
          />
        </div>

        {/* Results */}
        <div style={{ maxHeight: 360, overflowY: 'auto' }}>
          {allResults.length === 0 && (
            <div style={{ padding: '24px 16px', textAlign: 'center', fontSize: 13, color: 'var(--fg-dim)' }}>
              No results for "{query}"
            </div>
          )}

          {filteredPages.length > 0 && (
            <div style={{ padding: '8px 16px 4px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-faint)', fontWeight: 500 }}>
              Pages
            </div>
          )}
          {filteredPages.map((p, i) => {
            const IconEl = Icon[p.icon] || Icon.Home;
            return (
              <button key={p.id}
                onClick={() => execute({ ...p, type: 'page' })}
                onMouseEnter={() => setSelected(i)}
                style={{
                  width: '100%', display: 'flex', alignItems: 'center', gap: 10,
                  height: 36, padding: '0 16px',
                  background: selected === i ? 'var(--surface-3)' : 'transparent',
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
            <div style={{ padding: '8px 16px 4px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-faint)', fontWeight: 500, borderTop: pageCount > 0 ? '1px solid var(--hairline)' : 'none' }}>
              Actions
            </div>
          )}
          {filteredActions.map((a, i) => {
            const idx = pageCount + i;
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

        {/* Footer */}
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
