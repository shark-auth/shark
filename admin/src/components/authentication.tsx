// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'

// Authentication Config page — read-only display of auth method configuration

export function Authentication() {
  const { data: config, loading, error } = useAPI('/admin/config');
  const [previewTpl, setPreviewTpl] = React.useState(null);
  const [previewHTML, setPreviewHTML] = React.useState(null);
  const [previewErr, setPreviewErr] = React.useState(null);

  const openPreview = async (tpl) => {
    setPreviewTpl(tpl);
    setPreviewHTML(null);
    setPreviewErr(null);
    try {
      const res = await API.get('/admin/email-preview/' + tpl);
      setPreviewHTML(res?.html || '');
    } catch (e) {
      setPreviewErr(e.message || 'Failed to render preview');
    }
  };

  if (loading) {
    return (
      <div style={{ padding: 24 }}>
        <div className="faint" style={{ fontSize: 12 }}>Loading authentication config…</div>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ padding: 24 }}>
        <div style={{ color: 'var(--danger)', fontSize: 12 }}>Failed to load config: {error}</div>
      </div>
    );
  }

  const cfg = config || {};
  const oauthProviders = cfg.oauth_providers || [];
  const smtpConfigured = cfg.smtp_configured === true;
  const jwtMode = cfg.jwt_mode === true;

  const OAUTH_PROVIDERS = [
    { id: 'google',  label: 'Google',  initial: 'G' },
    { id: 'github',  label: 'GitHub',  initial: 'GH' },
    { id: 'apple',   label: 'Apple',   initial: 'A' },
    { id: 'discord', label: 'Discord', initial: 'D' },
  ];

  return (
    <div style={{ height: '100%', overflowY: 'auto', padding: 20 }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16, maxWidth: 860 }}>

        {/* Page header */}
        <div style={{ marginBottom: 4 }}>
          <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Authentication</h1>
          <p className="faint" style={{ margin: '3px 0 0', fontSize: 11.5 }}>
            Read-only view of active authentication configuration · <span className="mono">GET /admin/config</span>
          </p>
        </div>

        {/* Password Policy */}
        <div className="card">
          <div className="card-header">
            <div className="row" style={{ gap: 8 }}>
              <Icon.Lock width={13} height={13} style={{ opacity: 0.6 }}/>
              <span>Password Policy</span>
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0 }}>
            <AuthCell
              label="Minimum length"
              value="8 characters"
            />
            <AuthCell
              label="Complexity"
              value="Uppercase + lowercase + digit"
              border="left"
            />
            <AuthCell
              label="Lockout"
              value="5 attempts / 15 min cooldown"
              border="left"
            />
          </div>
        </div>

        {/* OAuth Providers */}
        <div className="card">
          <div className="card-header">
            <div className="row" style={{ gap: 8 }}>
              <Icon.Globe width={13} height={13} style={{ opacity: 0.6 }}/>
              <span>OAuth Providers</span>
            </div>
            <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>
              {oauthProviders.length} configured
            </span>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 0 }}>
            {OAUTH_PROVIDERS.map((p, i) => {
              const configured = oauthProviders.includes(p.id);
              return (
                <div
                  key={p.id}
                  style={{
                    padding: '14px 16px',
                    borderRight: i < 3 ? '1px solid var(--hairline)' : 'none',
                    opacity: configured ? 1 : 0.45,
                  }}
                >
                  <div className="row" style={{ gap: 10, marginBottom: 10 }}>
                    <div style={{
                      width: 28, height: 28, borderRadius: 6,
                      background: configured ? 'var(--surface-3)' : 'var(--surface-2)',
                      border: '1px solid var(--hairline-strong)',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      fontSize: 10, fontWeight: 700, color: 'var(--fg)', flexShrink: 0,
                      letterSpacing: '-0.01em',
                    }}>
                      {p.initial}
                    </div>
                    <span style={{ fontSize: 12.5, fontWeight: 500 }}>{p.label}</span>
                  </div>
                  <div className="row" style={{ gap: 5 }}>
                    <span
                      className={'dot ' + (configured ? 'success' : '')}
                      style={{ opacity: configured ? 1 : 0.5 }}
                    />
                    <span style={{ fontSize: 11, color: configured ? 'var(--success)' : 'var(--fg-dim)' }}>
                      {configured ? 'Configured' : 'Not configured'}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        {/* Two-column row: Magic Links + Passkeys */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>

          {/* Magic Links */}
          <div className="card">
            <div className="card-header">
              <div className="row" style={{ gap: 8 }}>
                <Icon.Mail width={13} height={13} style={{ opacity: 0.6 }}/>
                <span>Magic Links</span>
              </div>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0 }}>
              <AuthCell
                label="Status"
                value={smtpConfigured ? 'Available' : 'Not configured'}
                valueColor={smtpConfigured ? 'var(--success)' : 'var(--fg-dim)'}
                dot={smtpConfigured ? 'success' : ''}
              />
              <AuthCell
                label="Token lifetime"
                value="15 minutes"
                border="left"
              />
            </div>
          </div>

          {/* Passkeys */}
          <div className="card">
            <div className="card-header">
              <div className="row" style={{ gap: 8 }}>
                <Icon.Key width={13} height={13} style={{ opacity: 0.6 }}/>
                <span>Passkeys</span>
              </div>
              {cfg.passkey_count != null && (
                <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>
                  {cfg.passkey_count} registered
                </span>
              )}
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0 }}>
              <AuthCell
                label="Status"
                value="Enabled"
                valueColor="var(--success)"
                dot="success"
              />
              <AuthCell
                label="RP Name"
                value={cfg.rp_name || 'Default'}
                border="left"
              />
              <AuthCell
                label="RP ID"
                value={cfg.rp_id || 'Default'}
                mono
                border="left"
              />
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 0, borderTop: '1px solid var(--hairline)' }}>
              <AuthCell
                label="Origin"
                value={cfg.passkey_origin || cfg.base_url || window.location.origin}
                mono
              />
              <AuthCell
                label="User Verification"
                value={cfg.passkey_uv || 'preferred'}
                border="left"
              />
              <AuthCell
                label="Attestation"
                value={cfg.passkey_attestation || 'none'}
                border="left"
              />
            </div>
          </div>
        </div>

        {/* Two-column row: Email Verification + JWT Mode */}
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>

          {/* Email Verification */}
          <div className="card">
            <div className="card-header">
              <div className="row" style={{ gap: 8 }}>
                <Icon.Check width={13} height={13} style={{ opacity: 0.6 }}/>
                <span>Email Verification</span>
              </div>
            </div>
            <div style={{ padding: '12px 14px' }}>
              <div className="row" style={{ gap: 8, marginBottom: 6 }}>
                <span className={'dot ' + (smtpConfigured ? 'success' : '')} style={{ opacity: smtpConfigured ? 1 : 0.5 }}/>
                <span style={{
                  fontSize: 13, fontWeight: 500,
                  color: smtpConfigured ? 'var(--fg)' : 'var(--fg-dim)',
                }}>
                  {smtpConfigured ? 'Available' : 'Not configured'}
                </span>
              </div>
              {!smtpConfigured && (
                <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
                  SMTP not configured — verification emails won't send
                </div>
              )}
              {smtpConfigured && (
                <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
                  SMTP configured — verification emails will send on registration
                </div>
              )}
              <div style={{ marginTop: 10, borderTop: '1px solid var(--hairline)', paddingTop: 8 }}>
                <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 6 }}>
                  Template preview
                </div>
                <div className="row" style={{ gap: 6 }}>
                  {['verify_email', 'magic_link', 'password_reset', 'organization_invitation'].map(t => (
                    <button key={t} className="btn ghost sm" onClick={() => openPreview(t)}
                      style={{ fontSize: 10 }}>
                      {t.replace(/_/g, ' ')}
                    </button>
                  ))}
                </div>
                <div className="faint" style={{ fontSize: 10, marginTop: 4 }}>
                  Renders against sample data via <span className="mono">GET /admin/email-preview/&#123;template&#125;</span>.
                </div>
              </div>
            </div>
          </div>

          {/* JWT Mode */}
          <div className="card">
            <div className="card-header">
              <div className="row" style={{ gap: 8 }}>
                <Icon.Token width={13} height={13} style={{ opacity: 0.6 }}/>
                <span>Session Mode</span>
              </div>
            </div>
            <div style={{ padding: '12px 14px' }}>
              <div style={{ display: 'flex', gap: 10, marginBottom: 10 }}>
                {[
                  { id: 'cookie', label: 'Cookie sessions', desc: 'Server-side sessions stored in DB' },
                  { id: 'jwt', label: 'JWT mode', desc: 'Stateless tokens signed with JWKS' },
                ].map(opt => {
                  const active = opt.id === 'jwt' ? jwtMode : !jwtMode;
                  return (
                    <div key={opt.id} style={{
                      flex: 1, padding: '10px 12px', borderRadius: 5,
                      border: `1px solid ${active ? 'var(--fg)' : 'var(--hairline-strong)'}`,
                      background: active ? 'var(--surface-3)' : 'var(--surface-1)',
                      cursor: 'not-allowed', opacity: active ? 1 : 0.5,
                    }}>
                      <div className="row" style={{ gap: 6 }}>
                        <span style={{
                          width: 12, height: 12, borderRadius: 99,
                          border: `2px solid ${active ? 'var(--fg)' : 'var(--hairline-bright)'}`,
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                        }}>
                          {active && <span style={{ width: 5, height: 5, borderRadius: 99, background: 'var(--fg)' }}/>}
                        </span>
                        <span style={{ fontSize: 12.5, fontWeight: 500 }}>{opt.label}</span>
                      </div>
                      <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 4, marginLeft: 18 }}>{opt.desc}</div>
                    </div>
                  );
                })}
              </div>
              {jwtMode && (
                <div className="row" style={{ gap: 8 }}>
                  <span style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Algorithm</span>
                  <span className="mono" style={{ fontSize: 12 }}>{cfg.jwt_algorithm || 'ES256'}</span>
                </div>
              )}
              <div style={{
                marginTop: 10, padding: '8px 10px', borderRadius: 4,
                background: 'var(--warn-bg)', border: '1px solid color-mix(in oklch, var(--warn) 25%, var(--hairline))',
                fontSize: 11, color: 'var(--warn)', lineHeight: 1.5,
              }}>
                <Icon.Warn width={11} height={11} style={{ verticalAlign: -2, marginRight: 6 }}/>
                Switching mode invalidates all existing sessions. Edit <span className="mono">auth.jwt.mode</span> in sharkauth.yaml and restart.
              </div>
            </div>
          </div>

        </div>

        {/* Base URL + CORS */}
        {(cfg.base_url || (cfg.cors_origins && cfg.cors_origins.length > 0)) && (
          <div className="card">
            <div className="card-header">
              <div className="row" style={{ gap: 8 }}>
                <Icon.Globe width={13} height={13} style={{ opacity: 0.6 }}/>
                <span>Server</span>
              </div>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 0 }}>
              {cfg.base_url && (
                <AuthCell
                  label="Base URL"
                  value={cfg.base_url}
                  mono
                />
              )}
              {cfg.cors_origins && cfg.cors_origins.length > 0 && (
                <AuthCell
                  label={`CORS origins (${cfg.cors_origins.length})`}
                  value={cfg.cors_origins.join(', ')}
                  mono
                  border={cfg.base_url ? 'left' : undefined}
                />
              )}
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="row" style={{ fontSize: 10.5, gap: 8, color: 'var(--fg-faint)', paddingBottom: 8 }}>
          <Icon.Lock width={11} height={11} style={{ opacity: 0.4 }}/>
          <span>Read-only config display</span>
          <span className="faint">·</span>
          <span className="mono faint">GET /admin/config</span>
          {cfg.dev_mode && (
            <>
              <span className="faint">·</span>
              <span className="chip warn" style={{ height: 16, fontSize: 10, padding: '0 6px' }}>dev mode</span>
            </>
          )}
        </div>

      </div>

      {/* Email preview modal */}
      {previewTpl && (
        <div onClick={() => setPreviewTpl(null)} style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
          display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50,
        }}>
          <div onClick={e => e.stopPropagation()} style={{
            background: 'var(--surface-1)', border: '1px solid var(--hairline-bright)',
            borderRadius: 6, width: 720, maxWidth: '90vw', maxHeight: '85vh',
            display: 'flex', flexDirection: 'column',
          }}>
            <div className="row" style={{ padding: '10px 14px', borderBottom: '1px solid var(--hairline)' }}>
              <span style={{ fontSize: 13, fontWeight: 500 }}>Preview · {previewTpl.replace(/_/g, ' ')}</span>
              <div style={{ flex: 1 }}/>
              <button className="btn ghost sm" onClick={() => setPreviewTpl(null)}>Close</button>
            </div>
            <div style={{ flex: 1, overflow: 'auto', background: '#fff' }}>
              {previewErr ? (
                <div style={{ padding: 16, color: 'var(--danger)', fontSize: 12 }}>Failed: {previewErr}</div>
              ) : previewHTML == null ? (
                <div className="faint" style={{ padding: 24, fontSize: 12 }}>Rendering…</div>
              ) : (
                <iframe srcDoc={previewHTML} sandbox="" style={{
                  width: '100%', height: 520, border: 0, background: '#fff',
                }}/>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// AuthCell — a single labeled value cell used inside section cards
function AuthCell({ label, value, valueColor, mono, dot, border }) {
  return (
    <div style={{
      padding: '12px 14px',
      borderLeft: border === 'left' ? '1px solid var(--hairline)' : undefined,
    }}>
      <div style={{
        fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
        color: 'var(--fg-dim)', marginBottom: 5,
      }}>
        {label}
      </div>
      <div className="row" style={{ gap: 6, alignItems: 'center' }}>
        {dot !== undefined && (
          <span className={'dot ' + dot} style={{ opacity: dot ? 1 : 0.4, flexShrink: 0 }}/>
        )}
        <span style={{
          fontSize: 12.5,
          fontWeight: 500,
          fontFamily: mono ? 'var(--font-mono)' : undefined,
          color: valueColor || 'var(--fg)',
        }}>
          {value}
        </span>
      </div>
    </div>
  );
}

