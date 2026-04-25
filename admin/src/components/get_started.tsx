// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'

// Get Started — dual-path operator console.
//
// Two integration paths sit side by side as checklists:
//   Path A — Proxy mode + hosted components (zero-code in user's app)
//   Path B — SDK (user builds UI, SharkAuth handles state + tokens)
//
// The admin focuses on one path at a time but can switch any time without
// losing progress. Each task has a status (pending/done/skipped), a deep
// link into the right page, and an optional right-side detail drawer with
// real config snippets / commands.
//
// Aesthetic: square (3-6px radius), 13px base, hairline borders, dense.
// Reference pages: users.tsx, agents_manage.tsx, sessions.tsx.

const STORAGE_KEY = 'shark_admin_onboarding_state_v2';

// ─── Persistence ──────────────────────────────────────────────────────────────

function loadState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const v = JSON.parse(raw);
    if (typeof v !== 'object' || v == null) return null;
    return v;
  } catch { return null; }
}

function saveState(s) {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(s)); } catch {}
}

function defaultState() {
  return {
    activePath: 'proxy', // 'proxy' | 'sdk'
    proxy: {},           // { taskId: 'done' | 'skipped' }
    sdk: {},
    drawer: null,        // { path, taskId } | null
  };
}

function markOnboarded() {
  try { localStorage.setItem('shark_admin_onboarded', '1'); } catch {}
}

// ─── Task definitions ─────────────────────────────────────────────────────────
// Each task: { id, title, summary, kind, ... }
//   kind: 'cmd' (CLI snippet), 'code' (code block), 'link' (deep link),
//         'check' (auto-detect via API)
//   target: page id for "Open" deep link
//   verify: optional async () => bool to auto-mark done

const PROXY_TASKS = [
  {
    id: 'proxy.upstream',
    title: 'Configure proxy upstream',
    summary: 'Point SharkAuth at the app you want to protect.',
    target: 'proxy',
    body: ({ devMode }) => ({
      lead: 'SharkAuth runs as a reverse proxy in front of your app. Define the upstream URL once — every request flows through SharkAuth before hitting your service.',
      blocks: [
        { kind: 'cmd', label: 'Quick start (CLI)', value: 'shark serve --proxy-upstream http://localhost:3000' },
        { kind: 'note', value: 'Or use the Proxy page to define multiple upstreams with path matching, header rules, and tier gates.' },
      ],
      cta: { label: 'Open Proxy page', target: 'proxy' },
    }),
  },
  {
    id: 'proxy.rule',
    title: 'Add a protect rule',
    summary: 'Decide which paths require auth — anonymous, authenticated, role, or scope.',
    target: 'proxy',
    body: () => ({
      lead: 'Rules use the `require:` grammar. Stack as many as you want — first match wins. Default-deny is a single rule away.',
      blocks: [
        {
          kind: 'code', label: 'Example rule',
          value: `name:    api-protected
match:   /api/*
require: authenticated
allow:   ""`,
        },
        { kind: 'note', value: 'Valid require values: anonymous, authenticated, agent, role:<name>, scope:<value>, tier:<tier>.' },
      ],
      cta: { label: 'Manage rules', target: 'proxy' },
    }),
  },
  {
    id: 'proxy.auth',
    title: 'Pick a login method',
    summary: 'Password, magic link, passkey, SSO — toggle what you want enabled.',
    target: 'authentication',
    body: () => ({
      lead: 'Hosted components use whatever methods you enable here. You can mix all of them; users see only what is on.',
      blocks: [
        { kind: 'note', value: 'Recommended starter: password + magic link. Add SSO once you have a provider.' },
      ],
      cta: { label: 'Configure auth methods', target: 'authentication' },
    }),
  },
  {
    id: 'proxy.redirect',
    title: 'Set the post-login redirect',
    summary: 'Where users land after successful login (defaults to "/").',
    target: 'branding',
    body: () => ({
      lead: 'The hosted login page sends users to this URL after their session is established. Use a path on your app, or a full URL for cross-origin redirects.',
      blocks: [
        { kind: 'code', label: 'Branding → Auth copy', value: 'post_login_redirect: /dashboard' },
      ],
      cta: { label: 'Open branding', target: 'branding' },
    }),
  },
  {
    id: 'proxy.brand',
    title: 'Brand the hosted login page',
    summary: 'Logo, colors, copy — everything that ships with the proxy.',
    target: 'branding',
    body: () => ({
      lead: 'Hosted pages render with your branding. Default is fine for prototypes; swap to your logo + accent before production.',
      blocks: [
        { kind: 'note', value: 'Logo, primary color, font, and per-surface copy live on one page. Use the live preview pane to iterate without reloading.' },
      ],
      cta: { label: 'Open BrandStudio', target: 'branding' },
    }),
  },
  {
    id: 'proxy.test',
    title: 'Test the proxy URL',
    summary: 'Hit your proxied app and confirm the login redirect.',
    target: 'proxy',
    verify: async () => {
      try {
        const s = await API.get('/admin/proxy/status');
        return s && (s.state === 'running' || s.running === true);
      } catch { return false; }
    },
    body: ({ origin }) => ({
      lead: 'Open your protected URL in a fresh tab. An unauthenticated request should redirect you to the SharkAuth hosted login page; logging in should drop you back at the original path.',
      blocks: [
        { kind: 'cmd', label: 'Smoke test', value: `curl -i ${origin}/` },
        { kind: 'note', value: 'Look for `Location: /shark/login` on a 302 response. That is the proxy doing its job.' },
      ],
      cta: { label: 'Open status panel', target: 'proxy' },
    }),
  },
];

const SDK_TASKS = [
  {
    id: 'sdk.app',
    title: 'Create an Application',
    summary: 'You need a publishable key — it lives on the App.',
    target: 'apps',
    verify: async () => {
      try {
        const a = await API.get('/admin/apps');
        const list = Array.isArray(a) ? a : (a?.apps || []);
        return list.length > 0;
      } catch { return false; }
    },
    body: () => ({
      lead: 'Apps map a frontend to its OAuth client_id. The publishable key (pk_live_…) is safe to ship in your client bundle.',
      blocks: [
        { kind: 'note', value: 'You can have many apps — one per environment, or one per frontend codebase. Each gets its own redirect URIs and rate limits.' },
      ],
      cta: { label: 'Open Applications', target: 'apps' },
    }),
  },
  {
    id: 'sdk.install',
    title: 'Install the SDK package',
    summary: 'React, vanilla TS, or Python — pick the one that fits.',
    target: null,
    body: () => ({
      lead: 'The React package wraps everything: provider, hooks, prebuilt components. Other stacks consume the TypeScript SDK directly.',
      blocks: [
        { kind: 'cmd', label: 'React (recommended)', value: 'npm install @shark-auth/react' },
        { kind: 'cmd', label: 'Vanilla TS / Node', value: 'npm install @shark-auth/sdk' },
        { kind: 'cmd', label: 'Python', value: 'pip install shark-auth' },
      ],
    }),
  },
  {
    id: 'sdk.provider',
    title: 'Mount SharkProvider',
    summary: 'Wrap your app once with the publishable key + auth URL.',
    target: null,
    body: ({ origin, firstAppKey }) => ({
      lead: 'The provider holds the access token, refreshes it silently, and exposes hooks like `useAuth`, `useUser`, `useSession`.',
      blocks: [
        {
          kind: 'code', label: 'app/layout.tsx',
          value: `import { SharkProvider } from '@shark-auth/react'

export default function Layout({ children }) {
  return (
    <SharkProvider
      authUrl="${origin}"
      publishableKey="${firstAppKey || 'pk_live_xxx'}"
    >
      {children}
    </SharkProvider>
  )
}`,
        },
      ],
    }),
  },
  {
    id: 'sdk.signin',
    title: 'Wire a SignIn button',
    summary: 'Drop in `<SignIn />` or call `startAuthFlow` yourself.',
    target: null,
    body: () => ({
      lead: 'The prebuilt component handles the OAuth 2.1 + PKCE handshake. State and code_verifier are persisted to sessionStorage; SharkAuth redirects back to /shark/callback when the user is done.',
      blocks: [
        {
          kind: 'code', label: 'Prebuilt button',
          value: `import { SignIn } from '@shark-auth/react'

<SignIn>Log in</SignIn>`,
        },
        {
          kind: 'code', label: 'Manual flow (advanced)',
          value: `import { startAuthFlow } from '@shark-auth/react/core'

const { url } = await startAuthFlow(
  window.location.href,
  authUrl,
  publishableKey,
)
window.location.href = url`,
        },
      ],
    }),
  },
  {
    id: 'sdk.callback',
    title: 'Mount the callback route',
    summary: 'Required: a /shark/callback route that runs `<SharkCallback />`.',
    target: null,
    body: () => ({
      lead: 'The callback URI is fixed at `${origin}/shark/callback`. SharkCallback parses `?code=`, exchanges it for an access token, then routes the user back to where they started.',
      blocks: [
        {
          kind: 'code', label: 'app/shark/callback/page.tsx',
          value: `import { SharkCallback } from '@shark-auth/react'

export default function Page() {
  return <SharkCallback />
}`,
        },
      ],
    }),
  },
  {
    id: 'sdk.session',
    title: 'Read the session',
    summary: 'Use `useUser()` / `useSession()` anywhere in your tree.',
    target: null,
    body: () => ({
      lead: 'Once a user signs in, hooks resolve synchronously on every render. Tokens refresh in the background — you never have to think about expiry.',
      blocks: [
        {
          kind: 'code', label: 'Component',
          value: `import { useUser } from '@shark-auth/react'

function Profile() {
  const { user, isLoading } = useUser()
  if (isLoading) return null
  if (!user) return <SignIn>Log in</SignIn>
  return <div>Hi, {user.email}</div>
}`,
        },
      ],
    }),
  },
];

const PATHS = {
  proxy: {
    id: 'proxy',
    title: 'Proxy mode',
    subtitle: 'Hosted components, zero code in your app',
    blurb: 'SharkAuth runs as a reverse proxy in front of your service. Login UI, callbacks, sessions — all handled by SharkAuth’s hosted pages.',
    icon: <Icon.Proxy width={14} height={14}/>,
    tasks: PROXY_TASKS,
    cli: 'shark serve --proxy-upstream http://localhost:3000',
  },
  sdk: {
    id: 'sdk',
    title: 'SDK',
    subtitle: 'You build the UI, we handle auth state',
    blurb: 'Install the React or TypeScript SDK in your app. Use hooks and components to drive login, sessions, and token refresh from your own UI.',
    icon: <Icon.Bolt width={14} height={14}/>,
    tasks: SDK_TASKS,
    cli: 'npm install @shark-auth/react',
  },
};

// ─── Tiny atoms ───────────────────────────────────────────────────────────────

const HAIRLINE = '1px solid var(--hairline)';
const HAIRLINE_STRONG = '1px solid var(--hairline-strong)';

function StatusDot({ status }) {
  const color =
    status === 'done' ? 'var(--success)' :
    status === 'skipped' ? 'var(--fg-faint)' :
    'var(--fg-dim)';
  const fill =
    status === 'done' ? 'var(--success)' :
    'transparent';
  return (
    <div style={{
      width: 14, height: 14, flexShrink: 0,
      borderRadius: 3,
      border: `1px solid ${color}`,
      background: fill,
      display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
    }}>
      {status === 'done' && <Icon.Check width={9} height={9} style={{ color: '#000' }}/>}
      {status === 'skipped' && (
        <div style={{ width: 6, height: 1, background: 'var(--fg-faint)' }}/>
      )}
    </div>
  );
}

function Kbd({ children }) {
  return (
    <span style={{
      fontFamily: 'var(--font-mono)',
      fontSize: 10,
      padding: '1px 5px',
      border: HAIRLINE_STRONG,
      borderRadius: 3,
      color: 'var(--fg-muted)',
      background: 'var(--surface-1)',
    }}>{children}</span>
  );
}

function CopyChip({ text, label = 'Copy' }) {
  const [copied, setCopied] = React.useState(false);
  return (
    <button
      onClick={(e) => {
        e.stopPropagation();
        try {
          navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 1100);
        } catch {}
      }}
      className="btn ghost sm"
      style={{ height: 22, padding: '0 7px', fontSize: 11 }}
    >
      {copied
        ? <><Icon.Check width={10} height={10} style={{ color: 'var(--success)' }}/> Copied</>
        : <><Icon.Copy width={10} height={10} style={{ opacity: 0.7 }}/> {label}</>}
    </button>
  );
}

// ─── Path tab strip ───────────────────────────────────────────────────────────

function PathTabs({ active, onChange, progress }) {
  return (
    <div style={{
      display: 'flex',
      border: HAIRLINE,
      borderRadius: 4,
      overflow: 'hidden',
      background: 'var(--surface-1)',
    }}>
      {Object.values(PATHS).map((p) => {
        const on = active === p.id;
        const prog = progress[p.id];
        return (
          <button
            key={p.id}
            onClick={() => onChange(p.id)}
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              gap: 10,
              padding: '12px 14px',
              background: on ? 'var(--surface-3)' : 'transparent',
              border: 'none',
              borderRight: p.id === 'proxy' ? HAIRLINE : 'none',
              cursor: 'pointer',
              textAlign: 'left',
              color: on ? 'var(--fg)' : 'var(--fg-muted)',
              transition: 'background 100ms',
            }}
            onMouseEnter={(e) => { if (!on) e.currentTarget.style.background = 'var(--surface-2)'; }}
            onMouseLeave={(e) => { if (!on) e.currentTarget.style.background = 'transparent'; }}
          >
            <div style={{
              width: 22, height: 22, flexShrink: 0,
              border: HAIRLINE_STRONG,
              borderRadius: 4,
              background: 'var(--surface-2)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              {p.icon}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
                <span style={{
                  fontFamily: 'var(--font-display)',
                  fontWeight: 600,
                  fontSize: 13,
                  letterSpacing: '-0.005em',
                }}>
                  {p.title}
                </span>
                <span style={{ fontSize: 10, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
                  Path {p.id === 'proxy' ? 'A' : 'B'}
                </span>
              </div>
              <div style={{
                fontSize: 11.5,
                color: 'var(--fg-dim)',
                marginTop: 2,
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
              }}>
                {p.subtitle}
              </div>
            </div>
            <div style={{
              fontSize: 10.5,
              fontFamily: 'var(--font-mono)',
              color: prog.done === prog.total ? 'var(--success)' : 'var(--fg-muted)',
              padding: '2px 6px',
              border: HAIRLINE_STRONG,
              borderRadius: 3,
              background: 'var(--surface-1)',
              flexShrink: 0,
            }}>
              {prog.done}/{prog.total}
            </div>
          </button>
        );
      })}
    </div>
  );
}

// ─── Task row ─────────────────────────────────────────────────────────────────

function TaskRow({ task, status, index, onOpen, onToggleDone, onSkip }) {
  const isDone = status === 'done';
  const isSkipped = status === 'skipped';
  return (
    <div
      onClick={onOpen}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: '10px 14px',
        borderBottom: HAIRLINE,
        background: 'var(--surface-1)',
        cursor: 'pointer',
        transition: 'background 80ms',
      }}
      onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--surface-2)'; }}
      onMouseLeave={(e) => { e.currentTarget.style.background = 'var(--surface-1)'; }}
    >
      <button
        onClick={(e) => { e.stopPropagation(); onToggleDone(); }}
        title={isDone ? 'Mark as pending' : 'Mark as done'}
        style={{
          background: 'none', border: 'none', padding: 0, cursor: 'pointer',
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        }}
      >
        <StatusDot status={status} />
      </button>
      <div style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 10.5,
        color: 'var(--fg-faint)',
        width: 22,
        flexShrink: 0,
      }}>
        {String(index + 1).padStart(2, '0')}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontSize: 13,
          fontWeight: 500,
          color: isDone || isSkipped ? 'var(--fg-muted)' : 'var(--fg)',
          textDecoration: isSkipped ? 'line-through' : 'none',
          textDecorationColor: 'var(--fg-faint)',
        }}>
          {task.title}
        </div>
        <div style={{ fontSize: 11.5, color: 'var(--fg-dim)', marginTop: 2 }}>
          {task.summary}
        </div>
      </div>
      <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
        {!isDone && !isSkipped && (
          <button
            className="btn ghost sm"
            onClick={(e) => { e.stopPropagation(); onSkip(); }}
            style={{ fontSize: 11, height: 24 }}
          >
            Skip
          </button>
        )}
        <button
          className="btn sm"
          onClick={(e) => { e.stopPropagation(); onOpen(); }}
          style={{ fontSize: 11, height: 24 }}
        >
          {isDone ? 'Review' : 'Open'} <Icon.ChevronRight width={10} height={10}/>
        </button>
      </div>
    </div>
  );
}

// ─── Detail drawer ────────────────────────────────────────────────────────────

function TaskDrawer({ pathId, task, status, onClose, onMarkDone, onSkip, onUnskip, onGoTo, ctx }) {
  if (!task) return null;
  const body = task.body ? task.body(ctx) : { lead: '', blocks: [] };
  const cta = body.cta;

  return (
    <>
      <div
        onClick={onClose}
        style={{
          position: 'fixed', inset: 0,
          background: 'rgba(0,0,0,0.45)',
          zIndex: 90,
        }}
      />
      <aside
        style={{
          position: 'fixed',
          right: 0, top: 0, bottom: 0,
          width: 440,
          background: 'var(--surface-0)',
          borderLeft: HAIRLINE_STRONG,
          zIndex: 100,
          display: 'flex',
          flexDirection: 'column',
          animation: 'shark-drawer-in 140ms ease-out',
        }}
      >
        {/* Header */}
        <div style={{
          padding: '14px 18px',
          borderBottom: HAIRLINE,
          display: 'flex',
          alignItems: 'flex-start',
          gap: 10,
          background: 'var(--surface-0)',
        }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{
              fontSize: 10,
              textTransform: 'uppercase',
              letterSpacing: '0.08em',
              color: 'var(--fg-dim)',
              marginBottom: 4,
            }}>
              Path {pathId === 'proxy' ? 'A · Proxy' : 'B · SDK'} · Task
            </div>
            <div style={{
              fontFamily: 'var(--font-display)',
              fontWeight: 600,
              fontSize: 17,
              letterSpacing: '-0.01em',
              lineHeight: 1.2,
            }}>
              {task.title}
            </div>
          </div>
          <button
            onClick={onClose}
            className="btn ghost icon sm"
            title="Close (Esc)"
          >
            <Icon.X width={11} height={11}/>
          </button>
        </div>

        {/* Status row */}
        <div style={{
          padding: '10px 18px',
          borderBottom: HAIRLINE,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          background: 'var(--surface-1)',
        }}>
          <StatusDot status={status}/>
          <span style={{ fontSize: 12, color: 'var(--fg-muted)' }}>
            {status === 'done' ? 'Completed' : status === 'skipped' ? 'Skipped' : 'Pending'}
          </span>
          <div style={{ flex: 1 }}/>
          {status === 'skipped'
            ? <button className="btn sm" onClick={onUnskip} style={{ fontSize: 11 }}>Restore</button>
            : status === 'done'
              ? <button className="btn ghost sm" onClick={onMarkDone} style={{ fontSize: 11 }}>Mark pending</button>
              : <>
                  <button className="btn ghost sm" onClick={onSkip} style={{ fontSize: 11 }}>Skip</button>
                  <button className="btn primary sm" onClick={onMarkDone} style={{ fontSize: 11 }}>
                    <Icon.Check width={10} height={10}/> Mark done
                  </button>
                </>
          }
        </div>

        {/* Body */}
        <div style={{
          flex: 1, overflow: 'auto', padding: '16px 18px 24px',
        }}>
          {body.lead && (
            <p style={{
              fontSize: 13,
              lineHeight: 1.55,
              color: 'var(--fg-muted)',
              margin: '0 0 16px',
              maxWidth: '60ch',
            }}>
              {body.lead}
            </p>
          )}

          {body.blocks?.map((b, i) => {
            if (b.kind === 'note') {
              return (
                <div key={i} style={{
                  padding: '8px 12px',
                  border: HAIRLINE,
                  background: 'var(--surface-1)',
                  borderRadius: 4,
                  marginBottom: 12,
                  display: 'flex',
                  gap: 8,
                  alignItems: 'flex-start',
                }}>
                  <Icon.Info width={12} height={12} style={{ marginTop: 2, color: 'var(--fg-dim)', flexShrink: 0 }}/>
                  <div style={{ fontSize: 12, color: 'var(--fg-muted)', lineHeight: 1.5 }}>
                    {b.value}
                  </div>
                </div>
              );
            }
            return (
              <div key={i} style={{ marginBottom: 14 }}>
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: 6,
                }}>
                  <span style={{
                    fontSize: 10,
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                    color: 'var(--fg-dim)',
                  }}>
                    {b.label}
                  </span>
                  <CopyChip text={b.value}/>
                </div>
                <pre style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 12,
                  lineHeight: 1.55,
                  color: 'var(--fg)',
                  background: 'var(--surface-1)',
                  border: HAIRLINE,
                  borderRadius: 4,
                  padding: '10px 12px',
                  margin: 0,
                  overflow: 'auto',
                  whiteSpace: 'pre',
                }}>
                  {b.kind === 'cmd' ? '$ ' + b.value : b.value}
                </pre>
              </div>
            );
          })}

          {cta && (
            <div style={{
              marginTop: 18,
              paddingTop: 14,
              borderTop: HAIRLINE,
              display: 'flex',
              gap: 8,
            }}>
              <button
                className="btn primary"
                onClick={() => { onGoTo(cta.target); }}
                style={{ fontSize: 12 }}
              >
                {cta.label} <Icon.External width={11} height={11}/>
              </button>
            </div>
          )}
        </div>
      </aside>
    </>
  );
}

// ─── Header ───────────────────────────────────────────────────────────────────

function PageHeader({ overallProgress, onSkipAll }) {
  const { done, total } = overallProgress;
  const pct = total === 0 ? 0 : Math.round((done / total) * 100);
  return (
    <div style={{
      padding: '20px 24px 18px',
      borderBottom: HAIRLINE,
      background: 'var(--surface-0)',
      display: 'flex',
      alignItems: 'flex-end',
      gap: 24,
    }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{
          fontSize: 10,
          textTransform: 'uppercase',
          letterSpacing: '0.1em',
          color: 'var(--fg-dim)',
          marginBottom: 6,
        }}>
          Onboarding · Two paths, your pick
        </div>
        <div style={{
          fontFamily: 'var(--font-display)',
          fontWeight: 700,
          fontSize: 26,
          letterSpacing: '-0.02em',
          lineHeight: 1.05,
        }}>
          Get SharkAuth in front of your app.
        </div>
        <div style={{
          fontSize: 13,
          color: 'var(--fg-muted)',
          marginTop: 6,
          maxWidth: '64ch',
          lineHeight: 1.5,
        }}>
          Two ways to ship: run SharkAuth as a proxy with hosted login pages, or
          drop our SDK into your own UI. Pick one to focus on — switch any time
          without losing progress on the other.
        </div>
      </div>
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-end',
        gap: 8,
        flexShrink: 0,
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '6px 10px',
          border: HAIRLINE_STRONG,
          background: 'var(--surface-1)',
          borderRadius: 4,
        }}>
          <div style={{
            width: 80, height: 4,
            background: 'var(--surface-3)',
            borderRadius: 2,
            overflow: 'hidden',
          }}>
            <div style={{
              width: `${pct}%`,
              height: '100%',
              background: pct === 100 ? 'var(--success)' : 'var(--fg)',
              transition: 'width 200ms ease-out',
            }}/>
          </div>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--fg-muted)',
          }}>
            {done}/{total}
          </span>
        </div>
        <button
          className="btn ghost sm"
          onClick={onSkipAll}
          style={{ fontSize: 11 }}
        >
          Skip onboarding <Kbd>esc</Kbd>
        </button>
      </div>
    </div>
  );
}

// ─── Main ─────────────────────────────────────────────────────────────────────

export function GetStarted({ setPage }) {
  const toast = useToast();

  const [state, setState] = React.useState(() => {
    const loaded = loadState();
    return loaded || defaultState();
  });

  React.useEffect(() => { saveState(state); }, [state]);

  const { data: appsData } = useAPI('/admin/apps');
  const apps = React.useMemo(() => {
    if (!appsData) return [];
    return Array.isArray(appsData) ? appsData : (appsData.apps || []);
  }, [appsData]);
  const firstAppKey = apps?.[0]?.publishable_key || apps?.[0]?.client_id || '';

  const ctx = React.useMemo(() => ({
    origin: typeof window !== 'undefined' ? window.location.origin : 'https://auth.example.com',
    devMode: false,
    firstAppKey,
  }), [firstAppKey]);

  // Auto-verify tasks (best-effort, silent on error)
  React.useEffect(() => {
    let cancelled = false;
    (async () => {
      const upd = { proxy: { ...state.proxy }, sdk: { ...state.sdk } };
      let changed = false;
      for (const path of ['proxy', 'sdk']) {
        for (const t of PATHS[path].tasks) {
          if (t.verify && !upd[path][t.id]) {
            try {
              const ok = await t.verify();
              if (ok && !cancelled) { upd[path][t.id] = 'done'; changed = true; }
            } catch {}
          }
        }
      }
      if (changed && !cancelled) {
        setState((s) => ({ ...s, proxy: upd.proxy, sdk: upd.sdk }));
      }
    })();
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Esc closes drawer or skips onboarding
  React.useEffect(() => {
    const onKey = (e) => {
      if (e.key !== 'Escape') return;
      if (state.drawer) setState((s) => ({ ...s, drawer: null }));
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [state.drawer]);

  const switchPath = (id) => setState((s) => ({ ...s, activePath: id }));
  const setTaskStatus = (pathId, taskId, status) => {
    setState((s) => {
      const next = { ...s };
      next[pathId] = { ...s[pathId] };
      if (status == null) delete next[pathId][taskId];
      else next[pathId][taskId] = status;
      return next;
    });
  };

  const openTask = (pathId, taskId) => setState((s) => ({ ...s, drawer: { pathId, taskId } }));
  const closeDrawer = () => setState((s) => ({ ...s, drawer: null }));

  const handleSkipAll = () => {
    markOnboarded();
    if (setPage) setPage('overview');
  };

  const goTo = (target) => {
    if (!target) return;
    if (setPage) setPage(target);
  };

  const progress = React.useMemo(() => {
    const m = {};
    for (const id of ['proxy', 'sdk']) {
      const tasks = PATHS[id].tasks;
      const done = tasks.filter((t) => state[id][t.id] === 'done').length;
      m[id] = { done, total: tasks.length };
    }
    return m;
  }, [state.proxy, state.sdk]);

  const overall = React.useMemo(() => {
    const t = progress.proxy.total + progress.sdk.total;
    const d = progress.proxy.done + progress.sdk.done;
    return { done: d, total: t };
  }, [progress]);

  const activePath = PATHS[state.activePath];
  const inactivePath = PATHS[state.activePath === 'proxy' ? 'sdk' : 'proxy'];
  const inactiveProg = progress[inactivePath.id];

  const drawerTask = state.drawer
    ? PATHS[state.drawer.pathId]?.tasks.find((t) => t.id === state.drawer.taskId)
    : null;
  const drawerStatus = state.drawer
    ? (state[state.drawer.pathId]?.[state.drawer.taskId] || null)
    : null;

  // Auto-mark complete -> celebrate once
  const allDone = overall.done === overall.total && overall.total > 0;
  const celebratedRef = React.useRef(false);
  React.useEffect(() => {
    if (allDone && !celebratedRef.current) {
      celebratedRef.current = true;
      markOnboarded();
      try { toast?.show?.('Onboarding complete'); } catch {}
    }
  }, [allDone, toast]);

  return (
    <div style={{
      height: '100%',
      overflow: 'auto',
      background: 'var(--bg)',
      color: 'var(--fg)',
    }}>
      <style>{`
        @keyframes shark-drawer-in {
          from { transform: translateX(8px); opacity: 0; }
          to   { transform: translateX(0);   opacity: 1; }
        }
      `}</style>

      <PageHeader overallProgress={overall} onSkipAll={handleSkipAll}/>

      <div style={{ padding: '20px 24px', maxWidth: 1100 }}>
        <PathTabs
          active={state.activePath}
          onChange={switchPath}
          progress={progress}
        />

        {/* Active path block */}
        <div style={{
          marginTop: 20,
          border: HAIRLINE,
          borderRadius: 5,
          background: 'var(--surface-0)',
          overflow: 'hidden',
        }}>
          <div style={{
            padding: '14px 18px',
            borderBottom: HAIRLINE,
            background: 'var(--surface-0)',
            display: 'flex',
            alignItems: 'flex-start',
            gap: 14,
          }}>
            <div style={{
              width: 28, height: 28,
              border: HAIRLINE_STRONG,
              borderRadius: 4,
              background: 'var(--surface-2)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              flexShrink: 0,
            }}>
              {activePath.icon}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{
                display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 4,
              }}>
                <span style={{
                  fontFamily: 'var(--font-display)',
                  fontWeight: 600,
                  fontSize: 15,
                  letterSpacing: '-0.01em',
                }}>
                  {activePath.title}
                </span>
                <span className="chip ghost" style={{ height: 18, fontSize: 10 }}>
                  Active
                </span>
              </div>
              <div style={{
                fontSize: 12.5,
                color: 'var(--fg-muted)',
                lineHeight: 1.5,
                maxWidth: '70ch',
              }}>
                {activePath.blurb}
              </div>
            </div>
            <div style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10.5,
              color: 'var(--fg-muted)',
              padding: '4px 8px',
              border: HAIRLINE,
              borderRadius: 3,
              background: 'var(--surface-1)',
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              flexShrink: 0,
            }}>
              <span>{activePath.cli}</span>
              <CopyChip text={activePath.cli} label=""/>
            </div>
          </div>

          {/* Task list */}
          <div>
            {activePath.tasks.map((t, i) => {
              const status = state[activePath.id][t.id] || null;
              return (
                <TaskRow
                  key={t.id}
                  task={t}
                  status={status}
                  index={i}
                  onOpen={() => openTask(activePath.id, t.id)}
                  onToggleDone={() => setTaskStatus(activePath.id, t.id, status === 'done' ? null : 'done')}
                  onSkip={() => setTaskStatus(activePath.id, t.id, 'skipped')}
                />
              );
            })}
          </div>
        </div>

        {/* Inactive path teaser — switch any time */}
        <div style={{
          marginTop: 16,
          padding: '12px 16px',
          border: HAIRLINE,
          borderRadius: 4,
          background: 'var(--surface-1)',
          display: 'flex',
          alignItems: 'center',
          gap: 14,
        }}>
          <div style={{
            width: 22, height: 22,
            border: HAIRLINE_STRONG,
            borderRadius: 4,
            background: 'var(--surface-2)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            flexShrink: 0,
          }}>
            {inactivePath.icon}
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{
              fontSize: 12.5,
              color: 'var(--fg-muted)',
              lineHeight: 1.5,
            }}>
              <span style={{ color: 'var(--fg)', fontWeight: 500 }}>
                Path {inactivePath.id === 'proxy' ? 'A' : 'B'} · {inactivePath.title}
              </span>
              {' '}— {inactivePath.subtitle}.
              {inactiveProg.done > 0 && (
                <> Progress saved: <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--fg-muted)' }}>
                  {inactiveProg.done}/{inactiveProg.total}
                </span>.</>
              )}
            </div>
          </div>
          <button
            className="btn sm"
            onClick={() => switchPath(inactivePath.id)}
            style={{ fontSize: 11 }}
          >
            Switch to {inactivePath.title} <Icon.ChevronRight width={10} height={10}/>
          </button>
        </div>

        {/* Footer hints */}
        <div style={{
          marginTop: 24,
          padding: '14px 16px',
          border: HAIRLINE,
          borderRadius: 4,
          background: 'var(--surface-0)',
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
          gap: 14,
        }}>
          <FooterHint
            icon={<Icon.Terminal width={12} height={12}/>}
            title="Prefer the CLI?"
            body={<>Run <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11.5, color: 'var(--fg)' }}>shark serve --dev</code> for an ephemeral instance with relaxed CORS and dev inbox.</>}
          />
          <FooterHint
            icon={<Icon.Schema width={12} height={12}/>}
            title="Need agent auth?"
            body="Both paths support agent OAuth clients. Configure them under Agents once humans are flowing."
          />
          <FooterHint
            icon={<Icon.Info width={12} height={12}/>}
            title="Stuck on a step?"
            body={<>Each task links to the page where it lives. Hit <Kbd>cmd</Kbd>+<Kbd>k</Kbd> to jump anywhere.</>}
          />
        </div>
      </div>

      {state.drawer && drawerTask && (
        <TaskDrawer
          pathId={state.drawer.pathId}
          task={drawerTask}
          status={drawerStatus}
          ctx={ctx}
          onClose={closeDrawer}
          onMarkDone={() => setTaskStatus(state.drawer.pathId, state.drawer.taskId, drawerStatus === 'done' ? null : 'done')}
          onSkip={() => setTaskStatus(state.drawer.pathId, state.drawer.taskId, 'skipped')}
          onUnskip={() => setTaskStatus(state.drawer.pathId, state.drawer.taskId, null)}
          onGoTo={(t) => { closeDrawer(); goTo(t); }}
        />
      )}
    </div>
  );
}

function FooterHint({ icon, title, body }) {
  return (
    <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
      <div style={{
        width: 22, height: 22,
        border: HAIRLINE_STRONG,
        borderRadius: 4,
        background: 'var(--surface-2)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        flexShrink: 0,
        color: 'var(--fg-muted)',
      }}>
        {icon}
      </div>
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, marginBottom: 2 }}>{title}</div>
        <div style={{ fontSize: 11.5, color: 'var(--fg-dim)', lineHeight: 1.5 }}>
          {body}
        </div>
      </div>
    </div>
  );
}

export default GetStarted;
