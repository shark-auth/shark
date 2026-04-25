// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { Login } from './login'
import { Sidebar, TopBar } from './layout'
import { Overview } from './overview'
import { Users } from './users'
import { Sessions } from './sessions'
import { Organizations } from './organizations'
import { Agents } from './agents_manage'
import { DeviceFlow } from './device_flow'
import { Audit } from './audit'
import { Applications } from './applications'
import { ApiKeys } from './api_keys'
import { Webhooks } from './webhooks'
import { RBAC } from './rbac'
import { DevEmail } from './dev_email'
import { SSO } from './sso'
import { SigningKeys } from './signing_keys'
import { Settings } from './settings'
import { Consents } from './consents_manage'
import { Vault } from './vault_manage'
import { Tokens, APIExplorer, EventSchemas, SessionDebugger, OIDCProvider, Impersonation, Migrations } from './empty_shell'
import { Branding } from './branding'
import { CompliancePage } from './compliance'
import { Proxy } from './proxy_config'
import { GetStarted } from './get_started'
import { Setup } from './setup'
import { Walkthrough } from './Walkthrough'
import { useKeyboardShortcuts, KeyboardCheatsheet } from './useKeyboardShortcuts'
import { CommandPalette } from './CommandPalette'
import { HelpButton } from './HelpButton'

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "sidebarCollapsed": false,
  "font": "manrope",
  "showPreview": false
}/*EDITMODE-END*/;

function useTweaks() {
  const [t, setT] = React.useState(() => {
    try { return { ...TWEAK_DEFAULTS, ...(JSON.parse(localStorage.getItem('tweaks') || '{}')) }; }
    catch { return TWEAK_DEFAULTS; }
  });
  const set = (k, v) => setT(prev => {
    const next = { ...prev, [k]: v };
    try { localStorage.setItem('tweaks', JSON.stringify(next)); } catch {}
    try { window.parent.postMessage({ type: '__edit_mode_set_keys', edits: { [k]: v } }, '*'); } catch {}
    return next;
  });
  return [t, set];
}

export function App() {
  const [apiKey, setApiKey] = React.useState(() => localStorage.getItem('shark_admin_key') || '');

  const getPageFromURL = () => {
    const path = window.location.pathname.replace(/^\/admin\/?/, '').replace(/\/$/, '');
    if (!path) return 'overview';
    return path.split('/')[0] || 'overview';
  };

  const [page, setPageState] = React.useState(getPageFromURL);

  const setPage = (p, extra) => {
    let url = '/admin/' + p;
    if (typeof extra === 'string' && extra) {
      url += extra.startsWith('/') ? extra : '/' + extra;
    } else if (extra && typeof extra === 'object') {
      const qs = new URLSearchParams();
      for (const [k, v] of Object.entries(extra)) {
        if (v != null && v !== '') qs.set(k, String(v));
      }
      const s = qs.toString();
      if (s) url += '?' + s;
    }
    window.history.pushState(null, '', url);
    setPageState(p);
  };

  React.useEffect(() => {
    if (apiKey) {
      localStorage.setItem('shark_admin_key', apiKey);
    } else {
      localStorage.removeItem('shark_admin_key');
    }
  }, [apiKey]);

  const [tweaks, setTweak] = useTweaks();
  const [tweaksOpen, setTweaksOpen] = React.useState(false);
  const [editMode, setEditMode] = React.useState(false);

  React.useEffect(() => {
    document.body.className = 'font-' + tweaks.font;
  }, [tweaks.font]);

  React.useEffect(() => {
    const handler = () => setApiKey('');
    window.addEventListener('shark-auth-expired', handler);
    return () => window.removeEventListener('shark-auth-expired', handler);
  }, []);

  React.useEffect(() => {
    const handler = (e) => {
      if (e.data?.type === '__activate_edit_mode') { setEditMode(true); setTweaksOpen(true); }
      if (e.data?.type === '__deactivate_edit_mode') { setEditMode(false); setTweaksOpen(false); }
    };
    window.addEventListener('message', handler);
    try { window.parent.postMessage({ type: '__edit_mode_available' }, '*'); } catch {}
    return () => window.removeEventListener('message', handler);
  }, []);

  const { cheatsheetOpen, setCheatsheetOpen } = useKeyboardShortcuts(setPage);
  const [paletteOpen, setPaletteOpen] = React.useState(false);
  const [walkthroughOpen, setWalkthroughOpen] = React.useState(false);

  React.useEffect(() => {
    if (localStorage.getItem('shark_admin_onboarded') === '1' && !localStorage.getItem('shark_walkthrough_seen')) {
      setWalkthroughOpen(true);
    }
  }, [page]);

  React.useEffect(() => {
    const handler = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') { e.preventDefault(); setPaletteOpen(v => !v); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  const [adminConfig, setAdminConfig] = React.useState(null);
  React.useEffect(() => {
    if (!apiKey) return;
    fetch('/api/v1/admin/config', { headers: { Authorization: `Bearer ${apiKey}` } })
      .then(r => r.ok ? r.json() : null)
      .then(d => { if (d) setAdminConfig(d); })
      .catch(err => { console.error('[admin] config fetch failed:', err); });
  }, [apiKey]);
  const devMode = adminConfig?.dev_mode ?? false;
  const emailProvider: string = adminConfig?.email?.provider ?? '';

  const redirectedRef = React.useRef(false);
  React.useEffect(() => {
    if (!apiKey || redirectedRef.current) return;
    if (localStorage.getItem('shark_admin_onboarded') === '1') return;
    if (page !== 'overview') return;
    fetch('/api/v1/admin/stats', { headers: { Authorization: `Bearer ${apiKey}` } })
      .then(r => r.ok ? r.json() : null)
      .then(d => {
        if (!d) return;
        const total = d?.users?.total ?? null;
        if (total === 0 && !redirectedRef.current) {
          redirectedRef.current = true;
          setPage('get-started');
        }
      })
      .catch(err => { console.error('[admin] stats fetch failed:', err); });
  }, [apiKey, page]);

  React.useEffect(() => {
    if (adminConfig !== null && emailProvider !== '' && emailProvider !== 'dev' && page === 'dev-email') {
      setPage('overview');
    }
  }, [adminConfig, emailProvider, page]);

  // Setup page: reachable without a login session (first-boot flow).
  // Check pathname (no query-string) so /admin/setup?token=... also matches.
  if (window.location.pathname.replace(/\/$/, '') === '/admin/setup') {
    return <Setup />;
  }

  if (!apiKey) {
    return <Login onLogin={(k) => setApiKey(k)} />;
  }

  const Page = {
    overview: Overview,
    users: Users,
    sessions: Sessions,
    orgs: Organizations,
    agents: Agents,
    'device-flow': DeviceFlow,
    audit: Audit,
    apps: Applications,
    keys: ApiKeys,
    webhooks: Webhooks,
    rbac: RBAC,
    'dev-email': DevEmail,
    sso: SSO,
    signing: SigningKeys,
    settings: Settings,
    consents: Consents,
    tokens: Tokens,
    vault: Vault,
    explorer: APIExplorer,
    debug: SessionDebugger,
    schemas: EventSchemas,
    proxy: Proxy,
    oidc: OIDCProvider,
    impersonation: Impersonation,
    compliance: CompliancePage,
    migrations: Migrations,
    branding: Branding,
    'get-started': GetStarted,
  }[page] || Overview;

  return (
    <div style={{ display: 'flex', height: '100vh', background: 'var(--bg)' }}>
      <Sidebar page={page} setPage={setPage}
        collapsed={tweaks.sidebarCollapsed}
        setCollapsed={(v) => setTweak('sidebarCollapsed', v)}
        devMode={devMode}
        emailProvider={emailProvider}
        showPreview={tweaks.showPreview}/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}
           data-screen-label={'01 ' + page}>
        <TopBar page={page} setTweaksOpen={setTweaksOpen} onOpenPalette={() => setPaletteOpen(true)} setPage={setPage}/>
        <div style={{ flex: 1, overflow: 'hidden', animation: 'fadeIn 160ms ease-out' }}>
          <Page setPage={setPage}/>
        </div>
      </div>

      {cheatsheetOpen && <KeyboardCheatsheet onClose={() => setCheatsheetOpen(false)}/>}
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} setPage={setPage}/>
      <HelpButton/>
      {walkthroughOpen && (
        <Walkthrough onComplete={() => {
          localStorage.setItem('shark_walkthrough_seen', '1');
          setWalkthroughOpen(false);
        }}/>
      )}

      {tweaksOpen && (
        <div className="tweaks-panel">
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
            <h3 style={{margin:0}}>Tweaks</h3>
            <button className="btn ghost icon sm" onClick={() => setTweaksOpen(false)}><Icon.X width={11} height={11}/></button>
          </div>
          <div className="tweak-row">
            <label>Sidebar</label>
            <div className="seg">
              <button className={!tweaks.sidebarCollapsed ? 'on' : ''} onClick={() => setTweak('sidebarCollapsed', false)}>Expanded</button>
              <button className={tweaks.sidebarCollapsed ? 'on' : ''} onClick={() => setTweak('sidebarCollapsed', true)}>Collapsed</button>
            </div>
          </div>
          <div className="tweak-row">
            <label>Font</label>
            <div className="seg">
              <button className={tweaks.font === 'manrope' ? 'on' : ''} onClick={() => setTweak('font', 'manrope')}>Manrope</button>
              <button className={tweaks.font === 'inter' ? 'on' : ''} onClick={() => setTweak('font', 'inter')}>Inter</button>
              <button className={tweaks.font === 'ibm' ? 'on' : ''} onClick={() => setTweak('font', 'ibm')}>IBM</button>
              <button className={tweaks.font === 'space' ? 'on' : ''} onClick={() => setTweak('font', 'space')}>Space</button>
            </div>
          </div>
          <div className="tweak-row">
            <label>Preview features</label>
            <div className="seg">
              <button className={!tweaks.showPreview ? 'on' : ''} onClick={() => setTweak('showPreview', false)}>Hidden</button>
              <button className={tweaks.showPreview ? 'on' : ''} onClick={() => setTweak('showPreview', true)}>Shown</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
