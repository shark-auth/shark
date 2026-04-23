// @ts-nocheck
import React from 'react'

import sharkyIconPng from '../assets/sharky-icon.png'
import sharkyFullPng from '../assets/sharky-full.png'
import sharkyGlyphPng from '../assets/sharky-glyph.png'
import sharkyWordmarkPng from '../assets/sharky-wordmark.png'

export { sharkyGlyphPng, sharkyWordmarkPng }

export const Icon = {
  Home: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M2 6.5L8 2l6 4.5V13a1 1 0 01-1 1h-3v-4H7v4H4a1 1 0 01-1-1V6.5z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Users: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="6" cy="5" r="2.3" stroke="currentColor" strokeWidth="1.3"/><path d="M2 13c0-2.2 1.8-4 4-4s4 1.8 4 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/><circle cx="11.5" cy="5.5" r="1.6" stroke="currentColor" strokeWidth="1.2"/><path d="M10 10.2c.5-.15 1-.2 1.5-.2 1.4 0 2.5 1.1 2.5 2.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/></svg>,
  Session: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2" y="3.5" width="12" height="8" rx="1" stroke="currentColor" strokeWidth="1.3"/><path d="M5 14h6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Org: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2.5" y="6" width="4" height="8" stroke="currentColor" strokeWidth="1.3"/><rect x="9.5" y="2" width="4" height="12" stroke="currentColor" strokeWidth="1.3"/><path d="M4 8.5h1M4 10.5h1M11 4.5h1M11 6.5h1M11 8.5h1M11 10.5h1" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round"/></svg>,
  Agent: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M3 6l5-3 5 3v5l-5 3-5-3V6z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/><circle cx="8" cy="8.2" r="1.3" fill="currentColor"/></svg>,
  Consent: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M2.5 4a1 1 0 011-1h9a1 1 0 011 1v5a4 4 0 01-4 4H6.5L3 15v-2.5a4 4 0 01-.5-2V4z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/><path d="M5.5 7l1.7 1.7L10.5 5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Token: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="6" cy="8" r="3" stroke="currentColor" strokeWidth="1.3"/><path d="M9 8h5M12 6v4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Vault: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2" y="3" width="12" height="10" rx="1" stroke="currentColor" strokeWidth="1.3"/><circle cx="8" cy="8" r="2.2" stroke="currentColor" strokeWidth="1.3"/><path d="M8 5.5v.5M8 10v.5M5.5 8h.5M10 8h.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Device: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="4" y="2" width="8" height="12" rx="1.5" stroke="currentColor" strokeWidth="1.3"/><circle cx="8" cy="12" r="0.6" fill="currentColor"/></svg>,
  App: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2" y="2" width="5" height="5" rx="0.8" stroke="currentColor" strokeWidth="1.3"/><rect x="9" y="2" width="5" height="5" rx="0.8" stroke="currentColor" strokeWidth="1.3"/><rect x="2" y="9" width="5" height="5" rx="0.8" stroke="currentColor" strokeWidth="1.3"/><rect x="9" y="9" width="5" height="5" rx="0.8" stroke="currentColor" strokeWidth="1.3"/></svg>,
  Lock: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="3" y="7" width="10" height="7" rx="1" stroke="currentColor" strokeWidth="1.3"/><path d="M5.5 7V5a2.5 2.5 0 015 0v2" stroke="currentColor" strokeWidth="1.3"/></svg>,
  SSO: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M2 8h5M9 8h5M7 6l2 2-2 2M9 6l-2 2 2 2" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Shield: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 2l5 2v4c0 3-2 5-5 6-3-1-5-3-5-6V4l5-2z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Key: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="5" cy="8" r="2.5" stroke="currentColor" strokeWidth="1.3"/><path d="M7.5 8H14M12 8v2.5M10 8v1.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Audit: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M3 2h7l3 3v9H3V2z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/><path d="M10 2v3h3M5.5 7.5h5M5.5 9.5h5M5.5 11.5h3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Webhook: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="5" r="2" stroke="currentColor" strokeWidth="1.3"/><circle cx="4" cy="12" r="1.6" stroke="currentColor" strokeWidth="1.3"/><circle cx="12" cy="12" r="1.6" stroke="currentColor" strokeWidth="1.3"/><path d="M8 7l-3 4M8 7l3 4M5.5 12h5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Mail: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2" y="3.5" width="12" height="9" rx="1" stroke="currentColor" strokeWidth="1.3"/><path d="M2.5 4.5l5.5 4 5.5-4" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Signing: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M3 13l4-4 6-6 2 2-6 6-4 4H3v-2z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Settings: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.3"/><path d="M8 1.5v1.5M8 13v1.5M14.5 8H13M3 8H1.5M12.5 3.5l-1 1M4.5 11.5l-1 1M12.5 12.5l-1-1M4.5 4.5l-1-1" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Explorer: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M4 4l-2 4 2 4M12 4l2 4-2 4M9.5 3l-3 10" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Debug: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="4" y="5" width="8" height="8" rx="3" stroke="currentColor" strokeWidth="1.3"/><path d="M2 8h2M12 8h2M2.5 4l2 1.5M13.5 4l-2 1.5M2.5 12l2-1.5M13.5 12l-2-1.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Schema: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2" y="2" width="5" height="5" stroke="currentColor" strokeWidth="1.3"/><rect x="9" y="9" width="5" height="5" stroke="currentColor" strokeWidth="1.3"/><path d="M7 4.5h2M4.5 7v2M9 11.5h-2M11.5 9v-2" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Proxy: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="3" cy="8" r="1.5" stroke="currentColor" strokeWidth="1.3"/><circle cx="13" cy="8" r="1.5" stroke="currentColor" strokeWidth="1.3"/><rect x="6" y="5.5" width="4" height="5" stroke="currentColor" strokeWidth="1.3"/><path d="M4.5 8h1.5M10 8h1.5" stroke="currentColor" strokeWidth="1.3"/></svg>,
  Impersonate: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="5" r="2.3" stroke="currentColor" strokeWidth="1.3"/><path d="M3 14c0-2.5 2.2-4.5 5-4.5s5 2 5 4.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/><path d="M11 2.5l1.5 1.5M12.5 2.5L11 4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/></svg>,
  Compliance: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 2l5 2v4c0 3-2 5-5 6-3-1-5-3-5-6V4l5-2z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/><path d="M5.5 8l1.5 1.5L10.5 6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Migration: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M2 8h10M9 5l3 3-3 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/><path d="M14 3v10" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Brand: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="8" r="5.5" stroke="currentColor" strokeWidth="1.3"/><circle cx="6" cy="7" r="0.8" fill="currentColor"/><circle cx="10" cy="7" r="0.8" fill="currentColor"/><circle cx="5" cy="10" r="0.8" fill="currentColor"/><circle cx="11" cy="10" r="0.8" fill="currentColor"/></svg>,
  Flow: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="1.5" y="6" width="4" height="4" stroke="currentColor" strokeWidth="1.3"/><rect x="10.5" y="2" width="4" height="4" stroke="currentColor" strokeWidth="1.3"/><rect x="10.5" y="10" width="4" height="4" stroke="currentColor" strokeWidth="1.3"/><path d="M5.5 8H8v-4h2.5M8 8v4h2.5" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Search: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="7" cy="7" r="4" stroke="currentColor" strokeWidth="1.3"/><path d="M10 10l3 3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Plus: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 3v10M3 8h10" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/></svg>,
  Bell: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M4 6.5a4 4 0 018 0c0 3 1.5 4 1.5 4h-11s1.5-1 1.5-4z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/><path d="M6.5 13a1.5 1.5 0 003 0" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  ChevronDown: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M4 6l4 4 4-4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  ChevronLeft: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M10 4l-4 4 4 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  ChevronRight: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M6 4l4 4-4 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  ArrowUp: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 13V3M4 7l4-4 4 4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  ArrowDown: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 3v10M4 9l4 4 4-4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Check: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M3 8l3 3 7-7" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  X: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  More: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="3.5" cy="8" r="1.1" fill="currentColor"/><circle cx="8" cy="8" r="1.1" fill="currentColor"/><circle cx="12.5" cy="8" r="1.1" fill="currentColor"/></svg>,
  Copy: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="5" y="5" width="8" height="9" rx="1" stroke="currentColor" strokeWidth="1.3"/><path d="M3 11V3a1 1 0 011-1h7" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Filter: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M2 3h12l-4.5 6v4L6 14V9L2 3z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Warn: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 2l6.5 11h-13L8 2z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/><path d="M8 6.5v3M8 11.5v.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Info: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.3"/><path d="M8 7v4M8 5v.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Bolt: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M9 2L3 9h4l-1 5 6-7H8l1-5z" stroke="currentColor" strokeWidth="1.3" strokeLinejoin="round"/></svg>,
  Collapse: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2.5" y="3" width="11" height="10" rx="1" stroke="currentColor" strokeWidth="1.2"/><path d="M6 3v10" stroke="currentColor" strokeWidth="1.2"/><path d="M10 6l-2 2 2 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Expand: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="2.5" y="3" width="11" height="10" rx="1" stroke="currentColor" strokeWidth="1.2"/><path d="M6 3v10" stroke="currentColor" strokeWidth="1.2"/><path d="M8 6l2 2-2 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Terminal: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><rect x="1.5" y="3" width="13" height="10" rx="1" stroke="currentColor" strokeWidth="1.3"/><path d="M4 7l2 1.5L4 10M8 10h4" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  Clock: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="8" r="5.5" stroke="currentColor" strokeWidth="1.3"/><path d="M8 5v3.2l2 1.3" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/></svg>,
  Globe: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><circle cx="8" cy="8" r="5.5" stroke="currentColor" strokeWidth="1.3"/><path d="M2.5 8h11M8 2.5c2 2 2 9 0 11M8 2.5c-2 2-2 9 0 11" stroke="currentColor" strokeWidth="1.2"/></svg>,
  Sparkle: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 2v3M8 11v3M2 8h3M11 8h3M4 4l2 2M12 12l-2-2M4 12l2-2M12 4l-2 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/></svg>,
  Eye: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M1.5 8s2.5-4 6.5-4 6.5 4 6.5 4-2.5 4-6.5 4-6.5-4-6.5-4z" stroke="currentColor" strokeWidth="1.3"/><circle cx="8" cy="8" r="1.8" stroke="currentColor" strokeWidth="1.3"/></svg>,
  Refresh: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M13 4.5A5.5 5.5 0 008 3a5 5 0 00-5 5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/><path d="M13 2v2.5h-2.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/><path d="M3 11.5A5.5 5.5 0 008 13a5 5 0 005-5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/><path d="M3 14v-2.5h2.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  External: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M7 3H3v10h10V9" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round"/><path d="M9 3h4v4M7 9l6-6" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round"/></svg>,
  DPoP: (p) => <svg viewBox="0 0 16 16" fill="none" {...p}><path d="M8 2l5 2v5c0 2.5-2 4.5-5 5-3-0.5-5-2.5-5-5V4l5-2z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round"/><path d="M8 6v4M6 8h4" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/></svg>,
};

export function SharkIcon({ size = 22 }) {
  return (
    <div style={{
      width: size, height: size,
      borderRadius: Math.max(3, size * 0.18),
      background: '#000',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      overflow: 'hidden',
      flexShrink: 0,
    }}>
      <img
        src={sharkyIconPng}
        alt=""
        width={size} height={size}
        style={{ display: 'block', width: '100%', height: '100%', objectFit: 'contain' }}
      />
    </div>
  );
}

export function SharkFullLogo({ width = 200 }) {
  return (
    <img
      src={sharkyFullPng}
      alt="Shark"
      style={{ display: 'block', width, height: 'auto', borderRadius: Math.round(width * 0.03) }}
    />
  );
}

export function hashColor(s) {
  let h = 0; for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
  const hue = Math.abs(h) % 360;
  return `oklch(0.3 0.04 ${hue})`;
}

export function Avatar({ name, email, size = 22, agent = false }) {
  const seed = email || name || 'x';
  const initials = (name || email || '?').split(/[\s@.]/).filter(Boolean).slice(0,2).map(s => s[0]).join('').toUpperCase();
  return (
    <span className={"avatar" + (agent ? " agent" : "") + (size > 30 ? " lg" : "")}
      style={{ background: agent ? undefined : hashColor(seed), width: size, height: size, fontSize: size <= 24 ? 10 : 13 }}>
      {initials || '?'}
    </span>
  );
}

export function CopyField({ value, mono = true, truncate = 24 }) {
  const [copied, setCopied] = React.useState(false);
  const display = truncate && value.length > truncate ? value.slice(0, truncate - 4) + '\u2026' + value.slice(-3) : value;
  return (
    <button
      className={mono ? 'mono' : ''}
      onClick={(e) => { e.stopPropagation(); navigator.clipboard?.writeText(value); setCopied(true); setTimeout(() => setCopied(false), 900); }}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 5,
        padding: '2px 5px', borderRadius: 3,
        border: '1px solid var(--hairline)',
        background: 'var(--surface-2)',
        fontSize: 11, color: 'var(--fg-muted)',
        cursor: 'pointer',
      }}
      title={value}
    >
      <span>{display}</span>
      {copied ? <Icon.Check width={10} height={10} style={{color:'var(--success)'}}/> : <Icon.Copy width={10} height={10} style={{opacity:0.6}}/>}
    </button>
  );
}

export function Sparkline({ data, height = 28, color = 'var(--fg)' }) {
  const w = 120;
  const max = Math.max(...data), min = Math.min(...data);
  const rng = max - min || 1;
  const step = w / (data.length - 1);
  const pts = data.map((v, i) => `${i * step},${height - ((v - min) / rng) * (height - 2) - 1}`).join(' ');
  const area = `M0,${height} L${pts.split(' ').join(' L')} L${w},${height} Z`;
  return (
    <svg className="spark" viewBox={`0 0 ${w} ${height}`} preserveAspectRatio="none" style={{ width: '100%', height }}>
      <path d={area} className="area" fill={color} opacity="0.08"/>
      <polyline points={pts} fill="none" stroke={color} strokeWidth="1.2"/>
    </svg>
  );
}

export function Donut({ segments, size = 92, thickness = 14 }) {
  const total = segments.reduce((a, s) => a + s.value, 0);
  const r = (size - thickness) / 2;
  const c = 2 * Math.PI * r;
  let offset = 0;
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle cx={size/2} cy={size/2} r={r} fill="none" stroke="var(--surface-3)" strokeWidth={thickness}/>
      {segments.map((s, i) => {
        const len = (s.value / total) * c;
        const el = (
          <circle key={i} cx={size/2} cy={size/2} r={r} fill="none"
            stroke={s.color} strokeWidth={thickness}
            strokeDasharray={`${len} ${c - len}`}
            strokeDashoffset={-offset}
            transform={`rotate(-90 ${size/2} ${size/2})`}
            strokeLinecap="butt"/>
        );
        offset += len;
        return el;
      })}
    </svg>
  );
}

export function Kbd({ keys }) {
  return <span className="row" style={{gap:3}}>{keys.split(' ').map((k, i) => <span key={i} className="kbd">{k}</span>)}</span>;
}
