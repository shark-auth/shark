// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { ProxyWizard } from './proxy_wizard'
import { CLIFooter } from './CLIFooter'

// Dedicated onboarding page reached on first login (shark_admin_onboarded !== '1'
// AND stats users=0). Combines Overview hero messaging + Proxy wizard + a
// checklist of alternative first-actions. Marks the flag on skip/completion
// so subsequent sessions land on Overview instead.

function markOnboarded() {
  try { sessionStorage.setItem('shark_admin_onboarded', '1'); } catch {}
}

function ChecklistRow({ icon, title, sub, onClick }) {
  return (
    <button onClick={onClick}
      className="card"
      style={{
        display: 'flex', alignItems: 'center', gap: 14,
        padding: '14px 16px',
        textAlign: 'left',
        cursor: 'pointer',
        border: '1px solid var(--hairline)',
        background: 'var(--surface-1)',
        color: 'inherit',
      }}>
      <div style={{
        width: 28, height: 28, borderRadius: 6,
        background: 'var(--surface-3)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: 'var(--fg-muted)',
        flexShrink: 0,
      }}>
        {icon}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 600, fontFamily: 'var(--font-display)' }}>{title}</div>
        <div className="faint" style={{ fontSize: 11.5, marginTop: 2 }}>{sub}</div>
      </div>
      <Icon.ChevronRight width={12} height={12} style={{ color: 'var(--fg-dim)' }}/>
    </button>
  );
}

export function GetStarted({ setPage }) {
  const go = (p) => {
    markOnboarded();
    if (typeof setPage === 'function') setPage(p);
  };
  const skip = () => {
    markOnboarded();
    if (typeof setPage === 'function') setPage('overview');
  };

  return (
    <div style={{ height: '100%', overflow: 'auto', padding: 24 }}>
      <div style={{ maxWidth: 720, margin: '0 auto' }}>
        {/* Header */}
        <div style={{ marginBottom: 20 }}>
          <div className="row" style={{ gap: 8, marginBottom: 10 }}>
            <span className="chip agent" style={{ height: 18, fontSize: 10 }}>Get started</span>
            <span className="faint mono" style={{ fontSize: 11 }}>first-run</span>
          </div>
          <h1 style={{
            margin: 0, fontSize: 28, fontWeight: 600,
            fontFamily: 'var(--font-display)', letterSpacing: '-0.02em', lineHeight: 1.1,
          }}>
            Ready to ship auth in 60 seconds
          </h1>
          <p style={{
            margin: '10px 0 0',
            fontSize: 14, color: 'var(--fg-muted)', lineHeight: 1.55,
          }}>
            SharkAuth protects any upstream with zero code changes. Paste a URL,
            pick routes, and you're live. Or explore the dashboard first —
            both work.
          </p>
          <div className="row" style={{ marginTop: 12, gap: 10 }}>
            <button className="btn ghost sm" onClick={skip}>
              Skip to dashboard
            </button>
          </div>
        </div>

        {/* Proxy wizard */}
        <div style={{ marginBottom: 24 }}>
          <ProxyWizard autofocus onComplete={() => { markOnboarded(); }}/>
        </div>

        {/* Alternative paths */}
        <div style={{ marginBottom: 14 }}>
          <div style={{
            fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
            color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 10,
          }}>
            Or explore first
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: 8 }}>
            <ChecklistRow
              icon={<Icon.Users width={13} height={13}/>}
              title="Create your first user"
              sub="Email + password or magic-link invite."
              onClick={() => go('users')}
            />
            <ChecklistRow
              icon={<Icon.Key width={13} height={13}/>}
              title="Create an API key"
              sub="Scoped admin keys for CI, migrations, and scripts."
              onClick={() => go('keys')}
            />
            <ChecklistRow
              icon={<Icon.Info width={13} height={13}/>}
              title="Set up SMTP"
              sub="Enable magic links, email verification, and alerts."
              onClick={() => go('auth')}
            />
          </div>
        </div>

        <div style={{ marginTop: 24 }}>
          <CLIFooter command="shark serve --proxy-upstream http://localhost:3000"/>
        </div>
      </div>
    </div>
  );
}
