// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { ProxyWizard } from './proxy_wizard'
import { CLIFooter } from './CLIFooter'

// Champion Tier Onboarding — High-Density, Sovereign, Agent-First.
// Implementation of Task T11 from DASHBOARD_DX_EXECUTION_PLAN.

function markOnboarded() {
  try { localStorage.setItem('shark_admin_onboarded', '1'); } catch {}
}

function PathCard({ icon, title, subtitle, benefits, onClick, primary = false }) {
  return (
    <button onClick={onClick} className={"path-card" + (primary ? " primary" : "")}>
      {primary && <div className="badge">Recommended</div>}
      <div className="icon-wrap">{icon}</div>
      <div className="content">
        <div className="title">{title}</div>
        <div className="subtitle">{subtitle}</div>
        <ul className="benefits">
          {benefits.map((b, i) => (
            <li key={i}><Icon.Check width={12}/> {b}</li>
          ))}
        </ul>
      </div>
      <div className="footer">
        <span>Get started</span> <Icon.ChevronRight width={14}/>
      </div>
    </button>
  );
}

function FeatureBox({ icon, title, description, group }) {
  return (
    <div className="feature-box">
      <div className="group-label">{group}</div>
      <div className="row" style={{ gap: 12, alignItems: 'center', marginBottom: 8 }}>
        <div className="icon-sm">{icon}</div>
        <div className="feature-title">{title}</div>
      </div>
      <div className="feature-desc">{description}</div>
    </div>
  );
}

export function GetStarted({ setPage }) {
  const [stage, setStep] = React.useState('hero'); // hero → matrix → selection → wizard

  const go = (p) => {
    markOnboarded();
    if (typeof setPage === 'function') setPage(p);
  };

  const skip = () => {
    markOnboarded();
    if (typeof setPage === 'function') setPage('overview');
  };

  return (
    <div className="onboarding-overlay">
      <div className="onboarding-container">
        
        {stage === 'hero' && (
          <div className="hero-stage" style={{ animation: 'fadeIn 400ms ease-out' }}>
            <div className="logo-badge">
              <Icon.Sparkle width={24} height={24}/>
              <span>Sovereign Identity</span>
            </div>
            <h1>Identity infrastructure <br/>built for AI agents.</h1>
            <p>SharkAuth is a single-binary, no-bullshit auth platform that treats autonomous agents as first-class citizens.</p>
            
            <div className="action-row">
              <button className="btn primary lg" onClick={() => setStep('matrix')}>
                Begin Setup <Icon.ChevronRight width={18}/>
              </button>
              <button className="btn ghost" onClick={skip}>Skip to dashboard</button>
            </div>
          </div>
        )}

        {stage === 'matrix' && (
          <div className="matrix-stage" style={{ animation: 'slideUp 300ms ease-out' }}>
            <header>
              <button className="btn ghost sm" onClick={() => setStep('hero')}><Icon.ChevronLeft width={12}/> Back</button>
              <h2>What you can do</h2>
              <p>A complete toolkit for human authentication and agentic delegation.</p>
            </header>

            <div className="matrix-grid">
              <div className="matrix-col">
                <FeatureBox 
                  group="HUMAN IDENTITY"
                  icon={<Icon.Lock width={14}/>} 
                  title="Full Auth Stack" 
                  description="Passwords, MFA, Organizations, and RBAC with Argon2id."
                />
                <FeatureBox 
                  group="HUMAN IDENTITY"
                  icon={<Icon.Globe width={14}/>} 
                  title="Social & SSO" 
                  description="Native Google, GitHub, and Enterprise SAML/OIDC."
                />
              </div>
              <div className="matrix-col">
                <FeatureBox 
                  group="AGENTIC FLOWS"
                  icon={<Icon.Agent width={14}/>} 
                  title="Verifiable Delegation" 
                  description="RFC 8693 Token Exchange for cross-agent authority."
                />
                <FeatureBox 
                  group="AGENTIC FLOWS"
                  icon={<Icon.Vault width={14}/>} 
                  title="Token Vault" 
                  description="Secure 3rd-party OAuth management for autonomous bots."
                />
              </div>
            </div>

            <footer className="matrix-footer">
              <button className="btn primary" onClick={() => setStep('selection')}>
                Continue to Integration <Icon.ChevronRight width={16}/>
              </button>
            </footer>
          </div>
        )}

        {stage === 'selection' && (
          <div className="selection-stage" style={{ animation: 'slideUp 300ms ease-out' }}>
            <header>
              <button className="btn ghost sm" onClick={() => setStep('matrix')}><Icon.ChevronLeft width={12}/> Back</button>
              <h2>Choose your path</h2>
              <p>How would you like to integrate SharkAuth into your application?</p>
            </header>

            <div className="path-grid">
              <PathCard 
                primary
                icon={<Icon.Proxy width={24}/>}
                title="Proxy Gateway"
                subtitle="The Edge Enforcer"
                benefits={[
                  "Codeless authentication",
                  "Identity header injection",
                  "Hosted login/signup UI"
                ]}
                onClick={() => setStep('wizard')}
              />
              <PathCard 
                icon={<Icon.App width={24}/>}
                title="SDK / API"
                subtitle="The Identity Mesh"
                benefits={[
                  "Drop-in React components",
                  "Full REST API control",
                  "Custom branding & flows"
                ]}
                onClick={() => go('apps')}
              />
            </div>
          </div>
        )}

        {stage === 'wizard' && (
          <div className="wizard-stage" style={{ animation: 'slideUp 300ms ease-out' }}>
            <header style={{ marginBottom: 32 }}>
              <button className="btn ghost sm" onClick={() => setStep('selection')}><Icon.ChevronLeft width={12}/> Back</button>
              <h2>Setup Proxy Gateway</h2>
              <p>Sit SharkAuth in front of your upstream app to handle auth at the edge.</p>
            </header>
            <ProxyWizard autofocus onComplete={() => { markOnboarded(); go('proxy'); }}/>
          </div>
        )}

        <div className="onboarding-footer">
          <CLIFooter command="shark serve --dev --proxy-upstream http://localhost:3000"/>
        </div>

      </div>

      <style>{`
        .onboarding-overlay {
          position: fixed; inset: 0; background: var(--bg); z-index: 1000;
          display: flex; align-items: center; justify-content: center;
          overflow: auto; padding: 40px 20px;
        }
        .onboarding-container {
          max-width: 800px; width: 100%; margin: auto;
        }
        
        .hero-stage { text-align: center; }
        .hero-stage h1 { font-size: 56px; font-weight: 800; letter-spacing: -0.04em; line-height: 0.95; margin: 24px 0; font-family: var(--font-display); }
        .hero-stage p { font-size: 20px; color: var(--fg-muted); line-height: 1.5; max-width: 500px; margin: 0 auto 40px; }
        .logo-badge { display: inline-flex; align-items: center; gap: 8px; background: var(--surface-2); padding: 6px 12px; borderRadius: 20px; border: 1px solid var(--hairline); }
        .logo-badge span { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.1em; color: var(--accent); }

        .matrix-stage header, .selection-stage header { text-align: center; margin-bottom: 48px; }
        .matrix-stage h2, .selection-stage h2 { font-size: 32px; font-weight: 700; margin: 12px 0 8px; font-family: var(--font-display); }
        
        .matrix-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
        .feature-box { background: var(--surface-1); border: 1px solid var(--hairline); padding: 20px; border-radius: 12px; }
        .group-label { font-size: 9px; font-weight: 800; letter-spacing: 0.1em; color: var(--fg-faint); margin-bottom: 12px; }
        .icon-sm { width: 28px; height: 28px; background: var(--surface-3); border-radius: 6px; display: flex; alignItems: center; justifyContent: center; color: var(--accent); }
        .feature-title { font-size: 14px; font-weight: 600; }
        .feature-desc { font-size: 12px; color: var(--fg-dim); line-height: 1.5; }

        .path-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
        .path-card { 
          background: var(--surface-1); border: 1px solid var(--hairline); padding: 32px; border-radius: 16px;
          text-align: left; cursor: pointer; transition: all 150ms; position: relative;
        }
        .path-card:hover { border-color: var(--fg-muted); transform: translateY(-2px); box-shadow: 0 12px 32px rgba(0,0,0,0.15); }
        .path-card.primary { border: 2px solid var(--accent); }
        .path-card .badge { position: absolute; top: -12px; right: 24px; background: var(--accent); color: #000; font-size: 10px; font-weight: 800; padding: 2px 8px; border-radius: 4px; }
        .path-card .icon-wrap { width: 48px; height: 48px; background: var(--surface-2); border-radius: 12px; display: flex; align-items: center; justify-content: center; margin-bottom: 20px; color: var(--fg); }
        .path-card.primary .icon-wrap { background: var(--accent-dim); color: var(--accent); }
        .path-card .title { font-size: 20px; font-weight: 700; font-family: var(--font-display); }
        .path-card .subtitle { font-size: 13px; color: var(--fg-muted); margin-bottom: 20px; }
        .path-card .benefits { list-style: none; padding: 0; margin: 0; display: flex; flexDirection: column; gap: 8px; }
        .path-card .benefits li { font-size: 12.5px; color: var(--fg-dim); display: flex; align-items: center; gap: 8px; }
        .path-card .footer { margin-top: 32px; display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 600; }

        .onboarding-footer { margin-top: 80px; border-top: 1px solid var(--hairline); padding-top: 24px; }
        
        .btn.lg { height: 56px; padding: 0 32px; font-size: 16px; font-weight: 700; border-radius: 12px; }

        @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
        @keyframes slideUp { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </div>
  );
}
