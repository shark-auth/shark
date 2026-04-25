// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Get Started — onboarding checklist.
// Sibling page to users.tsx: monochrome, square, hairline, 13px base, dense rows.
// Two integration paths shown via a tab strip. Per-task status persists in
// localStorage; auto-verify probes /admin/proxy/status and /admin/apps to flip
// task state when admin completes config elsewhere.

const STORAGE_KEY = 'shark_get_started_state_v1';

// ─── Task definitions ────────────────────────────────────────────────────────
// Tasks are derived from the live admin surface (proxy_handlers.go,
// application_handlers.go, branding admin, shark-auth-react SDK).
// IDs are stable — they key the localStorage status map.

const PROXY_TASKS = [
  {
    id: 'proxy.upstream',
    title: 'Configure proxy upstream',
    summary: 'Point SharkAuth at the app it should sit in front of. The reverse proxy forwards every request and stamps identity headers.',
    page: 'proxy',
    pageLabel: 'Proxy → Lifecycle',
    cli: 'shark serve --dev --proxy-upstream http://localhost:3000',
    autoCheck: 'proxyEnabled',
  },
  {
    id: 'proxy.rule',
    title: 'Add a protect rule',
    summary: 'Pick which paths require auth. A rule matches a path pattern + method, requires an identity (user / role / scope), and is evaluated at the edge before the upstream sees the request.',
    page: 'proxy',
    pageLabel: 'Proxy → Rules',
    cli: 'curl -sH "Authorization: Bearer $KEY" /api/v1/admin/proxy/rules',
    autoCheck: 'proxyHasRules',
  },
  {
    id: 'proxy.methods',
    title: 'Pick login methods',
    summary: 'Decide which sign-in surfaces the hosted login page exposes — passwords, magic links, social, SSO, passkeys. Disabled methods do not render.',
    page: 'authentication',
    pageLabel: 'Authentication',
    cli: 'shark methods enable password,oauth_google',
  },
  {
    id: 'proxy.redirect',
    title: 'Set post-login redirect',
    summary: 'Where the hosted login page sends users once authentication completes. Must match an allowed redirect on the proxy app.',
    page: 'proxy',
    pageLabel: 'Proxy → Config',
    cli: 'shark proxy set-redirect /home',
  },
  {
    id: 'proxy.brand',
    title: 'Brand the hosted login page',
    summary: 'Upload a logo and override design tokens. Branding lives in /admin/branding and is content-addressed for cache safety.',
    page: 'branding',
    pageLabel: 'Branding',
    cli: 'curl -X PATCH /api/v1/admin/branding -d \'{"name":"Acme"}\'',
    autoCheck: 'brandingSet',
  },
  {
    id: 'proxy.test',
    title: 'Test the proxy URL',
    summary: 'Hit the proxy from a browser. You should land on the hosted login page, sign in, and bounce to the upstream with X-Shark-User-Id stamped on the request.',
    page: 'proxy',
    pageLabel: 'Proxy → Simulate',
    cli: 'curl -i http://localhost:8080/',
  },
];

const SDK_TASKS = [
  {
    id: 'sdk.app',
    title: 'Create an Application',
    summary: 'Register an OAuth/OIDC client. You get a publishable key (public) and an issuer URL the SDK points at.',
    page: 'apps',
    pageLabel: 'Applications',
    cli: 'shark apps create --name "My App" --redirect http://localhost:5173/shark/callback',
    autoCheck: 'hasApp',
  },
  {
    id: 'sdk.install',
    title: 'Install the SDK package',
    summary: 'Pick the SDK that matches your stack. The React package wraps the TypeScript core SDK with hooks and components.',
    page: null,
    pageLabel: null,
    cli: 'npm install @sharkauth/react',
    extraSnippets: [
      { label: 'TypeScript core', code: 'npm install @shark-auth/sdk' },
      { label: 'Python', code: 'pip install shark-auth' },
    ],
  },
  {
    id: 'sdk.provider',
    title: 'Mount SharkProvider',
    summary: 'Wrap the app root with SharkProvider. It loads tokens from storage, refreshes on schedule, and exposes the session via context.',
    page: null,
    pageLabel: null,
    code: `import { SharkProvider } from '@sharkauth/react';

export default function App() {
  return (
    <SharkProvider
      authUrl="https://auth.example.com"
      publishableKey="pk_live_…"
    >
      <Routes/>
    </SharkProvider>
  );
}`,
  },
  {
    id: 'sdk.signin',
    title: 'Wire a Sign-in button',
    summary: 'Trigger the OAuth 2.1 + PKCE flow. startAuthFlow stores the verifier in sessionStorage and redirects to /oauth/authorize on the issuer.',
    page: null,
    pageLabel: null,
    code: `import { useShark } from '@sharkauth/react';

export function SignInButton() {
  const { signIn } = useShark();
  return <button onClick={() => signIn()}>Sign in</button>;
}`,
  },
  {
    id: 'sdk.callback',
    title: 'Mount the callback route',
    summary: 'The redirect_uri is /shark/callback by default. The route exchanges ?code= for tokens via /oauth/token, then bounces to the post-login redirect saved in sessionStorage.',
    page: null,
    pageLabel: null,
    code: `import { SharkCallback } from '@sharkauth/react';

// In your router:
{ path: '/shark/callback', element: <SharkCallback/> }`,
  },
  {
    id: 'sdk.session',
    title: 'Read the session',
    summary: 'Use the useShark hook anywhere under the provider. session.user is null until exchange completes; use isLoading to guard.',
    page: null,
    pageLabel: null,
    code: `const { session, isLoading } = useShark();
if (isLoading) return null;
return session?.user
  ? <p>Hi {session.user.email}</p>
  : <p>Signed out</p>;`,
  },
];

// ─── State helpers ───────────────────────────────────────────────────────────

function loadState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { tab: 'proxy', tasks: {} };
    const parsed = JSON.parse(raw);
    return { tab: parsed.tab === 'sdk' ? 'sdk' : 'proxy', tasks: parsed.tasks || {} };
  } catch { return { tab: 'proxy', tasks: {} }; }
}

function persist(state) {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch {}
}

// ─── Status dot — only colored element on the page ───────────────────────────
//    pending → neutral, done → success, skipped → warn, blocked → danger
function StatusDot({ status }) {
  const color = status === 'done' ? 'var(--success)'
    : status === 'skipped' ? 'var(--warn)'
    : status === 'blocked' ? 'var(--danger)'
    : 'var(--fg-faint)';
  const fill = status === 'done' || status === 'blocked' || status === 'skipped';
  return (
    <span
      style={{
        width: 8, height: 8, borderRadius: '50%',
        background: fill ? color : 'transparent',
        border: '1px solid ' + color,
        flexShrink: 0,
      }}
      aria-label={status || 'pending'}
    />
  );
}

// ─── Auto-verify hook ────────────────────────────────────────────────────────
//   Probes /admin/proxy/status (404 when proxy disabled), /admin/proxy/rules,
//   /admin/apps, /admin/branding. Returns booleans for autoCheck keys.

function useAutoChecks() {
  const [checks, setChecks] = React.useState({
    proxyEnabled: false,
    proxyHasRules: false,
    hasApp: false,
    brandingSet: false,
  });

  const probe = React.useCallback(async () => {
    const next = { proxyEnabled: false, proxyHasRules: false, hasApp: false, brandingSet: false };
    // Proxy enabled: status returns 200 with data; 404 when disabled.
    try {
      const r = await API.get('/admin/proxy/status');
      next.proxyEnabled = !!(r && (r.data || r.state || r.upstream));
    } catch {}
    try {
      const r = await API.get('/admin/proxy/rules');
      const rules = r?.data || r?.rules || (Array.isArray(r) ? r : []);
      next.proxyHasRules = Array.isArray(rules) && rules.length > 0;
    } catch {}
    try {
      const r = await API.get('/admin/apps');
      const apps = r?.applications || r?.data || (Array.isArray(r) ? r : []);
      next.hasApp = Array.isArray(apps) && apps.length > 0;
    } catch {}
    try {
      const r = await API.get('/admin/branding');
      const b = r?.data || r;
      next.brandingSet = !!(b && (b.name || b.logo_url || b.logo));
    } catch {}
    setChecks(next);
  }, []);

  React.useEffect(() => { probe(); }, [probe]);

  return { checks, refresh: probe };
}

// ─── Component ───────────────────────────────────────────────────────────────

export function GetStarted({ setPage }) {
  const initial = React.useMemo(loadState, []);
  const [tab, setTab] = React.useState(initial.tab);
  const [tasks, setTasks] = React.useState(initial.tasks);
  const [selectedId, setSelectedId] = React.useState(null);
  const toast = useToast();
  const { checks, refresh: refreshChecks } = useAutoChecks();

  React.useEffect(() => { persist({ tab, tasks }); }, [tab, tasks]);

  // Apply auto-checks: any pending task whose autoCheck flips true → done.
  React.useEffect(() => {
    setTasks(prev => {
      const next = { ...prev };
      let changed = false;
      for (const t of [...PROXY_TASKS, ...SDK_TASKS]) {
        if (t.autoCheck && checks[t.autoCheck] && next[t.id] !== 'done' && next[t.id] !== 'skipped') {
          next[t.id] = 'done';
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [checks]);

  const list = tab === 'proxy' ? PROXY_TASKS : SDK_TASKS;
  const selected = list.find(t => t.id === selectedId) || null;

  const setStatus = (id, status) => {
    setTasks(prev => {
      const next = { ...prev };
      if (status === null) delete next[id]; else next[id] = status;
      return next;
    });
  };

  const total = list.length;
  const done = list.filter(t => tasks[t.id] === 'done').length;
  const skipped = list.filter(t => tasks[t.id] === 'skipped').length;
  const pending = total - done - skipped;
  const pct = Math.round((done / Math.max(total, 1)) * 100);

  const dismissOnboarding = () => {
    try { localStorage.setItem('shark_admin_onboarded', '1'); } catch {}
    if (typeof setPage === 'function') setPage('overview');
  };

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Toolbar — same rhythm as users.tsx */}
        <div style={{
          padding: '10px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 8, alignItems: 'center',
          background: 'var(--surface-0)',
        }}>
          <div style={{ display: 'flex', gap: 0 }}>
            <TabBtn on={tab === 'proxy'} onClick={() => { setTab('proxy'); setSelectedId(null); }}>
              Proxy + hosted UI
            </TabBtn>
            <TabBtn on={tab === 'sdk'} onClick={() => { setTab('sdk'); setSelectedId(null); }}>
              SDK
            </TabBtn>
          </div>
          <div style={{ flex: 1 }}/>
          <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
            {done}/{total} done{skipped ? ` · ${skipped} skipped` : ''}
          </span>
          <div style={{
            width: 80, height: 4, background: 'var(--surface-2)',
            borderRadius: 2, overflow: 'hidden',
          }}>
            <div style={{ width: pct + '%', height: '100%', background: 'var(--fg)' }}/>
          </div>
          <button className="btn ghost sm" onClick={refreshChecks} title="Re-probe state">
            <Icon.Refresh width={11} height={11}/>Recheck
          </button>
          <button className="btn sm" onClick={dismissOnboarding}>
            Skip to dashboard
          </button>
        </div>

        {/* Sub-header — small framing line, no hero */}
        <div style={{
          padding: '7px 16px',
          borderBottom: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)',
        }}>
          {tab === 'proxy'
            ? 'SharkAuth runs as a reverse proxy in front of your app. Auth happens at the edge — no SDK needed.'
            : 'Embed SharkAuth in your own UI. Drop in the SDK, mount the provider, wire sign-in.'}
        </div>

        {/* Task table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table className="tbl">
            <thead style={{ background: 'var(--surface-0)' }}>
              <tr>
                <th style={{ width: 28, paddingLeft: 16, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}/>
                <th style={{ position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Task</th>
                <th style={{ width: 110, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Status</th>
                <th style={{ width: 160, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-muted)' }}>Goes to</th>
                <th style={{ width: 60, position: 'sticky', top: 0, zIndex: 1, background: 'var(--surface-0)' }}/>
              </tr>
            </thead>
            <tbody>
              {list.map((t, idx) => {
                const status = tasks[t.id] || 'pending';
                const auto = t.autoCheck && checks[t.autoCheck];
                return (
                  <tr key={t.id}
                      className={selectedId === t.id ? 'active' : ''}
                      onClick={() => setSelectedId(t.id)}
                      style={{ cursor: 'pointer' }}>
                    <td style={{ paddingLeft: 16 }}>
                      <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
                        {String(idx + 1).padStart(2, '0')}
                      </span>
                    </td>
                    <td>
                      <div style={{ minWidth: 0 }}>
                        <div style={{ fontWeight: 500, fontSize: 13 }}>{t.title}</div>
                        <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 560 }}>
                          {t.summary}
                        </div>
                      </div>
                    </td>
                    <td>
                      <span className="row" style={{ gap: 6 }}>
                        <StatusDot status={status}/>
                        <span style={{ fontSize: 12, color: 'var(--fg-muted)', lineHeight: 1.5, textTransform: 'capitalize' }}>{status}</span>
                        {auto && status === 'done' && (
                          <span className="faint mono" style={{ fontSize: 10, lineHeight: 1.5 }}>auto</span>
                        )}
                      </span>
                    </td>
                    <td>
                      {t.pageLabel ? (
                        <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{t.pageLabel}</span>
                      ) : (
                        <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>—</span>
                      )}
                    </td>
                    <td onClick={e => e.stopPropagation()}>
                      <Icon.ChevronRight width={12} height={12} style={{ opacity: 0.5 }}/>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* Footer */}
        <div style={{
          padding: '8px 16px',
          borderTop: '1px solid var(--hairline)',
          display: 'flex', alignItems: 'center', gap: 8,
          background: 'var(--surface-0)',
          fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)',
        }}>
          <span>{pending} pending</span>
          <span>·</span>
          <span>{done} done</span>
          {skipped > 0 && <><span>·</span><span>{skipped} skipped</span></>}
          <div style={{ flex: 1 }}/>
          <span className="faint">Status auto-syncs from /admin/proxy/status, /admin/apps, /admin/branding.</span>
        </div>

        <CLIFooter command="shark help"/>
      </div>

      {selected && (
        <TaskDrawer
          task={selected}
          status={tasks[selected.id] || 'pending'}
          onClose={() => setSelectedId(null)}
          onStatus={(s) => {
            setStatus(selected.id, s);
            if (s === 'done') toast.success('Marked done');
            else if (s === 'skipped') toast.success('Skipped');
            else if (s === null) toast.success('Reset');
          }}
          onGo={(p) => {
            try { localStorage.setItem('shark_admin_onboarded', '1'); } catch {}
            if (typeof setPage === 'function') setPage(p);
          }}
        />
      )}
    </div>
  );
}

// ─── Tab button — matches users.tsx slideover tab idiom ──────────────────────
function TabBtn({ on, onClick, children }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '6px 14px',
        height: 28,
        fontSize: 12, fontWeight: on ? 600 : 500,
        background: on ? 'var(--surface-3)' : 'var(--surface-2)',
        color: on ? 'var(--fg)' : 'var(--fg-muted)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 5,
        marginRight: -1,
        cursor: 'pointer',
      }}
    >{children}</button>
  );
}

// ─── Drawer ──────────────────────────────────────────────────────────────────
function TaskDrawer({ task, status, onClose, onStatus, onGo }) {
  const toast = useToast();
  // ESC closes
  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const copy = async (text) => {
    try { await navigator.clipboard.writeText(text); toast.success('Copied'); }
    catch { toast.error('Copy failed'); }
  };

  return (
    <div style={{
      width: 420, borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column',
      flexShrink: 0,
      animation: 'slideIn 140ms ease-out',
    }}>
      {/* Header */}
      <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
          <button className="btn ghost sm" onClick={onClose}>
            <Icon.X width={12} height={12}/>Close
          </button>
          <span className="faint mono" style={{ fontSize: 11 }}>{task.id}</span>
        </div>
        <div style={{ fontSize: 18, fontWeight: 500, letterSpacing: '-0.01em', fontFamily: 'var(--font-display)', lineHeight: 1.25 }}>
          {task.title}
        </div>
        <div className="row" style={{ gap: 6, marginTop: 8 }}>
          <StatusDot status={status}/>
          <span style={{ fontSize: 12, color: 'var(--fg-muted)', textTransform: 'capitalize' }}>{status}</span>
        </div>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
        <Field label="What this does">
          <div style={{ fontSize: 13, color: 'var(--fg-muted)', lineHeight: 1.55 }}>
            {task.summary}
          </div>
        </Field>

        {task.pageLabel && task.page && (
          <div style={{ marginTop: 16 }}>
            <Field label="Configure here">
              <div style={{
                padding: 10,
                border: '1px solid var(--hairline-strong)',
                borderRadius: 4,
                background: 'var(--surface-1)',
                display: 'flex', alignItems: 'center', gap: 8,
              }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: 13 }}>{task.pageLabel}</div>
                  <div className="faint mono" style={{ fontSize: 11 }}>/admin/{task.page}</div>
                </div>
                <button className="btn sm" onClick={() => onGo(task.page)}>
                  Open<Icon.External width={11} height={11}/>
                </button>
              </div>
            </Field>
          </div>
        )}

        {task.cli && (
          <div style={{ marginTop: 16 }}>
            <Field label="CLI">
              <Snippet text={task.cli} onCopy={copy} mono/>
            </Field>
          </div>
        )}

        {task.code && (
          <div style={{ marginTop: 16 }}>
            <Field label="Code">
              <Snippet text={task.code} onCopy={copy} mono multiline/>
            </Field>
          </div>
        )}

        {task.extraSnippets && task.extraSnippets.map(s => (
          <div key={s.label} style={{ marginTop: 12 }}>
            <Field label={s.label}>
              <Snippet text={s.code} onCopy={copy} mono/>
            </Field>
          </div>
        ))}

        {task.autoCheck && (
          <div style={{
            marginTop: 16, padding: 8,
            border: '1px solid var(--hairline)',
            borderRadius: 4,
            background: 'var(--surface-1)',
            fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)',
          }}>
            <Icon.Info width={11} height={11} style={{ verticalAlign: 'middle', marginRight: 6, opacity: 0.7 }}/>
            This task auto-completes once SharkAuth detects the corresponding state.
          </div>
        )}
      </div>

      {/* Footer actions */}
      <div style={{
        padding: 12, borderTop: '1px solid var(--hairline)',
        display: 'flex', gap: 6, alignItems: 'center',
        background: 'var(--surface-0)',
      }}>
        {status !== 'done' && (
          <button className="btn primary sm" onClick={() => onStatus('done')}>
            <Icon.Check width={11} height={11}/>Mark done
          </button>
        )}
        {status !== 'skipped' && (
          <button className="btn sm" onClick={() => onStatus('skipped')}>
            Skip
          </button>
        )}
        {status !== 'pending' && (
          <button className="btn ghost sm" onClick={() => onStatus(null)}>
            Reset
          </button>
        )}
        <div style={{ flex: 1 }}/>
        {task.page && (
          <button className="btn sm" onClick={() => onGo(task.page)}>
            Go<Icon.ChevronRight width={11} height={11}/>
          </button>
        )}
      </div>
    </div>
  );
}

function Field({ label, children }) {
  return (
    <div>
      <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 6, lineHeight: 1.5 }}>{label}</div>
      {children}
    </div>
  );
}

function Snippet({ text, onCopy, mono = true, multiline = false }) {
  return (
    <div style={{
      position: 'relative',
      border: '1px solid var(--hairline-strong)',
      borderRadius: 4,
      background: 'var(--surface-1)',
      padding: multiline ? '10px 36px 10px 10px' : '6px 36px 6px 10px',
      fontFamily: mono ? 'var(--font-mono)' : 'inherit',
      fontSize: 12, lineHeight: 1.55,
      color: 'var(--fg)',
      whiteSpace: multiline ? 'pre' : 'nowrap',
      overflowX: 'auto',
    }}>
      {text}
      <button
        onClick={() => onCopy(text)}
        title="Copy"
        style={{
          position: 'absolute', top: 4, right: 4,
          height: 22, width: 22,
          background: 'transparent', border: '1px solid transparent',
          color: 'var(--fg-muted)', cursor: 'pointer',
          borderRadius: 3,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}
        onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--hairline-strong)'; e.currentTarget.style.background = 'var(--surface-2)'; }}
        onMouseLeave={e => { e.currentTarget.style.borderColor = 'transparent'; e.currentTarget.style.background = 'transparent'; }}
      >
        <Icon.Copy width={11} height={11}/>
      </button>
    </div>
  );
}
