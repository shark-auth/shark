// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { TeachEmptyState } from './TeachEmptyState'

// Webhooks page — endpoint management + delivery history
// Table + create slide-over + detail slide-over w/ config/deliveries tabs

function relativeTime(ts) {
  if (!ts) return '—';
  const diff = Date.now() - new Date(ts).getTime();
  if (diff < 0) return 'just now';
  const s = Math.floor(diff / 1000);
  if (s < 60) return s + 's ago';
  const m = Math.floor(s / 60);
  if (m < 60) return m + 'm ago';
  const h = Math.floor(m / 60);
  if (h < 24) return h + 'h ago';
  const d = Math.floor(h / 24);
  if (d < 30) return d + 'd ago';
  return new Date(ts).toLocaleDateString();
}

function statusCodeChip(code) {
  if (!code) return <span className="faint" style={{fontSize:10.5}}>—</span>;
  const cls = code >= 500 ? 'danger' : code >= 400 ? 'warn' : code >= 200 ? 'success' : '';
  return <span className={'chip ' + cls} style={{height:16, fontSize:10, padding:'0 5px'}}>{code}</span>;
}

// Aligned with backend KnownWebhookEvents (internal/api/webhook_handlers.go).
// Backend uses canonical `organization.*` and `session.revoked` names.
const COMMON_EVENTS = [
  'user.created',
  'user.updated',
  'user.deleted',
  'session.created',
  'session.revoked',
  'mfa.enabled',
  'organization.created',
  'organization.member_added',
  'organization.deleted',
];

const wThStyle = { textAlign:'left', padding:'8px 14px', fontSize:10, fontWeight:500, color:'var(--fg-dim)', borderBottom:'1px solid var(--hairline)', background:'var(--surface-0)', position:'sticky', top:0, textTransform:'uppercase', letterSpacing:'0.05em' };
const wTdStyle = { padding:'9px 14px', borderBottom:'1px solid var(--hairline)', verticalAlign:'middle' };
const segStyle = { height:28, display:'inline-flex', border:'1px solid var(--hairline-strong)', borderRadius:3, overflow:'hidden' };
const segBtn = { padding:'0 10px', height:28, fontSize:11, borderRight:'1px solid var(--hairline)' };
const labelStyle = { display:'block', fontSize:10.5, textTransform:'uppercase', letterSpacing:'0.05em', color:'var(--fg-dim)', fontWeight:500, marginBottom:5 };
const inputStyle = { width:'100%', padding:'7px 10px', background:'var(--surface-1)', border:'1px solid var(--hairline-strong)', borderRadius:3, color:'var(--fg)', fontSize:12, outline:'none', boxSizing:'border-box' };
const modalBackdrop = { position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius:6, padding:18 };
const sectionLabelStyle = { fontSize:10, textTransform:'uppercase', letterSpacing:'0.08em', color:'var(--fg-dim)', fontWeight:500, marginBottom:6 };

export function Webhooks() {
  const [selected, setSelected] = React.useState(null);
  const [createOpen, setCreateOpen] = React.useState(false);
  const [query, setQuery] = React.useState('');
  const [revealSecret, setRevealSecret] = React.useState(null);
  const [toast, setToast] = React.useState(null);

  const { data: raw, loading, refresh } = useAPI('/webhooks');
  // Backend returns {data: [...]}; tolerate legacy {webhooks: [...]} or bare array.
  const webhooks = Array.isArray(raw?.data) ? raw.data
    : Array.isArray(raw?.webhooks) ? raw.webhooks
    : Array.isArray(raw) ? raw : [];

  const filtered = webhooks.filter(w => {
    if (!query) return true;
    return w.url?.toLowerCase().includes(query.toLowerCase()) ||
      (w.description || '').toLowerCase().includes(query.toLowerCase());
  });

  const showToast = (msg, type = 'success') => {
    setToast({ msg, type });
    setTimeout(() => setToast(null), 3000);
  };

  const handleCreate = async (form) => {
    const result = await API.post('/webhooks', {
      url: form.url,
      events: form.events,
      description: form.description || undefined,
    });
    if (result?.secret) setRevealSecret(result.secret);
    refresh();
  };

  const handleDelete = async (id) => {
    await API.del('/webhooks/' + id);
    if (selected?.id === id) setSelected(null);
    refresh();
  };

  const handleUpdate = async (id, updates) => {
    const result = await API.patch('/webhooks/' + id, updates);
    // Refresh selected with updated data
    if (selected?.id === id) setSelected(prev => ({ ...prev, ...updates }));
    refresh();
    showToast('Webhook updated');
  };

  const handleTest = async (id) => {
    try {
      await API.post('/webhooks/' + id + '/test');
      showToast('Test delivery sent');
    } catch (e) {
      showToast(e?.message || 'Test failed', 'danger');
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Secret reveal banner */}
      {revealSecret && (
        <div style={{
          background: '#000', color: '#fff', padding: '12px 20px',
          display: 'flex', alignItems: 'center', gap: 12, flexShrink: 0,
          borderBottom: '2px solid var(--success)',
        }}>
          <Icon.Webhook width={14} height={14} style={{ color: 'var(--success)', flexShrink: 0 }}/>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 11, color: 'var(--success)', fontWeight: 600, marginBottom: 3, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
              Webhook secret — copy now. This is the only time it will be shown.
            </div>
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak: 'break-all' }}>{revealSecret}</span>
          </div>
          <button className="btn ghost sm" style={{ color: '#fff', borderColor: 'rgba(255,255,255,0.25)', flexShrink: 0 }}
            onClick={() => navigator.clipboard?.writeText(revealSecret)}>
            <Icon.Copy width={10} height={10}/> Copy
          </button>
          <button className="btn ghost sm" style={{ color: '#fff', borderColor: 'rgba(255,255,255,0.25)', flexShrink: 0 }}
            onClick={() => setRevealSecret(null)}>
            I've saved it
          </button>
        </div>
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: 20, right: 20, zIndex: 100,
          padding: '9px 14px', borderRadius: 5,
          background: toast.type === 'danger' ? 'var(--danger-bg)' : 'var(--success-bg)',
          border: '1px solid ' + (toast.type === 'danger' ? 'var(--danger)' : 'var(--success)'),
          color: toast.type === 'danger' ? 'var(--danger)' : 'var(--success)',
          fontSize: 12, display: 'flex', alignItems: 'center', gap: 8,
          boxShadow: 'var(--shadow-lg)',
        }}>
          {toast.type === 'danger'
            ? <Icon.Warn width={12} height={12}/>
            : <Icon.Check width={12} height={12}/>}
          {toast.msg}
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 500px' : '1fr', flex: 1, overflow: 'hidden' }}>
        <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
          {/* Header */}
          <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
            <div className="row" style={{ gap: 12 }}>
              <div style={{ flex: 1 }}>
                <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Webhooks</h1>
                <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                  Notify your app when events happen in SharkAuth
                </p>
              </div>
              <button className="btn primary" onClick={() => setCreateOpen(true)}>
                <Icon.Plus width={11} height={11}/> New webhook
              </button>
            </div>
          </div>

          {/* Toolbar */}
          <div className="row" style={{ padding: '10px 20px', gap: 8, borderBottom: '1px solid var(--hairline)' }}>
            <div className="row" style={{
              flex: 1, gap: 6, padding: '0 8px', height: 28, maxWidth: 320,
              background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 4,
            }}>
              <Icon.Search width={12} height={12} style={{opacity:0.6}}/>
              <input placeholder="Filter by URL or description…"
                value={query} onChange={e => setQuery(e.target.value)}
                style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12 }}/>
            </div>
            <div style={{flex:1}}/>
            <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {webhooks.length}</span>
          </div>

          {/* Table */}
          <div style={{ flex: 1, overflow: 'auto' }}>
            {loading ? (
              <div className="faint" style={{ padding: 40, textAlign: 'center', fontSize: 12 }}>Loading…</div>
            ) : webhooks.length === 0 ? (
              <TeachEmptyState
                icon="Webhook"
                title="No webhooks yet"
                description="Webhooks notify your services when auth events happen — user signups, logins, MFA changes."
                createLabel="New Webhook"
                onCreate={() => setCreateOpen(true)}
                cliSnippet="shark webhook create --url https://..."
              />
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                <thead>
                  <tr>
                    <th style={wThStyle}>URL</th>
                    <th style={wThStyle}>Events</th>
                    <th style={wThStyle}>Status</th>
                    <th style={wThStyle}>Last Delivery</th>
                    <th style={wThStyle}>Success 7d</th>
                    <th style={{...wThStyle, width: 80}}/>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map(w => (
                    <WebhookRow
                      key={w.id}
                      w={w}
                      selected={selected?.id === w.id}
                      onSelect={() => setSelected(w)}
                      onTest={() => handleTest(w.id)}
                      onDelete={() => handleDelete(w.id)}
                    />
                  ))}
                  {filtered.length === 0 && !loading && (
                    <tr>
                      <td colSpan={6} style={{ padding: 40, textAlign: 'center' }}>
                        <span className="faint" style={{ fontSize: 12 }}>No webhooks match the current filter.</span>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            )}
          </div>

          {/* Footer */}
          <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
            <Icon.Webhook width={11} height={11} style={{opacity:0.5}}/>
            <span className="faint">HMAC-SHA256 signed · JSON payloads</span>
            <div style={{flex:1}}/>
            <span className="faint mono">POST /v1/webhooks</span>
          </div>
        </div>

        {selected && (
          <WebhookDetail
            w={selected}
            onClose={() => setSelected(null)}
            onUpdate={handleUpdate}
            onDelete={handleDelete}
            onTest={handleTest}
          />
        )}
      </div>

      {createOpen && (
        <CreateWebhookModal
          onClose={() => setCreateOpen(false)}
          onCreate={handleCreate}
        />
      )}
    </div>
  );
}

function WebhookRow({ w, selected, onSelect, onTest, onDelete }) {
  const active = w.enabled !== false;
  const events = w.events || [];
  const MAX_CHIPS = 3;

  const successRate = w.success_rate_7d != null
    ? Math.round(w.success_rate_7d * 100) + '%'
    : '—';

  const rateColor = w.success_rate_7d != null
    ? w.success_rate_7d >= 0.95 ? 'var(--success)'
    : w.success_rate_7d >= 0.7 ? 'var(--warn)'
    : 'var(--danger)'
    : 'var(--fg-faint)';

  return (
    <tr
      onClick={onSelect}
      style={{
        cursor: 'pointer',
        background: selected ? 'var(--surface-2)' : 'transparent',
        opacity: active ? 1 : 0.6,
      }}
    >
      <td style={wTdStyle}>
        <div className="row" style={{gap: 6, maxWidth: 260}}>
          <span className="mono" style={{
            fontSize: 11, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            maxWidth: 220, display: 'block',
          }}>{w.url}</span>
          <button className="btn ghost icon sm" style={{flexShrink:0}} onClick={(e) => { e.stopPropagation(); navigator.clipboard?.writeText(w.url); }} title="Copy URL">
            <Icon.Copy width={10} height={10}/>
          </button>
        </div>
        {w.description && (
          <div className="faint" style={{fontSize:10.5, marginTop:2, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap', maxWidth:240}}>{w.description}</div>
        )}
      </td>
      <td style={wTdStyle}>
        <div className="row" style={{gap: 3, flexWrap: 'wrap', maxWidth: 200}}>
          {events.slice(0, MAX_CHIPS).map(ev => (
            <span key={ev} className="chip mono" style={{height:15, fontSize:9, padding:'0 4px'}}>{ev}</span>
          ))}
          {events.length > MAX_CHIPS && (
            <span className="faint" style={{fontSize:10}}>+{events.length - MAX_CHIPS}</span>
          )}
          {events.length === 0 && <span className="faint" style={{fontSize:10.5}}>none</span>}
        </div>
      </td>
      <td style={wTdStyle}>
        <div className="row" style={{gap:5}}>
          <span className={'dot' + (active ? ' success' : '')} style={active ? {} : {background:'var(--fg-faint)'}}/>
          <span className="faint" style={{fontSize:10.5}}>{active ? 'active' : 'disabled'}</span>
        </div>
      </td>
      <td style={wTdStyle}>
        <div className="row" style={{gap:5}}>
          <span className="mono faint" style={{fontSize:10.5}}>{relativeTime(w.last_delivery_at)}</span>
          {w.last_delivery_status && statusCodeChip(w.last_delivery_status)}
        </div>
      </td>
      <td style={wTdStyle}>
        <span className="mono" style={{fontSize:11, color:rateColor, fontVariantNumeric:'tabular-nums'}}>{successRate}</span>
      </td>
      <td style={{...wTdStyle, textAlign:'right'}}>
        <div className="row" style={{gap:4, justifyContent:'flex-end'}}>
          <button className="btn ghost icon sm" title="Test" onClick={(e) => { e.stopPropagation(); onTest(); }}>
            <Icon.Bolt width={11} height={11}/>
          </button>
          <button className="btn ghost icon sm" style={{color:'var(--danger)'}} title="Delete" onClick={(e) => { e.stopPropagation(); onDelete(); }}>
            <Icon.X width={11} height={11}/>
          </button>
        </div>
      </td>
    </tr>
  );
}

function WebhookDetail({ w, onClose, onUpdate, onDelete, onTest }) {
  const [tab, setTab] = React.useState('config');

  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)', background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 10 }}>
          <div style={{
            width: 28, height: 28, background: '#000', borderRadius: 4,
            display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
          }}>
            <Icon.Webhook width={14} height={14} style={{color:'#fff'}}/>
          </div>
          <div style={{flex:1, minWidth:0}}>
            <div style={{fontWeight:500, fontSize:13, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap'}}>{w.url}</div>
            <div className="row" style={{gap:5, marginTop:2}}>
              <span className={'dot' + ((w.enabled !== false) ? ' success' : '')} style={(w.enabled !== false) ? {} : {background:'var(--fg-faint)'}}/>
              <span className="faint" style={{fontSize:10.5}}>{w.enabled !== false ? 'active' : 'disabled'}</span>
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        {/* Tab bar + Test button */}
        <div className="row" style={{marginTop:10, gap:8}}>
          <div className="seg" style={segStyle}>
            {[['config','Config'],['deliveries','Deliveries'],['test','Test Fire'],['verify','Sig Verify']].map(([v,l]) => (
              <button key={v} onClick={() => setTab(v)}
                style={{...segBtn, background: tab===v ? '#fafafa':'var(--surface-2)', color: tab===v ? '#000':'var(--fg-muted)'}}>
                {l}
              </button>
            ))}
          </div>
          <div style={{flex:1}}/>
          <button className="btn ghost sm" onClick={() => onTest(w.id)}>
            <Icon.Bolt width={10} height={10}/> Test
          </button>
        </div>
      </div>

      <div style={{flex:1, overflowY:'auto'}}>
        {tab === 'config' && <WebhookConfigTab w={w} onUpdate={onUpdate} onDelete={onDelete}/>}
        {tab === 'deliveries' && <WebhookDeliveriesTab webhookId={w.id}/>}
        {tab === 'test' && <WebhookTestFireTab webhookId={w.id}/>}
        {tab === 'verify' && <WebhookSigVerifyTab/>}
      </div>
    </aside>
  );
}

function WebhookConfigTab({ w, onUpdate, onDelete }) {
  const [url, setUrl] = React.useState(w.url || '');
  const [description, setDescription] = React.useState(w.description || '');
  const [events, setEvents] = React.useState(w.events || []);
  const [enabled, setEnabled] = React.useState(w.enabled !== false);
  const [freeformEvent, setFreeformEvent] = React.useState('');
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [confirmDelete, setConfirmDelete] = React.useState(false);

  const toggleEvent = (ev) => {
    setEvents(prev => prev.includes(ev) ? prev.filter(e => e !== ev) : [...prev, ev]);
  };

  const addFreeform = () => {
    const ev = freeformEvent.trim();
    if (ev && !events.includes(ev)) {
      setEvents(prev => [...prev, ev]);
    }
    setFreeformEvent('');
  };

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    try {
      await onUpdate(w.id, { url, events, description: description || null, enabled });
    } catch (e) {
      setError(e?.message || 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!confirmDelete) { setConfirmDelete(true); return; }
    await onDelete(w.id);
  };

  const customEvents = events.filter(e => !COMMON_EVENTS.includes(e));

  return (
    <div style={{padding: 16, display:'flex', flexDirection:'column', gap:16}}>
      {/* URL */}
      <div>
        <label style={labelStyle}>Endpoint URL</label>
        <input value={url} onChange={e => setUrl(e.target.value)}
          style={inputStyle} placeholder="https://your-app.com/webhooks"/>
      </div>

      {/* Description */}
      <div>
        <label style={labelStyle}>Description <span className="faint" style={{fontSize:10, textTransform:'none', letterSpacing:0}}>optional</span></label>
        <input value={description} onChange={e => setDescription(e.target.value)}
          style={inputStyle} placeholder="What is this webhook for?"/>
      </div>

      {/* Events */}
      <div>
        <label style={labelStyle}>Events · {events.length} selected</label>
        <div style={{border:'1px solid var(--hairline)', borderRadius:3, overflow:'hidden'}}>
          {COMMON_EVENTS.map(ev => (
            <label key={ev} className="row" style={{padding:'6px 10px', borderBottom:'1px solid var(--hairline)', gap:10, cursor:'pointer'}}>
              <input type="checkbox" checked={events.includes(ev)} onChange={() => toggleEvent(ev)}/>
              <span className="mono" style={{fontSize:11}}>{ev}</span>
            </label>
          ))}
        </div>
        {/* Custom events */}
        {customEvents.length > 0 && (
          <div style={{marginTop:6, display:'flex', flexWrap:'wrap', gap:4}}>
            {customEvents.map(ev => (
              <span key={ev} className="chip mono" style={{height:18, fontSize:10, padding:'0 6px', gap:5}}>
                {ev}
                <button style={{background:'none',border:0,padding:0,cursor:'pointer',color:'inherit',lineHeight:1}}
                  onClick={() => setEvents(prev => prev.filter(e => e !== ev))}>
                  <Icon.X width={9} height={9}/>
                </button>
              </span>
            ))}
          </div>
        )}
        {/* Freeform add */}
        <div className="row" style={{marginTop:6, gap:6}}>
          <input value={freeformEvent} onChange={e => setFreeformEvent(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addFreeform()}
            placeholder="Add custom event…"
            style={{...inputStyle, flex:1}}/>
          <button className="btn ghost sm" onClick={addFreeform} disabled={!freeformEvent.trim()}>Add</button>
        </div>
      </div>

      {/* Status toggle */}
      <div className="row" style={{gap:10}}>
        <label style={{...labelStyle, margin:0}}>Status</label>
        <div className="seg" style={segStyle}>
          {[['true','Active'],['false','Disabled']].map(([v,l]) => (
            <button key={v} onClick={() => setEnabled(v === 'true')}
              style={{...segBtn, background: String(enabled)===v ? '#fafafa':'var(--surface-2)', color: String(enabled)===v ? '#000':'var(--fg-muted)'}}>
              {l}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div style={{padding:'8px 10px', background:'var(--surface-1)', border:'1px solid var(--danger)', borderRadius:3, color:'var(--danger)', fontSize:11.5}}>
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="row" style={{justifyContent:'space-between', marginTop:4}}>
        <button
          className={'btn sm' + (confirmDelete ? ' danger' : ' ghost')}
          style={confirmDelete ? {} : {color:'var(--danger)'}}
          onClick={handleDelete}
        >
          {confirmDelete ? 'Confirm delete' : 'Delete'}
        </button>
        {confirmDelete && (
          <button className="btn ghost sm" onClick={() => setConfirmDelete(false)}>Cancel</button>
        )}
        <div style={{flex:1}}/>
        <button className="btn primary sm" onClick={handleSave} disabled={!url || saving}>
          {saving ? 'Saving…' : 'Save changes'}
        </button>
      </div>
    </div>
  );
}

function WebhookDeliveriesTab({ webhookId }) {
  const { data: raw, loading } = useAPI('/webhooks/' + webhookId + '/deliveries');
  // Backend returns {data, next_cursor}; tolerate legacy shapes.
  const deliveries = Array.isArray(raw?.data) ? raw.data
    : Array.isArray(raw?.deliveries) ? raw.deliveries
    : Array.isArray(raw) ? raw : [];
  const [expanded, setExpanded] = React.useState(null);

  if (loading) {
    return <div className="faint" style={{padding:40, textAlign:'center', fontSize:12}}>Loading…</div>;
  }

  if (deliveries.length === 0) {
    return (
      <div style={{padding:'40px 20px', textAlign:'center'}}>
        <div className="faint" style={{fontSize:12}}>No deliveries yet.</div>
      </div>
    );
  }

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11.5 }}>
      <thead>
        <tr>
          <th style={wThStyle}>Event</th>
          <th style={wThStyle}>Time</th>
          <th style={wThStyle}>Status</th>
          <th style={wThStyle}>Attempts</th>
          <th style={wThStyle}>Duration</th>
          <th style={{...wThStyle, width:28}}/>
        </tr>
      </thead>
      <tbody>
        {deliveries.map(d => {
          const isOpen = expanded === d.id;
          return (
            <React.Fragment key={d.id}>
              <tr
                onClick={() => setExpanded(isOpen ? null : d.id)}
                style={{ cursor: 'pointer' }}
              >
                <td style={wTdStyle}>
                  <span className="mono" style={{fontSize:10.5}}>{d.event || '—'}</span>
                </td>
                <td style={wTdStyle}>
                  <span className="mono faint" style={{fontSize:10.5}}>{relativeTime(d.created_at || d.timestamp)}</span>
                </td>
                <td style={wTdStyle}>
                  {statusCodeChip(d.status_code || d.response_status)}
                </td>
                <td style={wTdStyle}>
                  <span className="mono faint" style={{fontSize:10.5}}>{d.attempts ?? '—'}</span>
                </td>
                <td style={wTdStyle}>
                  <span className="mono faint" style={{fontSize:10.5}}>
                    {d.duration_ms != null ? d.duration_ms + 'ms' : '—'}
                  </span>
                </td>
                <td style={wTdStyle}>
                  <Icon.ChevronRight width={11} height={11} style={{
                    opacity:0.5, transform: isOpen ? 'rotate(90deg)' : 'none',
                    transition:'transform 120ms',
                  }}/>
                </td>
              </tr>
              {isOpen && (
                <tr>
                  <td colSpan={6} style={{padding:0, background:'var(--surface-1)', borderBottom:'1px solid var(--hairline)'}}>
                    <DeliveryExpanded d={d}/>
                  </td>
                </tr>
              )}
            </React.Fragment>
          );
        })}
      </tbody>
    </table>
  );
}

function DeliveryExpanded({ d }) {
  const [replayErr, setReplayErr] = React.useState(null);
  const [replayOk, setReplayOk] = React.useState(false);
  const prettyJson = (val) => {
    if (!val) return '—';
    if (typeof val === 'string') {
      try { return JSON.stringify(JSON.parse(val), null, 2); }
      catch { return val; }
    }
    try { return JSON.stringify(val, null, 2); }
    catch { return String(val); }
  };

  return (
    <div style={{padding:'12px 16px', display:'flex', flexDirection:'column', gap:12}}>
      {(d.request_body || d.payload) && (
        <div>
          <div style={sectionLabelStyle}>Request body</div>
          <pre style={{
            margin:0, padding:'8px 10px', borderRadius:3,
            background:'var(--surface-2)', border:'1px solid var(--hairline)',
            fontSize:10.5, fontFamily:'var(--font-mono)',
            overflow:'auto', maxHeight:160, color:'var(--fg-muted)',
            whiteSpace:'pre-wrap', wordBreak:'break-all',
          }}>{prettyJson(d.request_body || d.payload)}</pre>
        </div>
      )}
      {(d.response_body || d.response) && (
        <div>
          <div style={sectionLabelStyle}>Response body</div>
          <pre style={{
            margin:0, padding:'8px 10px', borderRadius:3,
            background:'var(--surface-2)', border:'1px solid var(--hairline)',
            fontSize:10.5, fontFamily:'var(--font-mono)',
            overflow:'auto', maxHeight:160, color:'var(--fg-muted)',
            whiteSpace:'pre-wrap', wordBreak:'break-all',
          }}>{prettyJson(d.response_body || d.response)}</pre>
        </div>
      )}
      {d.error && (
        <div style={{color:'var(--danger)', fontSize:11.5}}>
          <span className="faint" style={{marginRight:6}}>Error:</span>{d.error}
        </div>
      )}
      <div className="row" style={{ gap: 6, alignItems:'center' }}>
        <button className="btn sm" onClick={async () => {
          setReplayErr(null); setReplayOk(false);
          try {
            await API.post('/webhooks/' + d.webhook_id + '/deliveries/' + d.id + '/replay');
            setReplayOk(true);
          } catch (e) {
            setReplayErr(e?.message || 'Replay failed');
          }
        }}>
          <Icon.Refresh width={10} height={10}/> Replay
        </button>
        {replayErr && <span style={{color:'var(--danger)', fontSize:11.5}}>{replayErr}</span>}
        {replayOk && <span style={{color:'var(--success)', fontSize:11.5}}>Replay queued</span>}
      </div>
    </div>
  );
}

function WebhookTestFireTab({ webhookId }) {
  const [event, setEvent] = React.useState('user.created');
  const [firing, setFiring] = React.useState(false);
  const [result, setResult] = React.useState(null);

  // Aligned with backend KnownWebhookEvents.
  const events = ['user.created','user.updated','user.deleted','session.created','session.revoked','mfa.enabled','organization.created','organization.member_added','organization.deleted','webhook.test'];

  const handleFire = async () => {
    setFiring(true); setResult(null);
    try {
      const res = await API.post('/webhooks/' + webhookId + '/test', { event_type: event });
      setResult({ ok: true, status: res?.status_code || res?.status || 200 });
    } catch (e) {
      setResult({ ok: false, error: e.message });
    } finally {
      setFiring(false);
    }
  };

  return (
    <div style={{ padding: 16 }}>
      <div style={{ fontSize: 12, color: 'var(--fg-muted)', marginBottom: 12, lineHeight: 1.5 }}>
        Fire a test event with sample payload. Delivery appears in Deliveries tab.
      </div>
      <div className="row" style={{ gap: 8, marginBottom: 12 }}>
        <select value={event} onChange={e => setEvent(e.target.value)}
          style={{
            height: 28, padding: '0 8px', background: 'var(--surface-2)',
            border: '1px solid var(--hairline-strong)', borderRadius: 5,
            color: 'var(--fg)', fontSize: 12, fontFamily: 'var(--font-mono)',
            colorScheme: 'dark',
          }}>
          {events.map(e => <option key={e} value={e}>{e}</option>)}
        </select>
        <button className="btn sm" onClick={handleFire} disabled={firing}>
          <Icon.Bolt width={10} height={10}/>{firing ? 'Firing…' : 'Fire test event'}
        </button>
      </div>
      {result && (
        <div className={'chip ' + (result.ok ? 'success' : 'danger')} style={{ height: 20, fontSize: 11 }}>
          {result.ok ? `Delivered · ${result.status}` : `Failed: ${result.error}`}
        </div>
      )}
    </div>
  );
}

function WebhookSigVerifyTab() {
  const [payload, setPayload] = React.useState('');
  const [signature, setSignature] = React.useState('');
  const [secret, setSecret] = React.useState('');
  const [result, setResult] = React.useState(null);

  const handleVerify = async () => {
    if (!payload || !signature || !secret) return;
    try {
      const enc = new TextEncoder();
      const key = await crypto.subtle.importKey('raw', enc.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign']);
      const sig = await crypto.subtle.sign('HMAC', key, enc.encode(payload));
      const computed = Array.from(new Uint8Array(sig)).map(b => b.toString(16).padStart(2, '0')).join('');
      const inputSig = signature.replace(/^sha256=/, '').toLowerCase();
      setResult(computed === inputSig ? 'valid' : 'invalid');
    } catch {
      setResult('error');
    }
  };

  return (
    <div style={{ padding: 16 }}>
      <div style={{ fontSize: 12, color: 'var(--fg-muted)', marginBottom: 12, lineHeight: 1.5 }}>
        Paste a webhook payload + signature to verify HMAC-SHA256. Runs locally in browser.
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        <div>
          <label style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', display: 'block', marginBottom: 4 }}>Webhook secret</label>
          <input value={secret} onChange={e => setSecret(e.target.value)} placeholder="whsec_..."
            style={{ width: '100%', height: 28, padding: '0 8px', background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)', borderRadius: 4, fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg)' }}/>
        </div>
        <div>
          <label style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', display: 'block', marginBottom: 4 }}>Payload body</label>
          <textarea value={payload} onChange={e => setPayload(e.target.value)} rows={4} placeholder='{"event":"user.created",...}'
            style={{ width: '100%', padding: 8, background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)', borderRadius: 4, fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg)', resize: 'vertical' }}/>
        </div>
        <div>
          <label style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', display: 'block', marginBottom: 4 }}>Signature header</label>
          <input value={signature} onChange={e => setSignature(e.target.value)} placeholder="sha256=abc123..."
            style={{ width: '100%', height: 28, padding: '0 8px', background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)', borderRadius: 4, fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg)' }}/>
        </div>
        <div className="row" style={{ gap: 8 }}>
          <button className="btn sm" onClick={handleVerify} disabled={!payload || !signature || !secret}>Verify</button>
          {result === 'valid' && <span className="chip success" style={{ height: 20, fontSize: 11 }}>Valid signature</span>}
          {result === 'invalid' && <span className="chip danger" style={{ height: 20, fontSize: 11 }}>Invalid — signature mismatch</span>}
          {result === 'error' && <span className="chip danger" style={{ height: 20, fontSize: 11 }}>Verification error</span>}
        </div>
      </div>
    </div>
  );
}

function CreateWebhookModal({ onClose, onCreate }) {
  const [url, setUrl] = React.useState('');
  const [description, setDescription] = React.useState('');
  const [selectedEvents, setSelectedEvents] = React.useState(new Set(['user.created']));
  const [freeformEvent, setFreeformEvent] = React.useState('');
  const [customEvents, setCustomEvents] = React.useState([]);
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState(null);

  const toggleEvent = (ev) => {
    const next = new Set(selectedEvents);
    if (next.has(ev)) next.delete(ev); else next.add(ev);
    setSelectedEvents(next);
  };

  const addFreeform = () => {
    const ev = freeformEvent.trim();
    if (ev && !selectedEvents.has(ev) && !customEvents.includes(ev)) {
      setCustomEvents(prev => [...prev, ev]);
      setSelectedEvents(prev => new Set([...prev, ev]));
    }
    setFreeformEvent('');
  };

  const allSelectedEvents = [...selectedEvents];

  const handleSubmit = async () => {
    setSubmitting(true);
    setError(null);
    try {
      await onCreate({ url, events: allSelectedEvents, description });
      onClose();
    } catch (e) {
      setError(e?.message || 'Failed to create webhook');
      setSubmitting(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 520, maxHeight: '85vh', overflow:'auto'}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize:15, fontWeight:600}}>New webhook</h2>
        <p className="faint" style={{fontSize:12, marginTop:6}}>
          SharkAuth will POST signed JSON payloads to this URL when the selected events occur.
        </p>

        <div style={{marginTop:16}}>
          <label style={labelStyle}>Endpoint URL <span style={{color:'var(--danger)'}}>*</span></label>
          <input value={url} onChange={e => setUrl(e.target.value)}
            placeholder="https://your-app.com/webhooks/sharkauth"
            style={inputStyle}/>
        </div>

        <div style={{marginTop:14}}>
          <label style={labelStyle}>Description <span className="faint" style={{fontSize:10, textTransform:'none', letterSpacing:0}}>optional</span></label>
          <input value={description} onChange={e => setDescription(e.target.value)}
            placeholder="What is this webhook for?"
            style={inputStyle}/>
        </div>

        <div style={{marginTop:14}}>
          <label style={labelStyle}>Events · {allSelectedEvents.length} selected</label>
          <div style={{border:'1px solid var(--hairline)', borderRadius:3, maxHeight:220, overflow:'auto'}}>
            {COMMON_EVENTS.map(ev => (
              <label key={ev} className="row" style={{padding:'6px 10px', borderBottom:'1px solid var(--hairline)', gap:10, cursor:'pointer'}}>
                <input type="checkbox" checked={selectedEvents.has(ev)} onChange={() => toggleEvent(ev)}/>
                <span className="mono" style={{fontSize:11, flex:1}}>{ev}</span>
              </label>
            ))}
            {customEvents.map(ev => (
              <label key={ev} className="row" style={{padding:'6px 10px', borderBottom:'1px solid var(--hairline)', gap:10, cursor:'pointer'}}>
                <input type="checkbox" checked={selectedEvents.has(ev)} onChange={() => toggleEvent(ev)}/>
                <span className="mono" style={{fontSize:11, flex:1}}>{ev}</span>
                <span className="chip" style={{height:14, fontSize:9, padding:'0 4px'}}>custom</span>
              </label>
            ))}
          </div>
          {/* Freeform */}
          <div className="row" style={{marginTop:6, gap:6}}>
            <input value={freeformEvent} onChange={e => setFreeformEvent(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && (e.preventDefault(), addFreeform())}
              placeholder="Add custom event type…"
              style={{...inputStyle, flex:1}}/>
            <button className="btn ghost sm" onClick={addFreeform} disabled={!freeformEvent.trim()}>Add</button>
          </div>
        </div>

        {error && (
          <div style={{marginTop:12, padding:'8px 10px', background:'var(--surface-1)', border:'1px solid var(--danger)', borderRadius:3, color:'var(--danger)', fontSize:11.5}}>
            {error}
          </div>
        )}

        <div className="row" style={{marginTop:20, justifyContent:'flex-end', gap:8}}>
          <button className="btn ghost" onClick={onClose} disabled={submitting}>Cancel</button>
          <button className="btn primary" onClick={handleSubmit} disabled={!url || submitting}>
            {submitting ? 'Creating…' : 'Create webhook'}
          </button>
        </div>
      </div>
    </div>
  );
}


