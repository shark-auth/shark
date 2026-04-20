// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { useAPI } from './api'

// Help button — always-visible floating ? bottom-right. Opens menu with docs,
// changelog, GitHub, report bug. Report bug opens feedback modal with auto-filled
// page + version + recent console errors.

const REPO = 'https://github.com/shark-auth/shark';

// Buffer recent console errors so the feedback modal can attach them.
const consoleErrors = [];
(function installConsoleHook() {
  if (typeof window === 'undefined') return;
  if (window.__shark_console_hook) return;
  window.__shark_console_hook = true;
  const origErr = console.error;
  console.error = function (...args) {
    try {
      consoleErrors.push({
        ts: Date.now(),
        msg: args.map(a => {
          try { return typeof a === 'string' ? a : JSON.stringify(a); }
          catch { return String(a); }
        }).join(' ').slice(0, 400),
      });
      if (consoleErrors.length > 20) consoleErrors.shift();
    } catch {}
    return origErr.apply(console, args);
  };
  window.addEventListener('error', e => {
    consoleErrors.push({ ts: Date.now(), msg: `[window.error] ${e.message} @ ${e.filename}:${e.lineno}` });
    if (consoleErrors.length > 20) consoleErrors.shift();
  });
  window.addEventListener('unhandledrejection', e => {
    consoleErrors.push({ ts: Date.now(), msg: `[unhandledrejection] ${e.reason?.message || e.reason}` });
    if (consoleErrors.length > 20) consoleErrors.shift();
  });
})();

export function HelpButton() {
  const [open, setOpen] = React.useState(false);
  const [feedbackOpen, setFeedbackOpen] = React.useState(false);

  React.useEffect(() => {
    const handler = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === '/') { e.preventDefault(); setOpen(v => !v); }
      if (e.key === 'Escape') { setOpen(false); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  // Expose a global command router so CommandPalette can trigger actions.
  React.useEffect(() => {
    window.__shark_help = {
      openDocs:      () => window.open(REPO + '#readme', '_blank'),
      openGitHub:    () => window.open(REPO, '_blank'),
      openChangelog: () => window.open(REPO + '/blob/main/CHANGELOG.internal.md', '_blank'),
      openBug:       () => { setFeedbackOpen(true); setOpen(false); },
      openDiscord:   () => window.open(REPO + '#community', '_blank'),
    };
    return () => { delete window.__shark_help; };
  }, []);

  return (
    <>
      <button
        onClick={() => setOpen(v => !v)}
        title="Help (⌘/)"
        style={{
          position: 'fixed', bottom: 20, right: 20, zIndex: 80,
          width: 36, height: 36, borderRadius: '50%',
          background: 'var(--surface-3)',
          border: '1px solid var(--hairline-strong)',
          color: 'var(--fg)',
          cursor: 'pointer', fontSize: 18, fontWeight: 600,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          boxShadow: 'var(--shadow-lg)',
        }}
      >?</button>

      {open && <HelpMenu onClose={() => setOpen(false)} onReport={() => { setFeedbackOpen(true); setOpen(false); }}/>}
      {feedbackOpen && <FeedbackModal onClose={() => setFeedbackOpen(false)}/>}
    </>
  );
}

function HelpMenu({ onClose, onReport }) {
  const ref = React.useRef(null);
  React.useEffect(() => {
    const click = (e) => { if (!ref.current?.contains(e.target)) onClose(); };
    const t = setTimeout(() => window.addEventListener('mousedown', click), 0);
    return () => { clearTimeout(t); window.removeEventListener('mousedown', click); };
  }, [onClose]);

  const items = [
    { label: 'Documentation',      icon: 'Audit',   action: () => window.open(REPO + '#readme', '_blank'),                     hint: 'Readme + guides' },
    { label: 'Changelog',          icon: 'Clock',   action: () => window.open(REPO + '/blob/main/CHANGELOG.internal.md', '_blank'), hint: 'What shipped' },
    { label: 'GitHub repository',  icon: 'Terminal',action: () => window.open(REPO, '_blank'),                                 hint: 'Source + issues' },
    { label: 'Report a bug',       icon: 'Warn',    action: onReport,                                                          hint: 'Pre-filled with context' },
  ];

  return (
    <div ref={ref} style={{
      position: 'fixed', bottom: 64, right: 20, zIndex: 81,
      width: 260,
      background: 'var(--surface-1)',
      border: '1px solid var(--hairline-strong)',
      borderRadius: 'var(--radius)',
      boxShadow: 'var(--shadow-lg)',
      padding: 6,
      animation: 'fadeIn 80ms ease-out',
    }}>
      <div style={{ padding: '6px 10px 8px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>
        Help & feedback
      </div>
      {items.map(it => {
        const I = Icon[it.icon] || Icon.Info;
        return (
          <button
            key={it.label}
            onClick={() => { it.action(); onClose(); }}
            style={{
              width: '100%', display: 'flex', alignItems: 'center', gap: 10,
              padding: '8px 10px', borderRadius: 4,
              background: 'transparent', border: 0, cursor: 'pointer',
              color: 'var(--fg)', textAlign: 'left',
            }}
            onMouseEnter={e => e.currentTarget.style.background = 'var(--surface-3)'}
            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
          >
            <I width={13} height={13} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
            <div className="col" style={{ gap: 2, flex: 1 }}>
              <span style={{ fontSize: 12.5 }}>{it.label}</span>
              <span className="faint" style={{ fontSize: 10.5 }}>{it.hint}</span>
            </div>
          </button>
        );
      })}
    </div>
  );
}

function FeedbackModal({ onClose }) {
  const [title, setTitle] = React.useState('');
  const [body, setBody] = React.useState('');
  const { data: health } = useAPI('/admin/health');

  const page = window.location.pathname;
  const version = health?.version || 'unknown';
  const ua = navigator.userAgent;
  const errors = consoleErrors.slice(-10);

  const buildIssueBody = () => {
    const errorBlock = errors.length > 0
      ? '\n\n### Recent console errors\n```\n' + errors.map(e => `[${new Date(e.ts).toISOString()}] ${e.msg}`).join('\n') + '\n```'
      : '';
    return `### What happened
${body || '(describe what you did and what went wrong)'}

### Page
\`${page}\`

### Version
\`${version}\`

### User agent
\`${ua}\`${errorBlock}
`;
  };

  const submitGitHub = () => {
    const url = `${REPO}/issues/new?title=${encodeURIComponent(title || 'Dashboard bug: ' + page)}&body=${encodeURIComponent(buildIssueBody())}`;
    window.open(url, '_blank');
    onClose();
  };

  const submitMailto = () => {
    const url = `mailto:bugs@shark.invalid?subject=${encodeURIComponent(title || 'Dashboard bug: ' + page)}&body=${encodeURIComponent(buildIssueBody())}`;
    window.location.href = url;
    onClose();
  };

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 90,
      background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: 20,
    }} onClick={onClose}>
      <div
        onClick={e => e.stopPropagation()}
        style={{
          width: 520, maxWidth: '100%', maxHeight: '90vh', overflow: 'auto',
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 6,
          boxShadow: 'var(--shadow-lg)',
          padding: 20,
      }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
          <h2 style={{ fontSize: 15, margin: 0, fontWeight: 600 }}>Report a bug</h2>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <p className="faint" style={{ margin: '0 0 12px', fontSize: 12 }}>
          Page, version, and recent console errors are attached automatically. No data leaves until you hit submit.
        </p>

        <label style={lbl}>Short title</label>
        <input
          value={title}
          onChange={e => setTitle(e.target.value)}
          placeholder="Clicking X did Y when I expected Z"
          style={inp}
        />

        <label style={{ ...lbl, marginTop: 10 }}>What happened</label>
        <textarea
          value={body}
          onChange={e => setBody(e.target.value)}
          placeholder="Steps to reproduce, expected vs. actual…"
          style={{ ...inp, minHeight: 100, fontFamily: 'inherit', resize: 'vertical' }}
        />

        <details style={{ marginTop: 10 }}>
          <summary style={{ fontSize: 11, color: 'var(--fg-dim)', cursor: 'pointer' }}>
            Attached context ({errors.length} recent errors)
          </summary>
          <pre style={{
            margin: '8px 0 0', padding: 8, fontSize: 10.5, lineHeight: 1.5,
            background: 'var(--surface-2)', border: '1px solid var(--hairline)',
            borderRadius: 4, overflow: 'auto', maxHeight: 140,
            whiteSpace: 'pre-wrap',
          }}>{buildIssueBody()}</pre>
        </details>

        <div className="row" style={{ gap: 8, marginTop: 14, justifyContent: 'flex-end' }}>
          <button className="btn ghost sm" onClick={onClose}>Cancel</button>
          <button className="btn sm" onClick={submitMailto}>Open email</button>
          <button className="btn primary sm" onClick={submitGitHub}>Open GitHub issue</button>
        </div>
      </div>
    </div>
  );
}

const lbl = { display: 'block', fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 4 };
const inp = {
  width: '100%', padding: '8px 10px',
  background: 'var(--surface-2)',
  border: '1px solid var(--hairline-strong)',
  borderRadius: 4,
  color: 'var(--fg)',
  fontSize: 12.5,
  fontFamily: 'var(--font-mono)',
  outline: 'none',
  boxSizing: 'border-box',
};
