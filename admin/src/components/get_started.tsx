// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { ProxyWizard } from './proxy_wizard'
import { CLIFooter } from './CLIFooter'

// Champion Tier Onboarding — Guided, Opinionated, Effortless.
// Reached on first login when users=0.

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
        transition: 'transform 100ms, border-color 100ms',
      }}
      onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--fg-muted)'; e.currentTarget.style.transform = 'translateY(-1px)'; }}
      onMouseLeave={e => { e.currentTarget.style.borderColor = 'var(--hairline)'; e.currentTarget.style.transform = 'none'; }}
    >
      <div style={{
        width: 32, height: 32, borderRadius: 8,
        background: 'var(--surface-3)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: 'var(--fg)',
        flexShrink: 0,
      }}>
        {icon}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 600, fontFamily: 'var(--font-display)' }}>{title}</div>
        <div className="faint" style={{ fontSize: 11.5, marginTop: 2 }}>{sub}</div>
      </div>
      <Icon.ChevronRight width={14} height={14} style={{ color: 'var(--fg-dim)' }}/>
    </button>
  );
}

export function GetStarted({ setPage }) {
  const [stage, setStep] = React.useState('hero'); // hero → wizard | alternative

  const go = (p) => {
    markOnboarded();
    if (typeof setPage === 'function') setPage(p);
  };
  
  const skip = () => {
    markOnboarded();
    if (typeof setPage === 'function') setPage('overview');
  };

  if (stage === 'hero') {
    return (
      <div style={{ height: '100%', overflow: 'auto', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 40 }}>
        <div style={{ maxWidth: 560, width: '100%', textAlign: 'center' }}>
           <div className="row" style={{ justifyContent: 'center', gap: 10, marginBottom: 20 }}>
              <Icon.Sparkle width={24} height={24} style={{ color: 'var(--accent)' }}/>
              <span style={{ fontSize: 14, fontWeight: 600, letterSpacing: '0.1em', textTransform: 'uppercase', color: 'var(--fg-dim)' }}>Welcome to SharkAuth</span>
           </div>
           
           <h1 style={{ fontSize: 42, fontWeight: 700, fontFamily: 'var(--font-display)', letterSpacing: '-0.03em', lineHeight: 1, marginBottom: 20 }}>
             Authentication <br/>without the bullshit.
           </h1>
           
           <p style={{ fontSize: 17, color: 'var(--fg-muted)', lineHeight: 1.6, marginBottom: 32 }}>
             You're one step away from protecting your application. How would you like to start?
           </p>

           <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: 12 }}>
             <button className="btn primary" style={{ height: 50, fontSize: 15, fontWeight: 600 }} onClick={() => setStep('wizard')}>
               Protect an Existing App (Proxy Gateway)
             </button>
             <button className="btn ghost" style={{ height: 50, fontSize: 15 }} onClick={() => setStep('alternative')}>
               I'll use the SDK / API
             </button>
           </div>
           
           <button className="btn ghost sm" style={{ marginTop: 40, opacity: 0.5 }} onClick={skip}>
             Skip to dashboard
           </button>
        </div>
      </div>
    );
  }

  return (
    <div style={{ height: '100%', overflow: 'auto', padding: '40px 24px' }}>
      <div style={{ maxWidth: 720, margin: '0 auto' }}>
        <button className="btn ghost sm" onClick={() => setStep('hero')} style={{ marginBottom: 24, paddingLeft: 0 }}>
           <Icon.ChevronLeft width={12}/> Back
        </button>

        {stage === 'wizard' ? (
          <div>
            <div style={{ marginBottom: 24 }}>
              <h2 style={{ fontSize: 24, fontWeight: 600, fontFamily: 'var(--font-display)', marginBottom: 8 }}>Setup Proxy Gateway</h2>
              <p className="faint" style={{ fontSize: 14 }}>The easiest way to add auth. Sit SharkAuth in front of your app, and we'll handle the rest.</p>
            </div>
            <ProxyWizard autofocus onComplete={() => { markOnboarded(); go('proxy'); }}/>
          </div>
        ) : (
          <div>
            <div style={{ marginBottom: 24 }}>
              <h2 style={{ fontSize: 24, fontWeight: 600, fontFamily: 'var(--font-display)', marginBottom: 8 }}>SDK & Configuration</h2>
              <p className="faint" style={{ fontSize: 14 }}>Explore the dashboard and configure your identity mesh manually.</p>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: 10 }}>
              <ChecklistRow
                icon={<Icon.App width={16}/>}
                title="Create an Application"
                sub="Register an OAuth 2.1 client to get your keys."
                onClick={() => go('apps')}
              />
              <ChecklistRow
                icon={<Icon.Lock width={16}/>}
                title="Configure Identity Policies"
                sub="Set password rules, session TTLs, and MFA."
                onClick={() => go('auth')}
              />
              <ChecklistRow
                icon={<Icon.Users width={16}/>}
                title="Create your first user"
                sub="Add yourself or a test user to the system."
                onClick={() => go('users')}
              />
              <ChecklistRow
                icon={<Icon.Signing width={16}/>}
                title="Customize Branding"
                sub="Personalize the login page and email templates."
                onClick={() => go('branding')}
              />
            </div>
          </div>
        )}

        <div style={{ marginTop: 40, borderTop: '1px solid var(--hairline)', paddingTop: 24 }}>
           <div className="faint" style={{ fontSize: 11, marginBottom: 12, textTransform: 'uppercase', letterSpacing: '0.05em' }}>Pro tip: Try the CLI</div>
           <CLIFooter command="shark serve --dev --proxy-upstream http://localhost:3000"/>
        </div>
      </div>
    </div>
  );
}
