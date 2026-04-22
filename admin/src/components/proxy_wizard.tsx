// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Proxy empty-state onboarding wizard.
// Updated to support per-app gateway rule creation (Phase Wave E).

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

export function ProxyWizard({ appId, onComplete, autofocus }) {
  const toastRaw = useToast();
  // Defensive toast helper
  const toast = {
    success: (m) => toastRaw?.success ? toastRaw.success(m) : console.log('Toast:', m),
    error: (m) => toastRaw?.error ? toastRaw.error(m) : console.error('Toast Error:', m),
  };

  const [isLocal, setIsLocal] = React.useState(true);
  const [upstream, setUpstream] = React.useState('');
  const [path, setPath] = React.useState('/*');
  const [methods, setMethods] = React.useState(new Set(METHODS));
  const [ruleKind, setRuleKind] = React.useState('require');
  const [step, setStep] = React.useState(appId ? 2 : 1);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [launched, setLaunched] = React.useState(false);

  // Auto-detect local upstreams
  React.useEffect(() => {
    if (upstream.includes('localhost') || upstream.includes('127.0.0.1')) {
      setIsLocal(true);
    }
  }, [upstream]);

  const inputRef = React.useRef(null);
  React.useEffect(() => {
    if (autofocus && inputRef.current) {
      inputRef.current.focus();
    }
  }, [autofocus]);

  const step1Ok = appId ? true : isValidURL(upstream);
  const step2Ok = path.trim().length > 0 && (methods instanceof Set ? methods.size > 0 : true);

  const toggleMethod = (m) => {
    const next = new Set(methods);
    if (next.has(m)) next.delete(m); else next.add(m);
    setMethods(next);
  };

  const launch = async () => {
    setBusy(true); setError(null);
    try {
      let finalAppId = appId;
      
      // If we're creating from scratch (Step 1), we need an application first
      if (!finalAppId) {
        const urlObj = new URL(upstream);
        const appName = isLocal ? `Local Dev (${urlObj.port || '3000'})` : urlObj.hostname;
        const appResp = await API.post('/admin/apps', {
          name: appName,
          integration_mode: 'proxy',
          proxy_public_domain: isLocal ? 'localhost:8080' : urlObj.hostname,
          proxy_protected_url: upstream.trim(),
        });
        finalAppId = appResp.id;
      }

      // Champion Tier: The "Frontend First" Rule Pattern (Recommendation 1B)
      // We automatically inject a bypass for static assets so Next.js/Vite don't break.
      if (path === '/*') {
        await API.post('/admin/proxy/rules/db', {
          app_id: finalAppId,
          name: 'Static Assets Bypass',
          pattern: '/_next/*',
          methods: ['GET'],
          allow: 'anonymous',
          priority: 100,
        });
        await API.post('/admin/proxy/rules/db', {
          app_id: finalAppId,
          name: 'Common Assets Bypass',
          pattern: '/*.{css,js,png,jpg,svg,woff2}',
          methods: ['GET'],
          allow: 'anonymous',
          priority: 90,
        });
      }

      const payload = {
        app_id: finalAppId,
        name: `Main Route Protection`,
        pattern: path.trim(),
        methods: (methods instanceof Set && methods.size === METHODS.length) ? [] : Array.from(methods),
        enabled: true,
        priority: 0,
      };

      if (ruleKind === 'anonymous') {
        payload.allow = 'anonymous';
        payload.require = '';
      } else {
        payload.require = 'authenticated';
        payload.allow = '';
      }
      
      await API.post('/admin/proxy/rules/db', payload);
      toast.success('Proxy Gateway launched');
      setLaunched(true);
      setStep(3);
      if (typeof onComplete === 'function') onComplete({ upstream, path, appId: finalAppId });
    } catch (e) {
      setError(e?.message || 'Failed to create rule');
      toast.error('Launch failed: ' + (e?.message || 'unknown'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card" style={{ padding: 0, overflow: 'hidden', border: '1px solid var(--hairline-bright)' }}>
      <div style={{ padding: '14px 20px', borderBottom: HAIRLINE, background: 'var(--surface-1)' }}>
        <div className="row" style={{ gap: 8 }}>
          <span className="chip agent" style={{ height: 18, fontSize: 10 }}>Champion Tier</span>
          <span style={{ fontSize: 14, fontWeight: 600, fontFamily: 'var(--font-display)' }}>
            {appId ? 'Add path protection' : 'Launch Proxy Gateway in 60 seconds'}
          </span>
        </div>
      </div>

      <div style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 18 }}>
        {/* Step 1 - Upstream */}
        {!appId && (
          <div>
            <div className="row" style={{justifyContent: 'space-between', alignItems: 'center', marginBottom: 10}}>
               <StepHeader n={1} title="Target Service (Upstream)" active={step === 1} done={step > 1}/>
               {step === 1 && (
                 <div className="row" style={{gap: 6}}>
                   <button className={"btn sm " + (isLocal ? 'primary' : 'ghost')} onClick={() => setIsLocal(true)}>Local Dev</button>
                   <button className={"btn sm " + (!isLocal ? 'primary' : 'ghost')} onClick={() => setIsLocal(false)}>Production</button>
                 </div>
               )}
            </div>
            <div style={{ paddingLeft: 32 }}>
              {step === 1 ? (
                <>
                  <input
                    ref={inputRef}
                    value={upstream}
                    onChange={e => setUpstream(e.target.value)}
                    placeholder={isLocal ? "http://localhost:3000" : "https://internal-api.cluster.local"}
                    className="mono"
                    style={{ ...INPUT_STYLE, fontFamily: 'var(--font-mono)' }}
                    onKeyDown={e => { if (e.key === 'Enter' && step1Ok) setStep(2); }}
                  />
                  <div className="faint" style={{ fontSize: 11, marginTop: 8, lineHeight: 1.4 }}>
                    {isLocal 
                      ? "SharkAuth will tunnel traffic from :8080 to your local dev server. Public Domain will be set to localhost:8080."
                      : "Direct production proxy. You'll need to point your DNS CNAME to this SharkAuth instance."}
                  </div>
                  <div className="row" style={{ marginTop: 14 }}>
                    <button className="btn primary" disabled={!step1Ok} onClick={() => setStep(2)}>
                      Next: Configure Rules <Icon.ChevronRight width={10} height={10}/>
                    </button>
                  </div>
                </>
              ) : (
                <div className="mono" style={{fontSize: 12, color: 'var(--fg-dim)'}}>{upstream}</div>
              )}
            </div>
          </div>
        )}

        {/* Step 2 - Rules */}
        <div>
          <StepHeader n={appId ? 1 : 2} title="Route Protection" active={step === 2} done={step > 2}/>
          {step === 2 && (
            <div style={{ paddingLeft: 32, marginTop: 10, display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div style={{background: 'var(--surface-3)', padding: 10, borderRadius: 5, border: '1px solid var(--hairline)'}}>
                 <div className="row" style={{gap: 8, color: 'var(--success)', marginBottom: 4}}>
                   <Icon.Bolt width={12}/>
                   <span style={{fontSize: 11, fontWeight: 600}}>Champion Tier: Auto-Bypass Enabled</span>
                 </div>
                 <div className="faint" style={{fontSize: 10.5}}>
                   We'll automatically allow <code className="mono">/_next/*</code> and static assets (CSS/JS) to prevent breaking your UI.
                 </div>
              </div>

              <label className="col" style={{ gap: 4 }}>
                <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Protected Path</span>
                <input
                  value={path}
                  onChange={e => setPath(e.target.value)}
                  placeholder="/*"
                  className="mono"
                  style={{ ...INPUT_STYLE, fontFamily: 'var(--font-mono)' }}
                />
              </label>

              <div className="row" style={{gap: 20}}>
                <div className="col" style={{ gap: 6, flex: 1 }}>
                  <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Methods</span>
                  <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                    {METHODS.map(m => {
                      const on = methods instanceof Set ? methods.has(m) : false;
                      return (
                        <button key={m} onClick={() => toggleMethod(m)} className={"chip " + (on ? 'solid' : 'ghost')}
                          style={{ height: 22, fontSize: 10, cursor: 'pointer', padding: '0 6px' }}>{m}</button>
                      );
                    })}
                  </div>
                </div>

                <div className="col" style={{ gap: 6, flex: 1 }}>
                  <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Access Policy</span>
                  <div className="row" style={{ gap: 10 }}>
                    {RULE_KINDS.map(k => (
                      <label key={k.id} className="row" style={{ gap: 6, fontSize: 11.5, cursor: 'pointer' }}>
                        <input type="radio" checked={ruleKind === k.id} onChange={() => setRuleKind(k.id)}/>
                        {k.label}
                      </label>
                    ))}
                  </div>
                </div>
              </div>

              <div className="row" style={{ marginTop: 6 }}>
                <button className="btn primary" disabled={!step2Ok || busy} onClick={launch} style={{height: 38, padding: '0 24px'}}>
                  {busy ? 'Launching…' : (appId ? 'Add Protection' : 'Launch Gateway')}
                </button>
              </div>
            </div>
          )}
        </div>

        {/* Step 3 - Success */}
        {step === 3 && (
          <div style={{ paddingLeft: 32, marginTop: 4 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, color: 'var(--success)' }}>
              <Icon.Check width={16} height={16}/>
              <span style={{ fontSize: 15, fontWeight: 600 }}>Proxy Gateway Live</span>
            </div>
            <div style={{ marginTop: 12, background: 'var(--surface-2)', padding: 16, borderRadius: 6, border: '1px solid var(--hairline-strong)' }}>
               <div style={{fontSize: 12, fontWeight: 600, marginBottom: 8}}>Test your application:</div>
               <div className="row" style={{gap: 10, marginBottom: 12}}>
                  <div style={{flex: 1, padding: '8px 12px', background: '#000', borderRadius: 4, color: '#fff', fontSize: 13}} className="mono">
                    http://localhost:8080
                  </div>
                  <button className="btn primary sm" onClick={() => window.open('http://localhost:8080', '_blank')}>Open</button>
               </div>
               <div className="faint" style={{fontSize: 11, lineHeight: 1.5}}>
                 <b>Note:</b> Do not use <code className="mono">localhost:3000</code>. That bypasses the gateway. Send all traffic to the SharkAuth port (:8080).
               </div>
            </div>
            <button className="btn ghost sm" onClick={() => setStep(appId ? 2 : 1)} style={{ marginTop: 16 }}>
              Configure another route
            </button>
          </div>
        )}
      </div>
      {error && <div style={{ padding: '12px 20px', background: 'var(--danger-bg)', color: 'var(--danger)', fontSize: 12, borderTop: '1px solid var(--danger)' }}>{error}</div>}
    </div>
  );
}
