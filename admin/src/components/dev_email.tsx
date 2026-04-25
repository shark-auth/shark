// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'
import { CLIFooter } from './CLIFooter'
import { usePageActions } from './useKeyboardShortcuts'

// Dev Email — captures all outgoing mail when email.provider === 'dev'
// Visibility: only rendered when adminConfig.email.provider === 'dev'
// Right-side slide-over for detail. NEVER a modal.

const thStyle = {
  textAlign: 'left' as const,
  padding: '7px 14px',
  fontSize: 10,
  fontWeight: 500,
  color: 'var(--fg-dim)',
  borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-0)',
  position: 'sticky' as const,
  top: 0,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.06em',
};

const tdStyle = {
  padding: '7px 14px',
  borderBottom: '1px solid var(--hairline)',
  verticalAlign: 'middle' as const,
  fontSize: 12,
};

function detectEmailType(subject: string | null | undefined): string | null {
  if (!subject) return null;
  const s = subject.toLowerCase();
  if (s.includes('magic') || s.includes('sign in') || s.includes('login')) return 'magic_link';
  if (s.includes('verif') || s.includes('confirm')) return 'verify';
  if (s.includes('password') || s.includes('reset')) return 'password_reset';
  if (s.includes('invit')) return 'invite';
  return null;
}

function EmailTypeChip({ type }: { type: string | null }) {
  if (!type) return <span className="faint" style={{ fontSize: 10.5 }}>—</span>;
  const labels: Record<string, string> = {
    magic_link: 'magic link',
    verify: 'verify',
    password_reset: 'pwd reset',
    invite: 'invite',
  };
  return (
    <span className="chip mono" style={{ height: 16, fontSize: 9.5, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
      {labels[type] || type}
    </span>
  );
}

function relativeTime(ts: string | number | null | undefined): string {
  if (!ts) return '—';
  const t = typeof ts === 'number' ? ts : new Date(ts).getTime();
  const diff = Math.floor((Date.now() - t) / 1000);
  if (diff < 0) return 'just now';
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function extractMagicLink(html: string, text: string): string | null {
  const sources = [html || '', text || ''];
  for (const src of sources) {
    const hrefMatch = src.match(/href=["']([^"']*(?:magic|verify|confirm|token|otp|click)[^"']*)/i);
    if (hrefMatch) return hrefMatch[1];
    const urlMatch = src.match(/(https?:\/\/\S+(?:magic|verify|confirm|token|otp|click)\S+)/i);
    if (urlMatch) return urlMatch[1];
  }
  return null;
}

// ── Slide-over detail panel ──────────────────────────────────────────────────

function EmailSlideOver({ email, onClose }: { email: any; onClose: () => void }) {
  const [tab, setTab] = React.useState<'preview' | 'text' | 'raw'>('preview');
  const [detail, setDetail] = React.useState<any>(null);
  const [detailLoading, setDetailLoading] = React.useState(false);
  const [detailError, setDetailError] = React.useState<string | null>(null);
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!email?.id) return;
    // If full body already in summary, use directly
    if (email.html_body || email.text_body || email.body || email.html) {
      setDetail(email);
      return;
    }
    setDetailLoading(true);
    setDetailError(null);
    API.get(`/admin/dev/emails/${email.id}`)
      .then((d: any) => setDetail(d))
      .catch((e: any) => { setDetailError(e.message); setDetail(email); })
      .finally(() => setDetailLoading(false));
  }, [email?.id]);

  // Close on Escape
  React.useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onClose]);

  const d = detail || email;
  const htmlBody: string = d?.html_body || d?.html || d?.body_html || '';
  const textBody: string = d?.text_body || d?.text || d?.body_text || d?.body || '';
  const magicLink = extractMagicLink(htmlBody, textBody);
  const emailType = email.type || email.email_type || detectEmailType(email.subject);

  const copyMagicLink = () => {
    if (!magicLink) return;
    navigator.clipboard.writeText(magicLink).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    });
  };

  return (
    <aside style={{
      position: 'fixed',
      top: 0,
      right: 0,
      bottom: 0,
      width: 420,
      background: 'var(--surface-0)',
      borderLeft: '1px solid var(--hairline)',
      display: 'flex',
      flexDirection: 'column',
      zIndex: 200,
    }}>
      {/* Header */}
      <div className="row" style={{
        padding: '10px 14px',
        borderBottom: '1px solid var(--hairline)',
        gap: 8,
        flexShrink: 0,
        background: 'var(--surface-1)',
      }}>
        <Icon.Mail width={13} height={13} style={{ opacity: 0.5, flexShrink: 0 }}/>
        <span style={{
          fontSize: 12.5,
          fontWeight: 500,
          flex: 1,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}>
          {email.subject || '(no subject)'}
        </span>
        <button className="btn ghost icon sm" onClick={onClose} title="Close (Esc)">
          <Icon.X width={11} height={11}/>
        </button>
      </div>

      {/* Meta grid */}
      <div style={{
        padding: '10px 14px',
        borderBottom: '1px solid var(--hairline)',
        display: 'grid',
        gridTemplateColumns: '48px 1fr',
        gap: '5px 10px',
        fontSize: 11.5,
        flexShrink: 0,
      }}>
        <span style={{ color: 'var(--fg-dim)', fontSize: 11 }}>To</span>
        <span className="mono" style={{ fontSize: 11 }}>{email.to || email.to_addr || email.recipient || '—'}</span>
        <span style={{ color: 'var(--fg-dim)', fontSize: 11 }}>From</span>
        <span className="mono" style={{ fontSize: 11 }}>{email.from || email.sender || '—'}</span>
        <span style={{ color: 'var(--fg-dim)', fontSize: 11 }}>Type</span>
        <span style={{ paddingTop: 1 }}><EmailTypeChip type={emailType}/></span>
        <span style={{ color: 'var(--fg-dim)', fontSize: 11 }}>Received</span>
        <span className="mono" style={{ fontSize: 11, color: 'var(--fg-dim)' }}>
          {relativeTime(email.created_at || email.received_at || email.timestamp)}
        </span>
      </div>

      {/* Magic link action strip */}
      {magicLink && (
        <div style={{
          padding: '8px 14px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          flexShrink: 0,
        }}>
          <a
            href={magicLink}
            target="_blank"
            rel="noopener noreferrer"
            className="btn sm"
            style={{ textDecoration: 'none', fontSize: 11.5, gap: 5 }}
          >
            <Icon.External width={10} height={10}/>
            Open link
          </a>
          <button
            className="btn ghost sm"
            style={{ fontSize: 11.5 }}
            onClick={copyMagicLink}
          >
            {copied ? '✓ Copied' : 'Copy link'}
          </button>
          <span className="mono faint" style={{ fontSize: 9.5, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {magicLink.replace(/^https?:\/\/[^/]+/, '')}
          </span>
        </div>
      )}

      {/* View tabs */}
      <div style={{
        display: 'flex',
        borderBottom: '1px solid var(--hairline)',
        flexShrink: 0,
      }}>
        {(['preview', 'text', 'raw'] as const).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            style={{
              flex: 1,
              height: 30,
              fontSize: 11,
              fontWeight: tab === t ? 500 : 400,
              color: tab === t ? 'var(--fg)' : 'var(--fg-dim)',
              background: tab === t ? 'var(--surface-1)' : 'transparent',
              borderBottom: tab === t ? '2px solid var(--fg)' : '2px solid transparent',
              marginBottom: -1,
              cursor: 'pointer',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            {t === 'preview' ? 'HTML' : t === 'text' ? 'Text' : 'Raw'}
          </button>
        ))}
      </div>

      {/* Body area */}
      <div style={{ flex: 1, overflow: 'auto', position: 'relative' }}>
        {detailLoading && (
          <div className="faint" style={{ padding: '14px 16px', fontSize: 11 }}>Loading…</div>
        )}
        {detailError && (
          <div style={{ padding: '14px 16px', fontSize: 11, color: 'var(--danger)' }}>
            Could not load full email: {detailError}
          </div>
        )}
        {!detailLoading && (
          <>
            {tab === 'preview' && (
              htmlBody
                ? <iframe
                    srcDoc={htmlBody}
                    sandbox="allow-same-origin"
                    style={{ width: '100%', height: '100%', border: 'none', background: '#fff', display: 'block' }}
                    title="Email preview"
                  />
                : <div className="faint" style={{ padding: '24px 16px', fontSize: 12 }}>
                    No HTML body.{textBody ? ' Switch to Text.' : ''}
                  </div>
            )}
            {tab === 'text' && (
              textBody
                ? <pre style={{
                    margin: 0, padding: 16,
                    fontSize: 11.5, lineHeight: 1.65,
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--fg)',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                  }}>{textBody}</pre>
                : <div className="faint" style={{ padding: '24px 16px', fontSize: 12 }}>No text body.</div>
            )}
            {tab === 'raw' && (
              <pre style={{
                margin: 0, padding: 16,
                fontSize: 10.5, lineHeight: 1.6,
                fontFamily: 'var(--font-mono)',
                color: 'var(--fg)',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}>
                {htmlBody || textBody || JSON.stringify(d, null, 2) || '(no content)'}
              </pre>
            )}
          </>
        )}
      </div>
    </aside>
  );
}

// ── Inline clear-confirm strip ────────────────────────────────────────────────

function ClearConfirm({ onConfirm, onCancel, loading }: {
  onConfirm: () => void;
  onCancel: () => void;
  loading: boolean;
}) {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      padding: '8px 20px',
      borderBottom: '1px solid var(--hairline)',
      background: 'var(--surface-1)',
      fontSize: 12,
      flexShrink: 0,
    }}>
      <span style={{ color: 'var(--fg-dim)' }}>Clear all captured emails?</span>
      <button
        className="btn sm"
        style={{ color: 'var(--danger)', borderColor: 'var(--danger)' }}
        onClick={onConfirm}
        disabled={loading}
      >
        {loading ? 'Clearing…' : 'Yes, clear all'}
      </button>
      <button className="btn ghost sm" onClick={onCancel} disabled={loading}>
        Cancel
      </button>
    </div>
  );
}

// ── Main page ─────────────────────────────────────────────────────────────────

export function DevEmail() {
  const { data, loading, error, refresh } = useAPI('/admin/dev/emails');
  const toast = useToast();

  const [selected, setSelected] = React.useState<any>(null);
  const [confirmClear, setConfirmClear] = React.useState(false);
  const [clearing, setClearing] = React.useState(false);
  const [search, setSearch] = React.useState('');

  // Poll every 3s for live updates
  React.useEffect(() => {
    const id = setInterval(refresh, 3000);
    return () => clearInterval(id);
  }, [refresh]);

  usePageActions({ onRefresh: refresh });

  const allEmails: any[] = React.useMemo(() => {
    return data?.emails || data?.data || data?.items || (Array.isArray(data) ? data : []);
  }, [data]);

  const emails = React.useMemo(() => {
    if (!search.trim()) return allEmails;
    const q = search.toLowerCase();
    return allEmails.filter(e =>
      (e.to || e.to_addr || e.recipient || '').toLowerCase().includes(q) ||
      (e.subject || '').toLowerCase().includes(q)
    );
  }, [allEmails, search]);

  // Deselect if cleared
  React.useEffect(() => {
    if (selected && !allEmails.find((e: any) => e.id === selected.id)) {
      setSelected(null);
    }
  }, [allEmails, selected]);

  const handleClear = async () => {
    setClearing(true);
    try {
      await API.del('/admin/dev/emails');
      setSelected(null);
      setConfirmClear(false);
      refresh();
      toast?.success('Inbox cleared');
    } catch (e: any) {
      toast?.error('Clear failed: ' + (e?.message || 'unknown error'));
    } finally {
      setClearing(false);
    }
  };

  // 404 guard: dev mode not active
  if (!loading && error && (error.includes('404') || error.includes('Not Found'))) {
    return (
      <div style={{
        display: 'flex', flexDirection: 'column', alignItems: 'center',
        justifyContent: 'center', height: '100%', gap: 10,
      }}>
        <Icon.Mail width={30} height={30} style={{ opacity: 0.2 }}/>
        <p style={{ color: 'var(--fg-dim)', fontSize: 13, margin: 0 }}>
          Dev Email is only available in <strong>dev mode</strong>.
        </p>
        <p style={{ color: 'var(--fg-faint)', fontSize: 12, margin: 0 }}>
          Set <code>server.dev_mode = true</code> in your config, or use <code>shark serve --dev</code>.
        </p>
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Dev-mode banner */}
      <div style={{
        padding: '7px 20px',
        fontSize: 11.5,
        borderBottom: '1px solid var(--hairline)',
        color: 'var(--warn)',
        background: 'var(--warn-bg, rgba(0,0,0,0))',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexShrink: 0,
      }}>
        <Icon.Warn width={12} height={12} style={{ flexShrink: 0 }}/>
        <span>Dev email capture active. Switch provider before production.</span>
      </div>

      {/* Inline confirm strip (replaces modal) */}
      {confirmClear && (
        <ClearConfirm
          onConfirm={handleClear}
          onCancel={() => setConfirmClear(false)}
          loading={clearing}
        />
      )}

      {/* Toolbar */}
      <div className="row" style={{
        padding: '10px 20px',
        borderBottom: '1px solid var(--hairline)',
        gap: 8,
        flexShrink: 0,
      }}>
        {/* Search */}
        <div style={{ position: 'relative', flex: 1, maxWidth: 320 }}>
          <Icon.Search width={12} height={12} style={{
            position: 'absolute', left: 9, top: '50%', transform: 'translateY(-50%)',
            opacity: 0.45, pointerEvents: 'none',
          }}/>
          <input
            type="text"
            placeholder="Filter by to or subject…"
            value={search}
            onChange={e => setSearch(e.target.value)}
            style={{
              width: '100%',
              height: 28,
              padding: '0 9px 0 28px',
              fontSize: 12,
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline)',
              borderRadius: 3,
              color: 'var(--fg)',
              outline: 'none',
            }}
          />
        </div>

        <div style={{ flex: 1 }}/>

        {/* Count */}
        <span className="mono faint" style={{ fontSize: 11 }}>
          {search ? `${emails.length} / ${allEmails.length}` : allEmails.length}
        </span>

        <button className="btn ghost sm" onClick={refresh} disabled={loading} title="Refresh (r)">
          <Icon.Refresh width={11} height={11}/>
          Refresh
        </button>

        <button
          className="btn ghost sm"
          onClick={() => setConfirmClear(true)}
          disabled={allEmails.length === 0 || confirmClear}
          style={{ color: allEmails.length > 0 ? 'var(--danger)' : undefined, fontSize: 12 }}
        >
          <Icon.X width={11} height={11}/>
          Clear
        </button>
      </div>

      {/* Table area */}
      <div style={{ flex: 1, overflow: 'auto', paddingRight: selected ? 420 : 0 }}>
        {loading && allEmails.length === 0 && (
          <div className="faint" style={{ padding: '10px 20px', fontSize: 11 }}>Loading…</div>
        )}
        {error && !error.includes('404') && (
          <div style={{ padding: '10px 20px', fontSize: 11, color: 'var(--danger)' }}>
            Error: {error}
          </div>
        )}

        {!loading && !error && allEmails.length === 0 && (
          <div style={{ padding: '64px 20px', textAlign: 'center' }}>
            <Icon.Mail width={26} height={26} style={{ opacity: 0.18, display: 'block', margin: '0 auto 14px' }}/>
            <p style={{ color: 'var(--fg-dim)', fontSize: 13, margin: '0 0 6px' }}>
              No emails captured yet.
            </p>
            <p style={{ color: 'var(--fg-faint)', fontSize: 11.5, margin: 0 }}>
              Trigger a magic link or signup to see them here.
            </p>
          </div>
        )}

        {!loading && !error && allEmails.length > 0 && emails.length === 0 && (
          <div style={{ padding: '40px 20px', textAlign: 'center' }}>
            <p style={{ color: 'var(--fg-dim)', fontSize: 12, margin: 0 }}>No results for "{search}"</p>
          </div>
        )}

        {emails.length > 0 && (
          <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={thStyle}>To</th>
                <th style={thStyle}>Subject</th>
                <th style={thStyle}>Type</th>
                <th style={{ ...thStyle, width: 96, textAlign: 'right' }}>Received</th>
              </tr>
            </thead>
            <tbody>
              {emails.map((email: any) => {
                const active = selected?.id === email.id;
                return (
                  <tr
                    key={email.id}
                    onClick={() => setSelected(active ? null : email)}
                    style={{
                      cursor: 'pointer',
                      background: active ? 'var(--surface-2)' : 'transparent',
                    }}
                    onMouseEnter={e => !active && ((e.currentTarget as HTMLElement).style.background = 'var(--surface-1)')}
                    onMouseLeave={e => !active && ((e.currentTarget as HTMLElement).style.background = 'transparent')}
                  >
                    <td style={tdStyle}>
                      <span className="mono" style={{ fontSize: 11.5 }}>
                        {email.to || email.to_addr || email.recipient || '—'}
                      </span>
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
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Right-side slide-over — fixed panel, not a modal */}
      {selected && (
        <EmailSlideOver email={selected} onClose={() => setSelected(null)}/>
      )}

      <CLIFooter command="shark dev email tail"/>
    </div>
  );
}
