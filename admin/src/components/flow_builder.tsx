// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// ---------------------------------------------------------------------------
// Phase 6 F4 — Auth Flow Builder
// ---------------------------------------------------------------------------
// Top-level: FlowBuilder renders a list of flows OR an editor for a selected
// flow. The list uses the Shark table convention (see agents_manage.tsx);
// the editor is a three-pane layout (palette · canvas · config) with tabs
// for Steps, Conditions, Preview, and History.
//
// MVP simplifications (document, don't skip):
//   - TODO(F4.1): drag-and-drop reordering. Insertion is palette-click only.
//   - TODO(F4.1): forked canvas for conditional branches. We render the
//     conditional step as a regular card with nested then/else lists
//     indented beneath it.
//   - TODO(F4.1): canvas keyboard nav (Delete / ↑↓ / Enter).
// ---------------------------------------------------------------------------

const TRIGGERS = [
  { id: 'signup',         label: 'signup' },
  { id: 'login',          label: 'login' },
  { id: 'oauth_callback', label: 'oauth_callback' },
  { id: 'password_reset', label: 'password_reset' },
  { id: 'magic_link',     label: 'magic_link' },
];

const STEP_TYPES = {
  // Block — red dot
  require_email_verification: { family: 'block',  label: 'Require email verification', wired: true },
  require_mfa_enrollment:     { family: 'block',  label: 'Require MFA enrollment',     wired: true },
  require_mfa_challenge:      { family: 'block',  label: 'Require MFA challenge',      wired: false },
  require_password_strength:  { family: 'block',  label: 'Require password strength',  wired: true },
  custom_check:               { family: 'block',  label: 'Custom check',               wired: false },
  // Side effect — amber dot
  webhook:                    { family: 'effect', label: 'Webhook',                    wired: true },
  set_metadata:               { family: 'effect', label: 'Set metadata',               wired: false },
  assign_role:                { family: 'effect', label: 'Assign role',                wired: false },
  add_to_org:                 { family: 'effect', label: 'Add to org',                 wired: false },
  delay:                      { family: 'effect', label: 'Delay',                      wired: false },
  // Branch — purple dot
  conditional:                { family: 'branch', label: 'Conditional',                wired: true },
  redirect:                   { family: 'branch', label: 'Redirect',                   wired: true },
};

const FAMILIES = {
  block:  { label: 'Block',        color: 'var(--danger)' },
  prompt: { label: 'Prompt',       color: 'oklch(0.74 0.14 250)' },
  effect: { label: 'Side effect',  color: 'var(--warn)' },
  branch: { label: 'Branch',       color: 'oklch(0.74 0.14 310)' },
};

const PALETTE_LAYOUT = [
  { family: 'block',  items: ['require_email_verification','require_mfa_enrollment','require_mfa_challenge','require_password_strength','custom_check'] },
  { family: 'effect', items: ['webhook','set_metadata','assign_role','add_to_org','delay'] },
  { family: 'branch', items: ['conditional','redirect'] },
];

function relativeTime(val) {
  if (!val) return '—';
  const ms = typeof val === 'string' ? new Date(val).getTime() : val;
  const diff = Date.now() - ms;
  if (diff < 0) return 'just now';
  if (diff < 60e3) return Math.floor(diff / 1e3) + 's ago';
  if (diff < 3600e3) return Math.floor(diff / 60e3) + 'm ago';
  if (diff < 86400e3) return Math.floor(diff / 3600e3) + 'h ago';
  return Math.floor(diff / 86400e3) + 'd ago';
}

function formatDuration(ns) {
  if (ns == null) return '—';
  if (ns < 1e6) return (ns / 1e3).toFixed(0) + 'µs';
  if (ns < 1e9) return (ns / 1e6).toFixed(1) + 'ms';
  return (ns / 1e9).toFixed(2) + 's';
}

// Count steps deeply (including then/else branches).
function countSteps(steps) {
  if (!Array.isArray(steps)) return 0;
  let n = 0;
  for (const s of steps) {
    n += 1;
    if (s?.type === 'conditional') {
      n += countSteps(s.then);
      n += countSteps(s.else);
    }
  }
  return n;
}

// Icon for a family dot (decorative only).
function FamilyDot({ family }) {
  const color = FAMILIES[family]?.color || 'var(--fg-dim)';
  return <span style={{
    display:'inline-block', width:6, height:6, borderRadius:99,
    background: color, flexShrink: 0,
  }}/>;
}

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

export function FlowBuilder() {
  const [selectedId, setSelectedId] = React.useState(null);

  if (!selectedId) {
    return <FlowsList onSelect={setSelectedId}/>;
  }
  return <FlowEditor id={selectedId} onBack={() => setSelectedId(null)}/>;
}

// ---------------------------------------------------------------------------
// FlowsList
// ---------------------------------------------------------------------------

const thStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const tdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };

function FlowsList({ onSelect }) {
  const toast = useToast();
  const [filter, setFilter] = React.useState('all');
  const [query, setQuery] = React.useState('');
  const [creating, setCreating] = React.useState(false);

  const path = filter === 'all' ? '/admin/flows' : `/admin/flows?trigger=${filter}`;
  const { data, loading, refresh } = useAPI(path);
  const flows = data?.data || [];

  const filtered = flows.filter(f => {
    if (!query) return true;
    const q = query.toLowerCase();
    return (f.name || '').toLowerCase().includes(q) || (f.id || '').toLowerCase().includes(q);
  });

  const createNew = async () => {
    setCreating(true);
    try {
      const created = await API.post('/admin/flows', {
        name: 'Untitled flow',
        trigger: filter === 'all' ? 'signup' : filter,
        steps: [],
        enabled: false,
        priority: 10,
        conditions: {},
      });
      toast.success('Flow created');
      refresh();
      onSelect(created.id);
    } catch (e) {
      toast.error(e.message || 'Failed to create flow');
    } finally {
      setCreating(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600, fontFamily: 'var(--font-display)' }}>Flows</h1>
            <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
              Customize what happens after signup, login, password reset, OAuth callback, and magic link verification.
            </p>
          </div>
          <span className="chip" style={{ height: 22 }}>
            {flows.length} {flows.length === 1 ? 'flow' : 'flows'}
          </span>
          <button className="btn primary" disabled={creating} onClick={createNew}>
            <Icon.Plus width={11} height={11}/> New flow
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
          <input placeholder="Filter by name or id…"
            value={query} onChange={e => setQuery(e.target.value)}
            style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12 }}/>
        </div>
        <div className="row" style={{ gap: 2, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', padding: 2, height: 28 }}>
          {[['all', 'All'], ...TRIGGERS.map(t => [t.id, t.label])].map(([v, l]) => (
            <button key={v} onClick={() => setFilter(v)} style={{
              padding: '0 10px', fontSize: 11, height: 22,
              background: filter === v ? 'var(--surface-3)' : 'transparent',
              color: filter === v ? 'var(--fg)' : 'var(--fg-muted)',
              border: 0, borderRadius: 3, cursor: 'pointer', fontWeight: filter === v ? 500 : 400,
              fontFamily: filter === v ? 'var(--font-sans)' : 'var(--font-mono)',
            }}>{l}</button>
          ))}
        </div>
        <div style={{flex:1}}/>
        <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {flows.length}</span>
      </div>

      {/* Table */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {loading ? (
          <div className="faint" style={{padding: 20, fontSize: 12}}>Loading flows…</div>
        ) : filtered.length === 0 ? (
          <div style={{ display:'flex', alignItems:'center', justifyContent:'center', height:'100%', padding: 40, textAlign:'center' }}>
            <div style={{ maxWidth: 440 }}>
              <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--fg)' }}>
                {flows.length === 0 ? 'No flows configured.' : 'No flows match your filters.'}
              </div>
              {flows.length === 0 && (
                <div style={{ marginTop: 8, fontSize: 12.5, color: 'var(--fg-dim)', lineHeight: 1.6 }}>
                  Flows customize what happens after signup, login, password reset, OAuth callback, and magic link verification. Start with a signup flow.
                </div>
              )}
            </div>
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={thStyle}>Name</th>
                <th style={thStyle}>Trigger</th>
                <th style={thStyle}>Steps</th>
                <th style={thStyle}>Priority</th>
                <th style={thStyle}>Status</th>
                <th style={thStyle}>Updated</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(f => (
                <tr key={f.id}
                  onClick={() => onSelect(f.id)}
                  style={{ cursor:'pointer' }}
                  onMouseEnter={e => e.currentTarget.style.background = 'var(--surface-1)'}
                  onMouseLeave={e => e.currentTarget.style.background = 'transparent'}>
                  <td style={tdStyle}>
                    <div style={{ fontWeight: 500 }}>{f.name || 'Untitled'}</div>
                    <div className="mono faint" style={{ fontSize: 10.5, marginTop: 1 }}>{f.id}</div>
                  </td>
                  <td style={tdStyle}>
                    <span className="chip" style={{ height: 18, fontSize: 10 }}>{f.trigger}</span>
                  </td>
                  <td style={tdStyle}>
                    <span className="chip" style={{ height: 18, fontSize: 10 }}>
                      {countSteps(f.steps)} {countSteps(f.steps) === 1 ? 'step' : 'steps'}
                    </span>
                  </td>
                  <td style={tdStyle}>
                    <span className="mono faint" style={{ fontSize: 11 }}>{f.priority ?? 0}</span>
                  </td>
                  <td style={tdStyle}>
                    {f.enabled
                      ? <span className="chip success" style={{ height: 18, fontSize: 10 }}><span className="dot success"/>enabled</span>
                      : <span className="chip" style={{ height: 18, fontSize: 10 }}><span className="dot"/>disabled</span>}
                  </td>
                  <td style={tdStyle}>
                    <span className="mono faint" style={{ fontSize: 11 }}>{relativeTime(f.updated_at)}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <CLIFooter command={filter === 'all' ? 'shark flows list' : `shark flows list --trigger ${filter}`}/>
    </div>
  );
}

// ---------------------------------------------------------------------------
// FlowEditor
// ---------------------------------------------------------------------------

function FlowEditor({ id, onBack }) {
  const toast = useToast();
  const { data: flow, loading, refresh, setData: setFlow } = useAPI(`/admin/flows/${id}`);
  const [draft, setDraft] = React.useState(null);
  const [tab, setTab] = React.useState('steps');
  const [selectedPath, setSelectedPath] = React.useState(null); // path = number[]: indices into top->then/else
  const [saving, setSaving] = React.useState(false);
  const [enableConfirm, setEnableConfirm] = React.useState(false);
  const [deleteConfirm, setDeleteConfirm] = React.useState(false);

  React.useEffect(() => {
    if (flow) setDraft(JSON.parse(JSON.stringify(flow)));
  }, [flow]);

  const dirty = React.useMemo(() => {
    if (!flow || !draft) return false;
    return JSON.stringify(flow) !== JSON.stringify(draft);
  }, [flow, draft]);

  if (loading || !draft) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div className="faint" style={{ padding: 20, fontSize: 12 }}>Loading flow…</div>
      </div>
    );
  }

  const saveDraft = async (patchOverride) => {
    setSaving(true);
    try {
      const patch = patchOverride || {
        name: draft.name,
        trigger: draft.trigger,
        steps: draft.steps,
        enabled: draft.enabled,
        priority: draft.priority,
        conditions: draft.conditions || {},
      };
      const updated = await API.patch(`/admin/flows/${id}`, patch);
      setFlow(updated);
      toast.success('Flow saved');
    } catch (e) {
      toast.error(e.message || 'Failed to save flow');
    } finally {
      setSaving(false);
    }
  };

  const revertDraft = () => {
    if (flow) setDraft(JSON.parse(JSON.stringify(flow)));
    toast.info('Reverted');
  };

  const deleteFlow = async () => {
    try {
      await API.del(`/admin/flows/${id}`);
      toast.success('Flow deleted');
      onBack();
    } catch (e) {
      toast.error(e.message || 'Failed to delete flow');
    }
  };

  const toggleEnabled = async () => {
    if (!draft.enabled) { // about to enable
      setEnableConfirm(true);
      return;
    }
    const next = { ...draft, enabled: false };
    setDraft(next);
    await saveDraft({ enabled: false });
  };

  const confirmEnable = async () => {
    const next = { ...draft, enabled: true };
    setDraft(next);
    setEnableConfirm(false);
    await saveDraft({ enabled: true });
  };

  // ----- steps manipulation helpers -----

  // path is a sequence of {index, branch?} or simple indices. We keep it
  // as a plain array of numbers interleaved with 'then'|'else' strings:
  //   top-level step 2 → [2]
  //   then-branch of step 2, step 0 → [2, 'then', 0]
  //   else-branch of step 2, then-branch of step 0, step 1 → [2, 'else', 0, 'then', 1]
  const getStepsRef = (steps, path) => {
    if (path.length === 0) return { parent: null, list: steps, index: -1 };
    const p = [...path];
    let list = steps;
    let parent = null;
    while (p.length > 1) {
      const idx = p.shift();
      const branch = p.shift(); // 'then' or 'else'
      parent = list[idx];
      list = parent[branch] = parent[branch] || [];
    }
    return { parent, list, index: p[0] };
  };

  const getStepAt = (steps, path) => {
    const { list, index } = getStepsRef(steps, path);
    return list[index];
  };

  const appendStep = (type) => {
    const cfg = { type, config: {} };
    if (type === 'conditional') { cfg.condition = ''; cfg.then = []; cfg.else = []; }
    const next = JSON.parse(JSON.stringify(draft));
    if (!selectedPath || selectedPath.length === 0) {
      next.steps.push(cfg);
      setDraft(next);
      setSelectedPath([next.steps.length - 1]);
      return;
    }
    // Insert after the selected node at its level
    const { list, index } = getStepsRef(next.steps, selectedPath);
    list.splice(index + 1, 0, cfg);
    setDraft(next);
    const newPath = [...selectedPath.slice(0, -1), index + 1];
    setSelectedPath(newPath);
  };

  const deleteStep = (path) => {
    const next = JSON.parse(JSON.stringify(draft));
    const { list, index } = getStepsRef(next.steps, path);
    list.splice(index, 1);
    setDraft(next);
    setSelectedPath(null);
  };

  const duplicateStep = (path) => {
    const next = JSON.parse(JSON.stringify(draft));
    const { list, index } = getStepsRef(next.steps, path);
    const copy = JSON.parse(JSON.stringify(list[index]));
    list.splice(index + 1, 0, copy);
    setDraft(next);
    setSelectedPath([...path.slice(0, -1), index + 1]);
  };

  const moveStep = (path, delta) => {
    const next = JSON.parse(JSON.stringify(draft));
    const { list, index } = getStepsRef(next.steps, path);
    const target = index + delta;
    if (target < 0 || target >= list.length) return;
    [list[index], list[target]] = [list[target], list[index]];
    setDraft(next);
    setSelectedPath([...path.slice(0, -1), target]);
  };

  const updateStepConfig = (path, patch) => {
    const next = JSON.parse(JSON.stringify(draft));
    const step = getStepAt(next.steps, path);
    Object.assign(step, patch);
    setDraft(next);
  };

  const selectedStep = selectedPath ? getStepAt(draft.steps, selectedPath) : null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header bar */}
      <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)' }}>
        <div className="row" style={{ gap: 10 }}>
          <button className="btn ghost icon sm" onClick={onBack} title="Back to flows">
            <Icon.ChevronRight width={11} height={11} style={{ transform: 'rotate(180deg)' }}/>
          </button>
          <InlineName
            value={draft.name}
            onChange={v => setDraft({ ...draft, name: v })}
          />
          <TriggerPicker
            value={draft.trigger}
            onChange={v => setDraft({ ...draft, trigger: v })}
          />
          <span className="faint mono" style={{ fontSize: 10.5 }}>{draft.id}</span>
          <div style={{ flex: 1 }}/>
          {dirty && (
            <>
              <button className="btn ghost sm" onClick={revertDraft} disabled={saving}>Revert</button>
              <button className="btn primary sm" onClick={() => saveDraft()} disabled={saving}>
                {saving ? 'Saving…' : 'Save'}
              </button>
            </>
          )}
          <label className="row" style={{ gap: 6, cursor: 'pointer' }} onClick={toggleEnabled}>
            <span className={'dot' + (draft.enabled ? ' success' : '')} style={!draft.enabled ? { background: 'var(--fg-faint)' } : {}}/>
            <span className="mono" style={{ fontSize: 11, color: draft.enabled ? 'var(--success)' : 'var(--fg-muted)' }}>
              {draft.enabled ? 'enabled' : 'disabled'}
            </span>
          </label>
          <button className="btn ghost sm danger" onClick={() => setDeleteConfirm(true)}>
            <Icon.X width={11} height={11}/>
          </button>
        </div>

        {/* Tabs */}
        <div className="row" style={{ gap: 2, marginTop: 10, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', padding: 2, width: 'fit-content' }}>
          {[['steps','Steps'],['conditions','Trigger conditions'],['preview','Preview'],['history','History']].map(([v,l]) => (
            <button key={v} onClick={() => setTab(v)} style={{
              padding: '0 12px', fontSize: 11.5, height: 22,
              background: tab === v ? '#fafafa' : 'transparent',
              color: tab === v ? '#000' : 'var(--fg-muted)',
              border: 0, borderRadius: 3, cursor: 'pointer', fontWeight: tab === v ? 600 : 400,
            }}>{l}</button>
          ))}
        </div>
      </div>

      {/* Disabled banner */}
      {!draft.enabled && tab !== 'history' && (
        <div style={{ padding: '8px 16px', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)' }}>
          <div className="row" style={{ gap: 8 }}>
            <Icon.Warn width={11} height={11} style={{ color: 'var(--warn)', flexShrink: 0 }}/>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 12, fontWeight: 500 }}>This flow is disabled.</div>
              <div className="faint" style={{ fontSize: 11 }}>
                Enable to run on <span className="mono">{draft.trigger}</span> events.
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tab body */}
      <div style={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        {tab === 'steps' && (
          <StepsTab
            draft={draft}
            selectedPath={selectedPath}
            setSelectedPath={setSelectedPath}
            selectedStep={selectedStep}
            onAppend={appendStep}
            onDelete={deleteStep}
            onDuplicate={duplicateStep}
            onMove={moveStep}
            onUpdateStep={updateStepConfig}
          />
        )}
        {tab === 'conditions' && (
          <ConditionsTab
            conditions={draft.conditions || {}}
            onChange={c => setDraft({ ...draft, conditions: c })}
          />
        )}
        {tab === 'preview' && <PreviewTab flowId={id} trigger={draft.trigger} steps={draft.steps}/>}
        {tab === 'history' && <HistoryTab flowId={id}/>}
      </div>

      <CLIFooter command={`shark flows apply ${draft.trigger}.yaml`}/>

      {enableConfirm && (
        <ConfirmDialog
          title="Enable flow?"
          body={<>Flow will run on next <span className="mono">{draft.trigger}</span> event.</>}
          confirmLabel="Enable"
          onConfirm={confirmEnable}
          onClose={() => setEnableConfirm(false)}
        />
      )}
      {deleteConfirm && (
        <ConfirmDialog
          title="Delete flow?"
          body={<>This permanently removes the flow and its run history. This cannot be undone.</>}
          confirmLabel="Delete"
          danger
          onConfirm={() => { setDeleteConfirm(false); deleteFlow(); }}
          onClose={() => setDeleteConfirm(false)}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Header widgets
// ---------------------------------------------------------------------------

function InlineName({ value, onChange }) {
  const [editing, setEditing] = React.useState(false);
  const [local, setLocal] = React.useState(value || '');
  React.useEffect(() => { setLocal(value || ''); }, [value]);

  if (editing) {
    return (
      <input
        autoFocus
        value={local}
        onChange={e => setLocal(e.target.value)}
        onBlur={() => { setEditing(false); if (local !== value) onChange(local); }}
        onKeyDown={e => {
          if (e.key === 'Enter') e.currentTarget.blur();
          if (e.key === 'Escape') { setLocal(value || ''); setEditing(false); }
        }}
        style={{
          fontFamily: 'var(--font-display)', fontWeight: 600, fontSize: 16,
          padding: '2px 6px', borderRadius: 3,
          border: '1px solid var(--hairline-bright)', background: 'var(--surface-2)',
          color: 'var(--fg)', minWidth: 200,
        }}
      />
    );
  }
  return (
    <button onClick={() => setEditing(true)} style={{
      fontFamily: 'var(--font-display)', fontWeight: 600, fontSize: 16,
      padding: '2px 6px', borderRadius: 3, cursor: 'text',
      color: 'var(--fg)',
    }}>{value || 'Untitled flow'}</button>
  );
}

function TriggerPicker({ value, onChange }) {
  const [open, setOpen] = React.useState(false);
  return (
    <div style={{ position: 'relative' }}>
      <button className="chip" onClick={() => setOpen(v => !v)} style={{ cursor: 'pointer' }}>
        {value} <Icon.ChevronDown width={9} height={9}/>
      </button>
      {open && (
        <div style={{
          position: 'absolute', top: 24, left: 0, zIndex: 10,
          background: 'var(--surface-1)', border: '1px solid var(--hairline-bright)', borderRadius: 4,
          minWidth: 160, padding: 4, boxShadow: 'var(--shadow-lg)',
        }}>
          {TRIGGERS.map(t => (
            <button key={t.id} onClick={() => { onChange(t.id); setOpen(false); }}
              className="mono" style={{
                display: 'block', width: '100%', textAlign: 'left',
                padding: '6px 8px', fontSize: 11, borderRadius: 3,
                background: t.id === value ? 'var(--surface-3)' : 'transparent',
                color: 'var(--fg)',
              }}>{t.label}</button>
          ))}
        </div>
      )}
    </div>
  );
}

function ConfirmDialog({ title, body, confirmLabel, danger, onConfirm, onClose }) {
  return (
    <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 100 }}>
      <div style={{ background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18, width: 400 }}>
        <div style={{ fontWeight: 600, fontSize: 14, marginBottom: 6, fontFamily:'var(--font-display)' }}>{title}</div>
        <div style={{ fontSize: 12.5, color: 'var(--fg-muted)', marginBottom: 14, lineHeight: 1.5 }}>{body}</div>
        <div className="row" style={{ justifyContent: 'flex-end', gap: 8 }}>
          <button className="btn ghost sm" onClick={onClose}>Cancel</button>
          <button className={'btn sm ' + (danger ? 'danger' : 'primary')} onClick={onConfirm}>{confirmLabel}</button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Steps tab (palette · canvas · config)
// ---------------------------------------------------------------------------

function StepsTab({ draft, selectedPath, setSelectedPath, selectedStep, onAppend, onDelete, onDuplicate, onMove, onUpdateStep }) {
  return (
    <div style={{
      flex: 1, display: 'grid',
      gridTemplateColumns: '200px 1fr 320px',
      minHeight: 0, overflow: 'hidden',
    }}>
      <Palette onPick={onAppend}/>
      <Canvas
        steps={draft.steps}
        trigger={draft.trigger}
        selectedPath={selectedPath}
        setSelectedPath={setSelectedPath}
        onDelete={onDelete}
        onDuplicate={onDuplicate}
        onMove={onMove}
      />
      <ConfigPanel
        step={selectedStep}
        path={selectedPath}
        onChange={(patch) => onUpdateStep(selectedPath, patch)}
        onDelete={() => { onDelete(selectedPath); }}
      />
    </div>
  );
}

function Palette({ onPick }) {
  return (
    <div style={{
      borderRight: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      overflow: 'auto',
      padding: '12px 10px',
    }}>
      {PALETTE_LAYOUT.map(section => (
        <div key={section.family} style={{ marginBottom: 16 }}>
          <div className="row" style={{ gap: 6, marginBottom: 6 }}>
            <FamilyDot family={section.family}/>
            <span style={{
              fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
              color: 'var(--fg-dim)', fontWeight: 500,
            }}>{FAMILIES[section.family].label}</span>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            {section.items.map(type => {
              const meta = STEP_TYPES[type];
              return (
                <button key={type} onClick={() => onPick(type)}
                  title={meta.label}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 6,
                    padding: '6px 8px',
                    border: '1px solid var(--hairline)',
                    background: 'var(--surface-1)',
                    borderRadius: 4, cursor: 'pointer',
                    textAlign: 'left',
                    transition: 'background 80ms ease-out, border-color 80ms ease-out',
                  }}
                  onMouseEnter={e => { e.currentTarget.style.background = 'var(--surface-2)'; e.currentTarget.style.borderColor = 'var(--hairline-strong)'; }}
                  onMouseLeave={e => { e.currentTarget.style.background = 'var(--surface-1)'; e.currentTarget.style.borderColor = 'var(--hairline)'; }}
                >
                  <span className="mono" style={{ fontSize: 10.5, color: 'var(--fg)', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {type}
                  </span>
                  {!meta.wired && (
                    <span className="chip" style={{ height: 14, padding: '0 4px', fontSize: 9, color: 'var(--warn)', borderColor: 'color-mix(in oklch, var(--warn) 35%, var(--hairline-strong))' }}>
                      preview
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

function Canvas({ steps, trigger, selectedPath, setSelectedPath, onDelete, onDuplicate, onMove }) {
  return (
    <div style={{
      overflow: 'auto',
      padding: '24px 28px',
      background: 'var(--bg)',
      display: 'flex', flexDirection: 'column', alignItems: 'center',
    }}>
      {/* Trigger pseudo-node */}
      <div style={{
        padding: '8px 14px',
        border: '1px solid var(--hairline-bright)',
        borderRadius: 20,
        background: 'var(--surface-2)',
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        color: 'var(--fg)',
        display: 'inline-flex', alignItems: 'center', gap: 6,
      }}>
        <Icon.Bolt width={10} height={10} style={{ opacity: 0.6 }}/>
        <span style={{ color: 'var(--fg-dim)' }}>trigger</span>
        <span>{trigger}</span>
      </div>

      {steps.length === 0 ? (
        <>
          <Connector/>
          <div style={{
            border: '1px dashed var(--hairline-bright)',
            borderRadius: 6,
            padding: '24px 32px',
            color: 'var(--fg-dim)',
            fontSize: 12.5,
            maxWidth: 360, textAlign: 'center',
          }}>
            Click a step in the palette to begin.
          </div>
        </>
      ) : (
        <CanvasList
          steps={steps}
          path={[]}
          selectedPath={selectedPath}
          setSelectedPath={setSelectedPath}
          onDelete={onDelete}
          onDuplicate={onDuplicate}
          onMove={onMove}
        />
      )}

      {/* Done pseudo-node */}
      {steps.length > 0 && (
        <>
          <Connector/>
          <div style={{
            padding: '6px 12px',
            border: '1px solid var(--hairline)',
            borderRadius: 20,
            background: 'transparent',
            fontFamily: 'var(--font-mono)',
            fontSize: 10.5,
            color: 'var(--fg-dim)',
          }}>done</div>
        </>
      )}
    </div>
  );
}

function Connector({ dashed }) {
  return (
    <div style={{
      width: 1, height: 20,
      borderLeft: `1px ${dashed ? 'dashed' : 'solid'} var(--hairline-strong)`,
    }}/>
  );
}

function CanvasList({ steps, path, selectedPath, setSelectedPath, onDelete, onDuplicate, onMove }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: '100%', maxWidth: 520 }}>
      {steps.map((step, i) => {
        const fullPath = [...path, i];
        const selected = selectedPath && selectedPath.length === fullPath.length && selectedPath.every((v, k) => v === fullPath[k]);
        const configured = isStepConfigured(step);
        return (
          <React.Fragment key={i}>
            <Connector dashed={!configured}/>
            <StepNode
              step={step}
              path={fullPath}
              selected={selected}
              configured={configured}
              onSelect={() => setSelectedPath(fullPath)}
              onDelete={() => onDelete(fullPath)}
              onDuplicate={() => onDuplicate(fullPath)}
              onMoveUp={() => onMove(fullPath, -1)}
              onMoveDown={() => onMove(fullPath, 1)}
            />
            {step.type === 'conditional' && (
              // MVP linear display — TODO(F4.1) forked canvas
              <div style={{ width: '100%', paddingLeft: 24, marginTop: 6 }}>
                <BranchBlock
                  label="then"
                  steps={step.then || []}
                  path={[...fullPath, 'then']}
                  selectedPath={selectedPath}
                  setSelectedPath={setSelectedPath}
                  onDelete={onDelete}
                  onDuplicate={onDuplicate}
                  onMove={onMove}
                />
                <BranchBlock
                  label="else"
                  steps={step.else || []}
                  path={[...fullPath, 'else']}
                  selectedPath={selectedPath}
                  setSelectedPath={setSelectedPath}
                  onDelete={onDelete}
                  onDuplicate={onDuplicate}
                  onMove={onMove}
                />
              </div>
            )}
          </React.Fragment>
        );
      })}
    </div>
  );
}

function BranchBlock({ label, steps, path, selectedPath, setSelectedPath, onDelete, onDuplicate, onMove }) {
  return (
    <div style={{
      marginTop: 8,
      borderLeft: '1px dashed var(--hairline-strong)',
      paddingLeft: 16,
    }}>
      <div className="mono" style={{ fontSize: 10, color: 'var(--fg-dim)', marginBottom: 4, textTransform: 'uppercase', letterSpacing: '0.08em' }}>
        {label}
      </div>
      {steps.length === 0 ? (
        <div className="faint" style={{ fontSize: 11, padding: '4px 0' }}>(no steps)</div>
      ) : (
        <CanvasList
          steps={steps}
          path={path}
          selectedPath={selectedPath}
          setSelectedPath={setSelectedPath}
          onDelete={onDelete}
          onDuplicate={onDuplicate}
          onMove={onMove}
        />
      )}
    </div>
  );
}

function isStepConfigured(step) {
  if (!step) return false;
  const t = step.type;
  const cfg = step.config || {};
  if (t === 'redirect') return !!cfg.url;
  if (t === 'webhook') return !!cfg.url;
  if (t === 'conditional') return !!step.condition;
  if (t === 'custom_check') return !!cfg.url;
  return true; // steps with sensible defaults are considered configured
}

function stepSummary(step) {
  if (!step) return '';
  const cfg = step.config || {};
  if (step.type === 'redirect') return cfg.url || '(no url)';
  if (step.type === 'webhook') return [cfg.method || 'POST', cfg.url || '(no url)'].join(' ');
  if (step.type === 'require_password_strength') return `min_length=${cfg.min_length ?? 12}`;
  if (step.type === 'conditional') return step.condition ? `if ${step.condition}` : '(no condition)';
  if (step.type === 'delay') return cfg.seconds ? `${cfg.seconds}s` : '';
  if (step.type === 'assign_role' || step.type === 'add_to_org') return cfg.role || cfg.org_id || '';
  return '';
}

function StepNode({ step, path, selected, configured, onSelect, onDelete, onDuplicate, onMoveUp, onMoveDown }) {
  const [hover, setHover] = React.useState(false);
  const [menuOpen, setMenuOpen] = React.useState(false);
  const meta = STEP_TYPES[step.type] || { family: 'effect', label: step.type, wired: false };
  const summary = stepSummary(step);

  return (
    <div
      onClick={onSelect}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => { setHover(false); setMenuOpen(false); }}
      style={{
        width: '100%', maxWidth: 440,
        padding: '10px 12px',
        background: 'var(--surface-1)',
        border: '1px solid ' + (configured ? 'var(--hairline-strong)' : 'var(--hairline)'),
        borderRadius: 6,
        cursor: 'pointer',
        position: 'relative',
        boxShadow: selected ? 'inset 0 0 0 2px var(--fg)' : 'none',
        transition: 'box-shadow 120ms ease-out, border-color 80ms ease-out',
      }}
    >
      <div className="row" style={{ gap: 8 }}>
        <FamilyDot family={meta.family}/>
        <span className="mono" style={{ fontSize: 11.5, color: 'var(--fg)', fontWeight: 500 }}>
          {step.type}
        </span>
        <span className="faint" style={{ fontSize: 11 }}>{meta.label}</span>
        <div style={{ flex: 1 }}/>
        {!meta.wired && (
          <span className="chip" style={{ height: 16, padding: '0 5px', fontSize: 9.5, color: 'var(--warn)', borderColor: 'color-mix(in oklch, var(--warn) 35%, var(--hairline-strong))' }}>
            preview
          </span>
        )}
        {hover && (
          <button className="btn ghost icon sm"
            onClick={e => { e.stopPropagation(); setMenuOpen(v => !v); }}
            title="Options">
            <Icon.More width={11} height={11}/>
          </button>
        )}
      </div>
      {summary && (
        <div className="mono faint" style={{ fontSize: 10.5, marginTop: 4, paddingLeft: 12 }}>
          {summary}
        </div>
      )}
      {menuOpen && (
        <div style={{
          position: 'absolute', top: 32, right: 8, zIndex: 5,
          background: 'var(--surface-1)', border: '1px solid var(--hairline-bright)', borderRadius: 4,
          padding: 3, minWidth: 120, boxShadow: 'var(--shadow-lg)',
        }}
          onClick={e => e.stopPropagation()}
        >
          {[
            ['Move up',   onMoveUp],
            ['Move down', onMoveDown],
            ['Duplicate', onDuplicate],
            ['Delete',    onDelete],
          ].map(([l, fn]) => (
            <button key={l} onClick={() => { fn(); setMenuOpen(false); }} style={{
              display: 'block', width: '100%', textAlign: 'left',
              padding: '6px 8px', fontSize: 11, borderRadius: 3,
              color: l === 'Delete' ? 'var(--danger)' : 'var(--fg)',
            }}>{l}</button>
          ))}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Config panel
// ---------------------------------------------------------------------------

function ConfigPanel({ step, path, onChange, onDelete }) {
  if (!step) {
    return (
      <div style={{
        borderLeft: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        padding: 20,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        overflow: 'auto',
      }}>
        <div style={{ textAlign: 'center', maxWidth: 260 }}>
          <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)' }}>Pick a step to configure.</div>
          <div style={{ marginTop: 6, fontSize: 11.5, color: 'var(--fg-dim)', lineHeight: 1.5 }}>
            Click a node on the canvas.
          </div>
        </div>
      </div>
    );
  }

  const meta = STEP_TYPES[step.type] || { family: 'effect', label: step.type, wired: false };
  const cfg = step.config || {};
  const patchConfig = (patch) => onChange({ config: { ...cfg, ...patch } });

  return (
    <div style={{
      borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      padding: 16,
      overflow: 'auto',
      display: 'flex', flexDirection: 'column', gap: 14,
    }}>
      <div className="row" style={{ gap: 6 }}>
        <FamilyDot family={meta.family}/>
        <span className="mono" style={{ fontSize: 12, color: 'var(--fg)', fontWeight: 500 }}>{step.type}</span>
        <div style={{ flex: 1 }}/>
        <button className="btn ghost icon sm danger" onClick={onDelete} title="Delete step">
          <Icon.X width={11} height={11}/>
        </button>
      </div>
      <div className="faint" style={{ fontSize: 11 }}>{meta.label}</div>
      {!meta.wired && (
        <div style={{
          padding: '8px 10px', borderRadius: 4,
          background: 'color-mix(in oklch, var(--warn) 10%, var(--surface-1))',
          border: '1px solid color-mix(in oklch, var(--warn) 30%, var(--hairline-strong))',
          fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5,
        }}>
          Preview step — engine continues with a warning. Full execution ships in a later phase.
        </div>
      )}

      <StepConfigFields step={step} cfg={cfg} patchConfig={patchConfig} onChange={onChange}/>
    </div>
  );
}

function Field({ label, hint, children }) {
  return (
    <label style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <span style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.06em', color: 'var(--fg-dim)', fontWeight: 500 }}>{label}</span>
      {children}
      {hint && <span className="faint" style={{ fontSize: 10.5, lineHeight: 1.4 }}>{hint}</span>}
    </label>
  );
}

function TextInput({ value, onChange, placeholder, mono, type = 'text' }) {
  return (
    <input type={type} value={value || ''} placeholder={placeholder}
      onChange={e => onChange(type === 'number' ? (e.target.value === '' ? null : Number(e.target.value)) : e.target.value)}
      className={mono ? 'mono' : ''}
      style={{
        padding: '6px 8px', borderRadius: 4,
        border: '1px solid var(--hairline-strong)', background: 'var(--surface-2)',
        color: 'var(--fg)', fontSize: 12, width: '100%',
      }}
    />
  );
}

function TextArea({ value, onChange, placeholder, rows = 4, mono }) {
  return (
    <textarea value={value || ''} placeholder={placeholder} rows={rows}
      onChange={e => onChange(e.target.value)}
      className={mono ? 'mono' : ''}
      style={{
        padding: '6px 8px', borderRadius: 4,
        border: '1px solid var(--hairline-strong)', background: 'var(--surface-2)',
        color: 'var(--fg)', fontSize: 12, width: '100%', resize: 'vertical',
      }}
    />
  );
}

function Checkbox({ checked, onChange, label }) {
  return (
    <label className="row" style={{ gap: 8, cursor: 'pointer', fontSize: 12 }}>
      <span className={'cb' + (checked ? ' on' : '')} onClick={() => onChange(!checked)}>
        {checked && <Icon.Check width={10} height={10}/>}
      </span>
      <span>{label}</span>
    </label>
  );
}

function Select({ value, onChange, options }) {
  return (
    <select value={value} onChange={e => onChange(e.target.value)}
      style={{
        padding: '6px 8px', borderRadius: 4,
        border: '1px solid var(--hairline-strong)', background: 'var(--surface-2)',
        color: 'var(--fg)', fontSize: 12, fontFamily: 'var(--font-mono)',
      }}>
      {options.map(o => <option key={o} value={o}>{o}</option>)}
    </select>
  );
}

function KeyValueList({ entries, onChange, keyPlaceholder = 'key', valuePlaceholder = 'value' }) {
  const rows = Object.entries(entries || {});
  const setRow = (i, k, v) => {
    const next = {};
    rows.forEach(([rk, rv], idx) => {
      if (idx === i) next[k] = v;
      else next[rk] = rv;
    });
    onChange(next);
  };
  const removeRow = (i) => {
    const next = {};
    rows.forEach(([rk, rv], idx) => { if (idx !== i) next[rk] = rv; });
    onChange(next);
  };
  const addRow = () => onChange({ ...entries, '': '' });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {rows.map(([k, v], i) => (
        <div key={i} className="row" style={{ gap: 4 }}>
          <input value={k} placeholder={keyPlaceholder}
            onChange={e => setRow(i, e.target.value, v)}
            className="mono"
            style={{ flex: 1, padding: '4px 6px', borderRadius: 3, border: '1px solid var(--hairline-strong)', background: 'var(--surface-2)', color: 'var(--fg)', fontSize: 11 }}
          />
          <input value={v} placeholder={valuePlaceholder}
            onChange={e => setRow(i, k, e.target.value)}
            className="mono"
            style={{ flex: 1, padding: '4px 6px', borderRadius: 3, border: '1px solid var(--hairline-strong)', background: 'var(--surface-2)', color: 'var(--fg)', fontSize: 11 }}
          />
          <button className="btn ghost icon sm" onClick={() => removeRow(i)}><Icon.X width={10} height={10}/></button>
        </div>
      ))}
      <button className="btn ghost sm" onClick={addRow} style={{ alignSelf: 'flex-start' }}>
        <Icon.Plus width={10} height={10}/> Add
      </button>
    </div>
  );
}

function StepConfigFields({ step, cfg, patchConfig, onChange }) {
  switch (step.type) {
    case 'require_email_verification':
      return (
        <>
          <Field label="Redirect URL" hint="Optional. Where to send the user to verify.">
            <TextInput value={cfg.redirect} onChange={v => patchConfig({ redirect: v })} placeholder="/verify-email" mono/>
          </Field>
        </>
      );
    case 'require_mfa_enrollment':
      return (
        <>
          <Checkbox checked={!!cfg.skip_if_enrolled} onChange={v => patchConfig({ skip_if_enrolled: v })} label="Skip if already enrolled"/>
        </>
      );
    case 'require_password_strength':
      return (
        <>
          <Field label="Minimum length">
            <TextInput type="number" value={cfg.min_length ?? 12} onChange={v => patchConfig({ min_length: v })}/>
          </Field>
          <Checkbox checked={!!cfg.require_special} onChange={v => patchConfig({ require_special: v })} label="Require special character"/>
        </>
      );
    case 'redirect':
      return (
        <>
          <Field label="URL" hint="Required. Absolute or relative path.">
            <TextInput value={cfg.url} onChange={v => patchConfig({ url: v })} placeholder="https://example.com/welcome" mono/>
          </Field>
          <Field label="Delay (seconds)" hint="Optional. Wait before redirecting.">
            <TextInput type="number" value={cfg.delay ?? ''} onChange={v => patchConfig({ delay: v })}/>
          </Field>
        </>
      );
    case 'webhook':
      return (
        <>
          <Field label="URL" hint="Required. Called at this step.">
            <TextInput value={cfg.url} onChange={v => patchConfig({ url: v })} placeholder="https://example.com/hook" mono/>
          </Field>
          <Field label="Method">
            <Select value={cfg.method || 'POST'} onChange={v => patchConfig({ method: v })} options={['GET','POST','PUT','DELETE']}/>
          </Field>
          <Field label="Headers">
            <KeyValueList entries={cfg.headers || {}} onChange={v => patchConfig({ headers: v })} keyPlaceholder="X-Header" valuePlaceholder="value"/>
          </Field>
          <Field label="Timeout (seconds)">
            <TextInput type="number" value={cfg.timeout ?? 5} onChange={v => patchConfig({ timeout: v })}/>
          </Field>
        </>
      );
    case 'conditional':
      return (
        <>
          <Field label="Condition" hint='Expression. Example: user.email_domain == "acme.com"'>
            <TextArea value={step.condition} onChange={v => onChange({ condition: v })} placeholder='user.email_domain == "acme.com"' rows={3} mono/>
          </Field>
          <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
            Then/else branches are managed on the canvas. Add steps inside the branch blocks below the condition node.
          </div>
        </>
      );
    case 'custom_check':
      return (
        <>
          <Field label="URL" hint="Called server-side; must return JSON {allow: true|false, reason: string}.">
            <TextInput value={cfg.url} onChange={v => patchConfig({ url: v })} placeholder="https://example.com/check" mono/>
          </Field>
          <Field label="Timeout (seconds)">
            <TextInput type="number" value={cfg.timeout ?? 5} onChange={v => patchConfig({ timeout: v })}/>
          </Field>
        </>
      );
    case 'delay':
      return (
        <>
          <Field label="Seconds" hint="Pause execution before continuing.">
            <TextInput type="number" value={cfg.seconds ?? ''} onChange={v => patchConfig({ seconds: v })}/>
          </Field>
        </>
      );
    case 'assign_role':
      return (
        <Field label="Role slug">
          <TextInput value={cfg.role} onChange={v => patchConfig({ role: v })} placeholder="admin" mono/>
        </Field>
      );
    case 'add_to_org':
      return (
        <Field label="Org ID">
          <TextInput value={cfg.org_id} onChange={v => patchConfig({ org_id: v })} placeholder="org_xxx" mono/>
        </Field>
      );
    case 'set_metadata':
      return (
        <Field label="Metadata" hint="Key/value pairs merged into user metadata.">
          <KeyValueList entries={cfg.metadata || {}} onChange={v => patchConfig({ metadata: v })}/>
        </Field>
      );
    case 'require_mfa_challenge':
      return (
        <Field label="Factor">
          <Select value={cfg.factor || 'totp'} onChange={v => patchConfig({ factor: v })} options={['totp','webauthn']}/>
        </Field>
      );
    default:
      return (
        <Field label="Raw config (JSON)" hint="Generic editor — step type has no typed fields yet.">
          <JSONEditor value={cfg} onChange={v => onChange({ config: v })}/>
        </Field>
      );
  }
}

function JSONEditor({ value, onChange }) {
  const [local, setLocal] = React.useState(() => JSON.stringify(value || {}, null, 2));
  const [err, setErr] = React.useState(null);
  React.useEffect(() => { setLocal(JSON.stringify(value || {}, null, 2)); }, [JSON.stringify(value)]);
  return (
    <div>
      <textarea
        value={local}
        onChange={e => {
          setLocal(e.target.value);
          try { const v = JSON.parse(e.target.value || '{}'); setErr(null); onChange(v); }
          catch (x) { setErr(x.message); }
        }}
        className="mono"
        rows={6}
        style={{
          padding: '6px 8px', borderRadius: 4,
          border: '1px solid ' + (err ? 'var(--danger)' : 'var(--hairline-strong)'),
          background: 'var(--surface-2)', color: 'var(--fg)', fontSize: 11, width: '100%', resize: 'vertical',
        }}
      />
      {err && <div style={{ color: 'var(--danger)', fontSize: 10.5, marginTop: 4 }}>{err}</div>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Conditions tab
// ---------------------------------------------------------------------------

function ConditionsTab({ conditions, onChange }) {
  const [local, setLocal] = React.useState(() => JSON.stringify(conditions || {}, null, 2));
  const [err, setErr] = React.useState(null);

  React.useEffect(() => {
    setLocal(JSON.stringify(conditions || {}, null, 2));
  }, [JSON.stringify(conditions)]);

  const save = () => {
    try {
      const v = JSON.parse(local || '{}');
      setErr(null);
      onChange(v);
    } catch (x) { setErr(x.message); }
  };

  return (
    <div style={{ flex: 1, overflow: 'auto', padding: 24, display: 'flex', flexDirection: 'column', gap: 14, maxWidth: 720, width: '100%', margin: '0 auto' }}>
      <div>
        <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 15, margin: 0, fontWeight: 600 }}>Trigger conditions</h2>
        <p className="faint" style={{ fontSize: 12, marginTop: 4, lineHeight: 1.6 }}>
          Conditions decide whether this flow runs at all. Only matching flows are considered for a given trigger; highest-priority match wins.
        </p>
      </div>
      <Field label="Conditions (JSON)" hint='Example: {"email_domain": "acme.com", "has_metadata": "org_id"}'>
        <textarea
          value={local}
          onChange={e => { setLocal(e.target.value); setErr(null); }}
          className="mono"
          rows={10}
          style={{
            padding: '8px 10px', borderRadius: 4,
            border: '1px solid ' + (err ? 'var(--danger)' : 'var(--hairline-strong)'),
            background: 'var(--surface-2)', color: 'var(--fg)', fontSize: 12, width: '100%', resize: 'vertical',
          }}
        />
      </Field>
      {err && <div style={{ color: 'var(--danger)', fontSize: 11 }}>{err}</div>}
      <div className="row" style={{ gap: 8 }}>
        <button className="btn sm" onClick={() => setLocal(JSON.stringify(conditions || {}, null, 2))}>Reset</button>
        <button className="btn primary sm" onClick={save}>Apply</button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Preview tab
// ---------------------------------------------------------------------------

const PRESET_MOCKS = [
  { id: 'fresh',        label: 'Fresh signup',   user: { email: 'new@example.com',   email_verified: false, name: 'New Person' } },
  { id: 'existing',     label: 'Existing user',  user: { email: 'alice@example.com', email_verified: true,  name: 'Alice' } },
  { id: 'oauth',        label: 'OAuth user',     user: { email: 'github@example.com', email_verified: true, name: 'OAuth User' } },
  { id: 'org-admin',    label: 'Org admin',      user: { email: 'admin@acme.com',    email_verified: true,  name: 'Org Admin' }, metadata: { org_id: 'org_acme', role: 'admin' } },
];

const MOCKS_KEY = 'sharkauth.flow.mocks';

function loadSavedMocks() {
  try {
    const raw = localStorage.getItem(MOCKS_KEY);
    if (!raw) return PRESET_MOCKS;
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed) || parsed.length === 0) return PRESET_MOCKS;
    return parsed;
  } catch { return PRESET_MOCKS; }
}

function saveMocks(mocks) {
  try { localStorage.setItem(MOCKS_KEY, JSON.stringify(mocks)); } catch {}
}

function PreviewTab({ flowId, trigger, steps }) {
  const [mocks, setMocks] = React.useState(loadSavedMocks);
  const [presetId, setPresetId] = React.useState(mocks[0]?.id || 'fresh');
  const preset = mocks.find(m => m.id === presetId) || mocks[0];

  const [email, setEmail] = React.useState(preset?.user?.email || '');
  const [emailVerified, setEmailVerified] = React.useState(!!preset?.user?.email_verified);
  const [name, setName] = React.useState(preset?.user?.name || '');
  const [password, setPassword] = React.useState('');
  const [metadataJSON, setMetadataJSON] = React.useState(() => JSON.stringify(preset?.metadata || {}, null, 2));
  const [metaErr, setMetaErr] = React.useState(null);

  const [running, setRunning] = React.useState(false);
  const [result, setResult] = React.useState(null);
  const [runError, setRunError] = React.useState(null);

  React.useEffect(() => {
    if (!preset) return;
    setEmail(preset.user?.email || '');
    setEmailVerified(!!preset.user?.email_verified);
    setName(preset.user?.name || '');
    setMetadataJSON(JSON.stringify(preset.metadata || {}, null, 2));
    setMetaErr(null);
  }, [presetId]);

  const run = async () => {
    let metadata = {};
    try { metadata = JSON.parse(metadataJSON || '{}'); setMetaErr(null); }
    catch (x) { setMetaErr(x.message); return; }

    setRunning(true);
    setRunError(null);
    setResult(null);
    try {
      const body = {
        user: { email, email_verified: emailVerified, name },
        password: password || undefined,
        metadata,
      };
      const res = await API.post(`/admin/flows/${flowId}/test`, body);
      setResult(res);
    } catch (e) {
      setRunError(e.message || 'Test failed');
    } finally {
      setRunning(false);
    }
  };

  const saveCurrentAsPreset = () => {
    const id = `custom-${Date.now()}`;
    let metadata = {};
    try { metadata = JSON.parse(metadataJSON || '{}'); } catch {}
    const next = [...mocks, {
      id, label: `Custom ${mocks.filter(m => m.id.startsWith('custom-')).length + 1}`,
      user: { email, email_verified: emailVerified, name }, metadata,
    }];
    setMocks(next); saveMocks(next); setPresetId(id);
  };

  return (
    <div style={{ flex: 1, display: 'grid', gridTemplateColumns: '1fr 1fr', minHeight: 0, overflow: 'hidden' }}>
      {/* Form */}
      <div style={{ overflow: 'auto', padding: 20, borderRight: '1px solid var(--hairline)', display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div>
          <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 14, margin: 0, fontWeight: 600 }}>Mock user</h2>
          <p className="faint" style={{ fontSize: 11.5, marginTop: 2 }}>Simulate a {trigger} trigger with this data.</p>
        </div>
        <Field label="Preset">
          <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
            {mocks.map(m => (
              <button key={m.id} onClick={() => setPresetId(m.id)} className={m.id === presetId ? '' : 'mono'}
                style={{
                  padding: '4px 8px', fontSize: 11, borderRadius: 3,
                  border: '1px solid var(--hairline-strong)',
                  background: m.id === presetId ? 'var(--surface-3)' : 'var(--surface-1)',
                  color: 'var(--fg)',
                }}>{m.label}</button>
            ))}
          </div>
        </Field>
        <Field label="Email">
          <TextInput value={email} onChange={setEmail} mono/>
        </Field>
        <Checkbox checked={emailVerified} onChange={setEmailVerified} label="email_verified"/>
        <Field label="Name">
          <TextInput value={name} onChange={setName}/>
        </Field>
        {trigger === 'signup' && (
          <Field label="Password" hint="Used by require_password_strength steps.">
            <TextInput value={password} onChange={setPassword} mono/>
          </Field>
        )}
        <Field label="Metadata (JSON)">
          <textarea value={metadataJSON}
            onChange={e => { setMetadataJSON(e.target.value); setMetaErr(null); }}
            className="mono" rows={4}
            style={{
              padding: '6px 8px', borderRadius: 4,
              border: '1px solid ' + (metaErr ? 'var(--danger)' : 'var(--hairline-strong)'),
              background: 'var(--surface-2)', color: 'var(--fg)', fontSize: 11, width: '100%', resize: 'vertical',
            }}
          />
        </Field>
        {metaErr && <div style={{ color: 'var(--danger)', fontSize: 10.5 }}>{metaErr}</div>}
        <div className="row" style={{ gap: 8 }}>
          <button className="btn primary sm" onClick={run} disabled={running}>
            {running ? 'Running…' : 'Run'}
          </button>
          <button className="btn ghost sm" onClick={saveCurrentAsPreset}>Save as preset</button>
        </div>
      </div>

      {/* Result */}
      <div style={{ overflow: 'auto', padding: 20, display: 'flex', flexDirection: 'column', gap: 12 }}>
        {runError && (
          <div style={{
            padding: '10px 12px', borderRadius: 4,
            background: 'color-mix(in oklch, var(--danger) 12%, var(--surface-1))',
            border: '1px solid color-mix(in oklch, var(--danger) 35%, var(--hairline-strong))',
            color: 'var(--danger)', fontSize: 12,
          }}>{runError}</div>
        )}
        {!result && !runError && (
          <div style={{ display:'flex', flex: 1, alignItems:'center', justifyContent:'center', textAlign:'center', padding: 40 }}>
            <div style={{ maxWidth: 260 }}>
              <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)' }}>Run a simulation.</div>
              <div style={{ marginTop: 6, fontSize: 11.5, color: 'var(--fg-dim)', lineHeight: 1.5 }}>
                Pick a mock user, hit Run, watch the flow execute step-by-step.
              </div>
            </div>
          </div>
        )}
        {result && <PreviewResult result={result} steps={steps}/>}
      </div>
    </div>
  );
}

const OUTCOME_CHIP = {
  continue: 'success',
  block:    'danger',
  redirect: 'warn',
  error:    'danger',
};

function PreviewResult({ result, steps }) {
  const chipClass = OUTCOME_CHIP[result.outcome] || '';
  const blockedIndex = result.blocked_at_step;
  const stepName = blockedIndex != null && steps[blockedIndex] ? steps[blockedIndex].type : null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div className="row" style={{ gap: 8 }}>
        <span className={'chip ' + chipClass} style={{ height: 22, fontSize: 11 }}>
          <span className={'dot ' + chipClass}/>{result.outcome}
        </span>
        {result.redirect_url && (
          <span className="mono faint" style={{ fontSize: 11 }}>→ {result.redirect_url}</span>
        )}
      </div>

      {result.outcome === 'block' && blockedIndex != null && (
        <div style={{
          padding: '10px 12px', borderRadius: 4,
          background: 'color-mix(in oklch, var(--danger) 10%, var(--surface-1))',
          border: '1px solid color-mix(in oklch, var(--danger) 30%, var(--hairline-strong))',
        }}>
          <div style={{ fontSize: 12.5, fontWeight: 500 }}>
            Flow blocked at step {blockedIndex + 1}{stepName ? `: ${stepName}` : ''}.
          </div>
          {result.reason && (
            <div className="mono faint" style={{ fontSize: 11, marginTop: 4 }}>
              {result.reason}
            </div>
          )}
          <div className="faint" style={{ fontSize: 11, marginTop: 4, lineHeight: 1.5 }}>
            Configure the step or adjust your mock user.
          </div>
        </div>
      )}

      {result.reason && result.outcome !== 'block' && (
        <div className="mono faint" style={{ fontSize: 11 }}>{result.reason}</div>
      )}

      <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 4 }}>
        <div style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 4 }}>
          Timeline
        </div>
        {(result.timeline || []).map((entry, i) => {
          const skipped = blockedIndex != null && entry.index > blockedIndex;
          const entryChip = OUTCOME_CHIP[entry.outcome] || '';
          return (
            <div key={i}
              style={{
                display: 'grid', gridTemplateColumns: '24px 1fr auto auto',
                gap: 8, alignItems: 'center',
                padding: '6px 8px', borderBottom: '1px solid var(--hairline)',
                opacity: skipped ? 0.3 : 1,
                animation: `fadeIn 200ms ease-out ${i * 80}ms both`,
              }}>
              <span className="mono faint" style={{ fontSize: 10.5 }}>{entry.index + 1}</span>
              <span className="mono" style={{ fontSize: 11.5 }}>{entry.type}</span>
              <span className={'dot ' + entryChip}/>
              <span className="mono faint" style={{ fontSize: 10.5 }}>{formatDuration(entry.duration_ns)}</span>
              {entry.reason && (
                <div className="mono faint" style={{ gridColumn: '2 / 5', fontSize: 10.5, marginTop: 2 }}>
                  {entry.reason}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// History tab
// ---------------------------------------------------------------------------

function HistoryTab({ flowId }) {
  const { data, loading, refresh } = useAPI(`/admin/flows/${flowId}/runs?limit=50`);
  const runs = data?.data || [];

  return (
    <div style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <div className="row" style={{ padding: '10px 20px', borderBottom: '1px solid var(--hairline)', gap: 8 }}>
        <div style={{ flex: 1 }}>
          <h2 style={{ fontFamily: 'var(--font-display)', fontSize: 14, margin: 0, fontWeight: 600 }}>Recent runs</h2>
          <p className="faint" style={{ fontSize: 11, marginTop: 2 }}>Latest 50 executions of this flow.</p>
        </div>
        <button className="btn ghost sm" onClick={refresh} disabled={loading}>
          <Icon.Refresh width={11} height={11}/>
        </button>
      </div>

      {loading ? (
        <div className="faint" style={{ padding: 20, fontSize: 12 }}>Loading runs…</div>
      ) : runs.length === 0 ? (
        <div style={{ display:'flex', flex: 1, alignItems:'center', justifyContent:'center', padding: 40, textAlign: 'center' }}>
          <div style={{ maxWidth: 320 }}>
            <div style={{ fontSize: 13, fontWeight: 500 }}>No executions yet.</div>
            <div style={{ marginTop: 6, fontSize: 11.5, color: 'var(--fg-dim)', lineHeight: 1.5 }}>
              Runs appear here after the flow triggers on a real event.
            </div>
          </div>
        </div>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
          <thead>
            <tr>
              <th style={thStyle}>Started</th>
              <th style={thStyle}>User</th>
              <th style={thStyle}>Outcome</th>
              <th style={thStyle}>Blocked step</th>
              <th style={thStyle}>Duration</th>
              <th style={thStyle}>Reason</th>
            </tr>
          </thead>
          <tbody>
            {runs.map(r => {
              const durationMs = (new Date(r.finished_at).getTime() - new Date(r.started_at).getTime());
              const chipClass = OUTCOME_CHIP[r.outcome] || '';
              return (
                <tr key={r.id}>
                  <td style={tdStyle}><span className="mono faint" style={{ fontSize: 10.5 }}>{relativeTime(r.started_at)}</span></td>
                  <td style={tdStyle}>
                    {r.user_id
                      ? <span className="mono" style={{ fontSize: 10.5 }}>{r.user_id}</span>
                      : <span className="faint" style={{ fontSize: 10.5 }}>—</span>}
                  </td>
                  <td style={tdStyle}>
                    <span className={'chip ' + chipClass} style={{ height: 18, fontSize: 10 }}>
                      <span className={'dot ' + chipClass}/>{r.outcome}
                    </span>
                  </td>
                  <td style={tdStyle}>
                    {r.blocked_at_step != null
                      ? <span className="mono" style={{ fontSize: 10.5 }}>#{r.blocked_at_step + 1}</span>
                      : <span className="faint" style={{ fontSize: 10.5 }}>—</span>}
                  </td>
                  <td style={tdStyle}>
                    <span className="mono faint" style={{ fontSize: 10.5 }}>{durationMs >= 0 ? `${durationMs}ms` : '—'}</span>
                  </td>
                  <td style={tdStyle}>
                    {r.reason
                      ? <span className="mono faint" style={{ fontSize: 10.5, maxWidth: 300, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.reason}</span>
                      : <span className="faint" style={{ fontSize: 10.5 }}>—</span>}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}
