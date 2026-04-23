// @ts-nocheck
import React from 'react'
import { Icon } from './shared'

const STEPS = [
  {
    title: "Welcome to the Shark Dashboard",
    text: "This is your control plane for all things identity. Let's take a quick look around.",
    target: "body", // Full screen
  },
  {
    title: "Humans vs Agents",
    text: "SharkAuth treats agents as first-class citizens. Manage your users in HUMANS, and autonomous identities in AGENTS.",
    target: "[group='AGENTS']",
  },
  {
    title: "Unified Traffic Control",
    text: "Use the Proxy Gateway to protect existing apps with zero code changes. All routing rules are managed here.",
    target: "button[id='proxy']",
  },
  {
    title: "Identity Mesh",
    text: "Create Applications to register OAuth 2.1 clients and get your credentials for SDK integration.",
    target: "button[id='apps']",
  },
  {
    title: "Command Palette",
    text: "Press ⌘K anytime to search users, jump to pages, or execute CLI-like commands directly from the UI.",
    target: "button:has(svg[viewBox='0 0 16 16'])", // The search button
  }
];

export function Walkthrough({ onComplete }) {
  const [step, setStep] = React.useState(0);
  const current = STEPS[step];

  const next = () => {
    if (step < STEPS.length - 1) setStep(step + 1);
    else onComplete();
  };

  const skip = () => onComplete();

  // Position logic (very simple for now)
  const style = {
    position: 'fixed',
    bottom: 40,
    right: 40,
    width: 320,
    background: 'var(--surface-1)',
    border: '1px solid var(--accent)',
    borderRadius: 12,
    boxShadow: '0 20px 80px rgba(0,0,0,0.5), 0 0 0 100vmax rgba(0,0,0,0.4)',
    padding: 24,
    zIndex: 1000,
    animation: 'slideUp 260ms cubic-bezier(0.16, 1, 0.3, 1)',
  };

  React.useEffect(() => {
    if (!current.target || current.target === 'body') return;
    const el = document.querySelector(current.target);
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      el.style.outline = '2px solid var(--accent)';
      el.style.outlineOffset = '4px';
      el.style.transition = 'outline 200ms';
      return () => {
        el.style.outline = '';
        el.style.outlineOffset = '';
      };
    }
  }, [step]);

  return (
    <div style={style}>
      <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
        <span className="chip agent" style={{ height: 18, fontSize: 10 }}>Walkthrough</span>
        <button onClick={skip} className="btn ghost icon sm"><Icon.X width={12}/></button>
      </div>
      
      <div style={{ fontSize: 15, fontWeight: 700, marginBottom: 8, fontFamily: 'var(--font-display)' }}>
        {current.title}
      </div>
      <div style={{ fontSize: 12.5, color: 'var(--fg-dim)', lineHeight: 1.5, marginBottom: 20 }}>
        {current.text}
      </div>

      <div className="row" style={{ justifyContent: 'space-between', alignItems: 'center' }}>
        <div className="faint" style={{ fontSize: 11 }}>
          {step + 1} of {STEPS.length}
        </div>
        <div className="row" style={{ gap: 8 }}>
          <button className="btn ghost sm" onClick={skip}>Skip</button>
          <button className="btn primary sm" onClick={next}>
            {step === STEPS.length - 1 ? 'Finish' : 'Next'} <Icon.ChevronRight width={12}/>
          </button>
        </div>
      </div>

      <style>{`
        @keyframes slideUp { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </div>
  );
}
