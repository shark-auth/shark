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
import { DevInbox } from './dev_inbox'
import { SSO } from './sso'
import { Authentication } from './authentication'
import { SigningKeys } from './signing_keys'
import { Settings } from './settings'
import { Consents } from './consents_manage'
import { Vault } from './vault_manage'
import { Tokens, APIExplorer, SessionDebugger, EventSchemas, OIDCProvider, Impersonation, CompliancePage, Migrations, Branding } from './empty_shell'
import { FlowBuilder } from './flow_builder'
import { Proxy } from './proxy_config'
import { useKeyboardShortcuts, KeyboardCheatsheet } from './useKeyboardShortcuts'
import { CommandPalette } from './CommandPalette'

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "sidebarCollapsed": false,
  "font": "manrope"
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
  const [apiKey, setApiKey] = React.useState(() => sessionStorage.getItem('shark_admin_key') || '');

  // Page id is the FIRST path segment under /admin/. Anything after (e.g.
  // /admin/users/usr_abc/security) is preserved in the URL and exposed to
  // the page via window.location.pathname so deep-link routing works.
  const getPageFromURL = () => {
    const path = window.location.pathname.replace(/^\/admin\/?/, '').replace(/\/$/, '');
    if (!path) return 'overview';
    return path.split('/')[0] || 'overview';
  };

  const [page, setPageState] = React.useState(getPageFromURL);

  // setPage(id, paramsOrSubpath?)
  //   - id: page key (e.g. 'users')
  //   - second arg, when an object, becomes ?key=value query params
  //   - when a string, is appended as a sub-path (e.g. '/usr_abc/security')
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
    const handler = () => setPageState(getPageFromURL());
    window.addEventListener('popstate', handler);
    return () => window.removeEventListener('popstate', handler);
  }, []);

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

  React.useEffect(() => {
    const handler = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') { e.preventDefault(); setPaletteOpen(v => !v); }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

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
    'dev-inbox': DevInbox,
    sso: SSO,
    auth: Authentication,
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
    flow: FlowBuilder,
  }[page] || Overview;

  return (
    <div style={{ display: 'flex', height: '100vh', background: 'var(--bg)' }}>
      <Sidebar page={page} setPage={setPage}
        collapsed={tweaks.sidebarCollapsed}
        setCollapsed={(v) => setTweak('sidebarCollapsed', v)}/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}
           data-screen-label={'01 ' + page}>
        <TopBar page={page} setTweaksOpen={setTweaksOpen} onOpenPalette={() => setPaletteOpen(true)} setPage={setPage}/>
        <div style={{ flex: 1, overflow: 'hidden', animation: 'fadeIn 160ms ease-out' }}>
          <Page/>
        </div>
      </div>

      {cheatsheetOpen && <KeyboardCheatsheet onClose={() => setCheatsheetOpen(false)}/>}
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} setPage={setPage}/>

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
          <div className="faint" style={{ fontSize: 10.5, marginTop: 8, paddingTop: 8, borderTop: '1px solid var(--hairline)' }}>
            Dark theme, B&W palette. Dev preview.
          </div>
        </div>
      )}
    </div>
  );
}
