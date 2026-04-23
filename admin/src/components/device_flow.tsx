// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { API, useAPI } from './api'

// Device Flow page — user-code entry that hands off to the server-rendered
// approval page at /oauth/device/verify (session-authenticated).
//
// Backend reality: the approve/deny flow is implemented as server-rendered
// HTML (see internal/oauth/device.go). There is no JSON API for verify and
// no "list pending" admin endpoint, so this page is intentionally minimal:
// a code entry form that opens the server page in a new tab.

const ALLOWED_CODE_CHARS = /[A-HJ-NP-Z2-9]/; // RFC 8628 §6.1 base32-ish (no I/O/0/1)

function normalizeUserCode(raw) {
  const up = (raw || '').toUpperCase();
  let out = '';
  for (const ch of up) {
    if (ALLOWED_CODE_CHARS.test(ch)) {
      out += ch;
      if (out.length === 8) break;
    }
  }
  if (out.length > 4) out = out.slice(0, 4) + '-' + out.slice(4);
  return out;
}

function isCompleteCode(code) {
  return /^[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}$/.test(code || '');
}

export function DeviceFlow() {
  const toast = useToast();
  const [userCode, setUserCode] = React.useState('');
  const inputRef = React.useRef(null);

  const valid = isCompleteCode(userCode);

  const handleChange = (e) => {
    setUserCode(normalizeUserCode(e.target.value));
  };

  const handleApprove = () => {
    if (!valid) {
      toast?.warn?.('Enter all 8 characters (format XXXX-XXXX).');
      inputRef.current?.focus();
      return;
    }
    const url = `/oauth/device/verify?user_code=${encodeURIComponent(userCode)}`;
    window.open(url, '_blank', 'noopener');
    toast?.info?.('Opening approval page — confirm agent details in the new tab.');
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && valid) {
      e.preventDefault();
      handleApprove();
    }
  };

  const initialCurl = `curl -s -X POST ${location.origin}/oauth/device \\
  -H "Content-Type: application/x-www-form-urlencoded" \\
  -d "client_id=YOUR_CLIENT_ID&scope=openid profile"`;

  const pollCurl = `curl -s -X POST ${location.origin}/oauth/token \\
  -H "Content-Type: application/x-www-form-urlencoded" \\
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \\
  -d "device_code=DEVICE_CODE_FROM_STEP_1" \\
  -d "client_id=YOUR_CLIENT_ID"`;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)', flexShrink: 0 }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <div className="row" style={{ gap: 8, alignItems: 'center' }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Device flow</h1>
              <span className="chip mono" style={{ height: 18, fontSize: 10, padding: '0 6px' }}>
                POST /oauth/device
              </span>
            </div>
            <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
              OAuth 2.0 Device Authorization Grant · RFC 8628
            </p>
          </div>
        </div>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 20 }}>
        <div style={{ maxWidth: 720, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* Primary card — Approve a device code */}
          <div className="card">
            <div className="card-header">
              <span>Approve a device code</span>
              <span className="faint" style={{ fontSize: 10.5, textTransform: 'none', letterSpacing: 0 }}>
                session-authenticated handoff
              </span>
            </div>
            <div style={{ padding: 20 }}>
              <p className="faint" style={{ margin: 0, fontSize: 12, lineHeight: 1.55 }}>
                Enter the code shown on your device to approve OAuth access. You'll be redirected
                to a secure page to confirm the agent, scopes, and identity before issuing a token.
              </p>

              <div style={{ marginTop: 18 }}>
                <label
                  htmlFor="user-code-input"
                  className="faint"
                  style={{
                    display: 'block',
                    fontSize: 10,
                    textTransform: 'uppercase',
                    letterSpacing: '0.12em',
                    marginBottom: 6,
                  }}
                >
                  User code
                </label>
                <input
                  id="user-code-input"
                  ref={inputRef}
                  value={userCode}
                  onChange={handleChange}
                  onKeyDown={handleKeyDown}
                  placeholder="ABCD-EFGH"
                  autoComplete="off"
                  autoCapitalize="characters"
                  spellCheck={false}
                  maxLength={9}
                  aria-label="OAuth device user code"
                  style={{
                    width: '100%',
                    padding: '14px 16px',
                    fontSize: 22,
                    fontFamily: 'var(--font-mono)',
                    fontVariantNumeric: 'tabular-nums',
                    letterSpacing: '0.14em',
                    textAlign: 'center',
                    textTransform: 'uppercase',
                    background: 'var(--surface-0)',
                    border: '1px solid ' + (valid ? 'var(--success)' : 'var(--hairline-strong)'),
                    borderRadius: 6,
                    color: 'var(--fg)',
                    outline: 'none',
                    boxSizing: 'border-box',
                    transition: 'border-color 120ms ease',
                  }}
                />
                <div
                  className="faint"
                  style={{
                    marginTop: 6,
                    fontSize: 10.5,
                    fontFamily: 'var(--font-mono)',
                    letterSpacing: '0.02em',
                  }}
                >
                  Format XXXX-XXXX · uppercase letters (no I/O) and digits 2-9 · {userCode.replace('-', '').length}/8
                </div>
              </div>

              <div className="row" style={{ gap: 8, marginTop: 18 }}>
                <button
                  className="btn primary"
                  onClick={handleApprove}
                  disabled={!valid}
                  style={{ minWidth: 160 }}
                >
                  <Icon.External width={11} height={11}/>
                  Approve device
                </button>
                <button
                  className="btn ghost"
                  onClick={() => { setUserCode(''); inputRef.current?.focus(); }}
                  disabled={!userCode}
                >
                  Clear
                </button>
              </div>

              <p className="faint" style={{ margin: '14px 0 0', fontSize: 11, lineHeight: 1.5 }}>
                You'll be redirected to a secure page to confirm agent details and scopes.
                If you're not signed in, you'll be prompted to sign in first.
              </p>
            </div>
          </div>

          {/* Admin pending queue */}
          <PendingDeviceQueue/>

          {/* Secondary card — How device flow works */}
          <div className="card">
            <div className="card-header">
              <span>How device flow works</span>
              <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>
                RFC 8628
              </span>
            </div>
            <div style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 14 }}>
              <Step
                n={1}
                title="Client requests authorization"
                body={
                  <>Client calls <span className="mono">POST /oauth/device</span> with its
                  <span className="mono"> client_id</span> and desired scopes. The server returns a
                  <span className="mono"> device_code</span>, a short <span className="mono">user_code</span>,
                  and a <span className="mono">verification_uri</span>.</>
                }
                code={initialCurl}
              />
              <Step
                n={2}
                title="User approves on a trusted device"
                body={
                  <>The user visits the <span className="mono">verification_uri</span> (this page or
                  <span className="mono"> /oauth/device/verify</span>) and enters the user code.
                  After signing in, they confirm the agent identity and requested scopes.</>
                }
              />
              <Step
                n={3}
                title="Client polls for the token"
                body={
                  <>The client polls <span className="mono">POST /oauth/token</span> with
                  <span className="mono"> grant_type=urn:ietf:params:oauth:grant-type:device_code</span>.
                  Responses are <span className="mono">authorization_pending</span> until approval, then
                  a normal access token is issued.</>
                }
                code={pollCurl}
              />
            </div>
          </div>
        </div>
      </div>

      {/* CLI footer */}
      <CLIFooter
        command={`curl -s -X POST ${location.origin}/oauth/device -d "client_id=YOUR_CLIENT_ID&scope=openid profile"`}
      />
    </div>
  );
}

function Step({ n, title, body, code }) {
  return (
    <div className="row" style={{ alignItems: 'flex-start', gap: 12 }}>
      <div
        style={{
          flexShrink: 0,
          width: 22,
          height: 22,
          borderRadius: '50%',
          background: 'var(--surface-2)',
          border: '1px solid var(--hairline-strong)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 11,
          fontWeight: 600,
          fontFamily: 'var(--font-mono)',
          color: 'var(--fg-muted)',
        }}
      >
        {n}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 12.5, fontWeight: 500, marginBottom: 3 }}>{title}</div>
        <div className="faint" style={{ fontSize: 11.5, lineHeight: 1.55 }}>{body}</div>
        {code && (
          <pre
            style={{
              margin: '8px 0 0',
              padding: '10px 12px',
              background: 'var(--surface-0)',
              border: '1px solid var(--hairline)',
              borderRadius: 5,
              fontSize: 10.5,
              fontFamily: 'var(--font-mono)',
              color: 'var(--fg-muted)',
              overflowX: 'auto',
              whiteSpace: 'pre',
              lineHeight: 1.5,
            }}
          >
            {code}
          </pre>
        )}
      </div>
    </div>
  );
}

// PendingDeviceQueue — admin override surface. Lists pending device codes,
// 5s polling, with approve/deny actions. Bypasses the user-facing verify
// flow when the user can't reach the verify URL themselves.
function PendingDeviceQueue() {
  const toast = useToast();
  const { data, loading, error, refresh } = useAPI('/admin/oauth/device-codes');

  React.useEffect(() => {
    const id = setInterval(refresh, 5000);
    return () => clearInterval(id);
  }, [refresh]);

  const pending = data?.data || [];

  const decide = async (userCode, decision) => {
    try {
      await API.post(`/admin/oauth/device-codes/${encodeURIComponent(userCode)}/${decision}`);
      toast?.success?.(`${userCode} ${decision}`);
      refresh();
    } catch (e) {
      toast?.error?.(e.message || `${decision} failed`);
    }
  };

  return (
    <div className="card">
      <div className="card-header">
        <span>Pending device codes</span>
        <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>
          5s poll · admin override
        </span>
      </div>
      <div style={{ padding: 12 }}>
        {loading && pending.length === 0 ? (
          <div className="faint" style={{ padding: 16, fontSize: 12, textAlign: 'center' }}>Loading…</div>
        ) : error ? (
          <div style={{ padding: 12, color: 'var(--danger)', fontSize: 12 }}>Failed to load: {error}</div>
        ) : pending.length === 0 ? (
          <div className="faint" style={{ padding: 16, fontSize: 12, textAlign: 'center' }}>
            No pending device codes.
          </div>
        ) : (
          <table className="tbl" style={{ fontSize: 12 }}>
            <thead>
              <tr>
                <th style={{ textAlign: 'left' }}>User code</th>
                <th style={{ textAlign: 'left' }}>Agent</th>
                <th style={{ textAlign: 'left' }}>Scope</th>
                <th style={{ textAlign: 'left' }}>Expires</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {pending.map(dc => (
                <tr key={dc.user_code}>
                  <td className="mono" style={{ fontWeight: 600, fontSize: 13, letterSpacing: '0.08em' }}>{dc.user_code}</td>
                  <td>{dc.agent_name || <span className="mono faint" style={{ fontSize: 10.5 }}>{dc.client_id}</span>}</td>
                  <td className="mono faint" style={{ fontSize: 10.5 }}>{dc.scope || '—'}</td>
                  <td className="mono faint" style={{ fontSize: 10.5 }}>
                    {new Date(dc.expires_at).toLocaleTimeString()}
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <button className="btn primary sm" style={{ marginRight: 4 }}
                            onClick={() => decide(dc.user_code, 'approve')}>Approve</button>
                    <button className="btn ghost sm danger"
                            onClick={() => decide(dc.user_code, 'deny')}>Deny</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
