// @ts-nocheck
import React from 'react'

const NAV_KEYS = {
  u: 'users', o: 'orgs', a: 'audit', k: 'keys', w: 'webhooks',
  d: 'dev-inbox', g: 'agents', v: 'vault', t: 'tokens', s: 'settings',
  h: 'overview',
};

export function useKeyboardShortcuts(setPage) {
  const [cheatsheetOpen, setCheatsheetOpen] = React.useState(false);
  const gPending = React.useRef(false);
  const gTimer = React.useRef(null);

  React.useEffect(() => {
    function handler(e) {
      const tag = e.target?.tagName;
      const inInput = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || e.target?.isContentEditable;

      // Cmd+K handled by CommandPalette — skip here
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') return;

      // Esc — close panels
      if (e.key === 'Escape') {
        setCheatsheetOpen(false);
        window.dispatchEvent(new CustomEvent('shark-close-panel'));
        return;
      }

      if (inInput) return;

      // ? — toggle cheatsheet
      if (e.key === '?') {
        e.preventDefault();
        setCheatsheetOpen(v => !v);
        return;
      }

      // / — focus search
      if (e.key === '/') {
        e.preventDefault();
        const input = document.querySelector('input[type="text"], input[type="search"], input:not([type])');
        if (input) input.focus();
        return;
      }

      // g prefix
      if (e.key === 'g' && !gPending.current) {
        gPending.current = true;
        clearTimeout(gTimer.current);
        gTimer.current = setTimeout(() => { gPending.current = false; }, 500);
        return;
      }

      if (gPending.current) {
        gPending.current = false;
        clearTimeout(gTimer.current);
        const target = NAV_KEYS[e.key];
        if (target) {
          e.preventDefault();
          setPage(target);
        }
        return;
      }
    }

    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [setPage]);

  return { cheatsheetOpen, setCheatsheetOpen };
}

const SHORTCUT_GROUPS = [
  { title: 'Navigation', items: [
    ['g h', 'Overview'], ['g u', 'Users'], ['g o', 'Organizations'],
    ['g a', 'Audit Log'], ['g k', 'API Keys'], ['g w', 'Webhooks'],
    ['g d', 'Dev Inbox'], ['g g', 'Agents'], ['g v', 'Vault'],
    ['g t', 'Tokens'], ['g s', 'Settings'],
  ]},
  { title: 'Actions', items: [
    ['/', 'Focus search'], ['?', 'This cheatsheet'],
    ['Esc', 'Close panel'], ['⌘ K', 'Command palette'],
  ]},
];

export function KeyboardCheatsheet({ onClose }) {
  return (
    <div onClick={onClose} style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      zIndex: 60, animation: 'fadeIn 120ms ease-out',
    }}>
      <div onClick={e => e.stopPropagation()} style={{
        background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)',
        borderRadius: 'var(--radius-lg)', boxShadow: 'var(--shadow-lg)',
        padding: 20, width: 480, maxWidth: '90vw',
      }}>
        <div style={{ fontSize: 13, fontWeight: 600, fontFamily: 'var(--font-display)', marginBottom: 16 }}>
          Keyboard shortcuts
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
          {SHORTCUT_GROUPS.map(group => (
            <div key={group.title}>
              <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-faint)', marginBottom: 8, fontWeight: 500 }}>
                {group.title}
              </div>
              {group.items.map(([keys, label]) => (
                <div key={keys} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '3px 0' }}>
                  <span style={{ fontSize: 12, color: 'var(--fg-muted)' }}>{label}</span>
                  <span className="row" style={{ gap: 3 }}>
                    {keys.split(' ').map((k, i) => (
                      <span key={i} className="kbd">{k}</span>
                    ))}
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
