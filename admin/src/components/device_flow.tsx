// @ts-nocheck
import React from 'react'
import { Icon, Avatar } from './shared'
import { MOCK } from './mock'

// Device Flow page — live pending approvals

export function DeviceFlow() {
  const [approving, setApproving] = React.useState(null);
  const [dismissed, setDismissed] = React.useState(new Set());
  const pending = MOCK.devicePending.filter(d => !dismissed.has(d.id));

  return (
    <div style={{ padding: 16, overflowY: 'auto', height: '100%' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
        <div>
          <div className="row" style={{ gap: 8 }}>
            <h2 style={{ margin: 0, fontSize: 16, fontWeight: 500, letterSpacing: '-0.01em' }}>Pending device approvals</h2>
            <span className="chip warn"><span className="dot warn pulse"/>live</span>
          </div>
          <div className="faint" style={{ fontSize: 11.5, marginTop: 2 }}>
            Polled every 5s · OAuth 2.0 device authorization grant · RFC 8628
          </div>
        </div>
        <div style={{ flex: 1 }}/>
        <span className="faint mono" style={{ fontSize: 11 }}>updated 3s ago</span>
        <button className="btn sm"><Icon.Refresh width={11} height={11}/></button>
      </div>

      {/* Pending cards */}
      {pending.length === 0 ? (
        <div style={{
          border: '1px dashed var(--hairline-strong)',
          borderRadius: 8, padding: 48, textAlign: 'center',
          background: 'var(--surface-1)',
        }}>
          <Icon.Device width={28} height={28} style={{ color: 'var(--fg-dim)', marginBottom: 8 }}/>
          <div style={{ fontSize: 14, fontWeight: 500, marginBottom: 4 }}>No pending device approvals</div>
          <div className="faint" style={{ fontSize: 12 }}>Agents using device flow will appear here. User-facing approval at <span className="mono">/oauth/device/verify</span>.</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 12, marginBottom: 24 }}>
          {pending.map(d => (
            <DeviceCard key={d.id} d={d} onApprove={() => setApproving(d.id)} onDeny={() => setDismissed(new Set([...dismissed, d.id]))}/>
          ))}
        </div>
      )}

      {/* Recent */}
      <div className="card">
        <div className="card-header">
          <span>Recent activity · 6h</span>
          <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>142 approved · 8 denied · 3 expired</span>
        </div>
        <table className="tbl">
          <thead>
            <tr>
              <th style={{ paddingLeft: 16 }}>User code</th>
              <th>Agent</th>
              <th>User</th>
              <th style={{ width: 90 }}>Outcome</th>
              <th style={{ width: 90 }}>When</th>
            </tr>
          </thead>
          <tbody>
            {MOCK.deviceRecent.map((r, i) => (
              <tr key={i}>
                <td style={{ paddingLeft: 16 }}><span className="mono" style={{ fontSize: 11 }}>{r.userCode}</span></td>
                <td><div className="row" style={{ gap: 6 }}><Avatar name={r.agent} agent size={18}/><span style={{ fontSize: 11.5 }}>{r.agent}</span></div></td>
                <td style={{ fontSize: 11.5 }}>{r.user}</td>
                <td>
                  {r.outcome === 'approved' && <span className="chip success"><Icon.Check width={9} height={9}/>approved</span>}
                  {r.outcome === 'denied' && <span className="chip danger"><Icon.X width={9} height={9}/>denied</span>}
                  {r.outcome === 'expired' && <span className="chip"><Icon.Clock width={9} height={9}/>expired</span>}
                </td>
                <td className="mono faint" style={{ fontSize: 11 }}>{MOCK.relativeTime(r.when)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {approving && <ApprovalModal d={MOCK.devicePending.find(x => x.id === approving)} onClose={() => { setApproving(null); setDismissed(new Set([...dismissed, approving])); }}/>}
    </div>
  );
}

function DeviceCard({ d, onApprove, onDeny }) {
  // Time left = Math.abs(now - expiresAt) as positive minutes (expiresAt is stored as "future" relative time)
  const now = Date.now();
  const secsLeft = Math.max(0, Math.floor((now - d.expiresAt) / 1000) * -1);
  const mm = Math.floor(secsLeft / 60);
  const ss = String(secsLeft % 60).padStart(2, '0');
  const pctLeft = Math.max(0, Math.min(1, secsLeft / 600));

  return (
    <div style={{
      border: '1px solid var(--hairline-strong)',
      borderRadius: 8,
      background: 'var(--surface-1)',
      padding: 16,
      position: 'relative',
      overflow: 'hidden',
    }}>
      {/* countdown strip */}
      <div style={{ position: 'absolute', top: 0, left: 0, right: 0, height: 2, background: 'var(--surface-3)' }}>
        <div style={{ width: `${pctLeft * 100}%`, height: '100%', background: secsLeft < 120 ? 'var(--danger)' : 'var(--warn)', transition: 'width 1s linear' }}/>
      </div>

      {/* Header row */}
      <div className="row" style={{ gap: 10, marginBottom: 12 }}>
        <Avatar name={d.agentName} agent size={32}/>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 13, fontWeight: 500 }}>{d.agentName}</div>
          <div className="faint mono" style={{ fontSize: 10.5 }}>requests access on behalf of</div>
          <div style={{ fontSize: 11.5, fontWeight: 500 }}>{d.user}</div>
        </div>
        {d.dpop && <span className="chip success"><Icon.DPoP width={9} height={9}/>DPoP</span>}
      </div>

      {/* Big user code */}
      <div style={{
        padding: 14,
        background: 'var(--surface-0)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 6,
        textAlign: 'center',
        marginBottom: 12,
      }}>
        <div className="faint" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: 4 }}>User code</div>
        <div className="mono" style={{
          fontSize: 26, fontWeight: 500, letterSpacing: '0.08em',
          fontVariantNumeric: 'tabular-nums',
        }}>{d.userCode}</div>
        <div style={{ fontSize: 10.5, color: secsLeft < 120 ? 'var(--danger)' : 'var(--fg-dim)', marginTop: 4, fontFamily: 'var(--font-mono)' }}>
          expires in {mm}:{ss}
        </div>
      </div>

      {/* Scopes */}
      <div style={{ marginBottom: 10 }}>
        <div className="faint" style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 4 }}>Requesting</div>
        <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
          {d.scopes.map(s => <span key={s} className="chip mono" style={{ height: 17, fontSize: 10 }}>{s}</span>)}
        </div>
      </div>

      {/* Meta */}
      <div className="col" style={{ gap: 3, fontSize: 10.5, marginBottom: 12 }}>
        <div className="row"><Icon.Globe width={10} height={10} style={{opacity:0.5}}/><span className="mono faint">{d.resource}</span></div>
        <div className="row"><span className="faint" style={{width:10}}>•</span><span className="mono faint">{d.ip} · {d.location}</span></div>
        <div className="row"><span className="faint" style={{width:10}}>•</span><span className="mono faint">{d.device}</span></div>
        <div className="row"><Icon.Clock width={10} height={10} style={{opacity:0.5}}/><span className="mono faint">requested {MOCK.relativeTime(d.requestedAt)}</span></div>
      </div>

      <div className="row" style={{ gap: 6 }}>
        <button className="btn primary" style={{ flex: 1 }} onClick={onApprove}>
          <Icon.Check width={12} height={12}/>Approve
        </button>
        <button className="btn danger" onClick={onDeny}>
          <Icon.X width={11} height={11}/>Deny
        </button>
      </div>
    </div>
  );
}

function ApprovalModal({ d, onClose }) {
  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50,
      backdropFilter: 'blur(4px)',
    }} onClick={onClose}>
      <div style={{
        width: 420, background: 'var(--surface-1)',
        border: '1px solid var(--hairline-bright)',
        borderRadius: 8, padding: 20,
        boxShadow: 'var(--shadow-lg)',
      }} onClick={e => e.stopPropagation()}>
        <div className="row" style={{ marginBottom: 12 }}>
          <Avatar name={d.agentName} agent size={28}/>
          <div>
            <div style={{ fontSize: 14, fontWeight: 500 }}>Approve {d.agentName}?</div>
            <div className="faint" style={{ fontSize: 11 }}>Code {d.userCode} · {d.user}</div>
          </div>
        </div>
        <div style={{ padding: 10, background: 'var(--surface-0)', borderRadius: 5, border: '1px solid var(--hairline-strong)', marginBottom: 12 }}>
          <div className="faint" style={{ fontSize: 10.5, marginBottom: 6 }}>This will grant scopes:</div>
          <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
            {d.scopes.map(s => <span key={s} className="chip mono" style={{ height: 17, fontSize: 10 }}>{s}</span>)}
          </div>
        </div>
        <div className="faint" style={{ fontSize: 11, marginBottom: 14, lineHeight: 1.5 }}>
          Approving creates a consent + issues an access token. The agent will receive the token on its next poll. You can revoke at any time from the Agent's tokens tab.
        </div>
        <div className="row" style={{ gap: 6, justifyContent: 'flex-end' }}>
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn primary" onClick={onClose}><Icon.Check width={11} height={11}/>Approve & issue</button>
        </div>
      </div>
    </div>
  );
}

