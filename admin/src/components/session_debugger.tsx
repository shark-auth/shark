// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { CLIFooter } from './CLIFooter'

// Session Debugger — pure client-side JWT decode + JWKS signature validation.
// No backend beyond /.well-known/jwks.json. Paste a JWT (or session cookie value)
// and see header, payload, expiry status, and signature validity.

function b64urlDecode(s) {
  s = s.replace(/-/g, '+').replace(/_/g, '/');
  while (s.length % 4) s += '=';
  return atob(s);
}

function b64urlToBytes(s) {
  const bin = b64urlDecode(s);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function parseJwt(raw) {
  const parts = raw.trim().split('.');
  if (parts.length !== 3) throw new Error('Not a JWT (expected 3 dot-separated parts)');
  const header = JSON.parse(b64urlDecode(parts[0]));
  const payload = JSON.parse(b64urlDecode(parts[1]));
  return { header, payload, raw, parts };
}

async function importJwk(jwk, alg) {
  const algoMap = {
    RS256: { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-256' },
    RS384: { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-384' },
    RS512: { name: 'RSASSA-PKCS1-v1_5', hash: 'SHA-512' },
    ES256: { name: 'ECDSA', namedCurve: 'P-256', hash: 'SHA-256' },
    ES384: { name: 'ECDSA', namedCurve: 'P-384', hash: 'SHA-384' },
  };
  const cfg = algoMap[alg];
  if (!cfg) throw new Error(`Unsupported alg ${alg}`);
  return crypto.subtle.importKey('jwk', jwk, cfg, false, ['verify']);
}

async function verifyJwt(token, jwks) {
  const { header, parts } = parseJwt(token);
  if (!header.kid) throw new Error('Token header missing kid');
  const jwk = (jwks?.keys || []).find(k => k.kid === header.kid);
  if (!jwk) throw new Error(`No matching JWK for kid=${header.kid}`);
  const key = await importJwk(jwk, header.alg);
  const signed = new TextEncoder().encode(parts[0] + '.' + parts[1]);
  const sig = b64urlToBytes(parts[2]);
  const algo = header.alg.startsWith('RS')
    ? { name: 'RSASSA-PKCS1-v1_5' }
    : { name: 'ECDSA', hash: { name: 'SHA-' + header.alg.slice(2) } };
  return crypto.subtle.verify(algo, key, sig, signed);
}

export function SessionDebugger() {
  const [input, setInput] = React.useState('');
  const [jwks, setJwks] = React.useState(null);
  const [jwksErr, setJwksErr] = React.useState(null);
  const [sigValid, setSigValid] = React.useState(null);
  const [sigErr, setSigErr] = React.useState(null);
  const [verifying, setVerifying] = React.useState(false);

  React.useEffect(() => {
    fetch('/.well-known/jwks.json')
      .then(r => r.ok ? r.json() : Promise.reject(new Error('HTTP ' + r.status)))
      .then(setJwks)
      .catch(e => setJwksErr(e.message || 'fetch failed'));
  }, []);

  let parsed = null;
  let parseErr = null;
  if (input.trim()) {
    try { parsed = parseJwt(input); } catch (e) { parseErr = e.message; }
  }

  // Auto-verify when jwks + parsed token available
  React.useEffect(() => {
    if (!parsed || !jwks) { setSigValid(null); setSigErr(null); return; }
    setVerifying(true);
    setSigErr(null);
    verifyJwt(input, jwks)
      .then(v => { setSigValid(v); setVerifying(false); })
      .catch(e => { setSigErr(e.message); setSigValid(false); setVerifying(false); });
  }, [input, jwks]);

  const now = Math.floor(Date.now() / 1000);
  const claims = parsed?.payload || {};
  const expStatus = claims.exp != null
    ? (claims.exp < now ? 'expired' : 'valid')
    : 'no-exp';
  const nbfStatus = claims.nbf != null
    ? (claims.nbf > now ? 'not-yet-valid' : 'valid')
    : 'no-nbf';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)', flexShrink: 0 }}>
        <div className="row" style={{ gap: 8, alignItems: 'center' }}>
          <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Session Debugger</h1>
          <span className="chip mono" style={{ height: 18, fontSize: 10, padding: '0 6px' }}>JWT</span>
        </div>
        <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
          Paste a JWT or session cookie value to decode, validate signature against JWKS, and inspect claims. All decoding happens in your browser.
        </p>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 20, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        {/* Input */}
        <div className="col" style={{ gap: 10, gridColumn: '1 / -1' }}>
          <label style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Token</label>
          <textarea
            value={input}
            onChange={e => setInput(e.target.value)}
            placeholder="eyJhbGciOi..."
            spellCheck={false}
            style={{
              width: '100%', minHeight: 84, padding: 10,
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 5,
              color: 'var(--fg)',
              fontFamily: 'var(--font-mono)',
              fontSize: 12, lineHeight: 1.5,
              resize: 'vertical',
              outline: 'none',
            }}
          />
          {parseErr && <span style={{ color: 'var(--danger)', fontSize: 12 }}>{parseErr}</span>}
          {jwksErr && <span style={{ color: 'var(--warn)', fontSize: 12 }}>JWKS fetch failed: {jwksErr} (signature cannot be verified)</span>}
        </div>

        {parsed && (
          <>
            {/* Summary card */}
            <div className="card" style={{ gridColumn: '1 / -1', padding: 14 }}>
              <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 8 }}>Status</div>
              <div className="row" style={{ gap: 24, flexWrap: 'wrap' }}>
                <StatusPill
                  label="Signature"
                  state={verifying ? 'pending' : sigValid === true ? 'ok' : sigValid === false ? 'bad' : 'unknown'}
                  text={verifying ? 'verifying…' : sigValid === true ? 'valid' : sigValid === false ? (sigErr || 'invalid') : '—'}
                />
                <StatusPill
                  label="Expiry"
                  state={expStatus === 'valid' ? 'ok' : expStatus === 'expired' ? 'bad' : 'unknown'}
                  text={
                    expStatus === 'valid' ? `expires in ${humanDelta(claims.exp - now)}`
                    : expStatus === 'expired' ? `expired ${humanDelta(now - claims.exp)} ago`
                    : 'no exp claim'
                  }
                />
                <StatusPill
                  label="Not before"
                  state={nbfStatus === 'valid' ? 'ok' : nbfStatus === 'not-yet-valid' ? 'bad' : 'unknown'}
                  text={
                    nbfStatus === 'valid' ? 'active'
                    : nbfStatus === 'not-yet-valid' ? `activates in ${humanDelta(claims.nbf - now)}`
                    : 'no nbf claim'
                  }
                />
                <StatusPill
                  label="Algorithm"
                  state="info"
                  text={parsed.header.alg || '—'}
                />
                <StatusPill
                  label="Key ID"
                  state="info"
                  text={parsed.header.kid || '—'}
                />
              </div>
            </div>

            {/* Header */}
            <div className="card" style={{ padding: 14 }}>
              <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
                <span style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Header</span>
                <CopyButton text={JSON.stringify(parsed.header, null, 2)}/>
              </div>
              <pre style={prePaneStyle}>{JSON.stringify(parsed.header, null, 2)}</pre>
            </div>

            {/* Payload */}
            <div className="card" style={{ padding: 14 }}>
              <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
                <span style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Payload</span>
                <CopyButton text={JSON.stringify(parsed.payload, null, 2)}/>
              </div>
              <pre style={prePaneStyle}>{annotateClaims(parsed.payload)}</pre>
            </div>
          </>
        )}

        {!parsed && !parseErr && (
          <div className="faint" style={{ gridColumn: '1 / -1', textAlign: 'center', padding: 40, fontSize: 13 }}>
            Paste a JWT above to decode. Nothing is sent to the server; validation runs in your browser.
          </div>
        )}
      </div>

      <CLIFooter command="shark debug decode-jwt <token>"/>
    </div>
  );
}

function StatusPill({ label, state, text }) {
  const color = state === 'ok' ? 'var(--success)'
             : state === 'bad' ? 'var(--danger)'
             : state === 'pending' ? 'var(--warn)'
             : 'var(--fg-muted)';
  return (
    <div className="col" style={{ gap: 2 }}>
      <span style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>{label}</span>
      <span className="row" style={{ gap: 6, alignItems: 'center' }}>
        {state !== 'info' && <span className="dot" style={{ background: color, width: 7, height: 7 }}/>}
        <span style={{ fontSize: 12.5, fontWeight: 500, color: color, fontFamily: state === 'info' ? 'var(--font-mono)' : undefined }}>{text}</span>
      </span>
    </div>
  );
}

function CopyButton({ text }) {
  const [copied, setCopied] = React.useState(false);
  return (
    <button
      className="btn ghost icon sm"
      onClick={() => { navigator.clipboard?.writeText(text); setCopied(true); setTimeout(() => setCopied(false), 900); }}
      title="Copy JSON"
    >
      {copied ? <Icon.Check width={10} height={10} style={{ color: 'var(--success)' }}/> : <Icon.Copy width={10} height={10}/>}
    </button>
  );
}

function annotateClaims(payload) {
  // Add human-readable timestamp comments inline for exp/nbf/iat
  const out = {};
  for (const [k, v] of Object.entries(payload)) {
    out[k] = v;
  }
  let json = JSON.stringify(out, null, 2);
  for (const key of ['exp', 'nbf', 'iat', 'auth_time']) {
    if (typeof payload[key] === 'number') {
      const iso = new Date(payload[key] * 1000).toISOString();
      const re = new RegExp(`("${key}": ${payload[key]})`);
      json = json.replace(re, `$1  // ${iso}`);
    }
  }
  return json;
}

function humanDelta(secs) {
  secs = Math.abs(Math.floor(secs));
  if (secs < 60) return `${secs}s`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m`;
  if (secs < 86400) return `${Math.floor(secs / 3600)}h`;
  return `${Math.floor(secs / 86400)}d`;
}

const prePaneStyle = {
  margin: 0,
  padding: 10,
  fontFamily: 'var(--font-mono)',
  fontSize: 12,
  lineHeight: 1.6,
  background: 'var(--surface-1)',
  border: '1px solid var(--hairline)',
  borderRadius: 4,
  overflow: 'auto',
  maxHeight: 380,
  whiteSpace: 'pre',
};
