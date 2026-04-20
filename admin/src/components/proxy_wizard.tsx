// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Proxy empty-state onboarding wizard.
// Three steps in a vertical stepper:
//   1. Paste upstream URL (validates format)
//   2. Pick a route to protect (path + methods + rule type)
//   3. Launch — creates a DB-backed override rule via POST /admin/proxy/rules/db
//      and instructs the user to restart with --proxy-upstream since there is
//      no /admin/proxy/enable endpoint. Offers a "Open preview" link.
//
// Reusable: mounted in proxy_config.tsx (replaces ProxyDisabledEmpty) and in
// get_started.tsx as the centerpiece of the first-login flow.

const HAIRLINE = '1px solid var(--hairline)';
const INPUT_STYLE = {
  height: 34, padding: '0 12px',
  background: 'var(--surface-2)',
  border: '1px solid var(--hairline-strong)',
  borderRadius: 5,
  color: 'var(--fg)',
  fontSize: 13,
  outline: 'none',
  width: '100%',
};

const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];
const RULE_KINDS = [
  { id: 'require', label: 'Require auth' },
  { id: 'anonymous', label: 'Allow anonymous' },
  { id: 'optional', label: 'Optional' },
];

function isValidURL(s) {
  if (!s) return false;
  try {
    const u = new URL(s);
    return u.protocol === 'http:' || u.protocol === 'https:';
  } catch { return false; }
}

function StepHeader({ n, title, active, done }) {
  const bg = done ? 'var(--success)' : active ? 'var(--fg)' : 'var(--surface-3)';
  const fg = done || active ? 'var(--bg)' : 'var(--fg-muted)';
  return (
    <div className="row" style={{ gap: 10, alignItems: 'center' }}>
      <div style={{
        width: 22, height: 22, borderRadius: 11,
        background: bg, color: fg,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 11, fontWeight: 600, fontVariantNumeric: 'tabular-nums',
      }}>
        {done ? <Icon.Check width={11} height={11}/> : n}
      </div>
      <div style={{
        fontSize: 13, fontWeight: 600,
        fontFamily: 'var(--font-display)',
        color: active || done ? 'var(--fg)' : 'var(--fg-muted)',
      }}>{title}</div>
    </div>
  );
}

export function ProxyWizard({ onComplete, autofocus }) {
  const toast = useToast();
  const [upstream, setUpstream] = React.useState('');
  const [path, setPath] = React.useState('/*');
  const [methods, setMethods] = React.useState(new Set(METHODS));
  const [ruleKind, setRuleKind] = React.useState('require');
  const [step, setStep] = React.useState(1);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [launched, setLaunched] = React.useState(false);

  const inputRef = React.useRef(null);
  React.useEffect(() => {
    if (autofocus && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }
  }, [autofocus]);

  const step1Ok = isValidURL(upstream);
  const step2Ok = path.trim().length > 0 && methods.size > 0;

  const toggleMethod = (m) => {
    const next = new Set(methods);
    if (next.has(m)) next.delete(m); else next.add(m);
    setMethods(next);
  };

  const launch = async () => {
    setBusy(true); setError(null);
    try {
      const payload = {
        name: 'Onboarding: ' + (new URL(upstream).host || 'upstream'),
        pattern: path.trim(),
        methods: methods.size === METHODS.length ? [] : Array.from(methods),
        enabled: true,
        priority: 0,
      };
      if (ruleKind === 'anonymous') {
        payload.allow = 'anonymous';
        payload.require = '';
      } else if (ruleKind === 'optional') {
        // Optional = allow anonymous pass-through, agent/user headers still
        // injected when present. Backend models this as allow:anonymous since
        // the inject logic is upstream of the rule decision.
        payload.allow = 'anonymous';
        payload.require = '';
      } else {
        payload.require = 'authenticated';
        payload.allow = '';
      }
      await API.post('/admin/proxy/rules/db', payload);
      toast?.success?.('Proxy configured');
      setLaunched(true);
      setStep(3);
      if (typeof onComplete === 'function') onComplete({ upstream, path });
    } catch (e) {
      setError(e?.message || 'Failed to create rule');
      toast?.error?.('Rule create failed: ' + (e?.message || 'unknown'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
      <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
        <div className="row" style={{ gap: 8 }}>
          <span className="chip agent" style={{ height: 18, fontSize: 10 }}>Wizard</span>
          <span style={{ fontSize: 14, fontWeight: 600, fontFamily: 'var(--font-display)' }}>
            Protect an upstream in 60 seconds
          </span>
        </div>
        <div className="faint" style={{ fontSize: 11.5, marginTop: 4 }}>
          Creates a DB-backed proxy rule — no config file edits required.
        </div>
      </div>

      <div style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 18 }}>
        {/* Step 1 */}
        <div>
          <StepHeader n={1} title="Paste upstream URL" active={step === 1} done={step > 1}/>
          <div style={{ paddingLeft: 32, marginTop: 10 }}>
            <input
              ref={inputRef}
              value={upstream}
              onChange={e => setUpstream(e.target.value)}
              placeholder="http://localhost:3000"
              className="mono"
              style={{ ...INPUT_STYLE, fontFamily: 'var(--font-mono)' }}
              onKeyDown={e => { if (e.key === 'Enter' && step1Ok) setStep(2); }}
            />
            {!step1Ok && upstream.length > 0 && (
              <div style={{ fontSize: 11, color: 'var(--danger)', marginTop: 6 }}>
                Enter a full URL starting with http:// or https://.
              </div>
            )}
            {step === 1 && (
              <div className="row" style={{ marginTop: 10 }}>
                <button className="btn primary sm" disabled={!step1Ok} onClick={() => setStep(2)}>
                  Next <Icon.ChevronRight width={10} height={10}/>
                </button>
              </div>
            )}
          </div>
        </div>

        {/* Step 2 */}
        <div>
          <StepHeader n={2} title="Pick a route to protect" active={step === 2} done={step > 2}/>
          {step >= 2 && (
            <div style={{ paddingLeft: 32, marginTop: 10, display: 'flex', flexDirection: 'column', gap: 12 }}>
              <label className="col" style={{ gap: 4 }}>
                <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Path pattern</span>
                <input
                  value={path}
                  onChange={e => setPath(e.target.value)}
                  placeholder="/*"
                  className="mono"
                  style={{ ...INPUT_STYLE, fontFamily: 'var(--font-mono)' }}
                />
                <span className="faint" style={{ fontSize: 10.5 }}>
                  Default <span className="mono">/*</span> matches every path. Use
                  <span className="mono"> /api/* </span> to scope.
                </span>
              </label>

              <div className="col" style={{ gap: 4 }}>
                <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Methods</span>
                <div className="row" style={{ gap: 6, flexWrap: 'wrap' }}>
                  {METHODS.map(m => {
                    const on = methods.has(m);
                    return (
                      <button key={m} onClick={() => toggleMethod(m)}
                        className={'btn sm ' + (on ? '' : 'ghost')}
                        style={{ fontSize: 11, fontFamily: 'var(--font-mono)' }}>
                        {m}
                      </button>
                    );
                  })}
                </div>
              </div>

              <div className="col" style={{ gap: 4 }}>
                <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Rule type</span>
                <div className="row" style={{ gap: 0, border: HAIRLINE, borderRadius: 5, overflow: 'hidden', width: 'fit-content' }}>
                  {RULE_KINDS.map(k => {
                    const on = ruleKind === k.id;
                    return (
                      <button key={k.id} onClick={() => setRuleKind(k.id)}
                        style={{
                          padding: '6px 14px',
                          background: on ? 'var(--fg)' : 'transparent',
                          color: on ? 'var(--bg)' : 'var(--fg)',
                          border: 'none',
                          fontSize: 12,
                          cursor: 'pointer',
                          borderRight: HAIRLINE,
                        }}>{k.label}</button>
                    );
                  })}
                </div>
              </div>

              {step === 2 && (
                <div className="row" style={{ gap: 8 }}>
                  <button className="btn ghost sm" onClick={() => setStep(1)}>Back</button>
                  <button className="btn primary sm" disabled={!step2Ok} onClick={() => setStep(3)}>
                    Next <Icon.ChevronRight width={10} height={10}/>
                  </button>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Step 3 */}
        <div>
          <StepHeader n={3} title="Launch" active={step === 3} done={launched}/>
          {step >= 3 && (
            <div style={{ paddingLeft: 32, marginTop: 10 }}>
              {!launched ? (
                <>
                  <div style={{ fontSize: 12.5, color: 'var(--fg-muted)', lineHeight: 1.55, marginBottom: 12 }}>
                    We'll create the rule in the database. Restart Shark with the
                    upstream flag to flip the proxy on:
                  </div>
                  <CLIFooter command={`shark serve --proxy-upstream ${upstream}`}/>
                  {error && (
                    <div style={{
                      marginTop: 10, padding: 10, borderRadius: 4,
                      border: '1px solid color-mix(in oklch, var(--danger) 35%, var(--hairline-strong))',
                      background: 'color-mix(in oklch, var(--danger) 8%, var(--surface-2))',
                      color: 'var(--danger)', fontSize: 12,
                    }}>{error}</div>
                  )}
                  <div className="row" style={{ gap: 8, marginTop: 12 }}>
                    <button className="btn ghost sm" onClick={() => setStep(2)}>Back</button>
                    <button className="btn primary" disabled={busy} onClick={launch}>
                      {busy ? 'Creating rule…' : 'Create rule'}
                    </button>
                  </div>
                </>
              ) : (
                <div>
                  <div className="row" style={{ gap: 8, marginBottom: 10 }}>
                    <span className="chip success" style={{ height: 20 }}>
                      <Icon.Check width={10} height={10}/> Rule created
                    </span>
                  </div>
                  <div style={{ fontSize: 12.5, color: 'var(--fg-muted)', lineHeight: 1.55, marginBottom: 12 }}>
                    Restart <span className="mono">shark serve</span> with
                    <span className="mono"> --proxy-upstream {upstream}</span> to
                    activate. Your rule will apply immediately on restart.
                  </div>
                  <div className="row" style={{ gap: 8 }}>
                    <a className="btn sm" href={upstream} target="_blank" rel="noreferrer">
                      Open preview <Icon.ChevronRight width={10} height={10}/>
                    </a>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
