// Dev Inbox — captures all outgoing mail in dev mode

function DevInbox() {
  const { data, loading, error, refresh } = useAPI('/admin/dev/emails');
  const [selected, setSelected] = React.useState(null);
  const [clearing, setClearing] = React.useState(false);

  const emails = React.useMemo(() => {
    return data?.emails || data?.items || (Array.isArray(data) ? data : []);
  }, [data]);

  // Close detail if selected email is no longer in list (e.g. after clear)
  React.useEffect(() => {
    if (selected && !emails.find(e => e.id === selected.id)) {
      setSelected(null);
    }
  }, [emails, selected]);

  const handleClearAll = async () => {
    if (!window.confirm('Clear all captured emails? This cannot be undone.')) return;
    setClearing(true);
    try {
      await API.del('/admin/dev/emails');
      setSelected(null);
      refresh();
    } catch (e) {
      alert('Failed to clear emails: ' + e.message);
    } finally {
      setClearing(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Warning banner */}
      <div style={{
        background: 'var(--warn-bg)',
        color: 'var(--warn)',
        padding: '8px 20px',
        fontSize: 12,
        borderBottom: '1px solid var(--hairline)',
        display: 'flex', alignItems: 'center', gap: 8,
        flexShrink: 0,
      }}>
        <Icon.Warn width={13} height={13} style={{ flexShrink: 0 }}/>
        <span>Dev inbox captures all outgoing mail. Switch email provider before production.</span>
      </div>

      {/* Main area: split or full */}
      <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 460px' : '1fr', flex: 1, overflow: 'hidden', minHeight: 0 }}>
        {/* List panel */}
        <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          {/* Header */}
          <div className="row" style={{ padding: '12px 20px', borderBottom: '1px solid var(--hairline)', gap: 8, flexShrink: 0 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Dev Inbox</h1>
              <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                {emails.length > 0 ? `${emails.length} captured email${emails.length === 1 ? '' : 's'}` : 'No captured emails'}
              </p>
            </div>
            <button className="btn ghost" onClick={refresh} disabled={loading} title="Refresh">
              <Icon.Refresh width={12} height={12}/>
              Refresh
            </button>
            <button
              className="btn ghost"
              onClick={handleClearAll}
              disabled={clearing || emails.length === 0}
              style={{ color: emails.length > 0 ? 'var(--danger)' : undefined }}
            >
              <Icon.X width={12} height={12}/>
              Clear All
            </button>
          </div>

          {/* Status rows */}
          {loading && <div className="faint" style={{ padding: '8px 20px', fontSize: 11 }}>Loading…</div>}
          {error && <div style={{ padding: '8px 20px', fontSize: 11, color: 'var(--danger)' }}>Error: {error}</div>}

          {/* Table */}
          <div style={{ flex: 1, overflow: 'auto' }}>
            {!loading && !error && emails.length === 0 ? (
              <div style={{ padding: '60px 20px', textAlign: 'center' }} className="faint">
                <Icon.Mail width={28} height={28} style={{ opacity: 0.25, display: 'block', margin: '0 auto 12px' }}/>
                No captured emails. Send a magic link or password reset to see emails here.
              </div>
            ) : (
              <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                <thead>
                  <tr>
                    <th style={thStyle}>To</th>
                    <th style={thStyle}>Subject</th>
                    <th style={thStyle}>Type</th>
                    <th style={{ ...thStyle, width: 110, textAlign: 'right' }}>Received</th>
                  </tr>
                </thead>
                <tbody>
                  {emails.map(email => (
                    <tr
                      key={email.id}
                      onClick={() => setSelected(email)}
                      className={selected?.id === email.id ? 'row-selected' : ''}
                      style={{
                        cursor: 'pointer',
                        background: selected?.id === email.id ? 'var(--surface-2)' : 'transparent',
                      }}
                    >
                      <td style={tdStyle}>
                        <span className="mono" style={{ fontSize: 11.5 }}>{email.to || email.recipient || '—'}</span>
                      </td>
                      <td style={tdStyle}>
                        <span style={{ fontSize: 12 }}>{email.subject || '(no subject)'}</span>
                      </td>
                      <td style={tdStyle}>
                        <EmailTypeChip type={email.type || email.email_type || detectEmailType(email.subject)}/>
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right' }}>
                        <span className="mono faint" style={{ fontSize: 11 }}>
                          {relativeTime(email.created_at || email.received_at || email.timestamp)}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        {/* Detail panel */}
        {selected && (
          <EmailDetail email={selected} onClose={() => setSelected(null)}/>
        )}
      </div>
    </div>
  );
}

function detectEmailType(subject) {
  if (!subject) return null;
  const s = subject.toLowerCase();
  if (s.includes('magic') || s.includes('sign in') || s.includes('login')) return 'magic_link';
  if (s.includes('verif') || s.includes('confirm')) return 'verify';
  if (s.includes('password') || s.includes('reset')) return 'password_reset';
  return null;
}

function EmailTypeChip({ type }) {
  if (!type) return <span className="faint" style={{ fontSize: 11 }}>—</span>;
  const labels = {
    magic_link: 'magic link',
    verify: 'verify',
    password_reset: 'pwd reset',
  };
  return (
    <span className="chip mono" style={{ height: 16, fontSize: 9.5, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
      {labels[type] || type}
    </span>
  );
}

function relativeTime(ts) {
  if (!ts) return '—';
  const t = typeof ts === 'number' ? ts : new Date(ts).getTime();
  const diff = Math.floor((Date.now() - t) / 1000);
  if (diff < 0) return 'just now';
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function extractMagicLink(html, text) {
  // Look for magic link URLs in HTML or text body
  const sources = [html || '', text || ''];
  for (const src of sources) {
    // Common patterns: href="...magic...", href="...verify...", href="...confirm..."
    const hrefMatch = src.match(/href=["']([^"']*(?:magic|verify|confirm|token|otp|click)[^"']*)/i);
    if (hrefMatch) return hrefMatch[1];
    // Bare URL lines
    const urlMatch = src.match(/(https?:\/\/\S+(?:magic|verify|confirm|token|otp|click)\S+)/i);
    if (urlMatch) return urlMatch[1];
  }
  return null;
}

function EmailDetail({ email, onClose }) {
  const [showRaw, setShowRaw] = React.useState(false);
  const [detail, setDetail] = React.useState(null);
  const [detailLoading, setDetailLoading] = React.useState(false);
  const [detailError, setDetailError] = React.useState(null);

  // Fetch full email detail if we only have summary in list
  React.useEffect(() => {
    if (!email?.id) return;
    // If email already has html_body or text_body, use it directly
    if (email.html_body || email.text_body || email.body || email.html) {
      setDetail(email);
      return;
    }
    setDetailLoading(true);
    setDetailError(null);
    API.get(`/admin/dev/emails/${email.id}`)
      .then(d => setDetail(d))
      .catch(e => { setDetailError(e.message); setDetail(email); })
      .finally(() => setDetailLoading(false));
  }, [email?.id]);

  const d = detail || email;
  const htmlBody = d?.html_body || d?.html || d?.body_html || '';
  const textBody = d?.text_body || d?.text || d?.body_text || d?.body || '';
  const magicLink = extractMagicLink(htmlBody, textBody);

  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex',
      flexDirection: 'column',
      overflow: 'hidden',
    }}>
      {/* Detail header */}
      <div className="row" style={{ padding: '12px 16px', borderBottom: '1px solid var(--hairline)', gap: 8, flexShrink: 0 }}>
        <Icon.Mail width={14} height={14} style={{ opacity: 0.6, flexShrink: 0 }}/>
        <span style={{ fontSize: 13, fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {email.subject || '(no subject)'}
        </span>
        <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
      </div>

      {/* Email meta */}
      <div style={{
        padding: '10px 16px',
        borderBottom: '1px solid var(--hairline)',
        display: 'grid',
        gridTemplateColumns: 'auto 1fr',
        gap: '5px 12px',
        fontSize: 11.5,
        flexShrink: 0,
      }}>
        <span className="faint">To</span>
        <span className="mono">{email.to || email.recipient || '—'}</span>
        <span className="faint">From</span>
        <span className="mono">{email.from || email.sender || '—'}</span>
        <span className="faint">Type</span>
        <span><EmailTypeChip type={email.type || email.email_type || detectEmailType(email.subject)}/></span>
        <span className="faint">Received</span>
        <span className="mono faint">{relativeTime(email.created_at || email.received_at || email.timestamp)}</span>
      </div>

      {/* Magic link action */}
      {magicLink && (
        <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--hairline)', flexShrink: 0 }}>
          <a
            href={magicLink}
            target="_blank"
            rel="noopener noreferrer"
            className="btn"
            style={{ display: 'inline-flex', alignItems: 'center', gap: 6, textDecoration: 'none', fontSize: 12 }}
          >
            <Icon.External width={11} height={11}/>
            Open magic link
          </a>
        </div>
      )}

      {/* Toggle: Preview / Raw */}
      <div className="row" style={{ padding: '8px 16px', borderBottom: '1px solid var(--hairline)', gap: 6, flexShrink: 0 }}>
        <button
          className={'btn ghost' + (!showRaw ? ' primary' : '')}
          style={{ fontSize: 11, height: 24 }}
          onClick={() => setShowRaw(false)}
        >
          Preview
        </button>
        <button
          className={'btn ghost' + (showRaw ? ' primary' : '')}
          style={{ fontSize: 11, height: 24 }}
          onClick={() => setShowRaw(true)}
        >
          Raw source
        </button>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflow: 'auto', padding: showRaw ? 0 : 0 }}>
        {detailLoading && (
          <div className="faint" style={{ padding: '12px 16px', fontSize: 11 }}>Loading email…</div>
        )}
        {detailError && (
          <div style={{ padding: '12px 16px', fontSize: 11, color: 'var(--danger)' }}>Could not load full email: {detailError}</div>
        )}
        {!detailLoading && (
          showRaw ? (
            <pre style={{
              margin: 0,
              padding: 16,
              fontSize: 10.5,
              lineHeight: 1.55,
              fontFamily: 'var(--font-mono)',
              color: 'var(--fg)',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
            }}>
              {htmlBody || textBody || JSON.stringify(d, null, 2) || '(no content)'}
            </pre>
          ) : htmlBody ? (
            <iframe
              srcDoc={htmlBody}
              sandbox="allow-same-origin"
              style={{ width: '100%', height: '100%', border: 'none', background: '#fff', display: 'block' }}
              title="Email preview"
            />
          ) : textBody ? (
            <pre style={{
              margin: 0,
              padding: 16,
              fontSize: 12,
              lineHeight: 1.6,
              fontFamily: 'var(--font-sans)',
              color: 'var(--fg)',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
            }}>{textBody}</pre>
          ) : (
            <div className="faint" style={{ padding: '24px 16px', fontSize: 12 }}>No preview available.</div>
          )
        )}
      </div>
    </aside>
  );
}

const thStyle = {
  textAlign: 'left',
  padding: '7px 16px',
  fontSize: 10.5,
  fontWeight: 500,
  color: 'var(--fg-dim)',
  borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-0)',
  position: 'sticky',
  top: 0,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
};
const tdStyle = { padding: '7px 16px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };

Object.assign(window, { DevInbox });
