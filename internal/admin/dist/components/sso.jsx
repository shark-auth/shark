// SSO Connections page — SAML + OIDC enterprise single sign-on

function SSO() {
  const [slideOver, setSlideOver] = React.useState(null); // null | { mode: 'create' } | { mode: 'edit', conn: {...} }
  const [deleteTarget, setDeleteTarget] = React.useState(null);

  const { data, loading, refresh } = useAPI('/sso/connections');
  const connections = data?.connections || data || [];

  const handleDelete = async (conn) => {
    if (!confirm(`Delete "${conn.name}"? This cannot be undone.`)) return;
    try {
      await API.del('/sso/connections/' + conn.id);
      refresh();
    } catch (e) {
      alert('Delete failed: ' + e.message);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>SSO Connections</h1>
            <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
              SAML &amp; OIDC enterprise identity providers · {Array.isArray(connections) ? connections.length : 0} configured
            </p>
          </div>
          <button className="btn primary" onClick={() => setSlideOver({ mode: 'create' })}>
            <Icon.Plus width={11} height={11}/> New connection
          </button>
        </div>
      </div>

      {/* Table */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {loading ? (
          <div className="faint" style={{ padding: 20, fontSize: 12 }}>Loading SSO connections…</div>
        ) : !Array.isArray(connections) || connections.length === 0 ? (
          <SSOEmptyState onAdd={() => setSlideOver({ mode: 'create' })}/>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={ssoThStyle}>Name</th>
                <th style={ssoThStyle}>Type</th>
                <th style={ssoThStyle}>Domain</th>
                <th style={ssoThStyle}>Status</th>
                <th style={ssoThStyle}>Created</th>
                <th style={ssoThStyle}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {connections.map(conn => (
                <SSORow
                  key={conn.id}
                  conn={conn}
                  onEdit={() => setSlideOver({ mode: 'edit', conn })}
                  onDelete={() => handleDelete(conn)}
                />
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Footer */}
      <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
        <Icon.SSO width={11} height={11} style={{ opacity: 0.5 }}/>
        <span className="faint">SSO connections</span>
        <div style={{ flex: 1 }}/>
        <span className="faint mono">POST /sso/connections</span>
      </div>

      {slideOver && (
        <SSOSlideOver
          mode={slideOver.mode}
          conn={slideOver.conn || null}
          onClose={() => setSlideOver(null)}
          onSave={async (payload) => {
            if (slideOver.mode === 'create') {
              await API.post('/sso/connections', payload);
            } else {
              await API.put('/sso/connections/' + slideOver.conn.id, payload);
            }
            refresh();
            setSlideOver(null);
          }}
        />
      )}
    </div>
  );
}

function SSORow({ conn, onEdit, onDelete }) {
  return (
    <tr style={{ borderBottom: '1px solid var(--hairline)' }}>
      <td style={ssoTdStyle}>
        <div className="row" style={{ gap: 8 }}>
          <div style={{
            width: 22, height: 22, borderRadius: 4, background: 'var(--surface-3)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 10, fontWeight: 600, color: 'var(--fg)', border: '1px solid var(--hairline-strong)',
            flexShrink: 0,
          }}>{(conn.name || '?')[0].toUpperCase()}</div>
          <span style={{ fontWeight: 500 }}>{conn.name}</span>
        </div>
      </td>
      <td style={ssoTdStyle}>
        <SSOTypeChip type={conn.type}/>
      </td>
      <td style={ssoTdStyle}>
        <span className="mono faint" style={{ fontSize: 11 }}>{conn.domain || '—'}</span>
      </td>
      <td style={ssoTdStyle}>
        <span className="row" style={{ gap: 5, fontSize: 11 }}>
          <span className={'dot ' + (conn.enabled ? 'success' : '')} style={{ opacity: conn.enabled ? 1 : 0.4 }}/>
          <span className="faint">{conn.enabled ? 'enabled' : 'disabled'}</span>
        </span>
      </td>
      <td style={ssoTdStyle}>
        <span className="mono faint" style={{ fontSize: 10.5 }}>{ssoRelativeTime(conn.created_at)}</span>
      </td>
      <td style={ssoTdStyle}>
        <div className="row" style={{ gap: 4 }}>
          <button className="btn ghost sm" onClick={onEdit}>Edit</button>
          <button className="btn danger sm" onClick={onDelete}>Delete</button>
        </div>
      </td>
    </tr>
  );
}

function SSOTypeChip({ type }) {
  const t = (type || '').toUpperCase();
  const isSAML = t === 'SAML';
  return (
    <span className="chip" style={{
      height: 18, fontSize: 10, padding: '0 6px',
      background: isSAML ? 'var(--surface-3)' : 'var(--surface-2)',
      fontWeight: 600, letterSpacing: '0.05em',
    }}>
      {t || '—'}
    </span>
  );
}

function SSOEmptyState({ onAdd }) {
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      padding: '60px 24px', gap: 12, textAlign: 'center',
    }}>
      <Icon.SSO width={32} height={32} style={{ opacity: 0.25 }}/>
      <div style={{ fontSize: 13, fontWeight: 500 }}>No SSO connections</div>
      <div className="faint" style={{ fontSize: 12, maxWidth: 340, lineHeight: 1.6 }}>
        Add a SAML or OIDC connection to enable enterprise single sign-on for your organization.
      </div>
      <button className="btn primary" style={{ marginTop: 4 }} onClick={onAdd}>
        <Icon.Plus width={11} height={11}/> Add connection
      </button>
    </div>
  );
}

function SSOSlideOver({ mode, conn, onClose, onSave }) {
  const isEdit = mode === 'edit';

  // Determine initial type: default to oidc for new, use existing for edit
  const initialType = isEdit ? ((conn.type || 'oidc').toLowerCase()) : 'oidc';

  const [type, setType] = React.useState(initialType);
  const [name, setName] = React.useState(isEdit ? (conn.name || '') : '');
  const [domain, setDomain] = React.useState(isEdit ? (conn.domain || '') : '');
  const [enabled, setEnabled] = React.useState(isEdit ? (conn.enabled !== false) : true);

  // OIDC fields
  const [issuerUrl, setIssuerUrl] = React.useState(isEdit ? (conn.issuer_url || '') : '');
  const [clientId, setClientId] = React.useState(isEdit ? (conn.client_id || '') : '');
  const [clientSecret, setClientSecret] = React.useState(''); // never pre-fill secret

  // SAML fields
  const [idpUrl, setIdpUrl] = React.useState(isEdit ? (conn.idp_url || '') : '');
  const [idpCert, setIdpCert] = React.useState(isEdit ? (conn.idp_certificate || '') : '');
  const [spEntityId, setSpEntityId] = React.useState(isEdit ? (conn.sp_entity_id || '') : '');

  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [showSecret, setShowSecret] = React.useState(false);

  const handleSave = async () => {
    if (!name.trim()) { setError('Name is required.'); return; }
    if (!domain.trim()) { setError('Domain is required.'); return; }

    let payload = { name: name.trim(), domain: domain.trim(), type, enabled };

    if (type === 'oidc') {
      if (!issuerUrl.trim()) { setError('Issuer URL is required.'); return; }
      if (!clientId.trim()) { setError('Client ID is required.'); return; }
      payload.issuer_url = issuerUrl.trim();
      payload.client_id = clientId.trim();
      if (clientSecret.trim()) payload.client_secret = clientSecret.trim();
    } else {
      if (!idpUrl.trim()) { setError('IdP URL is required.'); return; }
      if (!idpCert.trim()) { setError('IdP Certificate is required.'); return; }
      payload.idp_url = idpUrl.trim();
      payload.idp_certificate = idpCert.trim();
      if (spEntityId.trim()) payload.sp_entity_id = spEntityId.trim();
    }

    setSaving(true);
    setError(null);
    try {
      await onSave(payload);
    } catch (e) {
      setError(e.message);
      setSaving(false);
    }
  };

  return (
    <div style={ssoModalBackdrop} onClick={onClose}>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0, width: 520,
        background: 'var(--surface-0)', borderLeft: '1px solid var(--hairline-bright)',
        display: 'flex', flexDirection: 'column', boxShadow: 'var(--shadow-lg)',
      }} onClick={e => e.stopPropagation()}>
        {/* Slide-over header */}
        <div className="row" style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)' }}>
          <h2 style={{ margin: 0, fontSize: 14, fontWeight: 600, flex: 1 }}>
            {isEdit ? 'Edit connection' : 'New SSO connection'}
          </h2>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        {/* Form body */}
        <div style={{ flex: 1, overflowY: 'auto', padding: 20, display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* Type selector */}
          <div>
            <label style={ssoLabelStyle}>Protocol</label>
            <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
              {['oidc', 'saml'].map(t => (
                <label key={t} style={{
                  display: 'flex', alignItems: 'center', gap: 7,
                  padding: '7px 12px', borderRadius: 4, cursor: 'pointer',
                  border: type === t ? '1px solid var(--fg)' : '1px solid var(--hairline-strong)',
                  background: type === t ? 'var(--surface-2)' : 'var(--surface-1)',
                  fontSize: 12, fontWeight: type === t ? 500 : 400,
                  flex: 1, justifyContent: 'center',
                }}>
                  <input
                    type="radio"
                    name="sso_type"
                    value={t}
                    checked={type === t}
                    onChange={() => setType(t)}
                    style={{ display: 'none' }}
                  />
                  {t.toUpperCase()}
                </label>
              ))}
            </div>
          </div>

          {/* Common fields */}
          <SSOField label="Name" required>
            <input
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="Acme Corp"
              style={ssoInputStyle}
            />
          </SSOField>

          <SSOField label="Domain" required hint="Users with this email domain will be redirected to this IdP">
            <input
              value={domain}
              onChange={e => setDomain(e.target.value)}
              placeholder="acme.com"
              style={ssoInputStyle}
            />
          </SSOField>

          {/* OIDC-specific fields */}
          {type === 'oidc' && (
            <>
              <SSOField label="Issuer URL" required>
                <input
                  value={issuerUrl}
                  onChange={e => setIssuerUrl(e.target.value)}
                  placeholder="https://accounts.google.com"
                  style={ssoInputStyle}
                />
              </SSOField>

              <SSOField label="Client ID" required>
                <input
                  value={clientId}
                  onChange={e => setClientId(e.target.value)}
                  placeholder="your-client-id"
                  style={ssoInputStyle}
                />
              </SSOField>

              <SSOField label="Client Secret" hint={isEdit ? 'Leave blank to keep existing secret' : undefined}>
                <div style={{ position: 'relative' }}>
                  <input
                    type={showSecret ? 'text' : 'password'}
                    value={clientSecret}
                    onChange={e => setClientSecret(e.target.value)}
                    placeholder={isEdit ? '••••••••' : 'your-client-secret'}
                    style={{ ...ssoInputStyle, paddingRight: 32 }}
                  />
                  <button
                    type="button"
                    onClick={() => setShowSecret(v => !v)}
                    style={{
                      position: 'absolute', right: 6, top: '50%', transform: 'translateY(-50%)',
                      background: 'none', border: 'none', cursor: 'pointer', color: 'var(--fg-muted)', padding: 2,
                    }}
                  >
                    <Icon.Eye width={12} height={12}/>
                  </button>
                </div>
              </SSOField>
            </>
          )}

          {/* SAML-specific fields */}
          {type === 'saml' && (
            <>
              <SSOField label="IdP SSO URL" required>
                <input
                  value={idpUrl}
                  onChange={e => setIdpUrl(e.target.value)}
                  placeholder="https://idp.acme.com/saml/sso"
                  style={ssoInputStyle}
                />
              </SSOField>

              <SSOField label="IdP Certificate" required hint="Paste the X.509 certificate from your identity provider">
                <textarea
                  value={idpCert}
                  onChange={e => setIdpCert(e.target.value)}
                  placeholder={"-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----"}
                  rows={5}
                  style={{
                    ...ssoInputStyle,
                    fontFamily: 'var(--font-mono)', fontSize: 11,
                    resize: 'vertical', minHeight: 80,
                  }}
                />
              </SSOField>

              <SSOField label="SP Entity ID" hint="Auto-generated if left blank">
                <input
                  value={spEntityId}
                  onChange={e => setSpEntityId(e.target.value)}
                  placeholder="https://auth.yourdomain.com/saml/metadata"
                  style={ssoInputStyle}
                />
              </SSOField>
            </>
          )}

          {/* Enabled toggle */}
          <div className="row" style={{ gap: 10, padding: '10px 0', borderTop: '1px solid var(--hairline)' }}>
            <div style={{ flex: 1 }}>
              <div style={{ fontSize: 12, fontWeight: 500 }}>Enable connection</div>
              <div className="faint" style={{ fontSize: 11, marginTop: 2 }}>
                When disabled, users will not be redirected to this IdP
              </div>
            </div>
            <button
              type="button"
              onClick={() => setEnabled(v => !v)}
              style={{
                width: 36, height: 20, borderRadius: 10, border: 'none', cursor: 'pointer',
                background: enabled ? 'var(--fg)' : 'var(--surface-3)',
                position: 'relative', transition: 'background 120ms', flexShrink: 0,
              }}
            >
              <span style={{
                position: 'absolute', top: 2, left: enabled ? 18 : 2,
                width: 16, height: 16, borderRadius: '50%',
                background: enabled ? 'var(--bg)' : 'var(--fg-dim)',
                transition: 'left 120ms',
              }}/>
            </button>
          </div>

          {error && (
            <div style={{ color: 'var(--danger)', fontSize: 11, padding: '7px 10px', background: 'var(--surface-1)', borderRadius: 3 }}>
              {error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="row" style={{ padding: 12, borderTop: '1px solid var(--hairline)', gap: 8, justifyContent: 'flex-end' }}>
          <button className="btn ghost" onClick={onClose} disabled={saving}>Cancel</button>
          <button className="btn primary" onClick={handleSave} disabled={saving}>
            {saving ? (isEdit ? 'Saving…' : 'Creating…') : (isEdit ? 'Save changes' : 'Create connection')}
          </button>
        </div>
      </div>
    </div>
  );
}

function SSOField({ label, required, hint, children }) {
  return (
    <div>
      <label style={ssoLabelStyle}>
        {label}
        {required && <span style={{ color: 'var(--danger)', marginLeft: 2 }}>*</span>}
      </label>
      {hint && <div className="faint" style={{ fontSize: 10.5, marginBottom: 4, lineHeight: 1.4 }}>{hint}</div>}
      <div style={{ marginTop: hint ? 0 : 4 }}>{children}</div>
    </div>
  );
}

function ssoRelativeTime(val) {
  if (!val) return '—';
  const ms = typeof val === 'string' ? new Date(val).getTime() : val;
  const diff = Date.now() - ms;
  if (diff < 0) return 'just now';
  if (diff < 60e3) return Math.floor(diff / 1e3) + 's ago';
  if (diff < 3600e3) return Math.floor(diff / 60e3) + 'm ago';
  if (diff < 86400e3) return Math.floor(diff / 3600e3) + 'h ago';
  return Math.floor(diff / 86400e3) + 'd ago';
}

const ssoThStyle = {
  textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500,
  color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-0)', position: 'sticky', top: 0,
  textTransform: 'uppercase', letterSpacing: '0.05em',
};
const ssoTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const ssoModalBackdrop = { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', zIndex: 50 };
const ssoLabelStyle = { display: 'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4 };
const ssoInputStyle = {
  width: '100%', boxSizing: 'border-box', fontSize: 12,
  padding: '6px 9px', border: '1px solid var(--hairline-strong)',
  borderRadius: 3, background: 'var(--surface-1)',
  color: 'var(--fg)', outline: 'none',
};

Object.assign(window, { SSO });
