// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Compliance page — audit export (wired) + GDPR/deletion stubs (CLI-backed for now).
// Demoted from Phase 9 stub: ships real audit CSV export against existing backend,
// surfaces CLI commands for GDPR operations pending full UI.

export function CompliancePage() {
  const [tab, setTab] = React.useState('audit');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)', flexShrink: 0 }}>
        <div className="row" style={{ gap: 8, alignItems: 'center' }}>
          <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Compliance</h1>
          <span className="chip mono" style={{ height: 18, fontSize: 10, padding: '0 6px' }}>SOC2 · GDPR</span>
        </div>
        <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
          Audit log export, access reviews, right-to-erasure workflows.
        </p>

        <div className="row" style={{ gap: 4, marginTop: 12 }}>
          <TabBtn active={tab === 'audit'}    onClick={() => setTab('audit')}>Audit Export</TabBtn>
          <TabBtn active={tab === 'gdpr'}     onClick={() => setTab('gdpr')}>Right to Erasure</TabBtn>
          <TabBtn active={tab === 'access'}   onClick={() => setTab('access')}>Access Review</TabBtn>
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
        {tab === 'audit' && <AuditExport/>}
        {tab === 'gdpr' && <GDPRStub/>}
        {tab === 'access' && <AccessReviewStub/>}
      </div>

      <CLIFooter command="shark audit export --format csv --since 2026-01-01"/>
    </div>
  );
}

function TabBtn({ active, onClick, children }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '6px 12px', fontSize: 12, fontWeight: 500,
        background: active ? 'var(--surface-3)' : 'transparent',
        color: active ? 'var(--fg)' : 'var(--fg-muted)',
        border: 0, borderRadius: 4, cursor: 'pointer',
      }}
    >{children}</button>
  );
}

function AuditExport() {
  const toast = useToast();
  const [since, setSince] = React.useState(() => {
    const d = new Date();
    d.setDate(d.getDate() - 30);
    return d.toISOString().slice(0, 10);
  });
  const [until, setUntil] = React.useState(() => new Date().toISOString().slice(0, 10));
  const [loading, setLoading] = React.useState(false);

  const presets = [
    { label: 'Last 7 days', days: 7 },
    { label: 'Last 30 days', days: 30 },
    { label: 'Last 90 days', days: 90 },
    { label: 'Year to date', days: null, ytd: true },
  ];

  const applyPreset = (p) => {
    const now = new Date();
    setUntil(now.toISOString().slice(0, 10));
    if (p.ytd) {
      setSince(`${now.getFullYear()}-01-01`);
    } else {
      const d = new Date();
      d.setDate(d.getDate() - p.days);
      setSince(d.toISOString().slice(0, 10));
    }
  };

  const download = async () => {
    setLoading(true);
    try {
      const key = localStorage.getItem('shark_admin_key');
      const fromISO = new Date(since + 'T00:00:00Z').toISOString();
      const toISO = new Date(until + 'T23:59:59Z').toISOString();
      const res = await fetch(`/api/v1/audit-logs/export`, {
        method: 'POST',
        headers: {
          'Authorization': 'Bearer ' + key,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ from: fromISO, to: toISO }),
      });
      if (!res.ok) throw new Error('HTTP ' + res.status);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit-${since}-to-${until}.csv`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      toast?.success(`Exported ${since} to ${until}`);
    } catch (e) {
      toast?.error('Export failed: ' + (e?.message || 'unknown'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="col" style={{ gap: 16, maxWidth: 640 }}>
      <div className="card" style={{ padding: 16 }}>
        <h2 style={{ fontSize: 14, margin: '0 0 4px', fontWeight: 600 }}>Export audit logs</h2>
        <p className="faint" style={{ margin: '0 0 12px', fontSize: 12 }}>
          Download a full audit log for compliance review (SOC2, access review, incident post-mortem).
        </p>

        <div className="row" style={{ gap: 6, marginBottom: 12, flexWrap: 'wrap' }}>
          {presets.map(p => (
            <button key={p.label} className="btn ghost sm" onClick={() => applyPreset(p)}>{p.label}</button>
          ))}
        </div>

        <div className="col" style={{ gap: 10 }}>
          <div className="row" style={{ gap: 10 }}>
            <DateField label="From" value={since} onChange={setSince}/>
            <DateField label="To" value={until} onChange={setUntil}/>
          </div>
          <button className="btn primary" onClick={download} disabled={loading} style={{ width: 'fit-content' }}>
            <Icon.Audit width={12} height={12}/>
            {loading ? 'Exporting…' : 'Download CSV export'}
          </button>
        </div>
      </div>
    </div>
  );
}

function GDPRStub() {
  return (
    <div className="col" style={{ gap: 16, maxWidth: 640 }}>
      <div className="card" style={{ padding: 16 }}>
        <h2 style={{ fontSize: 14, margin: '0 0 4px', fontWeight: 600 }}>Right to Erasure (GDPR Article 17)</h2>
        <p className="faint" style={{ margin: '0 0 12px', fontSize: 12, lineHeight: 1.6 }}>
          Full UI wizard ships with Phase 9 Migrations. For now, run the CLI to delete a user and all associated data.
          Deletion is cascading: sessions, consents, org memberships, API keys, webhooks owned by user.
        </p>
        <CopyField value="shark users delete --email user@example.com --purge"/>
        <p className="faint" style={{ margin: '12px 0 0', fontSize: 11 }}>
          Use <code>--dry-run</code> first to preview cascading deletes. Audit event: <code>admin.user.purge</code>.
        </p>
      </div>

      <div className="card" style={{ padding: 16 }}>
        <h2 style={{ fontSize: 14, margin: '0 0 4px', fontWeight: 600 }}>Data Export (GDPR Article 15)</h2>
        <p className="faint" style={{ margin: '0 0 12px', fontSize: 12, lineHeight: 1.6 }}>
          Export all data associated with a user for portability/access request.
        </p>
        <CopyField value="shark users export --email user@example.com --format json"/>
      </div>
    </div>
  );
}

function AccessReviewStub() {
  const { data: usersData } = useAPI('/users?per_page=50&mfa_enabled=false');
  const { data: adminStats } = useAPI('/admin/stats');

  const nonMfaCount = usersData?.total ?? 0;
  const failedLogins = adminStats?.failed_logins_24h ?? 0;

  return (
    <div className="col" style={{ gap: 16, maxWidth: 640 }}>
      <div className="card" style={{ padding: 16 }}>
        <h2 style={{ fontSize: 14, margin: '0 0 4px', fontWeight: 600 }}>Access Review Snapshot</h2>
        <p className="faint" style={{ margin: '0 0 12px', fontSize: 12 }}>Quick signals for SOC2/ISO27001 quarterly review.</p>

        <div className="col" style={{ gap: 8 }}>
          <ReviewRow label="Users without MFA" value={nonMfaCount} severity={nonMfaCount > 0 ? 'warn' : 'ok'}/>
          <ReviewRow label="Failed logins (24h)" value={failedLogins} severity={failedLogins > 10 ? 'warn' : 'ok'}/>
        </div>

        <p className="faint" style={{ margin: '16px 0 8px', fontSize: 11 }}>
          For full RBAC audit (which users hold which permissions), see <strong>Roles & Permissions</strong> page.
        </p>
      </div>
    </div>
  );
}

function ReviewRow({ label, value, severity }) {
  const color = severity === 'warn' ? 'var(--warn)' : 'var(--success)';
  return (
    <div className="row" style={{ gap: 8, padding: '8px 0', borderBottom: '1px solid var(--hairline)' }}>
      <span className="dot" style={{ background: color, width: 6, height: 6 }}/>
      <span style={{ flex: 1, fontSize: 12.5 }}>{label}</span>
      <span className="mono" style={{ fontSize: 13, fontWeight: 600, color }}>{value}</span>
    </div>
  );
}

function DateField({ label, value, onChange }) {
  return (
    <div className="col" style={{ gap: 4, flex: 1 }}>
      <label style={fieldLabel}>{label}</label>
      <input
        type="date"
        value={value}
        onChange={e => onChange(e.target.value)}
        style={inputStyle}
      />
    </div>
  );
}

const fieldLabel = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' };
const inputStyle = {
  height: 30, padding: '0 8px',
  background: 'var(--surface-1)',
  border: '1px solid var(--hairline-strong)',
  borderRadius: 4,
  color: 'var(--fg)',
  fontSize: 12,
  fontFamily: 'var(--font-mono)',
  outline: 'none',
};
