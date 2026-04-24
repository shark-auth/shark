// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { ProxyWizard } from './proxy_wizard'

const HAIRLINE = '1px solid var(--hairline)';
const SECTION_LABEL = {
  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
  color: 'var(--fg-dim)', fontWeight: 500,
};

// ─── Lifecycle ────────────────────────────────────────────────────────────────

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
    try {
      const r = await API.post('/admin/proxy/' + action);
      setStatus(r?.data || null);
    } catch (e) {
      throw e;
    }
  };

  return { status, loading, missing, refresh: fetch_, act };
}

function LifecycleBar({ status, act, toast }) {
  const state = status?.state_str || 'unknown';
  const badgeCls = state === 'running' ? 'success' : state === 'reloading' ? 'warn' : 'error';
  const [busy, setBusy] = React.useState(false);
  const [errExpanded, setErrExpanded] = React.useState(false);

  const do_ = async (action, confirm_) => {
    if (confirm_ && !window.confirm(confirm_)) return;
    setBusy(true);
    try {
      await act(action);
      toast.success(action.charAt(0).toUpperCase() + action.slice(1) + ' successful');
    } catch (e) {
      toast.error(e.message || 'Action failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
      <span className={'chip sm ' + badgeCls} style={{ display: 'inline-flex', alignItems: 'center', gap: 5 }}>
        {state === 'reloading' && <span className="dot warn pulse" style={{ width: 6, height: 6 }}/>}
        {state}
      </span>
      {status?.listeners != null && (
        <span className="mono faint" style={{ fontSize: 11 }}>{status.listeners} listener{status.listeners !== 1 ? 's' : ''}</span>
      )}
      {status?.rules_loaded != null && (
        <span className="mono faint" style={{ fontSize: 11 }}>{status.rules_loaded} rules</span>
      )}
      <div className="row" style={{ gap: 6 }}>
        <button className="btn sm" disabled={busy || state === 'running' || state === 'reloading'} onClick={() => do_('start')}>Start</button>
        <button className="btn sm" disabled={busy || state === 'stopped'} onClick={() => do_('stop', 'Stop the proxy? In-flight requests will drain.')}>Stop</button>
        <button className="btn ghost sm" disabled={busy || state === 'stopped'} onClick={() => do_('reload')}>
          <Icon.Refresh width={11}/>Reload
        </button>
      </div>
      {status?.last_error && (
        <button className="btn ghost sm" style={{ color: 'var(--error)' }} onClick={() => setErrExpanded(v => !v)}>
          <Icon.Warn width={11}/> Last error
        </button>
      )}
      {errExpanded && status?.last_error && (
        <div className="mono" style={{ fontSize: 11, color: 'var(--error)', background: 'var(--surface-2)', padding: '4px 8px', borderRadius: 4, maxWidth: 400 }}>
          {status.last_error}
        </div>
      )}
    </div>
  );
}

// ─── Require grammar validator ────────────────────────────────────────────────

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

// ─── Rule form (create / edit) ────────────────────────────────────────────────

const BLANK = { name: '', pattern: '', methods: '', require: 'authenticated', allow: '', scopes: '', enabled: true, priority: 0, tier_match: '', m2m: false, app_id: '' };

function RuleModal({ rule, apps, onSave, onClose }) {
  const [f, setF] = React.useState(() => rule ? {
    ...rule,
    methods: (rule.methods || []).join(', '),
    scopes: (rule.scopes || []).join(', '),
  } : { ...BLANK });
  const [errs, setErrs] = React.useState({});
  const [saving, setSaving] = React.useState(false);
  const [apiErr, setApiErr] = React.useState('');
  const toast = useToast();

  const set = (k, v) => setF(p => ({ ...p, [k]: v }));

  const submit = async () => {
    const e = validateRule(f);
    if (Object.keys(e).length > 0) { setErrs(e); return; }
    setSaving(true); setApiErr('');
    try {
      const body = {
        ...f,
        methods: f.methods ? f.methods.split(',').map(s => s.trim()).filter(Boolean) : [],
        scopes: f.scopes ? f.scopes.split(',').map(s => s.trim()).filter(Boolean) : [],
        priority: Number(f.priority) || 0,
      };
      let res;
      if (rule?.id) {
        res = await API.patch('/admin/proxy/rules/db/' + rule.id, body);
      } else {
        res = await API.post('/admin/proxy/rules/db', body);
      }
      if (res?.engine_refresh_error) {
        toast.warn('Saved — engine refresh warning: ' + res.engine_refresh_error);
      } else {
        toast.success(rule?.id ? 'Rule updated' : 'Rule created');
      }
      onSave();
    } catch (e) {
      setApiErr(e.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const field = (label, key, type = 'text', extra = {}) => (
    <div style={{ marginBottom: 14 }}>
      <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>{label}</label>
      <input
        type={type}
        value={f[key] ?? ''}
        onChange={e => { set(key, e.target.value); setErrs(p => ({ ...p, [key]: undefined })); }}
        style={{ width: '100%', boxSizing: 'border-box', ...(extra.style || {}) }}
        placeholder={extra.placeholder || ''}
        className={errs[key] ? 'input-err' : ''}
      />
      {errs[key] && <div style={{ fontSize: 11, color: 'var(--error)', marginTop: 3 }}>{errs[key]}</div>}
    </div>
  );

  return (
    <div style={{ background: 'var(--surface-0)', borderRadius: 8, border: HAIRLINE, padding: 20, width: 480 }} onClick={e => e.stopPropagation()}>
      <div className="row" style={{ justifyContent: 'space-between', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 15 }}>{rule?.id ? 'Edit Rule' : 'New Rule'}</h2>
        <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11}/></button>
      </div>

      {field('Name', 'name', 'text', { placeholder: 'block-unauth-writes' })}
      {field('Pattern', 'pattern', 'text', { placeholder: '/api/*' })}

      <div style={{ marginBottom: 14 }}>
        <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Require</label>
        <input
          value={f.require}
          onChange={e => { set('require', e.target.value); set('allow', ''); setErrs(p => ({ ...p, require: undefined })); }}
          placeholder="authenticated, role:admin, tier:pro, scope:read, agent…"
          style={{ width: '100%', boxSizing: 'border-box' }}
          className={errs.require ? 'input-err' : ''}
        />
        {errs.require && <div style={{ fontSize: 11, color: 'var(--error)', marginTop: 3 }}>{errs.require}</div>}
        <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
          anonymous · authenticated · agent · role:X · global_role:X · tier:X · scope:X · permission:X:Y
        </div>
      </div>

      <div style={{ marginBottom: 14 }}>
        <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Allow (public override)</label>
        <div className="row" style={{ gap: 8 }}>
          <button
            className={'btn sm ' + (f.allow === 'anonymous' ? '' : 'ghost')}
            onClick={() => { set('allow', 'anonymous'); set('require', ''); }}
          >anonymous</button>
          <button
            className={'btn ghost sm ' + (f.allow === '' && f.require ? '' : '')}
            onClick={() => { set('allow', ''); }}
          >clear allow</button>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <div style={{ marginBottom: 14 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Methods (comma-sep)</label>
          <input value={f.methods} onChange={e => set('methods', e.target.value)} placeholder="GET, POST — empty = any" style={{ width: '100%', boxSizing: 'border-box' }}/>
        </div>
        <div style={{ marginBottom: 14 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Priority</label>
          <input type="number" value={f.priority} onChange={e => set('priority', e.target.value)} style={{ width: '100%', boxSizing: 'border-box' }}/>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <div style={{ marginBottom: 14 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Tier match</label>
          <input value={f.tier_match} onChange={e => set('tier_match', e.target.value)} placeholder="free · pro · (empty)" style={{ width: '100%', boxSizing: 'border-box' }}/>
        </div>
        <div style={{ marginBottom: 14 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Scopes (comma-sep)</label>
          <input value={f.scopes} onChange={e => set('scopes', e.target.value)} placeholder="read:data" style={{ width: '100%', boxSizing: 'border-box' }}/>
        </div>
      </div>

      {apps.length > 0 && (
        <div style={{ marginBottom: 14 }}>
          <label style={{ display: 'block', fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Application (optional)</label>
          <select value={f.app_id} onChange={e => set('app_id', e.target.value)} style={{ width: '100%' }}>
            <option value="">Global</option>
            {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
        </div>
      )}

      <div className="row" style={{ gap: 16, marginBottom: 14 }}>
        <label className="row" style={{ gap: 6, cursor: 'pointer', fontSize: 13 }}>
          <input type="checkbox" checked={!!f.enabled} onChange={e => set('enabled', e.target.checked)}/>
          Enabled
        </label>
        <label className="row" style={{ gap: 6, cursor: 'pointer', fontSize: 13 }}>
          <input type="checkbox" checked={!!f.m2m} onChange={e => set('m2m', e.target.checked)}/>
          M2M only
        </label>
      </div>

      {apiErr && <div style={{ fontSize: 12, color: 'var(--error)', marginBottom: 12 }}>{apiErr}</div>}

      <div className="row" style={{ justifyContent: 'flex-end', gap: 8 }}>
        <button className="btn ghost sm" onClick={onClose}>Cancel</button>
        <button className="btn primary sm" disabled={saving} onClick={submit}>{saving ? 'Saving…' : (rule?.id ? 'Update' : 'Create')}</button>
      </div>
    </div>
  );
}

// ─── Import YAML modal ─────────────────────────────────────────────────────────

function ImportModal({ onClose, onDone }) {
  const [yaml, setYaml] = React.useState('');
  const [result, setResult] = React.useState(null);
  const [importing, setImporting] = React.useState(false);
  const [apiErr, setApiErr] = React.useState('');
  const toast = useToast();
  const fileRef = React.useRef(null);

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
    const f = e.target.files[0];
    if (!f) return;
    const reader = new FileReader();
    reader.onload = (ev) => setYaml(ev.target.result);
    reader.readAsText(f);
  };

  return (
    <div style={{ background: 'var(--surface-0)', borderRadius: 8, border: HAIRLINE, padding: 20, width: 560 }} onClick={e => e.stopPropagation()}>
      <div className="row" style={{ justifyContent: 'space-between', marginBottom: 14 }}>
        <h2 style={{ margin: 0, fontSize: 15 }}>Import YAML Rules</h2>
        <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11}/></button>
      </div>

      <div
        style={{ border: '2px dashed var(--hairline-strong)', borderRadius: 6, padding: '12px 16px', marginBottom: 10, textAlign: 'center', cursor: 'pointer', fontSize: 12, color: 'var(--fg-dim)' }}
        onClick={() => fileRef.current?.click()}
        onDragOver={e => e.preventDefault()}
        onDrop={e => { e.preventDefault(); const f = e.dataTransfer.files[0]; if (f) { const r = new FileReader(); r.onload = ev => setYaml(ev.target.result); r.readAsText(f); } }}
      >
        Drop .yaml file here or click to browse
        <input ref={fileRef} type="file" accept=".yaml,.yml" style={{ display: 'none' }} onChange={onFile}/>
      </div>

      <textarea
        value={yaml}
        onChange={e => setYaml(e.target.value)}
        placeholder="Or paste YAML here…"
        style={{ width: '100%', boxSizing: 'border-box', height: 160, fontFamily: 'var(--font-mono)', fontSize: 12, padding: 8, border: HAIRLINE, borderRadius: 4, background: 'var(--surface-1)', color: 'var(--fg)', resize: 'vertical' }}
      />

      {apiErr && <div style={{ fontSize: 12, color: 'var(--error)', margin: '6px 0' }}>{apiErr}</div>}

      {result && result.errors?.length > 0 && (
        <div style={{ marginTop: 10 }}>
          <div style={{ fontSize: 11, color: 'var(--warn)', marginBottom: 6 }}>{result.errors.length} import error{result.errors.length !== 1 ? 's' : ''}:</div>
          <table className="tbl" style={{ width: '100%', fontSize: 11 }}>
            <thead><tr><th style={{ padding: '4px 8px' }}>#</th><th style={{ padding: '4px 8px' }}>Name</th><th style={{ padding: '4px 8px' }}>Message</th></tr></thead>
            <tbody>
              {result.errors.map((err, i) => (
                <tr key={i} style={{ borderTop: HAIRLINE }}>
                  <td style={{ padding: '4px 8px' }} className="mono">{err.index ?? i}</td>
                  <td style={{ padding: '4px 8px' }}>{err.name || '—'}</td>
                  <td style={{ padding: '4px 8px', color: 'var(--error)' }}>{err.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {result?.imported > 0 && (
        <div style={{ fontSize: 12, color: 'var(--success)', marginTop: 8 }}>
          <Icon.Check width={12}/> {result.imported} rule{result.imported !== 1 ? 's' : ''} imported
        </div>
      )}

      <div className="row" style={{ justifyContent: 'flex-end', gap: 8, marginTop: 14 }}>
        <button className="btn ghost sm" onClick={onClose}>Close</button>
        <button className="btn primary sm" disabled={importing || !yaml.trim()} onClick={doImport}>{importing ? 'Importing…' : 'Import'}</button>
      </div>
    </div>
  );
}

// ─── Delete confirm ───────────────────────────────────────────────────────────

function DeleteConfirm({ rule, onConfirm, onClose }) {
  const [deleting, setDeleting] = React.useState(false);
  const toast = useToast();

  const go = async () => {
    setDeleting(true);
    try {
      await API.del('/admin/proxy/rules/db/' + rule.id);
      toast.success('Rule deleted');
      onConfirm();
    } catch (e) {
      toast.error(e.message || 'Delete failed');
      setDeleting(false);
    }
  };

  return (
    <div style={{ background: 'var(--surface-0)', borderRadius: 8, border: HAIRLINE, padding: 20, width: 380 }} onClick={e => e.stopPropagation()}>
      <h2 style={{ margin: '0 0 10px', fontSize: 15 }}>Delete rule?</h2>
      <div style={{ fontSize: 13, marginBottom: 16, color: 'var(--fg-muted)' }}>
        <span className="mono" style={{ fontWeight: 600 }}>{rule.name}</span> will be removed permanently.
      </div>
      <div className="row" style={{ justifyContent: 'flex-end', gap: 8 }}>
        <button className="btn ghost sm" onClick={onClose}>Cancel</button>
        <button className="btn sm" style={{ background: 'var(--error)', color: '#fff' }} disabled={deleting} onClick={go}>{deleting ? 'Deleting…' : 'Delete'}</button>
      </div>
    </div>
  );
}

// ─── Toggle enabled inline ────────────────────────────────────────────────────

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
    <button className={'btn ghost icon sm' + (busy ? ' disabled' : '')} onClick={toggle} title={rule.enabled ? 'Disable' : 'Enable'} style={{ opacity: busy ? 0.5 : 1 }}>
      {rule.enabled ? <Icon.Check width={12} style={{ color: 'var(--success)' }}/> : <Icon.X width={12} style={{ color: 'var(--fg-faint)' }}/>}
    </button>
  );
}

// ─── Main Proxy component ──────────────────────────────────────────────────────

export function Proxy() {
  const toast = useToast();
  const lifecycle = useLifecycle();
  const { data: appsRaw } = useAPI('/admin/apps');
  const apps = appsRaw?.data || [];
  const { data: rulesRaw, refresh: refreshRules } = useAPI('/admin/proxy/rules/db');
  const allRules = rulesRaw?.data || [];

  const [appFilter, setAppFilter] = React.useState('all');
  const [modal, setModal] = React.useState(null); // null | { type: 'create'|'edit'|'delete'|'import', rule? }

  const rules = appFilter === 'all' ? allRules : allRules.filter(r => r.app_id === appFilter);

  const closeModal = () => setModal(null);

  if (lifecycle.missing) {
    return (
      <div style={{ padding: 40, textAlign: 'center' }}>
        <div className="muted">Proxy Gateway disabled. Enable proxy in sharkauth.yaml.</div>
        <div style={{ marginTop: 24, maxWidth: 500, margin: '24px auto' }}>
          <ProxyWizard onComplete={() => lifecycle.refresh()}/>
        </div>
      </div>
    );
  }

  const state = lifecycle.status?.state_str || 'unknown';
  const badgeCls = state === 'running' ? 'success' : state === 'reloading' ? 'warn' : 'error';

  return (
    <div className="col" style={{ height: '100%', overflow: 'auto' }}>
      {/* Header */}
      <div style={{ padding: '12px 20px', borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between', flexWrap: 'wrap', gap: 10 }}>
          <div className="row" style={{ gap: 12 }}>
            <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600, fontFamily: 'var(--font-display)' }}>Proxy Gateway</h1>
            {!lifecycle.loading && lifecycle.status && (
              <LifecycleBar status={lifecycle.status} act={lifecycle.act} toast={toast}/>
            )}
            {!lifecycle.loading && !lifecycle.status && (
              <span className="chip sm error">offline</span>
            )}
          </div>
          <div className="row" style={{ gap: 8 }}>
            <button className="btn ghost sm" onClick={() => setModal({ type: 'import' })}>
              <Icon.Migration width={11}/> Import YAML
            </button>
            <button className="btn primary sm" onClick={() => setModal({ type: 'create' })}>
              <Icon.Plus width={11}/> New Rule
            </button>
          </div>
        </div>
      </div>

      <div style={{ maxWidth: 1000, margin: '20px auto', width: '100%', padding: '0 20px 100px' }}>

        {/* Host topology */}
        {apps.filter(a => a.proxy_public_domain).length > 0 && (
          <>
            <div style={{ ...SECTION_LABEL, marginBottom: 12 }}>Mesh Topology</div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 10, marginBottom: 28 }}>
              {apps.filter(a => a.proxy_public_domain).map(app => (
                <div key={app.id} className="card" style={{ padding: 10 }}>
                  <div className="row" style={{ gap: 8 }}>
                    <Icon.Globe width={13} style={{ color: 'var(--success)', flexShrink: 0 }}/>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div className="mono" style={{ fontWeight: 600, fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{app.proxy_public_domain}</div>
                      <div className="faint mono" style={{ fontSize: 10 }}>{app.proxy_protected_url}</div>
                    </div>
                    <span className="dot success pulse"/>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        {/* Rules table */}
        <div className="row" style={{ ...SECTION_LABEL, justifyContent: 'space-between', marginBottom: 10 }}>
          <span>Route Policies ({rules.length})</span>
          <select value={appFilter} onChange={e => setAppFilter(e.target.value)}
            style={{ height: 24, fontSize: 11, background: 'var(--surface-2)', border: HAIRLINE, color: 'var(--fg)', padding: '0 4px', borderRadius: 4 }}>
            <option value="all">All Applications</option>
            {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
        </div>

        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
          <table className="tbl" style={{ width: '100%', fontSize: 12.5 }}>
            <thead>
              <tr style={{ background: 'var(--surface-1)' }}>
                <th style={{ padding: '8px 12px', textAlign: 'left', width: 44 }}>Prio</th>
                <th style={{ padding: '8px 12px', textAlign: 'left' }}>Name</th>
                <th style={{ padding: '8px 12px', textAlign: 'left' }}>Pattern</th>
                <th style={{ padding: '8px 12px', textAlign: 'left' }}>Require</th>
                <th style={{ padding: '8px 12px', textAlign: 'left' }}>App</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', width: 60 }}>M2M</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', width: 60 }}>On</th>
                <th style={{ padding: '8px 12px', textAlign: 'right', width: 80 }}></th>
              </tr>
            </thead>
            <tbody>
              {rules.length === 0 ? (
                <tr><td colSpan={8} style={{ padding: 40, textAlign: 'center' }}>
                  <div className="faint">No rules. Click <strong>New Rule</strong> or import YAML to get started.</div>
                </td></tr>
              ) : rules.map(r => (
                <tr key={r.id} style={{ borderTop: HAIRLINE }}>
                  <td className="mono muted" style={{ padding: '8px 12px' }}>{r.priority}</td>
                  <td style={{ padding: '8px 12px', fontWeight: 500, maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.name}</td>
                  <td style={{ padding: '8px 12px' }}>
                    <div className="mono" style={{ fontSize: 12 }}>{r.pattern}</div>
                    {r.methods?.length > 0 && <div className="faint" style={{ fontSize: 10 }}>{r.methods.join(', ')}</div>}
                  </td>
                  <td style={{ padding: '8px 12px' }}>
                    <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                      {(r.require || r.allow) && (
                        <span className={'chip sm ' + (r.allow ? 'ghost' : 'solid')} style={{ fontSize: 10 }}>
                          {r.require || r.allow}
                        </span>
                      )}
                      {r.tier_match && <span className="chip sm warn" style={{ fontSize: 10 }}>{r.tier_match}</span>}
                    </div>
                  </td>
                  <td className="faint" style={{ padding: '8px 12px', fontSize: 11 }}>
                    {apps.find(a => a.id === r.app_id)?.name || <span style={{ opacity: 0.4 }}>global</span>}
                  </td>
                  <td style={{ padding: '8px 12px' }}>
                    {r.m2m && <span className="chip sm" style={{ fontSize: 10 }}>m2m</span>}
                  </td>
                  <td style={{ padding: '8px 12px' }}>
                    <EnabledToggle rule={r} onToggled={refreshRules}/>
                  </td>
                  <td style={{ padding: '8px 12px', textAlign: 'right' }}>
                    <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
                      <button className="btn ghost icon sm" title="Edit" onClick={() => setModal({ type: 'edit', rule: r })}>
                        <Icon.Settings width={11}/>
                      </button>
                      <button className="btn ghost icon sm" title="Delete" onClick={() => setModal({ type: 'delete', rule: r })}
                        style={{ color: 'var(--error)' }}>
                        <Icon.X width={11}/>
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Modals */}
      {modal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.55)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50 }}
          onClick={closeModal}>
          {modal.type === 'create' && (
            <RuleModal apps={apps} onSave={() => { closeModal(); refreshRules(); }} onClose={closeModal}/>
          )}
          {modal.type === 'edit' && (
            <RuleModal rule={modal.rule} apps={apps} onSave={() => { closeModal(); refreshRules(); }} onClose={closeModal}/>
          )}
          {modal.type === 'delete' && (
            <DeleteConfirm rule={modal.rule} onConfirm={() => { closeModal(); refreshRules(); }} onClose={closeModal}/>
          )}
          {modal.type === 'import' && (
            <ImportModal onClose={closeModal} onDone={refreshRules}/>
          )}
        </div>
      )}

      <CLIFooter command="shark proxy status"/>
    </div>
  );
}
