// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { ProxyWizard } from './proxy_wizard'

// Phase Wave E — Truly Transparent Gateway Dashboard (Unified Traffic Control).

const SECTION_LABEL = {
  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
  color: 'var(--fg-dim)', fontWeight: 500,
};

const HAIRLINE = '1px solid var(--hairline)';

function useProxyStats() {
  const [statsMap, setStatsMap] = React.useState(null);
  const [status, setStatus] = React.useState(0);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);
  const [probing, setProbing] = React.useState(false);

  const fetchOnce = React.useCallback(async () => {
    const key = localStorage.getItem('shark_admin_key');
    if (!key) return;
    try {
      const res = await fetch('/api/v1/admin/proxy/status', {
        headers: { 'Authorization': 'Bearer ' + key },
      });
      if (res.status === 401) {
        localStorage.removeItem('shark_admin_key');
        window.dispatchEvent(new Event('shark-auth-expired'));
        return;
      }
      setStatus(res.status);
      if (res.status === 404) {
        setStatsMap(null);
        setError(null);
      } else if (res.ok) {
        const j = await res.json();
        setStatsMap(j?.data || null);
        setError(null);
      } else {
        setError('HTTP ' + res.status);
      }
    } catch (e) {
      setError(e.message || 'network error');
    } finally {
      setLoading(false);
    }
  }, []);

  const refresh = React.useCallback(async () => {
    setProbing(true);
    await fetchOnce();
    setTimeout(() => setProbing(false), 600);
  }, [fetchOnce]);

  React.useEffect(() => {
    fetchOnce();
    const interval = setInterval(fetchOnce, 5000);
    return () => clearInterval(interval);
  }, [fetchOnce]);

  return { statsMap, status, loading, error, refresh, probing };
}

export function Proxy() {
  const { statsMap, status, loading, refresh, probing } = useProxyStats();
  const { data: appsRaw } = useAPI('/admin/apps');
  const apps = appsRaw?.data || [];
  
  const [appFilter, setAppFilter] = React.useState('all');
  const [creating, setCreating] = React.useState(false);

  // Fetch all rules from all apps
  const { data: rulesRaw, refresh: refreshRules } = useAPI('/admin/proxy/rules/db');
  const allRules = rulesRaw?.data || [];
  const rules = appFilter === 'all' ? allRules : allRules.filter(r => r.app_id === appFilter);

  if (status === 404) {
    return (
      <div style={{ padding: 40, textAlign: 'center' }}>
        <div className="muted">Proxy Gateway disabled. Enable proxy in sharkauth.yaml.</div>
        <div style={{ marginTop: 24, maxWidth: 500, margin: '24px auto' }}>
          <ProxyWizard onComplete={refresh}/>
        </div>
      </div>
    );
  }

  return (
    <div className="col" style={{ height: '100%', overflow: 'auto' }}>
      <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between' }}>
          <div className="row" style={{ gap: 12 }}>
            <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600, fontFamily: 'var(--font-display)' }}>Proxy Gateway</h1>
            <span className="chip success sm">ACTIVE</span>
          </div>
          <div className="row" style={{ gap: 10 }}>
             <button className="btn primary sm" onClick={() => setCreating(true)}>
               <Icon.Plus width={11}/> New Rule
             </button>
             <button className="btn ghost sm" onClick={refresh}>
               <Icon.Refresh width={11} style={{ animation: probing ? 'spin 800ms linear infinite' : 'none' }}/>
             </button>
          </div>
        </div>
      </div>

      <div style={{ maxWidth: 1000, margin: '20px auto', width: '100%', padding: '0 20px 100px' }}>
        
        {/* Host Map */}
        <div style={{ ...SECTION_LABEL, marginBottom: 12 }}>Mesh Topology (Hosts)</div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 12, marginBottom: 32 }}>
          {apps.filter(a => a.proxy_public_domain).map(app => (
            <div key={app.id} className="card" style={{ padding: 12 }}>
              <div className="row" style={{ gap: 10 }}>
                <Icon.Globe width={14} style={{ color: 'var(--success)' }}/>
                <div style={{ flex: 1 }}>
                  <div className="mono" style={{ fontWeight: 600, fontSize: 13 }}>{app.proxy_public_domain}</div>
                  <div className="faint mono" style={{ fontSize: 11 }}>{app.proxy_protected_url}</div>
                </div>
                <div className="dot success pulse"/>
              </div>
            </div>
          ))}
        </div>

        {/* Unified Rules */}
        <div className="row" style={{ ...SECTION_LABEL, justifyContent: 'space-between' }}>
          <span>Route Policies</span>
          <select value={appFilter} onChange={e => setAppFilter(e.target.value)} style={{ height: 24, fontSize: 11, background: 'var(--surface-2)', border: HAIRLINE, color: 'var(--fg)', padding: '0 4px', borderRadius: 4 }}>
            <option value="all">All Applications</option>
            {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
        </div>

        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
          <table className="tbl" style={{ width: '100%', fontSize: 12.5 }}>
            <thead>
              <tr style={{ background: 'var(--surface-1)' }}>
                <th style={{ padding: 12, textAlign: 'left', width: 60 }}>Prio</th>
                <th style={{ padding: 12, textAlign: 'left' }}>App</th>
                <th style={{ padding: 12, textAlign: 'left' }}>Pattern</th>
                <th style={{ padding: 12, textAlign: 'left' }}>Policy</th>
                <th style={{ padding: 12, textAlign: 'left' }}>Hits</th>
              </tr>
            </thead>
            <tbody>
              {rules.length === 0 ? <tr><td colSpan={5} style={{ padding: 40, textAlign: 'center' }}><div className="faint">No rules found.</div></td></tr> : rules.map(r => (
                <tr key={r.id} style={{ borderTop: HAIRLINE }}>
                  <td style={{ padding: 12 }} className="mono muted">{r.priority}</td>
                  <td style={{ padding: 12 }} className="mono">{apps.find(a => a.id === r.app_id)?.name || '(Global)'}</td>
                  <td style={{ padding: 12 }}>
                    <div className="mono" style={{ fontWeight: 500 }}>{r.pattern}</div>
                    <div className="faint" style={{ fontSize: 10 }}>{r.methods.length > 0 ? r.methods.join(', ') : 'ANY'}</div>
                  </td>
                  <td style={{ padding: 12 }}>
                    <span className={"chip sm " + (r.require ? 'solid' : 'ghost')}>{r.require || r.allow}</span>
                  </td>
                  <td style={{ padding: 12 }} className="mono muted">—</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

      </div>

      {creating && (
        <div style={{ position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 }} onClick={() => setCreating(false)}>
          <div style={{ width: 500 }} onClick={e => e.stopPropagation()}>
            <ProxyWizard 
              onComplete={() => { setCreating(false); refreshRules(); }}
            />
          </div>
        </div>
      )}

      <CLIFooter command="shark proxy status"/>
      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}
