// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { Login } from './login'
import { Sidebar, TopBar } from './layout'
import { Overview } from './overview'
import { Users } from './users'
import { MailEditor } from './MailEditor'
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
import { IdentityHub } from './identity_hub'
import { SSO } from './sso'
import { SigningKeys } from './signing_keys'
import { Settings } from './settings'
import { Consents } from './consents_manage'
import { Vault } from './vault_manage'
import { Tokens, APIExplorer, EventSchemas, SessionDebugger, OIDCProvider, Impersonation, Migrations } from './empty_shell'
import { Branding } from './branding'
import { EmailSettings } from './email_settings'
import { CompliancePage } from './compliance'
import { Proxy as ProxyPlaceholder } from './proxy_config'
import { Proxy as ProxyReal } from './proxy_config_real'
const Proxy = (import.meta as any).env?.VITE_FEATURE_PROXY === 'true' ? ProxyReal : ProxyPlaceholder
import { GetStarted } from './get_started'
import { Setup } from './setup'
import { Walkthrough } from './Walkthrough'
import { useKeyboardShortcuts, KeyboardCheatsheet } from './useKeyboardShortcuts'
import { CommandPalette } from './CommandPalette'
import { HelpButton } from './HelpButton'
import { DelegationChains } from './delegation_chains'

export function App() {
  const [apiKey, setApiKey] = React.useState(() => sessionStorage.getItem('shark_admin_key') || '');
  const [sidebarCollapsed, setSidebarCollapsed] = React.useState(false);

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
      sessionStorage.setItem('shark_admin_key', apiKey);
    } else {
      sessionStorage.removeItem('shark_admin_key');
    }
  }, [apiKey]);

  React.useEffect(() => {
    const handler = () => setApiKey('');
    window.addEventListener('shark-auth-expired', handler);
    return () => window.removeEventListener('shark-auth-expired', handler);
  }, []);

  const { cheatsheetOpen, setCheatsheetOpen } = useKeyboardShortcuts(setPage);
  const [paletteOpen, setPaletteOpen] = React.useState(false);
  const [walkthroughOpen, setWalkthroughOpen] = React.useState(false);

  // First-boot: pending admin key banner
  const [firstBootKey, setFirstBootKey] = React.useState<string>(() => {
    try { return sessionStorage.getItem('shark_first_boot_pending_key') || ''; } catch { return ''; }
  });
  const [bannerCopied, setBannerCopied] = React.useState(false);

  const dismissFirstBootBanner = () => {
    try {
      sessionStorage.removeItem('shark_first_boot_pending_key');
      localStorage.setItem('shark_first_boot_done', '1');
    } catch {}
    setFirstBootKey('');
    setPage('get-started');
  };

  const copyFirstBootKey = () => {
    if (!firstBootKey) return;
    navigator.clipboard.writeText(firstBootKey).then(() => {
      setBannerCopied(true);
      setTimeout(() => setBannerCopied(false), 2000);
    }).catch(() => {});
  };

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

  // W18: dev-email tab is always reachable regardless of email.provider.
  // Previous redirect kicked operators out when provider != 'dev'; the page
  // now self-handles the empty/wrong-provider state with a banner.

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
    auth: IdentityHub,
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
    email: EmailSettings,
    branding: Branding,
    emails: MailEditor,
    'get-started': GetStarted,
    'delegation-chains': DelegationChains,
  }[page] || Overview;

  return (
    <div style={{ display: 'flex', height: '100vh', background: 'var(--bg)' }}>
      <Sidebar page={page} setPage={setPage}
        collapsed={sidebarCollapsed}
        setCollapsed={setSidebarCollapsed}
        devMode={devMode}
        emailProvider={emailProvider}
        showPreview={false}/>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}
           data-screen-label={'01 ' + page}>
        {firstBootKey && (
          <div style={{
            background: 'var(--surface-1)',
            borderBottom: '1px solid var(--hairline-strong)',
            padding: '12px 24px',
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
            flexShrink: 0,
          }}>
            <div style={{ fontSize: 12, fontWeight: 600, letterSpacing: '0.04em', textTransform: 'uppercase', color: 'var(--warn)' }}>
              ⚠ ADMIN API KEY — YOU WILL ONLY SEE THIS ONCE
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
              <span style={{
                fontFamily: 'var(--font-mono, monospace)',
                fontSize: 13,
                color: 'var(--fg)',
                wordBreak: 'break-all',
                flex: 1,
                minWidth: 0,
              }}>{firstBootKey}</span>
              <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
                <button
                  onClick={copyFirstBootKey}
                  style={{
                    fontSize: 12,
                    padding: '4px 10px',
                    background: 'none',
                    border: '1px solid var(--hairline-strong)',
                    borderRadius: 4,
                    cursor: 'pointer',
                    color: 'var(--fg)',
                    fontFamily: 'inherit',
                    whiteSpace: 'nowrap',
                  }}
                >{bannerCopied ? 'Copied!' : 'Copy'}</button>
                <button
                  onClick={dismissFirstBootBanner}
                  style={{
                    fontSize: 12,
                    padding: '4px 10px',
                    background: 'none',
                    border: '1px solid var(--hairline-strong)',
                    borderRadius: 4,
                    cursor: 'pointer',
                    color: 'var(--fg)',
                    fontFamily: 'inherit',
                    whiteSpace: 'nowrap',
                  }}
                >Dismiss</button>
              </div>
            </div>
          </div>
        )}
        <TopBar page={page} onOpenPalette={() => setPaletteOpen(true)} setPage={setPage}/>
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

    </div>
  );
}
