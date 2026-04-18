// @ts-nocheck
import React from 'react'
import { Icon, Avatar, Kbd, CopyField, sharkyGlyphPng, sharkyWordmarkPng } from './shared'

// Sidebar + topbar + layout shell

export const NAV = [
  { group: null, items: [{ id: 'overview', label: 'Overview', icon: 'Home' }] },
  { group: 'HUMANS', items: [
    { id: 'users', label: 'Users', icon: 'Users' },
    { id: 'sessions', label: 'Sessions', icon: 'Session' },
    { id: 'orgs', label: 'Organizations', icon: 'Org' },
  ]},
  { group: 'AGENTS', items: [
    { id: 'agents', label: 'Agents', icon: 'Agent', badge: 'new' },
    { id: 'consents', label: 'Consents', icon: 'Consent' },
    { id: 'tokens', label: 'Tokens', icon: 'Token', badge: '1.2k' },
    { id: 'vault', label: 'Vault', icon: 'Vault' },
    { id: 'device-flow', label: 'Device Flow', icon: 'Device', badge: '4', live: true },
  ]},
  { group: 'ACCESS', items: [
    { id: 'apps', label: 'Applications', icon: 'App' },
    { id: 'auth', label: 'Authentication', icon: 'Lock' },
    { id: 'sso', label: 'SSO', icon: 'SSO' },
    { id: 'rbac', label: 'Roles & Permissions', icon: 'Shield' },
    { id: 'keys', label: 'API Keys', icon: 'Key' },
  ]},
  { group: 'OPERATIONS', items: [
    { id: 'audit', label: 'Audit Log', icon: 'Audit' },
    { id: 'webhooks', label: 'Webhooks', icon: 'Webhook' },
    { id: 'dev-inbox', label: 'Dev Inbox', icon: 'Mail', badge: 'dev' },
    { id: 'signing', label: 'Signing Keys', icon: 'Signing' },
    { id: 'settings', label: 'Settings', icon: 'Settings' },
  ]},
  { group: 'DEVELOPERS', items: [
    { id: 'explorer', label: 'API Explorer', icon: 'Explorer', ph: 5 },
    { id: 'debug', label: 'Session Debugger', icon: 'Debug', ph: 5 },
    { id: 'schemas', label: 'Event Schemas', icon: 'Schema', ph: 5 },
  ]},
];

export function Sidebar({ page, setPage, collapsed, setCollapsed }) {
  return (
    <aside style={{
      width: collapsed ? 'var(--sidebar-w-collapsed)' : 'var(--sidebar-w)',
      background: 'var(--surface-0)',
      borderRight: '1px solid var(--hairline)',
      display: 'flex', flexDirection: 'column',
      transition: 'width 160ms ease-out',
      flexShrink: 0,
      overflow: 'hidden',
    }}>
      {/* Logo */}
      <div style={{
        height: 'var(--topbar-h)',
        display: 'flex', alignItems: 'center', gap: 10,
        padding: collapsed ? '0' : '0 12px',
        justifyContent: collapsed ? 'center' : 'space-between',
        borderBottom: '1px solid var(--hairline)',
        background: '#000',
        color: '#fff',
      }}>
        {collapsed ? (
          <img
            src={sharkyGlyphPng}
            alt="Shark"
            style={{ display: 'block', height: 26, width: 'auto' }}
          />
        ) : (
          <>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, minWidth: 0 }}>
              <img
                src={sharkyGlyphPng}
                alt=""
                style={{ display: 'block', height: 26, width: 'auto', flexShrink: 0 }}
              />
              <img
                src={sharkyWordmarkPng}
                alt="Shark"
                style={{ display: 'block', height: 18, width: 'auto', flexShrink: 0 }}
              />
            </div>
            <span style={{
              height: 18, fontSize: 10, padding: '0 6px',
              display: 'inline-flex', alignItems: 'center',
              border: '1px solid rgba(255,255,255,0.25)', borderRadius: 3,
              color: 'rgba(255,255,255,0.7)', fontVariantNumeric: 'tabular-nums',
              flexShrink: 0,
            }}>v0.8.2</span>
          </>
        )}
      </div>

      {/* Nav */}
      <nav style={{ flex: 1, overflowY: 'auto', padding: '6px 0' }}>
        {NAV.map((section, i) => (
          <div key={i} style={{ marginBottom: 6 }}>
            {section.group && !collapsed && (
              <div style={{
                padding: '10px 14px 4px',
                fontSize: 10, letterSpacing: '0.1em',
                color: 'var(--fg-faint)', fontWeight: 500,
              }}>{section.group}</div>
            )}
            {section.group && collapsed && <div style={{ height: 8 }}/>}
            {section.items.map(item => {
              const IconEl = Icon[item.icon] || Icon.Home;
              const active = page === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => !item.disabled && setPage(item.id)}
                  title={collapsed ? item.label : undefined}
                  disabled={item.disabled}
                  style={{
                    width: '100%',
                    display: 'flex', alignItems: 'center', gap: 10,
                    padding: collapsed ? '0' : '0 14px',
                    justifyContent: collapsed ? 'center' : 'flex-start',
                    height: 28,
                    color: active ? 'var(--fg)' : item.disabled ? 'var(--fg-faint)' : 'var(--fg-muted)',
                    background: active ? 'var(--surface-3)' : 'transparent',
                    borderLeft: active ? '2px solid #fff' : '2px solid transparent',
                    paddingLeft: collapsed ? 0 : 12,
                    fontSize: 12.5,
                    fontWeight: active ? 500 : 400,
                    cursor: item.disabled ? 'default' : 'pointer',
                    opacity: item.disabled ? 0.55 : 1,
                    transition: 'background 60ms',
                  }}
                  onMouseEnter={e => !active && !item.disabled && (e.currentTarget.style.background = 'var(--surface-2)')}
                  onMouseLeave={e => !active && (e.currentTarget.style.background = 'transparent')}
                >
                  <IconEl width={14} height={14} style={{ flexShrink: 0 }}/>
                  {!collapsed && <>
                    <span style={{ flex: 1, textAlign: 'left' }}>{item.label}</span>
                    {item.badge && (
                      <span className={"chip" + (item.live ? " warn" : "")} style={{ height: 16, fontSize: 10, padding: '0 5px' }}>
                        {item.live && <span className="dot warn pulse" style={{width:4,height:4}}/>}
                        {item.badge}
                      </span>
                    )}
                    {item.ph && <span className="mono" style={{ fontSize: 9, color: 'var(--fg-faint)' }}>P{item.ph}</span>}
                  </>}
                </button>
              );
            })}
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div style={{
        borderTop: '1px solid var(--hairline)',
        padding: collapsed ? '8px 0' : '8px 12px',
        display: 'flex', flexDirection: collapsed ? 'column' : 'row',
        alignItems: 'center', gap: 8,
        fontSize: 11, color: 'var(--fg-dim)',
      }}>
        {!collapsed && (
          <div className="row" style={{ flex: 1, gap: 6 }}>
            <span className="dot success"/>
            <span className="mono">healthy</span>
            <span className="faint">·</span>
            <span className="mono">dev</span>
          </div>
        )}
        <button
          className="btn ghost icon sm"
          onClick={() => setCollapsed(!collapsed)}
          title={collapsed ? 'Expand' : 'Collapse'}
        >
          {collapsed ? <Icon.Expand width={13} height={13}/> : <Icon.Collapse width={13} height={13}/>}
        </button>
      </div>
    </aside>
  );
}

export function TopBar({ page, setTweaksOpen }) {
  const pageTitle = {
    overview: 'Overview',
    users: 'Users',
    sessions: 'Sessions',
    orgs: 'Organizations',
    agents: 'Agents',
    'device-flow': 'Device Flow',
    audit: 'Audit Log',
    apps: 'Applications',
    keys: 'API Keys',
    rbac: 'Roles & Permissions',
    'dev-inbox': 'Dev Inbox',
    sso: 'SSO Connections',
    auth: 'Authentication',
    signing: 'Signing Keys',
    settings: 'Settings',
    consents: 'Consents',
    tokens: 'Tokens',
    vault: 'Vault',
    explorer: 'API Explorer',
    debug: 'Session Debugger',
    schemas: 'Event Schemas',
  }[page] || page;

  return (
    <header style={{
      height: 'var(--topbar-h)',
      display: 'flex', alignItems: 'center',
      padding: '0 16px',
      background: 'var(--surface-0)',
      borderBottom: '1px solid var(--hairline)',
      gap: 12,
      flexShrink: 0,
    }}>
      <div className="row" style={{ gap: 6, minWidth: 0 }}>
        <span style={{ fontWeight: 500, fontSize: 13 }}>{pageTitle}</span>
        {page === 'device-flow' && (
          <span className="chip warn" style={{marginLeft: 8}}>
            <span className="dot warn pulse"/>
            live
          </span>
        )}
      </div>

      <div style={{ flex: 1 }}/>

      <button className="btn ghost" style={{
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        width: 280, justifyContent: 'flex-start',
      }}>
        <Icon.Search width={13} height={13} style={{opacity:0.6}}/>
        <span className="faint" style={{flex:1, textAlign:'left'}}>Search or run a command…</span>
        <Kbd keys="⌘ K"/>
      </button>

      <button className="btn">
        <Icon.Plus width={12} height={12}/>
        Quick create
        <Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/>
      </button>

      <button className="btn ghost icon" title="Notifications" style={{ position: 'relative' }}>
        <Icon.Bell width={14} height={14}/>
        <span style={{
          position: 'absolute', top: 6, right: 6,
          width: 6, height: 6, borderRadius: 99,
          background: 'var(--danger)',
        }}/>
      </button>

      <div style={{ width: 1, height: 18, background: 'var(--hairline-strong)' }}/>

      <button className="btn ghost" style={{ padding: '0 4px 0 8px', height: 28 }}>
        <span className="faint" style={{fontSize: 11}}>env</span>
        <span className="mono" style={{fontSize: 11}}>nimbus-prod</span>
        <Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/>
      </button>

      <button
        onClick={() => setTweaksOpen(v => !v)}
        className="btn ghost icon"
        title="Tweaks">
        <Icon.Sparkle width={13} height={13}/>
      </button>

      <button
        className="btn ghost sm"
        title="Sign out"
        onClick={() => { sessionStorage.removeItem('shark_admin_key'); window.location.reload(); }}
        style={{ color: 'var(--fg-dim)', fontSize: 11 }}
      >
        Sign out
      </button>

      <Avatar name="You" email="admin@shark" size={26}/>
    </header>
  );
}

