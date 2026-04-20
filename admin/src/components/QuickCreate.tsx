// @ts-nocheck
import React from 'react'
import { Icon } from './shared'

// Quick create dropdown — wired to the topbar "+ Quick create" button.
// Picking an item navigates to that page; the page's own create flow then
// fires via the ?new=1 query param (each list page checks it on mount).

const ITEMS = [
  { label: 'New User',         page: 'users',  icon: 'Users' },
  { label: 'New Agent',        page: 'agents', icon: 'Agent' },
  { label: 'New Application',  page: 'apps',   icon: 'App'   },
  { label: 'New Organization', page: 'orgs',   icon: 'Org'   },
  { label: 'New API Key',      page: 'keys',   icon: 'Key'   },
  { label: 'New Webhook',      page: 'webhooks', icon: 'Webhook' },
  { label: 'New Role',         page: 'rbac',   icon: 'Shield' },
  { label: 'New SSO Connection', page: 'sso',  icon: 'SSO'   },
];

export function QuickCreateMenu({ open, onClose, setPage, anchorRef }) {
  const ref = React.useRef(null);

  React.useEffect(() => {
    if (!open) return;
    const click = (e) => {
      if (ref.current?.contains(e.target)) return;
      if (anchorRef?.current?.contains(e.target)) return;
      onClose();
    };
    const esc = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('mousedown', click);
    window.addEventListener('keydown', esc);
    return () => {
      window.removeEventListener('mousedown', click);
      window.removeEventListener('keydown', esc);
    };
  }, [open, onClose, anchorRef]);

  if (!open) return null;

  // Position relative to anchor button in topbar
  const rect = anchorRef?.current?.getBoundingClientRect();
  const top = rect ? rect.bottom + 6 : 56;
  const right = rect ? Math.max(8, window.innerWidth - rect.right) : 16;

  return (
    <div ref={ref} style={{
      position: 'fixed', top, right,
      width: 220, zIndex: 60,
      background: 'var(--surface-1)',
      border: '1px solid var(--hairline-strong)',
      borderRadius: 'var(--radius)',
      boxShadow: 'var(--shadow-lg)',
      padding: 4,
      animation: 'fadeIn 80ms ease-out',
    }}>
      {ITEMS.map(it => {
        const IconEl = Icon[it.icon] || Icon.Plus;
        return (
          <button
            key={it.label}
            onClick={() => { setPage(it.page, { new: '1' }); onClose(); }}
            style={{
              width: '100%', display: 'flex', alignItems: 'center', gap: 9,
              padding: '7px 10px', borderRadius: 4,
              fontSize: 12.5, color: 'var(--fg)', textAlign: 'left',
              background: 'transparent', border: 0, cursor: 'pointer',
            }}
            onMouseEnter={e => e.currentTarget.style.background = 'var(--surface-3)'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
          >
            <IconEl width={13} height={13} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
            <span style={{ flex: 1 }}>{it.label}</span>
          </button>
        );
      })}
    </div>
  );
}
