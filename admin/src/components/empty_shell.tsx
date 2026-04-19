// @ts-nocheck
import React from 'react'

function EmptyShell({ title, phase, description }) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      height: '100%', textAlign: 'center', padding: 40,
    }}>
      <div style={{ maxWidth: 420 }}>
        <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--fg)' }}>{title}</div>
        <div style={{ marginTop: 10, fontSize: 13, color: 'var(--fg-dim)', lineHeight: 1.6 }}>{description}</div>
        <div className="chip ghost" style={{ marginTop: 16, fontSize: 11 }}>Phase {phase}</div>
      </div>
    </div>
  );
}

export function Consents() { return <EmptyShell title="Consents" phase={6} description="Per-user agent access grants. Manage which agents users have authorized and their scopes." />; }
export function Tokens() { return <EmptyShell title="Tokens" phase={6} description="Active OAuth tokens across agents and users. View, filter, and revoke tokens." />; }
export function Vault() { return <EmptyShell title="Vault" phase={6} description="Third-party OAuth token management. Store and refresh tokens for Google, Slack, GitHub, and more." />; }
export function APIExplorer() { return <EmptyShell title="API Explorer" phase={5} description="In-dashboard API playground. Try endpoints, copy curl commands and SDK snippets." />; }
export function SessionDebugger() { return <EmptyShell title="Session Debugger" phase={5} description="Paste a session cookie or JWT to decode, validate, and inspect claims." />; }
export function EventSchemas() { return <EmptyShell title="Event Schemas" phase={5} description="Reference browser for webhook and audit event payload schemas." />; }
export function Proxy() { return <EmptyShell title="Proxy" phase={7} description="Reverse proxy with automatic header injection. Route requests through Shark for transparent auth." />; }
export function OIDCProvider() { return <EmptyShell title="OIDC Provider" phase={8} description="Use Shark as an OpenID Connect identity provider. Federate with other services and IdPs." />; }
export function Impersonation() { return <EmptyShell title="Impersonation" phase={9} description="Sign in as any user for debugging and support. All actions are audit-logged with admin attribution." />; }
export function CompliancePage() { return <EmptyShell title="Compliance" phase={9} description="GDPR data export, right-to-erasure, SOC2 access reviews, and session geography analysis." />; }
export function Migrations() { return <EmptyShell title="Migrations" phase={9} description="Import users from Auth0, Clerk, or Supabase with field mapping, dry-run preview, and conflict resolution." />; }
export function Branding() { return <EmptyShell title="Branding" phase={9} description="Customize sign-in components, email templates, logos, and colors for your brand." />; }
export function FlowBuilder() { return <EmptyShell title="Flow Builder" phase={10} description="Visual drag-and-drop auth flow editor. Design authentication sequences and export as YAML." />; }
