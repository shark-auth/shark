// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// ─── Tokens ───────────────────────────────────────────────────────────────────
const HAIRLINE = '1px solid var(--hairline)';
const HAIRLINE_STRONG = '1px solid var(--hairline-strong)';
const SECTION_LABEL = {
  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
  color: 'var(--fg-dim)', fontWeight: 600,
};
const INPUT = {
  width: '100%', boxSizing: 'border-box', height: 28, padding: '0 8px',
  background: 'var(--surface-1)', border: HAIRLINE_STRONG, borderRadius: 4,
  fontSize: 13, color: 'var(--fg)',
};
const TEXTAREA = {
  ...INPUT, height: 'auto', padding: 8, minHeight: 64,
  fontFamily: 'var(--font-mono)', fontSize: 12, resize: 'vertical',
};

// ─── Lifecycle hook ───────────────────────────────────────────────────────────

function useLifecycle() {
  const [status, setStatus] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [missing, setMissing] = React.useState(false);

  const fetch_ = React.useCallback(async () => {
    try {
      const r = await API.get('/admin/proxy/lifecycle');
      setStatus(r?.data || null);
      setMissing(false);
    } catch (e) {
      if (e.message?.includes('404') || e.message === 'HTTP 404') {
        setMissing(true);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetch_();
    const iv = setInterval(fetch_, 5000);
    return () => clearInterval(iv);
  }, [fetch_]);

  const act = async (action) => {
    const r = await API.post('/admin/proxy/' + action);
    setStatus(r?.data || null);
  };

  return { status, loading, missing, refresh: fetch_, act };
}

// ─── Breaker / runtime status ─────────────────────────────────────────────────

function useRuntimeStatus() {
  const [data, setData] = React.useState(null);
  const fetch_ = React.useCallback(async () => {
    try {
      const r = await API.get('/admin/proxy/status');
      setData(r?.data || null);
    } catch (e) { /* 404 = disabled, ignore */ }
  }, []);
  React.useEffect(() => {
    fetch_();
    const iv = setInterval(fetch_, 5000);
    return () => clearInterval(iv);
  }, [fetch_]);
  return { data, refresh: fetch_ };
}

// ─── Rule require validator ───────────────────────────────────────────────────

const REQUIRE_RE = /^(anonymous|authenticated|agent|role:.+|global_role:.+|permission:.+:.+|scope:.+|tier:.+)$/;
const ALLOW_VALUES = ['', 'anonymous'];

function validateRule(f) {
  const errs = {};
  if (!f.name?.trim()) errs.name = 'Required';
  if (!f.pattern?.trim()) errs.pattern = 'Required';
  if (f.pattern && !f.pattern.startsWith('/')) errs.pattern = 'Must start with /';
  const hasRequire = f.require?.trim();
  const hasAllow = f.allow?.trim();
  if (!hasRequire && !hasAllow) errs.require = 'Set require or allow';
  if (hasRequire && hasAllow) errs.require = 'Set only one of require / allow';
  if (hasRequire && !REQUIRE_RE.test(f.require.trim())) errs.require = 'Unknown require value';
  if (hasAllow && !ALLOW_VALUES.includes(f.allow.trim())) errs.allow = 'Only "anonymous" accepted';
  return errs;
}

const BLANK = {
  name: '', pattern: '', methods: '', require: 'authenticated', allow: '',
  scopes: '', enabled: true, priority: 0, tier_match: '', m2m: false, app_id: '',
};

// ─── Power toggle (the headline button) ───────────────────────────────────────

function PowerToggle({ status, onAction }) {
  const [busy, setBusy] = React.useState(false);
  const toast = useToast();
  const state = status?.state_str || 'unknown';
  const isRunning = state === 'running';
  const isReloading = state === 'reloading';

  const click = async () => {
    if (busy || isReloading) return;
    if (isRunning && !window.confirm('Stop the proxy? In-flight requests will drain.')) return;
    setBusy(true);
    try {
      await onAction(isRunning ? 'stop' : 'start');
      toast.success(isRunning ? 'Proxy stopped' : 'Proxy started');
    } catch (e) {
      toast.error(e.message || 'Action failed');
    } finally {
      setBusy(false);
    }
  };

  const tone = isRunning ? 'success' : (isReloading ? 'warn' : 'fg-faint');
  const label = isReloading ? 'Reloading' : (isRunning ? 'Stop proxy' : 'Start proxy');
  const dotColor = isRunning ? 'var(--success)' : (isReloading ? 'var(--warn)' : 'var(--fg-faint)');

  return (
    <button
      onClick={click}
      disabled={busy || isReloading}
      title={isRunning ? 'Click to stop' : 'Click to start'}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 8,
        height: 30, padding: '0 12px',
        background: isRunning ? 'var(--surface-2)' : 'var(--fg)',
        color: isRunning ? 'var(--fg)' : 'var(--bg-0, #000)',
        border: isRunning ? HAIRLINE_STRONG : '1px solid var(--fg)',
        borderRadius: 5, cursor: busy ? 'wait' : 'pointer',
        fontSize: 12, fontWeight: 600, letterSpacing: '0.01em',
        transition: 'background 100ms ease',
      }}
    >
      <span style={{
        width: 8, height: 8, borderRadius: '50%',
        background: dotColor,
        boxShadow: isRunning ? `0 0 0 3px color-mix(in oklch, ${dotColor} 25%, transparent)` : 'none',
        animation: isReloading ? 'pulse 1s ease-in-out infinite' : 'none',
      }}/>
      {busy ? '…' : label}
    </button>
  );
}

// ─── Status strip ─────────────────────────────────────────────────────────────

function StatusStrip({ lifecycle, runtime }) {
  const state = lifecycle.status?.state_str || 'unknown';
  const stateTone =
    state === 'running' ? 'var(--success)' :
    state === 'reloading' ? 'var(--warn)' :
    state === 'error' ? 'var(--danger)' :
    'var(--fg-faint)';
  const breakerState = runtime?.state || '—';
  const breakerTone = breakerState === 'closed' ? 'var(--success)' : breakerState === 'open' ? 'var(--danger)' : 'var(--warn)';

  const startedAt = lifecycle.status?.started_at;
  const uptime = startedAt ? formatUptime(startedAt) : '—';

  const items = [
    { label: 'State', value: state, tone: stateTone, mono: true },
    { label: 'Listeners', value: lifecycle.status?.listeners ?? 0, mono: true },
    { label: 'Rules', value: lifecycle.status?.rules_loaded ?? 0, mono: true },
    { label: 'Uptime', value: uptime, mono: true },
    { label: 'Breaker', value: breakerState, tone: breakerTone, mono: true },
    { label: 'Failures', value: runtime?.failures ?? 0, mono: true },
    { label: 'Cache', value: runtime?.cache_size ?? 0, mono: true },
    { label: 'Last latency', value: runtime?.last_latency_ms != null ? `${runtime.last_latency_ms}ms` : '—', mono: true },
  ];

  return (
    <div style={{
      display: 'grid',
      gridTemplateColumns: 'repeat(auto-fit, minmax(110px, 1fr))',
      borderTop: HAIRLINE, borderBottom: HAIRLINE,
      background: 'var(--surface-1)',
    }}>
      {items.map((it, i) => (
        <div key={it.label} style={{
          padding: '8px 14px',
          borderRight: i < items.length - 1 ? HAIRLINE : 'none',
          minWidth: 0,
        }}>
          <div style={{ ...SECTION_LABEL, marginBottom: 3 }}>{it.label}</div>
          <div className={it.mono ? 'mono' : ''} style={{
            fontSize: 13, fontWeight: 500, color: it.tone || 'var(--fg)',
            whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
          }}>{it.value}</div>
        </div>
      ))}
    </div>
  );
}

function formatUptime(iso) {
  try {
    const ms = Date.now() - new Date(iso).getTime();
    if (ms < 0) return '—';
    const s = Math.floor(ms / 1000);
    if (s < 60) return s + 's';
    const m = Math.floor(s / 60);
    if (m < 60) return m + 'm';
    const h = Math.floor(m / 60);
    if (h < 24) return h + 'h ' + (m % 60) + 'm';
    const d = Math.floor(h / 24);
    return d + 'd ' + (h % 24) + 'h';
  } catch { return '—'; }
}

// ─── Topology card ────────────────────────────────────────────────────────────

function TopologyStrip({ apps, runtime, onReload, busy }) {
  const proxiedApps = apps.filter(a => a.proxy_public_domain);

  const upstreamFromConfig = runtime?.upstream;

  return (
    <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
      <div className="row" style={{ ...SECTION_LABEL, justifyContent: 'space-between', marginBottom: 8 }}>
        <span>Upstream Topology</span>
        <button className="btn ghost sm" onClick={onReload} disabled={busy}
          style={{ height: 22, fontSize: 11, padding: '0 8px' }}>
          <Icon.Refresh width={10}/> Reload engine
        </button>
      </div>

      {/* Default upstream from config (if proxy not multi-listener) */}
      {upstreamFromConfig && proxiedApps.length === 0 && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '8px 10px', border: HAIRLINE, borderRadius: 4,
          background: 'var(--surface-1)',
        }}>
          <Icon.Globe width={13} style={{ color: 'var(--fg-muted)' }}/>
          <div className="mono" style={{ fontSize: 12, fontWeight: 500 }}>
            default → <span className="faint">{upstreamFromConfig}</span>
          </div>
        </div>
      )}

      {proxiedApps.length > 0 && (
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
          gap: 6,
        }}>
          {proxiedApps.map(app => (
            <div key={app.id} style={{
              border: HAIRLINE, borderRadius: 4, padding: '8px 10px',
              background: 'var(--surface-1)',
              display: 'flex', alignItems: 'center', gap: 8, minWidth: 0,
            }}>
              <span style={{
                width: 6, height: 6, borderRadius: '50%',
                background: 'var(--success)',
                boxShadow: '0 0 0 3px color-mix(in oklch, var(--success) 22%, transparent)',
                flexShrink: 0,
              }}/>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="mono" style={{
                  fontSize: 12, fontWeight: 500,
                  whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
                }}>{app.proxy_public_domain}</div>
                <div className="faint mono" style={{
                  fontSize: 10,
                  whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
                }}>→ {app.proxy_protected_url || '(no upstream)'}</div>
              </div>
              <span className="faint" style={{ fontSize: 10 }}>{app.name}</span>
            </div>
          ))}
        </div>
      )}

      {proxiedApps.length === 0 && !upstreamFromConfig && (
        <div className="faint" style={{ fontSize: 12, padding: '6px 0' }}>
          No upstream configured. Add an application below or set <span className="mono">proxy.upstream</span> in <span className="mono">sharkauth.yaml</span>.
        </div>
      )}
    </div>
  );
}

// ─── Quick add upstream (replaces wizard) ─────────────────────────────────────

function AddUpstreamRow({ onCreated }) {
  const [open, setOpen] = React.useState(false);
  const [upstream, setUpstream] = React.useState('http://localhost:3000');
  const [name, setName] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const toast = useToast();

  const create = async () => {
    setErr('');
    let parsed;
    try { parsed = new URL(upstream); }
    catch { setErr('Invalid URL'); return; }
    setBusy(true);
    try {
      const slug = (name || parsed.hostname).toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
      const r = await API.post('/admin/apps', {
        name: name || parsed.hostname,
        slug,
        integration_mode: 'proxy',
        proxy_public_domain: parsed.host,
        proxy_protected_url: upstream,
      });
      toast.success('Upstream added');
      setOpen(false);
      setUpstream('http://localhost:3000');
      setName('');
      onCreated?.(r?.data);
    } catch (e) {
      setErr(e.message || 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  if (!open) {
    return (
      <button className="btn ghost sm" onClick={() => setOpen(true)}
        style={{ height: 24, fontSize: 11, padding: '0 8px' }}>
        <Icon.Plus width={10}/> Add upstream
      </button>
    );
  }

  return (
    <div className="row" style={{ gap: 6, alignItems: 'flex-start' }}>
      <input
        autoFocus
        placeholder="https://app.example.com"
        value={upstream}
        onChange={e => { setUpstream(e.target.value); setErr(''); }}
        style={{ ...INPUT, width: 240, height: 24, fontSize: 12 }}
      />
      <input
        placeholder="name (optional)"
        value={name}
        onChange={e => setName(e.target.value)}
        style={{ ...INPUT, width: 140, height: 24, fontSize: 12 }}
      />
      <button className="btn primary sm" onClick={create} disabled={busy}
        style={{ height: 24, fontSize: 11, padding: '0 10px' }}>
        {busy ? '…' : 'Create'}
      </button>
      <button className="btn ghost sm" onClick={() => { setOpen(false); setErr(''); }}
        style={{ height: 24, fontSize: 11, padding: '0 8px' }}>Cancel</button>
      {err && <div className="row" style={{ height: 24, alignItems: 'center', color: 'var(--danger)', fontSize: 11 }}>{err}</div>}
    </div>
  );
}

// ─── Right-drawer rule editor ─────────────────────────────────────────────────

function RuleDrawer({ rule, apps, onClose, onSaved }) {
  const isEdit = !!rule?.id;
  const [f, setF] = React.useState(() => rule ? {
    ...BLANK, ...rule,
    methods: (rule.methods || []).join(', '),
    scopes: (rule.scopes || []).join(', '),
  } : { ...BLANK });
  const [errs, setErrs] = React.useState({});
  const [saving, setSaving] = React.useState(false);
  const [apiErr, setApiErr] = React.useState('');
  const toast = useToast();
  const set = (k, v) => setF(p => ({ ...p, [k]: v }));

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const submit = async () => {
    const e = validateRule(f);
    if (Object.keys(e).length > 0) { setErrs(e); return; }
    setSaving(true); setApiErr('');
    try {
      const body = {
        ...f,
        methods: f.methods ? f.methods.split(',').map(s => s.trim().toUpperCase()).filter(Boolean) : [],
        scopes: f.scopes ? f.scopes.split(',').map(s => s.trim()).filter(Boolean) : [],
        priority: Number(f.priority) || 0,
      };
      const res = isEdit
        ? await API.patch('/admin/proxy/rules/db/' + rule.id, body)
        : await API.post('/admin/proxy/rules/db', body);
      if (res?.engine_refresh_error) {
        toast.warn('Saved — engine refresh: ' + res.engine_refresh_error);
      } else {
        toast.success(isEdit ? 'Rule updated' : 'Rule created');
      }
      onSaved();
    } catch (e) {
      setApiErr(e.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const del = async () => {
    if (!isEdit) return;
    if (!window.confirm(`Delete rule "${rule.name}"?`)) return;
    try {
      await API.del('/admin/proxy/rules/db/' + rule.id);
      toast.success('Rule deleted');
      onSaved();
    } catch (e) {
      toast.error(e.message || 'Delete failed');
    }
  };

  const Field = ({ label, hint, children, error }) => (
    <div style={{ marginBottom: 12 }}>
      <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4, fontWeight: 500 }}>{label}</label>
      {children}
      {error && <div style={{ fontSize: 11, color: 'var(--danger)', marginTop: 3 }}>{error}</div>}
      {hint && !error && <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>{hint}</div>}
    </div>
  );

  return (
    <>
      <div onClick={onClose} style={{
        position: 'fixed', inset: 0, zIndex: 40,
        background: 'rgba(0,0,0,0.35)',
      }}/>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0, width: 420, maxWidth: '100vw',
        zIndex: 41, borderLeft: HAIRLINE,
        background: 'var(--surface-0)', display: 'flex', flexDirection: 'column',
        animation: 'slideIn 140ms ease-out',
      }} onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div style={{ padding: 14, borderBottom: HAIRLINE }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
            <button className="btn ghost sm" onClick={onClose} disabled={saving}
              style={{ height: 24, fontSize: 11, padding: '0 8px' }}>
              <Icon.X width={10}/> Close
            </button>
            <span className="faint mono" style={{ fontSize: 10 }}>
              {isEdit ? `PATCH /admin/proxy/rules/db/${rule.id}` : 'POST /admin/proxy/rules/db'}
            </span>
          </div>
          <div style={{ fontSize: 16, fontWeight: 500, letterSpacing: '-0.01em', fontFamily: 'var(--font-display)' }}>
            {isEdit ? 'Edit rule' : 'New rule'}
          </div>
          <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
            Path-pattern policy applied at the proxy edge before requests hit upstream.
          </div>
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: 'auto', padding: 14 }}>
          <Field label="Name" error={errs.name}>
            <input
              autoFocus
              value={f.name}
              onChange={e => { set('name', e.target.value); setErrs(p => ({ ...p, name: undefined })); }}
              placeholder="block-unauth-writes"
              style={INPUT}
            />
          </Field>

          <Field label="Pattern" hint="chi-style: /api/*, /v1/orgs/{id}" error={errs.pattern}>
            <input
              value={f.pattern}
              onChange={e => { set('pattern', e.target.value); setErrs(p => ({ ...p, pattern: undefined })); }}
              placeholder="/api/*"
              style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
            />
          </Field>

          <Field
            label="Require"
            hint="anonymous · authenticated · agent · role:X · global_role:X · tier:X · scope:X · permission:X:Y"
            error={errs.require}
          >
            <input
              value={f.require}
              onChange={e => { set('require', e.target.value); set('allow', ''); setErrs(p => ({ ...p, require: undefined })); }}
              placeholder="authenticated"
              style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
            />
          </Field>

          <Field label="Allow override" hint="Public-by-pattern bypass — overrides require.">
            <div className="row" style={{ gap: 6 }}>
              <button
                className={'btn sm ' + (f.allow === 'anonymous' ? 'primary' : 'ghost')}
                onClick={() => { set('allow', 'anonymous'); set('require', ''); }}
                style={{ height: 24, fontSize: 11, padding: '0 8px' }}
              >anonymous</button>
              <button
                className="btn ghost sm"
                onClick={() => set('allow', '')}
                disabled={!f.allow}
                style={{ height: 24, fontSize: 11, padding: '0 8px' }}
              >clear</button>
            </div>
          </Field>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
            <Field label="Methods" hint="comma-sep · empty = any">
              <input
                value={f.methods}
                onChange={e => set('methods', e.target.value)}
                placeholder="GET, POST"
                style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
              />
            </Field>
            <Field label="Priority">
              <input
                type="number"
                value={f.priority}
                onChange={e => set('priority', e.target.value)}
                style={INPUT}
              />
            </Field>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
            <Field label="Tier match" hint="free · pro · (empty)">
              <input
                value={f.tier_match}
                onChange={e => set('tier_match', e.target.value)}
                placeholder="(any)"
                style={INPUT}
              />
            </Field>
            <Field label="Scopes" hint="comma-sep">
              <input
                value={f.scopes}
                onChange={e => set('scopes', e.target.value)}
                placeholder="read:data"
                style={{ ...INPUT, fontFamily: 'var(--font-mono)' }}
              />
            </Field>
          </div>

          {apps.length > 0 && (
            <Field label="Application" hint="Scope this rule to a single app.">
              <select
                value={f.app_id}
                onChange={e => set('app_id', e.target.value)}
                style={INPUT}
              >
                <option value="">Global (all apps)</option>
                {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
              </select>
            </Field>
          )}

          <div style={{ display: 'flex', gap: 16, padding: '8px 0', borderTop: HAIRLINE, marginTop: 8 }}>
            <label style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12, cursor: 'pointer' }}>
              <input type="checkbox" checked={!!f.enabled} onChange={e => set('enabled', e.target.checked)}/>
              Enabled
            </label>
            <label style={{ display: 'inline-flex', alignItems: 'center', gap: 6, fontSize: 12, cursor: 'pointer' }}>
              <input type="checkbox" checked={!!f.m2m} onChange={e => set('m2m', e.target.checked)}/>
              M2M only
            </label>
          </div>

          {apiErr && (
            <div style={{
              fontSize: 12, color: 'var(--danger)',
              padding: '6px 8px', border: '1px solid var(--danger)',
              borderRadius: 4, marginTop: 10, background: 'color-mix(in oklch, var(--danger) 8%, transparent)',
            }}>{apiErr}</div>
          )}
        </div>

        {/* Footer */}
        <div style={{ padding: 12, borderTop: HAIRLINE, display: 'flex', justifyContent: 'space-between', gap: 8 }}>
          <div>
            {isEdit && (
              <button className="btn ghost sm" onClick={del}
                style={{ height: 26, fontSize: 11, padding: '0 10px', color: 'var(--danger)' }}>
                Delete
              </button>
            )}
          </div>
          <div className="row" style={{ gap: 6 }}>
            <button className="btn ghost sm" onClick={onClose} disabled={saving}
              style={{ height: 26, fontSize: 11, padding: '0 10px' }}>Cancel</button>
            <button className="btn primary sm" onClick={submit} disabled={saving}
              style={{ height: 26, fontSize: 11, padding: '0 12px' }}>
              {saving ? 'Saving…' : (isEdit ? 'Update' : 'Create')}
            </button>
          </div>
        </div>
      </div>
    </>
  );
}

// ─── YAML import drawer ───────────────────────────────────────────────────────

function ImportDrawer({ onClose, onDone }) {
  const [yaml, setYaml] = React.useState('');
  const [result, setResult] = React.useState(null);
  const [importing, setImporting] = React.useState(false);
  const [apiErr, setApiErr] = React.useState('');
  const toast = useToast();
  const fileRef = React.useRef(null);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const doImport = async () => {
    if (!yaml.trim()) return;
    setImporting(true); setApiErr(''); setResult(null);
    try {
      const r = await API.post('/admin/proxy/rules/import', { yaml });
      setResult(r);
      if (r?.errors?.length === 0) {
        toast.success(`Imported ${r.imported} rule${r.imported !== 1 ? 's' : ''}`);
        onDone();
      } else {
        toast.warn(`Imported ${r.imported}, ${r.errors?.length} errors`);
      }
    } catch (e) {
      setApiErr(e.message || 'Import failed');
    } finally {
      setImporting(false);
    }
  };

  const onFile = (e) => {
    const f = e.target.files?.[0];
    if (!f) return;
    const reader = new FileReader();
    reader.onload = (ev) => setYaml(ev.target.result);
    reader.readAsText(f);
  };

  return (
    <>
      <div onClick={onClose} style={{ position: 'fixed', inset: 0, zIndex: 40, background: 'rgba(0,0,0,0.35)' }}/>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0, width: 460, maxWidth: '100vw',
        zIndex: 41, borderLeft: HAIRLINE,
        background: 'var(--surface-0)', display: 'flex', flexDirection: 'column',
        animation: 'slideIn 140ms ease-out',
      }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: 14, borderBottom: HAIRLINE }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
            <button className="btn ghost sm" onClick={onClose} style={{ height: 24, fontSize: 11, padding: '0 8px' }}>
              <Icon.X width={10}/> Close
            </button>
            <span className="faint mono" style={{ fontSize: 10 }}>POST /admin/proxy/rules/import</span>
          </div>
          <div style={{ fontSize: 16, fontWeight: 500, fontFamily: 'var(--font-display)' }}>Import YAML rules</div>
          <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
            Bulk-create rules from <span className="mono">sharkauth.yaml</span> snippets.
          </div>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: 14 }}>
          <div
            style={{
              border: '1px dashed var(--hairline-strong)', borderRadius: 4,
              padding: '14px 12px', textAlign: 'center', cursor: 'pointer',
              fontSize: 12, color: 'var(--fg-dim)', marginBottom: 10,
              background: 'var(--surface-1)',
            }}
            onClick={() => fileRef.current?.click()}
            onDragOver={e => e.preventDefault()}
            onDrop={e => {
              e.preventDefault();
              const f = e.dataTransfer.files?.[0];
              if (f) { const r = new FileReader(); r.onload = ev => setYaml(ev.target.result); r.readAsText(f); }
            }}
          >
            Drop .yaml file or click to browse
            <input ref={fileRef} type="file" accept=".yaml,.yml" style={{ display: 'none' }} onChange={onFile}/>
          </div>

          <textarea
            value={yaml}
            onChange={e => setYaml(e.target.value)}
            placeholder="Or paste YAML here…"
            style={{ ...TEXTAREA, height: 220 }}
          />

          {apiErr && (
            <div style={{ fontSize: 12, color: 'var(--danger)', marginTop: 8 }}>{apiErr}</div>
          )}

          {result?.errors?.length > 0 && (
            <div style={{ marginTop: 10 }}>
              <div style={{ ...SECTION_LABEL, color: 'var(--warn)', marginBottom: 6 }}>
                {result.errors.length} error{result.errors.length !== 1 ? 's' : ''}
              </div>
              <table className="tbl" style={{ width: '100%', fontSize: 11, borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ background: 'var(--surface-1)' }}>
                    <th style={{ padding: '4px 8px', textAlign: 'left' }}>#</th>
                    <th style={{ padding: '4px 8px', textAlign: 'left' }}>Name</th>
                    <th style={{ padding: '4px 8px', textAlign: 'left' }}>Message</th>
                  </tr>
                </thead>
                <tbody>
                  {result.errors.map((err, i) => (
                    <tr key={i} style={{ borderTop: HAIRLINE }}>
                      <td style={{ padding: '4px 8px' }} className="mono">{err.index ?? i}</td>
                      <td style={{ padding: '4px 8px' }}>{err.name || '—'}</td>
                      <td style={{ padding: '4px 8px', color: 'var(--danger)' }}>{err.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {result?.imported > 0 && (
            <div style={{ fontSize: 12, color: 'var(--success)', marginTop: 8 }}>
              <Icon.Check width={11}/> {result.imported} rule{result.imported !== 1 ? 's' : ''} imported
            </div>
          )}
        </div>

        <div style={{ padding: 12, borderTop: HAIRLINE, display: 'flex', justifyContent: 'flex-end', gap: 6 }}>
          <button className="btn ghost sm" onClick={onClose} style={{ height: 26, fontSize: 11, padding: '0 10px' }}>Close</button>
          <button className="btn primary sm" disabled={importing || !yaml.trim()} onClick={doImport}
            style={{ height: 26, fontSize: 11, padding: '0 12px' }}>
            {importing ? 'Importing…' : 'Import'}
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Simulate drawer (test policy) ────────────────────────────────────────────

function SimulateDrawer({ onClose }) {
  const [path, setPath] = React.useState('/api/users');
  const [method, setMethod] = React.useState('GET');
  const [identity, setIdentity] = React.useState('{"user_email":"alice@example.com","roles":["admin"]}');
  const [result, setResult] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const run = async () => {
    setBusy(true); setErr(''); setResult(null);
    try {
      let id = {};
      if (identity.trim()) {
        try { id = JSON.parse(identity); }
        catch { setErr('Identity must be valid JSON'); setBusy(false); return; }
      }
      const r = await API.post('/admin/proxy/simulate', { path, method, identity: id });
      setResult(r);
    } catch (e) {
      setErr(e.message || 'Simulation failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div onClick={onClose} style={{ position: 'fixed', inset: 0, zIndex: 40, background: 'rgba(0,0,0,0.35)' }}/>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0, width: 420, maxWidth: '100vw',
        zIndex: 41, borderLeft: HAIRLINE,
        background: 'var(--surface-0)', display: 'flex', flexDirection: 'column',
        animation: 'slideIn 140ms ease-out',
      }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: 14, borderBottom: HAIRLINE }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
            <button className="btn ghost sm" onClick={onClose} style={{ height: 24, fontSize: 11, padding: '0 8px' }}>
              <Icon.X width={10}/> Close
            </button>
            <span className="faint mono" style={{ fontSize: 10 }}>POST /admin/proxy/simulate</span>
          </div>
          <div style={{ fontSize: 16, fontWeight: 500, fontFamily: 'var(--font-display)' }}>Simulate request</div>
          <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
            Test what the engine would decide without sending a real request.
          </div>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: 14 }}>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Method</label>
            <select value={method} onChange={e => setMethod(e.target.value)} style={INPUT}>
              {['GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS'].map(m => <option key={m}>{m}</option>)}
            </select>
          </div>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Path</label>
            <input value={path} onChange={e => setPath(e.target.value)}
              style={{ ...INPUT, fontFamily: 'var(--font-mono)' }} placeholder="/api/users"/>
          </div>
          <div style={{ marginBottom: 12 }}>
            <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Identity (JSON)</label>
            <textarea value={identity} onChange={e => setIdentity(e.target.value)}
              style={{ ...TEXTAREA, height: 100 }}/>
            <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
              Fields: user_id, user_email, roles[], agent_id, scopes[], auth_method.
            </div>
          </div>

          <button className="btn primary sm" onClick={run} disabled={busy}
            style={{ height: 26, fontSize: 11, padding: '0 12px', width: '100%' }}>
            {busy ? 'Running…' : 'Run simulation'}
          </button>

          {err && <div style={{ fontSize: 12, color: 'var(--danger)', marginTop: 10 }}>{err}</div>}

          {result && (
            <div style={{ marginTop: 14, border: HAIRLINE, borderRadius: 4, padding: 10, background: 'var(--surface-1)' }}>
              <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
                <span style={SECTION_LABEL}>Result</span>
                <span className="mono faint" style={{ fontSize: 10 }}>{result.eval_us ?? '?'}μs</span>
              </div>
              <div style={{ display: 'flex', gap: 10, alignItems: 'baseline', marginBottom: 6 }}>
                <span style={{
                  fontSize: 13, fontWeight: 600,
                  color: result.decision === 'allow' ? 'var(--success)' : 'var(--danger)',
                  textTransform: 'uppercase', letterSpacing: '0.04em',
                }}>{result.decision}</span>
                <span className="faint mono" style={{ fontSize: 11 }}>{result.reason}</span>
              </div>
              {result.matched_rule && (
                <div style={{ marginTop: 6, paddingTop: 6, borderTop: HAIRLINE }}>
                  <div style={{ ...SECTION_LABEL, marginBottom: 3 }}>Matched rule</div>
                  <div className="mono" style={{ fontSize: 11 }}>{result.matched_rule.path}</div>
                  <div className="mono faint" style={{ fontSize: 10 }}>
                    {(result.matched_rule.methods || []).join(', ') || 'any'} · {result.matched_rule.require}
                  </div>
                </div>
              )}
              {result.injected_headers && Object.keys(result.injected_headers).length > 0 && (
                <div style={{ marginTop: 6, paddingTop: 6, borderTop: HAIRLINE }}>
                  <div style={{ ...SECTION_LABEL, marginBottom: 3 }}>Injected headers</div>
                  {Object.entries(result.injected_headers).map(([k, v]) => (
                    <div key={k} className="mono" style={{ fontSize: 11, lineHeight: 1.5 }}>
                      <span style={{ color: 'var(--fg-muted)' }}>{k}:</span> {v}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </>
  );
}

// ─── Inline enabled toggle ────────────────────────────────────────────────────

function EnabledToggle({ rule, onToggled }) {
  const [busy, setBusy] = React.useState(false);
  const toast = useToast();
  const toggle = async (e) => {
    e.stopPropagation();
    setBusy(true);
    try {
      await API.patch('/admin/proxy/rules/db/' + rule.id, { enabled: !rule.enabled });
      onToggled();
    } catch (err) {
      toast.error(err.message || 'Toggle failed');
    } finally {
      setBusy(false);
    }
  };
  return (
    <button
      onClick={toggle}
      title={rule.enabled ? 'Click to disable' : 'Click to enable'}
      disabled={busy}
      style={{
        width: 24, height: 14, padding: 0,
        borderRadius: 3, border: HAIRLINE_STRONG, cursor: 'pointer',
        background: rule.enabled ? 'var(--success)' : 'var(--surface-2)',
        position: 'relative', transition: 'background 100ms ease',
        opacity: busy ? 0.5 : 1,
      }}
    >
      <span style={{
        position: 'absolute', top: 1, left: rule.enabled ? 11 : 1,
        width: 10, height: 10, borderRadius: 2,
        background: rule.enabled ? '#000' : 'var(--fg-muted)',
        transition: 'left 100ms ease',
      }}/>
    </button>
  );
}

// ─── Disabled / not-wired state ───────────────────────────────────────────────

function DisabledState({ onRetry }) {
  return (
    <div style={{ padding: 24, height: '100%', overflow: 'auto' }}>
      <div style={{ maxWidth: 540 }}>
        <div className="row" style={{ gap: 8, marginBottom: 4 }}>
          <Icon.Globe width={16}/>
          <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600, fontFamily: 'var(--font-display)' }}>Proxy</h1>
          <span style={{
            display: 'inline-flex', alignItems: 'center', gap: 5,
            height: 20, padding: '0 7px', fontSize: 11, fontWeight: 500,
            border: HAIRLINE_STRONG, borderRadius: 3,
            color: 'var(--fg-muted)', background: 'var(--surface-2)',
          }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--fg-faint)' }}/>
            disabled
          </span>
        </div>
        <div className="faint" style={{ fontSize: 13, marginBottom: 18 }}>
          Reverse proxy is not wired in this build. Enable it to protect upstream services without code changes.
        </div>

        <div style={{ border: HAIRLINE, borderRadius: 5, padding: 14, background: 'var(--surface-1)' }}>
          <div style={{ ...SECTION_LABEL, marginBottom: 8 }}>Enable in sharkauth.yaml</div>
          <pre className="mono" style={{
            fontSize: 12, background: 'var(--surface-0)', border: HAIRLINE,
            borderRadius: 4, padding: 10, overflow: 'auto', margin: 0, lineHeight: 1.5,
          }}>{`proxy:
  enabled: true
  upstream: http://localhost:3000
  listen: :8080
  rules:
    - name: protected
      pattern: /api/*
      require: authenticated`}</pre>
          <div className="faint" style={{ fontSize: 11, marginTop: 8 }}>
            Restart the binary, then click below to retry.
          </div>
          <button className="btn primary sm" onClick={onRetry}
            style={{ height: 26, fontSize: 11, padding: '0 12px', marginTop: 10 }}>
            <Icon.Refresh width={10}/> Retry
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Main Proxy page ──────────────────────────────────────────────────────────

export function Proxy() {
  const lifecycle = useLifecycle();
  const runtime = useRuntimeStatus();
  const { data: appsRaw } = useAPI('/admin/apps');
  const apps = appsRaw?.data || [];
  const { data: rulesRaw, refresh: refreshRules } = useAPI('/admin/proxy/rules/db');
  const allRules = rulesRaw?.data || [];

  const [appFilter, setAppFilter] = React.useState('all');
  const [search, setSearch] = React.useState('');
  const [drawer, setDrawer] = React.useState(null); // {kind:'rule'|'import'|'simulate', rule?}
  const [reloadBusy, setReloadBusy] = React.useState(false);
  const toast = useToast();

  const rules = React.useMemo(() => {
    let out = allRules;
    if (appFilter !== 'all') out = out.filter(r => r.app_id === appFilter);
    if (search.trim()) {
      const q = search.toLowerCase();
      out = out.filter(r =>
        r.name?.toLowerCase().includes(q) ||
        r.pattern?.toLowerCase().includes(q) ||
        r.require?.toLowerCase().includes(q) ||
        r.allow?.toLowerCase().includes(q),
      );
    }
    return out;
  }, [allRules, appFilter, search]);

  const reload = async () => {
    setReloadBusy(true);
    try {
      await lifecycle.act('reload');
      toast.success('Engine reloaded');
      refreshRules();
    } catch (e) {
      toast.error(e.message || 'Reload failed');
    } finally {
      setReloadBusy(false);
    }
  };

  if (lifecycle.missing) return <DisabledState onRetry={lifecycle.refresh}/>;

  const lastError = lifecycle.status?.last_error;

  return (
    <div className="col" style={{ height: '100%', overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      {/* ─── Header ──────────────────────────────────────────────── */}
      <div style={{ padding: '12px 20px', borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 12 }}>
          <div className="row" style={{ gap: 10, alignItems: 'center', minWidth: 0 }}>
            <h1 style={{
              fontSize: 18, margin: 0, fontWeight: 600,
              fontFamily: 'var(--font-display)', letterSpacing: '-0.01em',
            }}>Proxy</h1>
            <span className="faint" style={{ fontSize: 12 }}>
              Reverse proxy engine · {allRules.length} rule{allRules.length !== 1 ? 's' : ''}
            </span>
          </div>
          <div className="row" style={{ gap: 6, alignItems: 'center' }}>
            <button className="btn ghost sm" onClick={() => setDrawer({ kind: 'simulate' })}
              style={{ height: 28, fontSize: 12, padding: '0 10px' }}>
              <Icon.Refresh width={11}/> Simulate
            </button>
            <button className="btn ghost sm" onClick={() => setDrawer({ kind: 'import' })}
              style={{ height: 28, fontSize: 12, padding: '0 10px' }}>
              <Icon.Migration width={11}/> Import YAML
            </button>
            <button className="btn primary sm" onClick={() => setDrawer({ kind: 'rule', rule: null })}
              style={{ height: 28, fontSize: 12, padding: '0 12px' }}>
              <Icon.Plus width={11}/> New rule
            </button>
            {/* The headline power toggle */}
            <div style={{ width: 1, alignSelf: 'stretch', background: 'var(--hairline)', margin: '0 4px' }}/>
            {!lifecycle.loading && <PowerToggle status={lifecycle.status} onAction={lifecycle.act}/>}
          </div>
        </div>
      </div>

      {/* ─── Status strip ────────────────────────────────────────── */}
      {!lifecycle.loading && lifecycle.status && (
        <StatusStrip lifecycle={lifecycle} runtime={runtime.data}/>
      )}

      {/* ─── Last error banner ───────────────────────────────────── */}
      {lastError && (
        <div style={{
          padding: '8px 20px', borderBottom: HAIRLINE,
          background: 'color-mix(in oklch, var(--danger) 8%, var(--surface-0))',
          display: 'flex', alignItems: 'center', gap: 10,
        }}>
          <Icon.Warn width={12} style={{ color: 'var(--danger)', flexShrink: 0 }}/>
          <span className="mono" style={{ fontSize: 11, color: 'var(--danger)', flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {lastError}
          </span>
        </div>
      )}

      {/* ─── Topology ────────────────────────────────────────────── */}
      <TopologyStrip apps={apps} runtime={runtime.data} onReload={reload} busy={reloadBusy}/>

      {/* ─── Rules section ───────────────────────────────────────── */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <div style={{
          padding: '10px 20px', borderBottom: HAIRLINE,
          display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          gap: 12, flexWrap: 'wrap',
        }}>
          <div className="row" style={{ gap: 10, alignItems: 'center' }}>
            <span style={SECTION_LABEL}>Route policies ({rules.length})</span>
            <AddUpstreamRow onCreated={() => { refreshRules(); }}/>
          </div>

          <div className="row" style={{ gap: 6 }}>
            <input
              placeholder="Filter rules…"
              value={search}
              onChange={e => setSearch(e.target.value)}
              style={{ ...INPUT, height: 24, width: 180, fontSize: 12 }}
            />
            <select
              value={appFilter}
              onChange={e => setAppFilter(e.target.value)}
              style={{ ...INPUT, height: 24, width: 'auto', fontSize: 12, padding: '0 6px' }}
            >
              <option value="all">All apps</option>
              {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
          </div>
        </div>

        <div style={{ flex: 1, overflowY: 'auto' }}>
          <table className="tbl" style={{ width: '100%', fontSize: 12.5, borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{
                background: 'var(--surface-1)', position: 'sticky', top: 0, zIndex: 1,
                borderBottom: HAIRLINE,
              }}>
                <th style={{ padding: '8px 12px', textAlign: 'left', width: 44, fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>Pri</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>Name</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>Pattern</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>Methods</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>Policy</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>Tier</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>App</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', width: 60, fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>On</th>
              </tr>
            </thead>
            <tbody>
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={8} style={{ padding: 50, textAlign: 'center' }}>
                    <div className="faint" style={{ fontSize: 12, marginBottom: 6 }}>
                      {allRules.length === 0 ? 'No rules yet.' : 'No rules match the current filter.'}
                    </div>
                    {allRules.length === 0 && (
                      <button className="btn primary sm" onClick={() => setDrawer({ kind: 'rule', rule: null })}
                        style={{ height: 26, fontSize: 11, padding: '0 12px' }}>
                        <Icon.Plus width={10}/> Create first rule
                      </button>
                    )}
                  </td>
                </tr>
              ) : rules.map(r => (
                <tr
                  key={r.id}
                  onClick={() => setDrawer({ kind: 'rule', rule: r })}
                  style={{
                    borderTop: HAIRLINE, cursor: 'pointer',
                    background: 'var(--surface-0)',
                    opacity: r.enabled ? 1 : 0.55,
                    transition: 'background 80ms ease',
                  }}
                  onMouseEnter={e => e.currentTarget.style.background = 'var(--surface-1)'}
                  onMouseLeave={e => e.currentTarget.style.background = 'var(--surface-0)'}
                >
                  <td className="mono" style={{ padding: '7px 12px', color: 'var(--fg-muted)' }}>{r.priority ?? 0}</td>
                  <td style={{ padding: '7px 12px', fontWeight: 500, maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {r.name}
                  </td>
                  <td style={{ padding: '7px 12px' }}>
                    <span className="mono" style={{ fontSize: 12 }}>{r.pattern}</span>
                  </td>
                  <td className="mono faint" style={{ padding: '7px 12px', fontSize: 11 }}>
                    {r.methods?.length ? r.methods.join(', ') : '—'}
                  </td>
                  <td style={{ padding: '7px 12px' }}>
                    <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                      {r.allow ? (
                        <span style={chip('var(--warn)')}>{r.allow}</span>
                      ) : r.require ? (
                        <span style={chip('var(--fg-muted)')} className="mono">{r.require}</span>
                      ) : null}
                      {r.m2m && <span style={chip('var(--agent, var(--fg-muted))')}>m2m</span>}
                    </div>
                  </td>
                  <td style={{ padding: '7px 12px' }}>
                    {r.tier_match
                      ? <span style={chip('var(--success)')}>{r.tier_match}</span>
                      : <span className="faint" style={{ fontSize: 11 }}>—</span>}
                  </td>
                  <td className="faint" style={{ padding: '7px 12px', fontSize: 11 }}>
                    {apps.find(a => a.id === r.app_id)?.name || <span style={{ opacity: 0.5 }}>global</span>}
                  </td>
                  <td style={{ padding: '7px 12px' }} onClick={e => e.stopPropagation()}>
                    <EnabledToggle rule={r} onToggled={refreshRules}/>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* ─── Drawers ─────────────────────────────────────────────── */}
      {drawer?.kind === 'rule' && (
        <RuleDrawer
          rule={drawer.rule}
          apps={apps}
          onClose={() => setDrawer(null)}
          onSaved={() => { setDrawer(null); refreshRules(); }}
        />
      )}
      {drawer?.kind === 'import' && (
        <ImportDrawer
          onClose={() => setDrawer(null)}
          onDone={() => { refreshRules(); }}
        />
      )}
      {drawer?.kind === 'simulate' && (
        <SimulateDrawer onClose={() => setDrawer(null)}/>
      )}

      <CLIFooter command="shark proxy status"/>
    </div>
  );
}

// ─── small helpers ────────────────────────────────────────────────────────────

function chip(color) {
  return {
    display: 'inline-flex', alignItems: 'center',
    height: 18, padding: '0 6px',
    fontSize: 10, fontWeight: 500,
    color,
    border: '1px solid ' + color,
    borderRadius: 3,
    background: 'color-mix(in oklch, ' + color + ' 8%, transparent)',
    whiteSpace: 'nowrap',
  };
}
