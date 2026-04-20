// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API } from './api'
import { CLIFooter } from './CLIFooter'
import { usePageActions } from './useKeyboardShortcuts'
import { TeachEmptyState } from './TeachEmptyState'

// Signing Keys page — JWKS / JWT signing key management

const skThStyle = {
  textAlign: 'left',
  padding: '8px 14px',
  fontSize: 10,
  fontWeight: 500,
  color: 'var(--fg-dim)',
  borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-0)',
  position: 'sticky',
  top: 0,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
};
const skTdStyle = {
  padding: '10px 14px',
  borderBottom: '1px solid var(--hairline)',
  verticalAlign: 'middle',
};

export function SigningKeys() {
  const [keys, setKeys] = React.useState([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);
  const [refreshTick, setRefreshTick] = React.useState(0);
  const [rotateTooltip, setRotateTooltip] = React.useState(null);

  const jwksUrl = window.location.origin + '/.well-known/jwks.json';

  React.useEffect(() => {
    setLoading(true);
    setError(null);
    fetch('/.well-known/jwks.json')
      .then(r => {
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.json();
      })
      .then(data => {
        setKeys(data.keys || []);
        setLoading(false);
      })
      .catch(e => {
        setError(e.message || 'Failed to load keys');
        setLoading(false);
      });
  }, [refreshTick]);

  const refresh = () => setRefreshTick(t => t + 1);

  usePageActions({ onRefresh: refresh });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)', flexShrink: 0 }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <div className="row" style={{ gap: 8, alignItems: 'center' }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Signing Keys</h1>
              <span className="chip mono" style={{ height: 18, fontSize: 10, padding: '0 6px' }}>JWKS</span>
            </div>
            <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
              JSON Web Key Set — public keys used to verify JWTs issued by this server
            </p>
          </div>
          <button className="btn ghost sm" onClick={refresh} disabled={loading} title="Refresh">
            <Icon.Refresh width={11} height={11}/>
            Refresh
          </button>
          <button
            className="btn ghost sm"
            onClick={() => window.open('/.well-known/jwks.json', '_blank')}
          >
            <Icon.External width={11} height={11}/>
            Download JWKS
          </button>
          <button className="btn sm" onClick={async () => {
            if (!confirm('Rotate signing key? Clients caching JWKS may see transient failures within cache TTL.')) return;
            try {
              const res = await API.post('/admin/auth/rotate-signing-key');
              alert('Key rotated. New KID: ' + (res?.kid || 'unknown'));
              setLoading(true);
              const jwks = await fetch('/.well-known/jwks.json').then(r => r.json());
              setKeys(jwks.keys || []);
              setLoading(false);
            } catch (e) {
              alert('Rotation failed: ' + (e?.message || 'unknown error'));
            }
          }}>
            <Icon.Signing width={11} height={11}/>
            Rotate
          </button>
        </div>

        {/* JWKS URL strip */}
        <div className="row" style={{
          marginTop: 12, gap: 10,
          padding: '8px 12px',
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline)',
          borderRadius: 4,
          fontSize: 11.5,
        }}>
          <Icon.Globe width={12} height={12} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
          <span className="faint" style={{ fontSize: 11, flexShrink: 0 }}>JWKS URL</span>
          <span className="mono" style={{ color: 'var(--fg-muted)', fontSize: 11 }}>{jwksUrl}</span>
          <CopyField value={jwksUrl} truncate={60}/>
        </div>
      </div>

      {/* Table */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {loading ? (
          <div className="faint" style={{ padding: 40, textAlign: 'center', fontSize: 12 }}>
            Loading…
          </div>
        ) : error ? (
          <div style={{
            margin: 32, padding: '12px 16px',
            background: 'var(--surface-1)', border: '1px solid var(--danger)',
            borderRadius: 4, fontSize: 12, color: 'var(--danger)',
            display: 'flex', alignItems: 'center', gap: 8,
          }}>
            <Icon.Warn width={13} height={13}/>
            <span>Failed to load JWKS: {error}</span>
          </div>
        ) : keys.length === 0 ? (
          <TeachEmptyState
            icon="Key"
            title="No signing keys"
            description="No signing keys configured. JWT mode may not be enabled, or initial key generation has not run."
            cliSnippet="shark keys rotate"
          />
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={skThStyle}>Key ID</th>
                <th style={skThStyle}>Algorithm</th>
                <th style={skThStyle}>Type</th>
                <th style={skThStyle}>Use</th>
                <th style={skThStyle}>Curve / Size</th>
                <th style={skThStyle}>Ops</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((k, i) => (
                <tr key={k.kid || i} style={{ background: 'transparent' }}>
                  <td style={skTdStyle}>
                    {k.kid
                      ? <CopyField value={k.kid} truncate={32}/>
                      : <span className="mono faint" style={{ fontSize: 11 }}>—</span>
                    }
                  </td>
                  <td style={skTdStyle}>
                    <span className="mono" style={{ fontSize: 11.5 }}>{k.alg || '—'}</span>
                  </td>
                  <td style={skTdStyle}>
                    <span className="mono" style={{ fontSize: 11.5 }}>{k.kty || '—'}</span>
                  </td>
                  <td style={skTdStyle}>
                    {k.use === 'sig' ? (
                      <span className="chip success" style={{ height: 17, fontSize: 10 }}>sig</span>
                    ) : k.use === 'enc' ? (
                      <span className="chip" style={{ height: 17, fontSize: 10 }}>enc</span>
                    ) : k.use ? (
                      <span className="chip" style={{ height: 17, fontSize: 10 }}>{k.use}</span>
                    ) : (
                      <span className="faint mono" style={{ fontSize: 10.5 }}>—</span>
                    )}
                  </td>
                  <td style={skTdStyle}>
                    <span className="mono faint" style={{ fontSize: 10.5 }}>
                      {k.crv || (k.n ? Math.floor(atob(k.n.replace(/-/g,'+').replace(/_/g,'/')).length * 8) + ' bit' : '—')}
                    </span>
                  </td>
                  <td style={skTdStyle}>
                    <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
                      {(k.key_ops || []).map(op => (
                        <span key={op} className="chip mono" style={{ height: 15, fontSize: 9, padding: '0 4px' }}>{op}</span>
                      ))}
                      {(!k.key_ops || k.key_ops.length === 0) && (
                        <span className="faint mono" style={{ fontSize: 10.5 }}>—</span>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Footer */}
      {!loading && !error && keys.length > 0 && (
        <div className="row" style={{
          padding: '8px 20px',
          borderTop: '1px solid var(--hairline)',
          fontSize: 10.5, gap: 10, flexShrink: 0,
        }}>
          <Icon.Info width={11} height={11} style={{ opacity: 0.5 }}/>
          <span className="faint">{keys.length} key{keys.length !== 1 ? 's' : ''} published</span>
          <div style={{ flex: 1 }}/>
          <span className="mono faint">GET /.well-known/jwks.json</span>
        </div>
      )}
      <CLIFooter command={`curl -s ${jwksUrl} | jq .`}/>
    </div>
  );
}

