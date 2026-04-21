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

export function Tokens() { return <EmptyShell title="Tokens" phase={6} description="Active OAuth tokens across agents and users. View, filter, and revoke tokens." />; }
export function APIExplorer() { return <EmptyShell title="API Explorer" phase={5} description="In-dashboard API playground. Try endpoints, copy curl commands and SDK snippets." />; }
export function EventSchemas() { return <EmptyShell title="Event Schemas" phase={5} description="Reference browser for webhook and audit event payload schemas." />; }
export function OIDCProvider() { return <EmptyShell title="OIDC Provider" phase={8} description="Use Shark as an OpenID Connect identity provider. Federate with other services and IdPs." />; }
export function Impersonation() { return <EmptyShell title="Impersonation" phase={9} description="Sign in as any user for debugging and support. All actions are audit-logged with admin attribution." />; }
export function Migrations() { return <EmptyShell title="Migrations" phase={9} description="Import users from Auth0, Clerk, or Supabase with field mapping, dry-run preview, and conflict resolution." />; }
