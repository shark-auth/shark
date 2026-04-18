// Empty state shell for future phase pages

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

function Consents() { return <EmptyShell title="Consents" phase={6} description="Per-user agent access grants. Manage which agents users have authorized and their scopes." />; }
function Tokens() { return <EmptyShell title="Tokens" phase={6} description="Active OAuth tokens across agents and users. View, filter, and revoke tokens." />; }
function Vault() { return <EmptyShell title="Vault" phase={6} description="Third-party OAuth token management. Store and refresh tokens for Google, Slack, GitHub, and more." />; }
function APIExplorer() { return <EmptyShell title="API Explorer" phase={5} description="In-dashboard API playground. Try endpoints, copy curl commands and SDK snippets." />; }
function SessionDebugger() { return <EmptyShell title="Session Debugger" phase={5} description="Paste a session cookie or JWT to decode, validate, and inspect claims." />; }
function EventSchemas() { return <EmptyShell title="Event Schemas" phase={5} description="Reference browser for webhook and audit event payload schemas." />; }
