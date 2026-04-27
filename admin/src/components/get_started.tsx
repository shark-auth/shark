// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Get Started — agent-first onboarding. Rebuilt Wave 1.7 Edit 4.
// Design: monochrome, square, hairline, 13px base, matches users.tsx density.
// Three tracks: Agent (primary), Human auth, Both. Track selection sticky in localStorage.
// Each section is an accordion row; clicking opens the task drawer.
// Auto-verify probes backend to flip tasks done.

const STORAGE_KEY = 'shark_get_started_v2';

// ─── Section markers (smoke-tested, do not rename) ───────────────────────────
export const SECTION_IDS = [
  'why-sharkauth',
  'sixty-second-setup',
  'your-first-agent',
  'dpop-token-ten-lines',
  'delegation-chain-demo',
  'next-steps',
];

// ─── Why SharkAuth (Section 1) ───────────────────────────────────────────────
const WHY_BULLETS = [
  'Per-token, per-agent, and per-customer revocation — five independent layers, RFC-correct.',
  'DPoP-bound tokens (RFC 9449): stolen access tokens cannot be replayed by a different client.',
  'Delegation chain audit (RFC 8693): every act claim is logged with full chain traceability.',
];

// ─── 60-second setup tasks (Section 2) ──────────────────────────────────────
const SETUP_TASKS = [
  {
    id: 'setup.serve',
    title: 'Run shark serve',
    summary: 'Starts the auth server on port 8080. First boot prints a one-time admin key.',
    cli: 'shark serve',
    extraSnippets: [
      { label: 'Dev mode (local email capture)', code: 'shark serve --dev' },
    ],
  },
  {
    id: 'setup.adminkey',
    title: 'Copy the admin key',
    summary: 'The one-time key appears in stdout on first boot. Paste it in the banner at the top of this dashboard. Shown once — store it.',
    page: 'overview',
    pageLabel: 'Overview → Admin key banner',
    autoCheck: 'hasAdminKey',
  },
];

// ─── Your first agent tasks (Section 3) ─────────────────────────────────────
const AGENT_TASKS = [
  {
    id: 'user.create',
    title: 'Create your first human user',
    summary: 'Your first end-user. Real humans authenticate against shark — sessions, MFA, password reset all built-in.',
    page: 'users',
    pageLabel: '→ /users (create-drawer)',
    cli: 'shark user create --email demo@example.com',
    autoCheck: 'hasUser',
  },
  {
    id: 'agent.register',
    title: 'Register your first agent',
    summary: 'Each agent gets a DPoP-bound credential. Tokens are cryptographically tied to the agent\'s keypair — token theft is useless without the private key.',
    page: 'agents',
    pageLabel: '→ /agents/new',
    cli: 'shark agent register --name my-first-agent',
    autoCheck: 'hasAgent',
  },
  {
    id: 'agent.credentials',
    title: 'Get credentials',
    summary: 'After registration, copy the client_id and client_secret from the agent detail drawer. The secret is shown once.',
    page: 'agents',
    pageLabel: '→ /agents',
    code: `# Store securely — shown once
CLIENT_ID=ag_01abc...
CLIENT_SECRET=sk_...`,
    autoCheck: 'hasAgent',
  },
  {
    id: 'agent.policy',
    title: 'Set a delegation policy',
    summary: 'Grant this agent permission to act on behalf of a user or another agent within a scope.',
    page: 'agents',
    pageLabel: '→ Agent detail → Delegation Policies',
    cli: 'shark agents policy set <agent-id> --may-act <other-agent-id> --scope email:read',
    autoCheck: 'hasAgentPolicy',
  },
];

// ─── DPoP-bound token in 10 lines (Section 4) ───────────────────────────────
// DATA-MARKER: dpop-python-10-liner — smoke test asserts this string is present in the bundle
const DPOP_PYTHON_SNIPPET = `import shark_auth

client = shark_auth.Client(
    issuer="https://auth.example.com",
    client_id=CLIENT_ID,
    client_secret=CLIENT_SECRET,
)

prover = shark_auth.DPoPProver.generate()
token = client.oauth.get_token_with_dpop(prover)
print(token.access_token)`;

const DPOP_CURL_SNIPPET = `# 1. Generate EC P-256 keypair
openssl ecparam -name prime256v1 -genkey -noout -out dpop.pem
openssl ec -in dpop.pem -pubout -out dpop_pub.pem

# 2. Request token with DPoP header
curl -X POST https://auth.example.com/oauth/token \\
  -H "DPoP: <signed-proof-jwt>" \\
  -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET"`;

const DPOP_TASKS = [
  {
    id: 'dpop.generate',
    title: 'Generate a DPoP keypair',
    summary: 'DPoP (RFC 9449) binds tokens to a private key. Stolen tokens cannot be replayed from a different client.',
    code: DPOP_PYTHON_SNIPPET,
    extraSnippets: [
      { label: 'curl (no SDK)', code: DPOP_CURL_SNIPPET },
      { label: 'TypeScript — coming W18', code: '// TypeScript SDK ships W18. Use curl alternative above.' },
    ],
    autoCheck: 'hasDPoPToken',
  },
];

// ─── Delegation chain demo (Section 5) ──────────────────────────────────────
const DELEGATION_TASKS = [
  {
    id: 'delegation.run',
    title: 'Run the delegation chain demo',
    summary: 'Issues a DPoP-bound token, exchanges it for a delegated sub-token, and prints the full act claim chain.',
    cli: 'shark demo delegation-with-trace',
    autoCheck: 'hasDelegationEvent',
  },
  {
    id: 'delegation.audit',
    title: 'Verify the audit trail',
    summary: 'Open the audit log filtered to delegation events. Each row shows the act claim (agent) alongside the subject (user).',
    page: 'audit',
    pageLabel: '→ /audit?delegation=true',
    cli: 'curl -sH "Authorization: Bearer $ADMIN_KEY" "http://localhost:8080/api/v1/admin/audit?delegation=true"',
    autoCheck: 'hasDelegationEvent',
  },
];

// ─── Next steps (Section 6) ──────────────────────────────────────────────────
const NEXT_STEPS = [
  {
    id: 'next.docs',
    title: 'Read the API reference',
    summary: 'Full REST + SDK docs including DPoP flows, token exchange parameters, revocation endpoints.',
    href: 'https://sharkauth.dev/docs',
    label: 'sharkauth.dev/docs →',
  },
  {
    id: 'next.screencast',
    title: 'Watch the screencast',
    summary: '90-second walkthrough: install, first agent, first DPoP token, delegation chain.',
    href: 'https://sharkauth.dev/screencast',
    label: 'sharkauth.dev/screencast →',
  },
  {
    id: 'next.github',
    title: 'GitHub',
    summary: 'Source code, issues, roadmap. Star if you find it useful.',
    href: 'https://github.com/sharkauth/sharkauth',
    label: 'github.com/sharkauth/sharkauth →',
  },
];

// ─── All sections in display order ───────────────────────────────────────────
// SECTION_IDS above are the smoke-verified marker strings.
const SECTIONS = [
  { id: 'why-sharkauth',       label: 'Why SharkAuth',              kind: 'why' },
  { id: 'sixty-second-setup',  label: '60-second setup',            kind: 'tasks', tasks: SETUP_TASKS },
  { id: 'your-first-agent',    label: 'Your first agent',           kind: 'tasks', tasks: AGENT_TASKS },
  { id: 'dpop-token-ten-lines',label: 'DPoP-bound token in 10 lines', kind: 'tasks', tasks: DPOP_TASKS },
  { id: 'delegation-chain-demo', label: 'Delegation chain demo',    kind: 'tasks', tasks: DELEGATION_TASKS },
  { id: 'next-steps',          label: 'Next steps',                 kind: 'next' },
];

// ─── Human auth tasks (secondary track) ──────────────────────────────────────
const HUMAN_TASKS = [
  {
    id: 'human.email',
    title: 'Configure email delivery',
    summary: 'SharkAuth needs an outbound mail provider to send magic links and password resets.',
    page: 'settings',
    pageLabel: 'Settings → Email Delivery',
  },
  {
    id: 'human.providers',
    title: 'Enable OAuth providers',
    summary: 'Add Google, GitHub, Apple, or Discord social login. Each provider needs a client_id + client_secret.',
    page: 'settings',
    pageLabel: 'Settings → OAuth & SSO',
  },
  {
    id: 'human.magiclink',
    title: 'Test magic-link login',
    summary: 'Send a magic link to your own address. Confirm delivery and that it redirects to the right post-login URL.',
    page: 'dev-email',
    pageLabel: 'Dev Email (captured locally in --dev mode)',
  },
  {
    id: 'human.sdk',
    title: 'Ship to your stack',
    summary: 'Python SDK (dogfood mode): install from git. TypeScript SDK coming v0.2 (W18-W19).',
    cli: 'pip install git+https://github.com/sharkauth/shark#subdirectory=sdk/python',
    extraSnippets: [
      { label: 'TypeScript SDK', code: '// TypeScript SDK coming v0.2 (W18-W19). No install snippet yet.' },
    ],
  },
];

// ─── State helpers ────────────────────────────────────────────────────────────
function loadState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { track: 'agent', tasks: {}, open: {} };
    const p = JSON.parse(raw);
    return {
      track: ['agent', 'human', 'both'].includes(p.track) ? p.track : 'agent',
      tasks: p.tasks || {},
      open: p.open || {},
    };
  } catch { return { track: 'agent', tasks: {}, open: {} }; }
}

function persist(state) {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(state)); } catch {}
}

// ─── Auto-verify hook ─────────────────────────────────────────────────────────
function useAutoChecks() {
  const [checks, setChecks] = React.useState({
    hasAdminKey: false,
    hasUser: false,
    hasAgent: false,
    hasAgentPolicy: false,
    hasDPoPToken: false,
    hasDelegationEvent: false,
  });

  const probe = React.useCallback(async () => {
    const next = { hasAdminKey: false, hasUser: false, hasAgent: false, hasAgentPolicy: false, hasDPoPToken: false, hasDelegationEvent: false };
    try {
      const r = await API.get('/admin/bootstrap/status');
      next.hasAdminKey = !!(r && (r.consumed || r.bootstrapped));
    } catch {}
    try {
      const r = await API.get('/admin/users?limit=1');
      const users = r?.users || r?.data || (Array.isArray(r) ? r : []);
      next.hasUser = Array.isArray(users) && users.length > 0;
    } catch {}
    try {
      const r = await API.get('/admin/agents?limit=1');
      const agents = r?.agents || r?.data || (Array.isArray(r) ? r : []);
      next.hasAgent = Array.isArray(agents) && agents.length > 0;
    } catch {}
    try {
      const r = await API.get('/admin/agents?limit=1');
      const agents = r?.agents || r?.data || (Array.isArray(r) ? r : []);
      if (Array.isArray(agents) && agents.length > 0) {
        const agentId = agents[0]?.id || agents[0]?.agent_id;
        if (agentId) {
          const pr = await API.get(`/admin/agents/${agentId}/policies`);
          const policies = pr?.policies || pr?.data || (Array.isArray(pr) ? pr : []);
          next.hasAgentPolicy = Array.isArray(policies) && policies.length > 0;
        }
      }
    } catch {}
    try {
      const r = await API.get('/admin/audit?event_type=oauth.token.issued&limit=5');
      const events = r?.events || r?.data || (Array.isArray(r) ? r : []);
      next.hasDPoPToken = Array.isArray(events) && events.some(e => e.dpop || e.metadata?.dpop);
    } catch {}
    try {
      const r = await API.get('/admin/audit?delegation=true&limit=1');
      const events = r?.events || r?.data || (Array.isArray(r) ? r : []);
      next.hasDelegationEvent = Array.isArray(events) && events.length > 0;
    } catch {}
    setChecks(next);
  }, []);

  React.useEffect(() => { probe(); }, [probe]);
  return { checks, refresh: probe };
}

// ─── Status dot ───────────────────────────────────────────────────────────────
function StatusDot({ status }) {
  const color = status === 'done' ? 'var(--success)'
    : status === 'skipped' ? 'var(--warn)'
    : status === 'blocked' ? 'var(--danger)'
    : 'var(--fg-faint)';
  const fill = status === 'done' || status === 'blocked' || status === 'skipped';
  return (
    <span style={{
      width: 8, height: 8, borderRadius: '50%',
      background: fill ? color : 'transparent',
      border: '1px solid ' + color,
      flexShrink: 0,
      display: 'inline-block',
    }} aria-label={status || 'pending'} />
  );
}

// ─── Main component ───────────────────────────────────────────────────────────
export function GetStarted({ setPage }) {
  const initial = React.useMemo(loadState, []);
  const [track, setTrack] = React.useState(initial.track);
  const [tasks, setTasks] = React.useState(initial.tasks);
  const [openSections, setOpenSections] = React.useState(
    initial.open && Object.keys(initial.open).length > 0
      ? initial.open
      : { 'why-sharkauth': true, 'sixty-second-setup': true, 'your-first-agent': true, 'dpop-token-ten-lines': true, 'delegation-chain-demo': false, 'next-steps': false }
  );
  const [selectedId, setSelectedId] = React.useState(null);
  const toast = useToast();
  const { checks, refresh: refreshChecks } = useAutoChecks();

  React.useEffect(() => { persist({ track, tasks, open: openSections }); }, [track, tasks, openSections]);

  // Apply auto-checks
  React.useEffect(() => {
    setTasks(prev => {
      const next = { ...prev };
      let changed = false;
      const allTasks = [...SETUP_TASKS, ...AGENT_TASKS, ...DPOP_TASKS, ...DELEGATION_TASKS, ...HUMAN_TASKS];
      for (const t of allTasks) {
        if (t.autoCheck && checks[t.autoCheck] && next[t.id] !== 'done' && next[t.id] !== 'skipped') {
          next[t.id] = 'done';
          changed = true;
        }
      }
      return changed ? next : prev;
    });
  }, [checks]);

  // All task lists for this track
  const trackTasks = React.useMemo(() => {
    if (track === 'human') return HUMAN_TASKS;
    if (track === 'both') return [...SETUP_TASKS, ...AGENT_TASKS, ...DPOP_TASKS, ...DELEGATION_TASKS, ...HUMAN_TASKS];
    return [...SETUP_TASKS, ...AGENT_TASKS, ...DPOP_TASKS, ...DELEGATION_TASKS];
  }, [track]);

  const total = trackTasks.length;
  const done = trackTasks.filter(t => tasks[t.id] === 'done').length;
  const skipped = trackTasks.filter(t => tasks[t.id] === 'skipped').length;
  const pct = Math.round((done / Math.max(total, 1)) * 100);

  // All task list for drawer lookup
  const allTasksFlat = [...SETUP_TASKS, ...AGENT_TASKS, ...DPOP_TASKS, ...DELEGATION_TASKS, ...HUMAN_TASKS];
  const selectedTask = allTasksFlat.find(t => t.id === selectedId) || null;

  const setStatus = (id, status) => {
    setTasks(prev => {
      const next = { ...prev };
      if (status === null) delete next[id]; else next[id] = status;
      return next;
    });
  };

  const toggleSection = (id) => {
    setOpenSections(prev => ({ ...prev, [id]: !prev[id] }));
  };

  const dismissOnboarding = () => {
    try { localStorage.setItem('shark_admin_onboarded', '1'); } catch {}
    if (typeof setPage === 'function') setPage('overview');
  };

  // Sections to display based on track
  const visibleSections = track === 'human'
    ? [{ id: 'why-sharkauth', label: 'Why SharkAuth', kind: 'why' }, { id: 'next-steps', label: 'Next steps', kind: 'next' }]
    : SECTIONS;

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>

        {/* ── Toolbar ─────────────────────────────────────────────────────── */}
        <div style={{
          padding: '10px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 8, alignItems: 'center',
          background: 'var(--surface-0)',
        }}>
          {/* Track selector */}
          <div style={{ display: 'flex', gap: 0 }}>
            <TabBtn on={track === 'agent'} onClick={() => { setTrack('agent'); setSelectedId(null); }}>
              Agent integration
            </TabBtn>
            <TabBtn on={track === 'human'} onClick={() => { setTrack('human'); setSelectedId(null); }}>
              Human auth
            </TabBtn>
            <TabBtn on={track === 'both'} onClick={() => { setTrack('both'); setSelectedId(null); }}>
              Both
            </TabBtn>
          </div>
          <div style={{ flex: 1 }}/>
          <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
            {done}/{total} done{skipped ? ` · ${skipped} skipped` : ''}
          </span>
          <div style={{ width: 80, height: 4, background: 'var(--surface-2)', borderRadius: 2, overflow: 'hidden' }}>
            <div style={{ width: pct + '%', height: '100%', background: 'var(--fg)' }}/>
          </div>
          <button className="btn ghost sm" onClick={refreshChecks} title="Re-probe state">
            <Icon.Refresh width={11} height={11}/>Recheck
          </button>
          <button className="btn sm" onClick={dismissOnboarding}>
            Skip to dashboard
          </button>
        </div>

        {/* ── Sub-header ──────────────────────────────────────────────────── */}
        <div style={{
          padding: '7px 16px',
          borderBottom: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)',
        }}>
          {track === 'human'
            ? 'Add human sign-in to your product: email, social, magic link, SSO, passkeys.'
            : track === 'both'
            ? 'Full integration: agent DPoP-bound tokens + human sign-in. Agent track first — it is the moat.'
            : 'Auth for products that give customers their own agents. Get to first DPoP-bound token in 5 minutes.'}
        </div>

        {/* ── Section accordion body ───────────────────────────────────────── */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {visibleSections.map(section => (
            <SectionBlock
              key={section.id}
              section={section}
              open={!!openSections[section.id]}
              onToggle={() => toggleSection(section.id)}
              tasks={section.tasks || []}
              taskStates={tasks}
              checks={checks}
              selectedId={selectedId}
              onSelect={setSelectedId}
              track={track}
              humanTasks={HUMAN_TASKS}
            />
          ))}
        </div>

        {/* ── Footer ──────────────────────────────────────────────────────── */}
        <div style={{
          padding: '8px 16px',
          borderTop: '1px solid var(--hairline)',
          display: 'flex', alignItems: 'center', gap: 8,
          background: 'var(--surface-0)',
          fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)',
        }}>
          <span>{total - done - skipped} pending</span>
          <span>·</span>
          <span>{done} done</span>
          {skipped > 0 && <><span>·</span><span>{skipped} skipped</span></>}
          <div style={{ flex: 1 }}/>
          <span className="faint">Status auto-syncs from /admin/agents, /admin/audit.</span>
        </div>

        <CLIFooter command="shark help"/>
      </div>

      {/* ── Task drawer ─────────────────────────────────────────────────── */}
      {selectedTask && (
        <TaskDrawer
          task={selectedTask}
          status={tasks[selectedTask.id] || 'pending'}
          onClose={() => setSelectedId(null)}
          onStatus={(s) => {
            setStatus(selectedTask.id, s);
            if (s === 'done') toast.success('Marked done');
            else if (s === 'skipped') toast.success('Skipped');
            else if (s === null) toast.success('Reset');
          }}
          onGo={(p) => {
            try { localStorage.setItem('shark_admin_onboarded', '1'); } catch {}
            if (typeof setPage === 'function') setPage(p);
          }}
          checks={checks}
        />
      )}
    </div>
  );
}

// ─── Section block ────────────────────────────────────────────────────────────
function SectionBlock({ section, open, onToggle, tasks: sectionTasks, taskStates, checks, selectedId, onSelect, track, humanTasks }) {
  return (
    <div data-section={section.id} style={{ borderBottom: '1px solid var(--hairline)' }}>
      {/* Section header — clickable to toggle */}
      <div
        onClick={onToggle}
        style={{
          padding: '9px 16px',
          display: 'flex', alignItems: 'center', gap: 8,
          cursor: 'pointer',
          background: 'var(--surface-1)',
          userSelect: 'none',
        }}
      >
        <Icon.ChevronRight
          width={11} height={11}
          style={{
            opacity: 0.6, flexShrink: 0,
            transform: open ? 'rotate(90deg)' : 'none',
            transition: 'transform 120ms',
          }}
        />
        <span style={{ fontSize: 12, fontWeight: 600, letterSpacing: '0.04em', textTransform: 'uppercase', color: 'var(--fg-muted)' }}>
          {section.label}
        </span>
        {section.kind === 'tasks' && sectionTasks.length > 0 && (
          <SectionProgress tasks={sectionTasks} taskStates={taskStates} />
        )}
      </div>

      {/* Section body */}
      {open && (
        <div>
          {section.kind === 'why' && <WhySection />}
          {section.kind === 'next' && <NextStepsSection />}
          {section.kind === 'tasks' && sectionTasks.map((t, idx) => {
            const status = taskStates[t.id] || 'pending';
            const autoVerified = t.autoCheck && checks[t.autoCheck] && status === 'done';
            return (
              <div
                key={t.id}
                onClick={() => onSelect(t.id)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 0,
                  borderBottom: '1px solid var(--hairline)',
                  cursor: 'pointer',
                  background: selectedId === t.id ? 'var(--surface-2)' : 'var(--surface-0)',
                  padding: '8px 16px',
                  transition: 'background 80ms',
                }}
                onMouseEnter={e => { if (selectedId !== t.id) e.currentTarget.style.background = 'var(--surface-1)'; }}
                onMouseLeave={e => { if (selectedId !== t.id) e.currentTarget.style.background = 'var(--surface-0)'; }}
              >
                <span className="mono faint" style={{ fontSize: 10.5, width: 22, flexShrink: 0 }}>
                  {String(idx + 1).padStart(2, '0')}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontWeight: 500, fontSize: 13 }}>{t.title}</div>
                  <div className="faint" style={{
                    fontSize: 11, lineHeight: 1.5,
                    overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 560,
                  }}>
                    {t.summary}
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginLeft: 12, flexShrink: 0 }}>
                  <StatusDot status={status}/>
                  <span style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'capitalize', minWidth: 48 }}>{status}</span>
                  {autoVerified && (
                    <span className="faint mono" style={{ fontSize: 10 }}>auto</span>
                  )}
                </div>
                <Icon.ChevronRight width={11} height={11} style={{ opacity: 0.4, marginLeft: 8, flexShrink: 0 }}/>
              </div>
            );
          })}
          {section.kind === 'tasks' && track === 'human' && section.id === 'next-steps' && (
            humanTasks.map((t, idx) => {
              const status = taskStates[t.id] || 'pending';
              return (
                <div key={t.id} onClick={() => onSelect(t.id)}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 0,
                    borderBottom: '1px solid var(--hairline)',
                    cursor: 'pointer',
                    background: selectedId === t.id ? 'var(--surface-2)' : 'var(--surface-0)',
                    padding: '8px 16px',
                  }}>
                  <span className="mono faint" style={{ fontSize: 10.5, width: 22, flexShrink: 0 }}>
                    {String(idx + 1).padStart(2, '0')}
                  </span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 500, fontSize: 13 }}>{t.title}</div>
                    <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 560 }}>
                      {t.summary}
                    </div>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginLeft: 12, flexShrink: 0 }}>
                    <StatusDot status={status}/>
                    <span style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'capitalize', minWidth: 48 }}>{status}</span>
                  </div>
                  <Icon.ChevronRight width={11} height={11} style={{ opacity: 0.4, marginLeft: 8, flexShrink: 0 }}/>
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}

function SectionProgress({ tasks, taskStates }) {
  const done = tasks.filter(t => taskStates[t.id] === 'done').length;
  const total = tasks.length;
  if (total === 0) return null;
  return (
    <span className="faint mono" style={{ fontSize: 10, marginLeft: 4, lineHeight: 1.5 }}>
      {done}/{total}
    </span>
  );
}

// ─── Why SharkAuth section body ───────────────────────────────────────────────
function WhySection() {
  return (
    <div style={{ padding: '12px 16px 14px 38px' }}>
      {WHY_BULLETS.map((b, i) => (
        <div key={i} style={{
          display: 'flex', gap: 8, alignItems: 'flex-start',
          marginBottom: i < WHY_BULLETS.length - 1 ? 8 : 0,
        }}>
          <span style={{ width: 4, height: 4, borderRadius: '50%', background: 'var(--fg-muted)', marginTop: 7, flexShrink: 0 }}/>
          <span style={{ fontSize: 13, lineHeight: 1.55, color: 'var(--fg-secondary)' }}>{b}</span>
        </div>
      ))}
    </div>
  );
}

// ─── Next steps section body ──────────────────────────────────────────────────
function NextStepsSection() {
  return (
    <div style={{ padding: '12px 16px 14px 38px', display: 'flex', flexDirection: 'column', gap: 10 }}>
      {NEXT_STEPS.map(ns => (
        <div key={ns.id} style={{
          border: '1px solid var(--hairline)',
          borderRadius: 4,
          padding: '10px 12px',
          background: 'var(--surface-0)',
        }}>
          <div style={{ fontWeight: 500, fontSize: 13, marginBottom: 3 }}>{ns.title}</div>
          <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginBottom: 6 }}>{ns.summary}</div>
          <a
            href={ns.href}
            target="_blank"
            rel="noopener noreferrer"
            style={{ fontSize: 11, color: 'var(--fg)', textDecoration: 'underline', fontFamily: 'var(--font-mono)' }}
          >
            {ns.label}
          </a>
        </div>
      ))}
    </div>
  );
}

// ─── Tab button ───────────────────────────────────────────────────────────────
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
        borderRadius: 4,
        marginRight: -1,
        cursor: 'pointer',
      }}
    >{children}</button>
  );
}

// ─── Task drawer ──────────────────────────────────────────────────────────────
function TaskDrawer({ task, status, onClose, onStatus, onGo, checks }) {
  const toast = useToast();
  const [snippetTab, setSnippetTab] = React.useState(0);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const copy = async (text) => {
    try { await navigator.clipboard.writeText(text); toast.success('Copied'); }
    catch { toast.error('Copy failed'); }
  };

  // Build snippet tabs: primary code first, then extraSnippets
  const snippets = [];
  if (task.code) snippets.push({ label: 'Python', code: task.code });
  if (task.cli) snippets.push({ label: 'CLI', code: task.cli });
  if (task.extraSnippets) snippets.push(...task.extraSnippets);

  const activeSnippet = snippets[Math.min(snippetTab, snippets.length - 1)];

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
          {task.autoCheck && checks[task.autoCheck] && status === 'done' && (
            <span className="faint mono" style={{ fontSize: 10 }}>auto-verified</span>
          )}
        </div>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
        <Field label="What this does">
          <div style={{ fontSize: 13, color: 'var(--fg-muted)', lineHeight: 1.55 }}>{task.summary}</div>
        </Field>

        {task.pageLabel && task.page && (
          <div style={{ marginTop: 16 }}>
            <Field label="Configure here">
              <div style={{
                padding: 10, border: '1px solid var(--hairline-strong)',
                borderRadius: 4, background: 'var(--surface-1)',
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

        {task.href && (
          <div style={{ marginTop: 16 }}>
            <Field label="Link">
              <a href={task.href} target="_blank" rel="noopener noreferrer"
                style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--fg)', textDecoration: 'underline' }}>
                {task.label || task.href}
              </a>
            </Field>
          </div>
        )}

        {/* Tabbed code/CLI snippets */}
        {snippets.length > 0 && (
          <div style={{ marginTop: 16 }}>
            <Field label="Code">
              {snippets.length > 1 && (
                <div style={{ display: 'flex', gap: 0, marginBottom: 6 }}>
                  {snippets.map((s, i) => (
                    <button
                      key={s.label}
                      onClick={() => setSnippetTab(i)}
                      style={{
                        padding: '3px 10px', fontSize: 11,
                        fontWeight: snippetTab === i ? 600 : 400,
                        background: snippetTab === i ? 'var(--surface-3)' : 'var(--surface-2)',
                        color: snippetTab === i ? 'var(--fg)' : 'var(--fg-muted)',
                        border: '1px solid var(--hairline)',
                        borderRadius: 3,
                        marginRight: -1,
                        cursor: 'pointer',
                      }}
                    >{s.label}</button>
                  ))}
                </div>
              )}
              {activeSnippet && (
                <Snippet text={activeSnippet.code} onCopy={copy} multiline={activeSnippet.code.includes('\n')}/>
              )}
            </Field>
          </div>
        )}

        {task.autoCheck && (
          <div style={{
            marginTop: 16, padding: 8,
            border: '1px solid var(--hairline)',
            borderRadius: 4, background: 'var(--surface-1)',
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
          <button className="btn sm" onClick={() => onStatus('skipped')}>Skip</button>
        )}
        {status !== 'pending' && (
          <button className="btn ghost sm" onClick={() => onStatus(null)}>Reset</button>
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
      <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 6, lineHeight: 1.5 }}>
        {label}
      </div>
      {children}
    </div>
  );
}

function Snippet({ text, onCopy, multiline = false }) {
  return (
    <div style={{
      position: 'relative',
      border: '1px solid var(--hairline-strong)',
      borderRadius: 4,
      background: 'var(--surface-1)',
      padding: multiline ? '10px 36px 10px 10px' : '6px 36px 6px 10px',
      fontFamily: 'var(--font-mono)',
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
