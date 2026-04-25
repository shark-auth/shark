// @ts-nocheck
import React from 'react'

// Coming-soon placeholder for surfaces not yet ready for v0.9.0.
// Distinct from "blank state" — these routes exist in the sidebar but
// the feature is on the post-launch roadmap. Show what it WILL do, not
// a phase number that leaks internal sequencing.
function ComingSoon({ title, description, eta }) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      height: '100%', textAlign: 'center', padding: 40,
    }}>
      <div style={{ maxWidth: 460, border: '1px solid var(--hairline-strong)', padding: '28px 32px', background: 'var(--surface-1)' }}>
        <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.12em', color: 'var(--fg-muted)', fontWeight: 600 }}>Coming soon</div>
        <div style={{ fontSize: 17, fontWeight: 600, color: 'var(--fg)', marginTop: 10 }}>{title}</div>
        <div style={{ marginTop: 12, fontSize: 13, color: 'var(--fg-dim)', lineHeight: 1.6 }}>{description}</div>
        {eta && <div className="faint mono" style={{ marginTop: 16, fontSize: 11 }}>{eta}</div>}
      </div>
    </div>
  );
}

export function Tokens() { return <ComingSoon title="Tokens" description="Active OAuth tokens across agents and users. View, filter, and revoke tokens individually or in bulk." />; }
export function APIExplorer() { return <ComingSoon title="API Explorer" description="In-dashboard API playground. Try endpoints, copy curl commands and SDK snippets." />; }
export function EventSchemas() { return <ComingSoon title="Event Schemas" description="Reference browser for webhook and audit event payload schemas." />; }
export function SessionDebugger() { return <ComingSoon title="Session Debugger" description="Inspect live sessions, decode JWTs, and trace token chains across delegations." />; }
export function OIDCProvider() { return <ComingSoon title="OIDC Provider" description="Use Shark as an OpenID Connect identity provider. Federate with other services and IdPs." />; }
export function Impersonation() { return <ComingSoon title="Impersonation" description="Sign in as any user for debugging and support. All actions are audit-logged with admin attribution." />; }
export function Migrations() { return <ComingSoon title="Migrations" description="Import users from Auth0, Clerk, or Supabase with field mapping, dry-run preview, and conflict resolution." />; }
