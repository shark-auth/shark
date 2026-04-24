// @ts-nocheck
// BrandStudio — 3-column editor replacing the old tabbed Branding page.
//
// Layout:
//   Left rail (220px)   — navigation tree, sections with deep-linkable surfaces
//   Center pane (fluid) — dense hairline-separated row editor for the selection
//   Right preview (420) — live preview, user-resizable via vertical drag handle
//
// State model:
//   brandCfg / brandDraft — /admin/branding (visuals: logo, colors, fonts, copy)
//   templates  / tplDraft — /admin/email-templates (email_tab preserved)
//   apps                  — /admin/apps (integrations_tab preserved)
//   authOverrides         — client-side copy overrides for each auth surface
//                           (stored in brand_draft.auth_copy — new field,
//                           persisted alongside branding on save)
//
// URL contract: ?section=<id>&surface=<id>&pv=<dark|light>
//
// Preserves ALL existing functionality and endpoints. Visual/structural
// refactor only. Old `branding/*_tab.tsx` files are now internal utility
// modules for their API hooks; the tabbed UI is gone.
import React from 'react'
import { HexColorPicker } from 'react-colorful'
import { API, useAPI } from './api'
import { useToast } from './toast'
import { usePageActions } from './useKeyboardShortcuts'
import { Icon } from './shared'
import {
  SignInForm,
  SignUpForm,
  MFAForm,
  ForgotPasswordForm,
  ResetPasswordForm,
  MagicLinkSent,
  EmailVerify,
  ErrorPage,
} from '../design/composed'

// ───────────────────────────────────────────────────────────────────────────
// Tree model
// ───────────────────────────────────────────────────────────────────────────

const SECTIONS: { id: string; label: string; icon: string; surfaces: { id: string; label: string }[] }[] = [
  { id: 'identity', label: 'Identity', icon: 'Brand', surfaces: [
    { id: 'logo',     label: 'Logo' },
    { id: 'wordmark', label: 'Wordmark' },
    { id: 'favicon',  label: 'Favicon' },
    { id: 'app-name', label: 'App name' },
  ]},
  { id: 'palette', label: 'Palette', icon: 'Palette', surfaces: [
    { id: 'primary',  label: 'Primary' },
    { id: 'surfaces', label: 'Surfaces' },
    { id: 'text',     label: 'Text' },
    { id: 'semantic', label: 'Semantic' },
  ]},
  { id: 'typography', label: 'Typography', icon: 'Signing', surfaces: [
    { id: 'display', label: 'Display font' },
    { id: 'body',    label: 'Body font' },
    { id: 'mono',    label: 'Mono font' },
    { id: 'scale',   label: 'Size scale' },
  ]},
  { id: 'auth', label: 'Auth surfaces', icon: 'Lock', surfaces: [
    { id: 'sign-in',          label: 'Sign-in' },
    { id: 'sign-up',          label: 'Sign-up' },
    { id: 'mfa',              label: 'MFA' },
    { id: 'forgot-password',  label: 'Forgot password' },
    { id: 'reset-password',   label: 'Reset password' },
    { id: 'magic-link-sent',  label: 'Magic link sent' },
    { id: 'email-verify',     label: 'Email verify' },
    { id: 'error-page',       label: 'Error page' },
  ]},
  { id: 'email', label: 'Email templates', icon: 'Mail', surfaces: [] /* populated dynamically */ },
  { id: 'integrations', label: 'Integrations', icon: 'Webhook', surfaces: [] /* populated dynamically */ },
]

const AUTH_SURFACES_FOR_PREVIEW = SECTIONS.find(s => s.id === 'auth')!.surfaces

// Default auth-copy overrides. Stored on branding.auth_copy[surface].
const AUTH_COPY_DEFAULTS: Record<string, { heading: string; subheading: string; cta: string; smallPrint: string }> = {
  'sign-in':         { heading: 'Sign in to your account', subheading: 'Welcome back',                                cta: 'Sign in',              smallPrint: '' },
  'sign-up':         { heading: 'Create an account',        subheading: 'Sign up to get started',                      cta: 'Create account',       smallPrint: 'By signing up you agree to our Terms.' },
  'mfa':             { heading: 'Two-factor',               subheading: 'Enter the 6-digit code from your authenticator', cta: 'Verify',            smallPrint: '' },
  'forgot-password': { heading: 'Forgot password',          subheading: 'We will email you a reset link',              cta: 'Send reset link',      smallPrint: '' },
  'reset-password':  { heading: 'Reset password',           subheading: 'Choose a new password',                       cta: 'Update password',      smallPrint: 'Minimum 8 characters.' },
  'magic-link-sent': { heading: 'Check your inbox',         subheading: 'We just sent you a magic link',               cta: 'Open mail app',        smallPrint: 'Link expires in 10 minutes.' },
  'email-verify':    { heading: 'Verify your email',        subheading: 'Click the link we sent to confirm',           cta: 'Resend email',         smallPrint: '' },
  'error-page':      { heading: 'Something went wrong',     subheading: 'This page is not available right now',        cta: 'Back to sign-in',      smallPrint: '' },
}

// ───────────────────────────────────────────────────────────────────────────
// Color utils (oklch ↔ hex, round-trip tolerant)
// ───────────────────────────────────────────────────────────────────────────

function clamp(n: number, lo: number, hi: number) { return Math.max(lo, Math.min(hi, n)) }

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  const n = h.length === 3 ? h.split('').map(c => c + c).join('') : h
  const r = parseInt(n.slice(0, 2), 16) / 255
  const g = parseInt(n.slice(2, 4), 16) / 255
  const b = parseInt(n.slice(4, 6), 16) / 255
  return [isNaN(r) ? 0 : r, isNaN(g) ? 0 : g, isNaN(b) ? 0 : b]
}
function rgbToHex(r: number, g: number, b: number): string {
  const toH = (v: number) => clamp(Math.round(v * 255), 0, 255).toString(16).padStart(2, '0')
  return '#' + toH(r) + toH(g) + toH(b)
}
// sRGB → OKLCH via linearization + OKLab conversion.
function hexToOklch(hex: string): { l: number; c: number; h: number } {
  const [r, g, b] = hexToRgb(hex)
  const lin = (v: number) => v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
  const R = lin(r), G = lin(g), B = lin(b)
  const l_ = Math.cbrt(0.4122214708 * R + 0.5363325363 * G + 0.0514459929 * B)
  const m_ = Math.cbrt(0.2119034982 * R + 0.6806995451 * G + 0.1073969566 * B)
  const s_ = Math.cbrt(0.0883024619 * R + 0.2817188376 * G + 0.6299787005 * B)
  const L = 0.2104542553 * l_ + 0.793617785 * m_ - 0.0040720468 * s_
  const A = 1.9779984951 * l_ - 2.428592205 * m_ + 0.4505937099 * s_
  const Bb = 0.0259040371 * l_ + 0.7827717662 * m_ - 0.808675766 * s_
  const C = Math.sqrt(A * A + Bb * Bb)
  let H = (Math.atan2(Bb, A) * 180) / Math.PI
  if (H < 0) H += 360
  return { l: L, c: C, h: H }
}
function oklchToHex(l: number, c: number, h: number): string {
  const A = Math.cos((h * Math.PI) / 180) * c
  const B = Math.sin((h * Math.PI) / 180) * c
  const l_ = l + 0.3963377774 * A + 0.2158037573 * B
  const m_ = l - 0.1055613458 * A - 0.0638541728 * B
  const s_ = l - 0.0894841775 * A - 1.291485548 * B
  const L = l_ * l_ * l_, M = m_ * m_ * m_, S = s_ * s_ * s_
  const R = 4.0767416621 * L - 3.3077115913 * M + 0.2309699292 * S
  const G = -1.2684380046 * L + 2.6097574011 * M - 0.3413193965 * S
  const Bb = -0.0041960863 * L - 0.7034186147 * M + 1.707614701 * S
  const delin = (v: number) => v <= 0.0031308 ? 12.92 * v : 1.055 * Math.pow(v, 1 / 2.4) - 0.055
  return rgbToHex(clamp(delin(R), 0, 1), clamp(delin(G), 0, 1), clamp(delin(Bb), 0, 1))
}
function parseOklchString(s: string): { l: number; c: number; h: number } | null {
  // Accepts "oklch(0.62 0.22 295)" or "#rrggbb".
  if (!s) return null
  if (s.startsWith('#')) return hexToOklch(s)
  const m = /oklch\(\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)\s*\)/.exec(s)
  if (!m) return null
  return { l: parseFloat(m[1]), c: parseFloat(m[2]), h: parseFloat(m[3]) }
}
function toOklchString(v: { l: number; c: number; h: number }): string {
  return `oklch(${v.l.toFixed(3)} ${v.c.toFixed(3)} ${v.h.toFixed(1)})`
}
function colorAsHex(v: string | undefined, fallback = '#7c3aed'): string {
  if (!v) return fallback
  if (v.startsWith('#')) return v
  const p = parseOklchString(v)
  return p ? oklchToHex(p.l, p.c, p.h) : fallback
}
function colorAsOklch(v: string | undefined): string {
  if (!v) return ''
  if (v.startsWith('oklch')) return v
  if (v.startsWith('#')) return toOklchString(hexToOklch(v))
  return v
}

// ───────────────────────────────────────────────────────────────────────────
// Defaults
// ───────────────────────────────────────────────────────────────────────────

const DEFAULT_PRIMARY = '#7c3aed'

const FONT_CURATED = [
  // Brand-safe, no reflex fonts.
  { value: 'Inter',           label: 'Inter' },
  { value: 'Manrope',         label: 'Manrope' },
  { value: 'Hanken Grotesk',  label: 'Hanken Grotesk' },
  { value: 'IBM Plex Sans',   label: 'IBM Plex Sans' },
  { value: 'Source Sans 3',   label: 'Source Sans 3' },
  { value: 'Work Sans',       label: 'Work Sans' },
  { value: 'DM Sans',         label: 'DM Sans' },
  { value: 'Nunito Sans',     label: 'Nunito Sans' },
  { value: 'system-ui',       label: 'System UI' },
]

const FONT_MONO = [
  { value: 'Azeret Mono',     label: 'Azeret Mono' },
  { value: 'JetBrains Mono',  label: 'JetBrains Mono' },
  { value: 'IBM Plex Mono',   label: 'IBM Plex Mono' },
  { value: 'Fira Code',       label: 'Fira Code' },
  { value: 'Source Code Pro', label: 'Source Code Pro' },
  { value: 'ui-monospace',    label: 'System Mono' },
]

// ───────────────────────────────────────────────────────────────────────────
// URL state
// ───────────────────────────────────────────────────────────────────────────

function useUrlState() {
  const [state, setState] = React.useState(() => {
    const p = new URLSearchParams(window.location.search)
    return {
      section: p.get('section') || 'identity',
      surface: p.get('surface') || 'logo',
      pv: (p.get('pv') === 'light' ? 'light' : 'dark') as 'dark' | 'light',
    }
  })
  const apply = React.useCallback((next: Partial<typeof state>) => {
    setState(prev => {
      const merged = { ...prev, ...next }
      const p = new URLSearchParams(window.location.search)
      p.set('section', merged.section)
      p.set('surface', merged.surface)
      p.set('pv', merged.pv)
      const u = `${window.location.pathname}?${p.toString()}${window.location.hash}`
      window.history.replaceState(null, '', u)
      return merged
    })
  }, [])
  return [state, apply] as const
}

// ───────────────────────────────────────────────────────────────────────────
// Main page
// ───────────────────────────────────────────────────────────────────────────

export function Branding() {
  const toast = useToast()
  const [urlState, setUrlState] = useUrlState()
  const [previewWidth, setPreviewWidth] = React.useState(420)

  // ── Branding (visuals + auth copy) ────────────────────────────────
  const { data: brandingResp, refresh: refreshBranding } = useAPI('/admin/branding')
  const [brandCfg, setBrandCfg] = React.useState<any>(null)
  const [brandDraft, setBrandDraft] = React.useState<any>(null)
  React.useEffect(() => {
    if (brandingResp?.branding) {
      setBrandCfg(brandingResp.branding)
      setBrandDraft(brandingResp.branding)
    }
  }, [brandingResp])

  // ── Email templates ───────────────────────────────────────────────
  const { data: tplResp, refresh: refreshTpl } = useAPI('/admin/email-templates')
  const [templates, setTemplates] = React.useState<any[]>([])
  const [tplDraftMap, setTplDraftMap] = React.useState<Record<string, any>>({})
  React.useEffect(() => {
    if (tplResp?.data) {
      setTemplates(tplResp.data)
      // seed drafts
      setTplDraftMap(prev => {
        const next = { ...prev }
        for (const t of tplResp.data) if (!next[t.id]) next[t.id] = t
        return next
      })
    }
  }, [tplResp])

  // ── Apps / integrations ───────────────────────────────────────────
  const { data: appsResp, refresh: refreshApps } = useAPI('/admin/apps')
  const apps = React.useMemo(() => (
    appsResp?.data || appsResp?.applications || appsResp?.items || []
  ), [appsResp])

  // Dynamically populate email + integrations surfaces.
  const sections = React.useMemo(() => {
    return SECTIONS.map(s => {
      if (s.id === 'email') {
        return { ...s, surfaces: templates.map(t => ({ id: t.id, label: t.id.replace(/_/g, ' ') })) }
      }
      if (s.id === 'integrations') {
        return { ...s, surfaces: (apps || []).map((a: any) => ({ id: a.id, label: a.name || a.client_id })) }
      }
      return s
    })
  }, [templates, apps])

  // If current surface is invalid for current section, snap to the first one.
  React.useEffect(() => {
    const sec = sections.find(s => s.id === urlState.section)
    if (!sec) return
    if (sec.surfaces.length === 0) return
    if (!sec.surfaces.find(sf => sf.id === urlState.surface)) {
      setUrlState({ surface: sec.surfaces[0].id })
    }
  }, [sections, urlState.section, urlState.surface, setUrlState])

  // ── Dirty / save ──────────────────────────────────────────────────
  const brandDirty = React.useMemo(() => (
    brandCfg && brandDraft && JSON.stringify(brandCfg) !== JSON.stringify(brandDraft)
  ), [brandCfg, brandDraft])

  const activeTplId = urlState.section === 'email' ? urlState.surface : null
  const activeTpl = activeTplId ? templates.find(t => t.id === activeTplId) : null
  const activeTplDraft = activeTplId ? tplDraftMap[activeTplId] : null
  const tplDirty = !!(activeTpl && activeTplDraft && JSON.stringify(activeTpl) !== JSON.stringify(activeTplDraft))

  const anyDirty = brandDirty || tplDirty

  const save = async () => {
    try {
      if (brandDirty) {
        const r = await API.patch('/admin/branding', brandDraft)
        setBrandCfg(r.branding)
        setBrandDraft(r.branding)
      }
      if (tplDirty && activeTplDraft) {
        const body = {
          subject: activeTplDraft.subject || '',
          preheader: activeTplDraft.preheader || '',
          header_text: activeTplDraft.header_text || '',
          body_paragraphs: activeTplDraft.body_paragraphs || [],
          cta_text: activeTplDraft.cta_text || '',
          cta_url_template: activeTplDraft.cta_url_template || '',
          footer_text: activeTplDraft.footer_text || '',
        }
        const r = await API.patch(`/admin/email-templates/${activeTplDraft.id}`, body)
        setTemplates(prev => prev.map(t => (t.id === r.id ? r : t)))
        setTplDraftMap(prev => ({ ...prev, [r.id]: r }))
      }
      toast.success('Branding saved')
    } catch (e: any) {
      toast.error(e?.message || 'Save failed')
    }
  }
  const revert = () => {
    if (brandCfg) setBrandDraft(brandCfg)
    if (activeTpl) setTplDraftMap(prev => ({ ...prev, [activeTpl.id]: activeTpl }))
  }

  // Import / export (command palette new-theme + export/import)
  const exportTheme = React.useCallback(() => {
    if (!brandDraft) return
    const payload = JSON.stringify({ branding: brandDraft, exported_at: new Date().toISOString() }, null, 2)
    const blob = new Blob([payload], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `brand-theme-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('Theme exported')
  }, [brandDraft, toast])

  const fileInputRef = React.useRef<HTMLInputElement | null>(null)
  const importTheme = React.useCallback(() => { fileInputRef.current?.click() }, [])
  const onImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    e.target.value = ''
    if (!f) return
    try {
      const text = await f.text()
      const parsed = JSON.parse(text)
      if (parsed?.branding) {
        setBrandDraft((prev: any) => ({ ...(prev || {}), ...parsed.branding }))
        toast.success('Theme imported — review and publish')
      } else {
        toast.error('Theme JSON missing "branding"')
      }
    } catch (err: any) {
      toast.error(err?.message || 'Invalid theme JSON')
    }
  }

  // CommandPalette BRANDING_ACTIONS hooks via window events.
  React.useEffect(() => {
    const handler = (e: any) => {
      const a = e.detail?.action
      if (a === 'branding:export') exportTheme()
      else if (a === 'branding:import') importTheme()
      else if (a === 'branding:reset-palette') {
        if (window.confirm('Reset palette to defaults? This discards current color edits.')) {
          setBrandDraft((d: any) => ({
            ...(d || {}),
            primary_color: DEFAULT_PRIMARY,
            secondary_color: '#121212',
            surface_color: '#000000',
            text_color: '#fafafa',
            success_color: 'oklch(0.74 0.14 150)',
            warn_color: 'oklch(0.82 0.14 80)',
            danger_color: 'oklch(0.7 0.2 25)',
          }))
          toast.success('Palette reset to defaults')
        }
      }
    }
    window.addEventListener('shark-branding-action', handler)
    return () => window.removeEventListener('shark-branding-action', handler)
  }, [exportTheme, importTheme, toast])

  // Keyboard: r = refresh, n = open import
  usePageActions({
    onRefresh: () => { refreshBranding(); refreshTpl(); refreshApps(); toast.success('Branding reloaded') },
    onNew: () => importTheme(),
  })

  if (!brandDraft) {
    return (
      <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
        Loading BrandStudio…
      </div>
    )
  }

  const currentSection = sections.find(s => s.id === urlState.section) || sections[0]
  const currentSurface = currentSection.surfaces.find(s => s.id === urlState.surface) || currentSection.surfaces[0]

  return (
    <div className="col" style={{ height: '100%', background: 'var(--bg)' }}>
      {/* Topbar — flat, identity_hub pattern */}
      <div style={{
        padding: '14px 20px', borderBottom: '1px solid var(--hairline)',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      }}>
        <div>
          <h1 style={{
            fontFamily: 'var(--font-display)', fontSize: 18, fontWeight: 600,
            margin: 0, letterSpacing: '-0.01em',
          }}>BrandStudio</h1>
          <div className="faint" style={{ fontSize: 11, marginTop: 2 }}>
            Tune identity, palette, typography and auth surfaces — live.
          </div>
        </div>
        <div className="row" style={{ gap: 8 }}>
          <button className="btn ghost sm" onClick={exportTheme} title="Export theme JSON">
            <Icon.External width={12} height={12}/> Export
          </button>
          <button className="btn ghost sm" onClick={importTheme} title="Import theme JSON">
            <Icon.ArrowUp width={12} height={12}/> Import
          </button>
          <input
            ref={fileInputRef} type="file" accept="application/json"
            style={{ display: 'none' }} onChange={onImportFile}
          />
        </div>
      </div>

      {/* 3-column body */}
      <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
        {/* ── Left rail ─────────────────────────────────────────── */}
        <LeftRail
          sections={sections}
          selected={urlState}
          onSelect={(section, surface) => setUrlState({ section, surface })}
        />

        {/* ── Center editor ─────────────────────────────────────── */}
        <div style={{
          flex: 1, minWidth: 0, overflow: 'auto',
          borderRight: '1px solid var(--hairline)',
        }}>
          <CenterEditor
            section={currentSection.id}
            surface={currentSurface?.id || ''}
            surfaceLabel={currentSurface?.label || ''}
            brandDraft={brandDraft}
            setBrandDraft={setBrandDraft}
            tplDraft={activeTplDraft}
            setTplDraft={(next: any) => {
              if (!activeTplId) return
              setTplDraftMap(prev => ({ ...prev, [activeTplId]: next }))
            }}
            apps={apps}
            onAppUpdate={async (app: any, patch: any) => {
              try {
                await API.patch(`/admin/apps/${app.id}`, patch)
                refreshApps()
                toast.success('Application updated')
              } catch (e: any) { toast.error(e?.message || 'Update failed') }
            }}
            onLogoUpload={async (file: File) => {
              try {
                const form = new FormData()
                form.append('logo', file)
                const res = await API.postFormData('/admin/branding/logo', form)
                setBrandDraft((d: any) => ({ ...d, logo_url: res.logo_url, logo_sha: res.logo_sha }))
                setBrandCfg((c: any) => ({ ...(c || {}), logo_url: res.logo_url, logo_sha: res.logo_sha }))
                toast.success('Logo uploaded')
              } catch (e: any) { toast.error(e?.message || 'Logo upload failed') }
            }}
            onLogoRemove={async () => {
              try {
                await API.del('/admin/branding/logo')
                setBrandDraft((d: any) => ({ ...d, logo_url: '', logo_sha: '' }))
                setBrandCfg((c: any) => ({ ...(c || {}), logo_url: '', logo_sha: '' }))
                toast.success('Logo removed')
              } catch (e: any) { toast.error(e?.message || 'Remove failed') }
            }}
            onTplSendTest={async () => {
              if (!activeTplId) return
              const to = window.prompt('Send test email to:')
              if (!to) return
              try {
                await API.post(`/admin/email-templates/${activeTplId}/send-test`, { to_email: to })
                toast.success(`Test sent to ${to}`)
              } catch (e: any) { toast.error(e?.message || 'Send test failed') }
            }}
            onTplReset={async () => {
              if (!activeTplId) return
              if (!window.confirm('Reset template to default? Current edits will be lost.')) return
              try {
                const r = await API.post(`/admin/email-templates/${activeTplId}/reset`, {})
                setTemplates(prev => prev.map(t => (t.id === r.id ? r : t)))
                setTplDraftMap(prev => ({ ...prev, [r.id]: r }))
                toast.success('Template reset')
              } catch (e: any) { toast.error(e?.message || 'Reset failed') }
            }}
          />

          {/* Sticky unsaved bar */}
          {anyDirty && (
            <div style={{
              position: 'sticky', bottom: 0, left: 0, right: 0,
              padding: '10px 20px',
              background: 'var(--surface-1)', borderTop: '1px solid var(--hairline-strong)',
              display: 'flex', alignItems: 'center', gap: 12,
              backdropFilter: 'saturate(1.2)',
            }}>
              <div style={{ width: 6, height: 6, borderRadius: 1, background: 'var(--warn)' }}/>
              <div style={{ fontSize: 12, color: 'var(--fg)' }}>
                Unsaved changes
              </div>
              <div style={{ flex: 1 }}/>
              <button className="btn ghost sm" onClick={revert}>Revert</button>
              <button className="btn primary sm" onClick={save}>Save</button>
            </div>
          )}
        </div>

        {/* ── Right preview ─────────────────────────────────────── */}
        <PreviewPane
          width={previewWidth}
          onWidthChange={setPreviewWidth}
          pv={urlState.pv}
          onPvChange={(pv) => setUrlState({ pv })}
          section={currentSection.id}
          surface={currentSurface?.id || ''}
          brandDraft={brandDraft}
          onPreviewSurface={(s) => setUrlState({ section: 'auth', surface: s })}
          tplDraft={activeTplDraft}
          templateId={activeTplId}
        />
      </div>
    </div>
  )
}

// ───────────────────────────────────────────────────────────────────────────
// Left rail
// ───────────────────────────────────────────────────────────────────────────

function LeftRail({ sections, selected, onSelect }: {
  sections: any[]
  selected: { section: string; surface: string }
  onSelect: (section: string, surface: string) => void
}) {
  return (
    <div style={{
      flex: '0 0 220px', width: 220,
      borderRight: '1px solid var(--hairline)',
      overflow: 'auto', padding: '12px 0',
      background: 'var(--surface-0)',
    }}>
      {sections.map(section => {
        const IconEl = Icon[section.icon] || Icon.Brand
        const isActiveSec = selected.section === section.id
        return (
          <div key={section.id} style={{ marginBottom: 10 }}>
            <div
              onClick={() => {
                const first = section.surfaces[0]
                onSelect(section.id, first ? first.id : '')
              }}
              style={{
                display: 'flex', alignItems: 'center', gap: 8,
                padding: '4px 14px',
                fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
                fontWeight: 600, color: 'var(--fg-dim)',
                cursor: 'pointer',
              }}
            >
              <IconEl width={12} height={12} style={{ color: 'var(--fg-dim)' }}/>
              {section.label}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', marginTop: 2 }}>
              {section.surfaces.length === 0 && (
                <div style={{ padding: '2px 14px 2px 34px', fontSize: 11, color: 'var(--fg-faint)' }}>
                  (empty)
                </div>
              )}
              {section.surfaces.map((sf: any) => {
                const active = isActiveSec && selected.surface === sf.id
                return (
                  <button
                    key={sf.id}
                    type="button"
                    onClick={() => onSelect(section.id, sf.id)}
                    style={{
                      all: 'unset',
                      display: 'flex', alignItems: 'center',
                      padding: '5px 14px 5px 34px',
                      fontSize: 12, lineHeight: 1.3,
                      color: active ? 'var(--fg)' : 'var(--fg-muted)',
                      background: active ? 'var(--surface-2)' : 'transparent',
                      cursor: 'pointer',
                      fontWeight: active ? 600 : 400,
                      textTransform: 'capitalize',
                    }}
                  >
                    {sf.label}
                  </button>
                )
              })}
            </div>
          </div>
        )
      })}
    </div>
  )
}

// ───────────────────────────────────────────────────────────────────────────
// Center editor — dispatches to per-surface editors
// ───────────────────────────────────────────────────────────────────────────

function CenterEditor(props: {
  section: string
  surface: string
  surfaceLabel: string
  brandDraft: any
  setBrandDraft: (updater: any) => void
  tplDraft: any
  setTplDraft: (next: any) => void
  apps: any[]
  onAppUpdate: (app: any, patch: any) => void
  onLogoUpload: (file: File) => void
  onLogoRemove: () => void
  onTplSendTest: () => void
  onTplReset: () => void
}) {
  const { section, surface, surfaceLabel, brandDraft, setBrandDraft } = props
  return (
    <div style={{ padding: '18px 24px 24px' }}>
      <div style={{ marginBottom: 16 }}>
        <div style={{
          fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
          color: 'var(--fg-dim)', fontWeight: 500,
        }}>{section}</div>
        <h2 style={{
          fontFamily: 'var(--font-display)', fontSize: 16, fontWeight: 600,
          margin: '2px 0 0', textTransform: 'capitalize',
        }}>{surfaceLabel}</h2>
      </div>

      {section === 'identity'    && <IdentityEditor  surface={surface} brandDraft={brandDraft} setBrandDraft={setBrandDraft} onLogoUpload={props.onLogoUpload} onLogoRemove={props.onLogoRemove}/>}
      {section === 'palette'     && <PaletteEditor   surface={surface} brandDraft={brandDraft} setBrandDraft={setBrandDraft}/>}
      {section === 'typography'  && <TypographyEditor surface={surface} brandDraft={brandDraft} setBrandDraft={setBrandDraft}/>}
      {section === 'auth'        && <AuthCopyEditor  surface={surface} brandDraft={brandDraft} setBrandDraft={setBrandDraft}/>}
      {section === 'email'       && <EmailEditor     tplDraft={props.tplDraft} setTplDraft={props.setTplDraft} onSendTest={props.onTplSendTest} onReset={props.onTplReset}/>}
      {section === 'integrations'&& <IntegrationsEditor surface={surface} apps={props.apps} onAppUpdate={props.onAppUpdate}/>}
    </div>
  )
}

// ── Row primitive — dense hairline pattern ────────────────────────────────
function Row({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <div style={{
      display: 'grid', gridTemplateColumns: '180px 1fr',
      gap: 16, padding: '12px 0',
      borderBottom: '1px solid var(--hairline)',
      alignItems: 'flex-start',
    }}>
      <div>
        <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--fg)' }}>{label}</div>
        {hint && <div style={{ fontSize: 10.5, color: 'var(--fg-dim)', marginTop: 2 }}>{hint}</div>}
      </div>
      <div style={{ minWidth: 0 }}>{children}</div>
    </div>
  )
}

const CTL_INPUT: React.CSSProperties = {
  width: '100%', height: 30, padding: '0 10px',
  fontSize: 12.5, color: 'var(--fg)',
  background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)',
  borderRadius: 4, outline: 'none',
}

// ── Identity editor ────────────────────────────────────────────────────────
function IdentityEditor({ surface, brandDraft, setBrandDraft, onLogoUpload, onLogoRemove }: any) {
  const patch = (p: any) => setBrandDraft((d: any) => ({ ...d, ...p }))
  if (surface === 'logo') {
    return (
      <div>
        <Row label="Logo" hint="PNG, SVG, or JPG · max 1MB · transparent bg recommended">
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
            <div style={{
              width: 140, height: 60, borderRadius: 4,
              background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>
              {brandDraft.logo_url ? (
                <img src={brandDraft.logo_url} alt="logo" style={{ maxHeight: 44, maxWidth: 124 }}/>
              ) : (
                <span style={{ fontSize: 10, color: 'var(--fg-faint)' }}>no logo</span>
              )}
            </div>
            <label className="btn sm" style={{ cursor: 'pointer' }}>
              Upload
              <input type="file" accept=".png,.svg,.jpg,.jpeg" style={{ display: 'none' }}
                onChange={(e) => { const f = e.target.files?.[0]; if (f) onLogoUpload(f); e.target.value = '' }}/>
            </label>
            {brandDraft.logo_url && (
              <button className="btn danger sm" onClick={onLogoRemove}>Remove</button>
            )}
          </div>
        </Row>
      </div>
    )
  }
  if (surface === 'wordmark') {
    return (
      <div>
        <Row label="Wordmark URL" hint="Optional text-logo image used alongside the glyph">
          <input style={CTL_INPUT} value={brandDraft.wordmark_url || ''}
            onChange={(e) => patch({ wordmark_url: e.target.value })}
            placeholder="https://example.com/wordmark.svg"/>
        </Row>
      </div>
    )
  }
  if (surface === 'favicon') {
    return (
      <div>
        <Row label="Favicon URL" hint="32×32 .ico or .png served on the hosted auth pages">
          <input style={CTL_INPUT} value={brandDraft.favicon_url || ''}
            onChange={(e) => patch({ favicon_url: e.target.value })}
            placeholder="https://example.com/favicon.ico"/>
        </Row>
      </div>
    )
  }
  if (surface === 'app-name') {
    return (
      <div>
        <Row label="App name" hint="Shown in sign-in / emails. Keep it short.">
          <input style={CTL_INPUT} value={brandDraft.app_name || ''}
            onChange={(e) => patch({ app_name: e.target.value })}
            placeholder="Acme"/>
        </Row>
        <Row label="Sender display name" hint="From-name on transactional emails">
          <input style={CTL_INPUT} value={brandDraft.email_from_name || ''}
            onChange={(e) => patch({ email_from_name: e.target.value })}
            placeholder="Acme Security"/>
        </Row>
        <Row label="Reply-to address" hint="From-address on transactional emails">
          <input style={CTL_INPUT} value={brandDraft.email_from_address || ''}
            onChange={(e) => patch({ email_from_address: e.target.value })}
            placeholder="no-reply@acme.com"/>
        </Row>
        <Row label="Footer text" hint="Used at the bottom of transactional emails">
          <textarea style={{ ...CTL_INPUT, height: 'auto', minHeight: 60, padding: '6px 10px', resize: 'vertical' }}
            value={brandDraft.footer_text || ''}
            onChange={(e) => patch({ footer_text: e.target.value })}
            placeholder="© Acme 2026 · support@acme.com"/>
        </Row>
      </div>
    )
  }
  return null
}

// ── Palette editor ─────────────────────────────────────────────────────────
function PaletteEditor({ surface, brandDraft, setBrandDraft }: any) {
  if (surface === 'primary') {
    return (
      <ColorRow
        label="Primary accent"
        hint="CSS var --shark-primary in the preview"
        cssVar="--shark-primary"
        value={brandDraft.primary_color}
        onChange={(v) => setBrandDraft((d: any) => ({ ...d, primary_color: v }))}
        onReset={() => setBrandDraft((d: any) => ({ ...d, primary_color: DEFAULT_PRIMARY }))}
      />
    )
  }
  if (surface === 'surfaces') {
    return (
      <>
        <ColorRow label="Base surface"    cssVar="--surface-0"
          value={brandDraft.surface_color} onChange={(v) => setBrandDraft((d: any) => ({ ...d, surface_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, surface_color: '#000000' }))}/>
        <ColorRow label="Card surface"    cssVar="--surface-1"
          value={brandDraft.secondary_color} onChange={(v) => setBrandDraft((d: any) => ({ ...d, secondary_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, secondary_color: '#0c0c0c' }))}/>
      </>
    )
  }
  if (surface === 'text') {
    return (
      <>
        <ColorRow label="Primary text"   cssVar="--fg"
          value={brandDraft.text_color}       onChange={(v) => setBrandDraft((d: any) => ({ ...d, text_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, text_color: '#fafafa' }))}/>
        <ColorRow label="Muted text"     cssVar="--fg-muted"
          value={brandDraft.text_muted_color} onChange={(v) => setBrandDraft((d: any) => ({ ...d, text_muted_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, text_muted_color: '#a3a3a3' }))}/>
      </>
    )
  }
  if (surface === 'semantic') {
    return (
      <>
        <ColorRow label="Success" cssVar="--success"
          value={brandDraft.success_color} onChange={(v) => setBrandDraft((d: any) => ({ ...d, success_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, success_color: 'oklch(0.74 0.14 150)' }))}/>
        <ColorRow label="Warning" cssVar="--warn"
          value={brandDraft.warn_color} onChange={(v) => setBrandDraft((d: any) => ({ ...d, warn_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, warn_color: 'oklch(0.82 0.14 80)' }))}/>
        <ColorRow label="Danger"  cssVar="--danger"
          value={brandDraft.danger_color} onChange={(v) => setBrandDraft((d: any) => ({ ...d, danger_color: v }))}
          onReset={() => setBrandDraft((d: any) => ({ ...d, danger_color: 'oklch(0.7 0.2 25)' }))}/>
      </>
    )
  }
  return null
}

function ColorRow({ label, hint, cssVar, value, onChange, onReset }: {
  label: string; hint?: string; cssVar: string; value?: string; onChange: (v: string) => void; onReset: () => void
}) {
  const [menuOpen, setMenuOpen] = React.useState(false)
  const hex = colorAsHex(value || '', '#7c3aed')
  const oklchStr = React.useMemo(() => colorAsOklch(value) || toOklchString(hexToOklch(hex)), [value, hex])
  const copy = async (text: string, label: string) => {
    try { await navigator.clipboard.writeText(text); /* toast is injected at parent level */ } catch (_) {}
    setMenuOpen(false)
  }
  return (
    <Row label={label} hint={hint || `CSS var ${cssVar}`}>
      <div style={{ display: 'flex', gap: 12, alignItems: 'flex-start', flexWrap: 'wrap', position: 'relative' }}>
        <div>
          <HexColorPicker color={hex} onChange={(h) => onChange(toOklchString(hexToOklch(h)))}
            style={{ width: 160, height: 120 }}/>
        </div>
        <div style={{ flex: 1, minWidth: 200, display: 'flex', flexDirection: 'column', gap: 6 }}>
          <div style={{ fontSize: 10, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
            Stored value (oklch)
          </div>
          <input style={{ ...CTL_INPUT, fontFamily: 'var(--font-mono)', fontSize: 11 }}
            value={oklchStr}
            onChange={(e) => onChange(e.target.value)}
            spellCheck={false}/>
          <div style={{ fontSize: 10, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.08em', marginTop: 4 }}>
            Derived hex (read-only)
          </div>
          <input readOnly style={{ ...CTL_INPUT, fontFamily: 'var(--font-mono)', fontSize: 11, opacity: 0.75 }}
            value={hex}/>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 6 }}>
            <div style={{
              width: 28, height: 28, borderRadius: 4,
              background: hex, border: '1px solid var(--hairline-strong)',
            }}/>
            <button className="btn ghost sm" onClick={() => setMenuOpen(v => !v)} title="More actions">
              <Icon.More width={12} height={12}/>
            </button>
            {menuOpen && (
              <div style={{
                position: 'absolute', top: 60, right: 0,
                background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)',
                borderRadius: 6, padding: 4, zIndex: 10, minWidth: 180,
                boxShadow: 'var(--shadow-lg)',
              }}>
                <MenuItem onClick={() => copy(oklchStr, 'oklch')}>Copy oklch value</MenuItem>
                <MenuItem onClick={() => copy(hex, 'hex')}>Copy hex</MenuItem>
                <MenuItem onClick={() => copy(cssVar, 'var')}>Copy CSS var name</MenuItem>
                <MenuItem onClick={() => { onReset(); setMenuOpen(false) }}>Reset to default</MenuItem>
              </div>
            )}
          </div>
        </div>
      </div>
    </Row>
  )
}

function MenuItem({ children, onClick }: { children: React.ReactNode; onClick: () => void }) {
  return (
    <button onClick={onClick} style={{
      all: 'unset', cursor: 'pointer',
      display: 'block', padding: '6px 10px', fontSize: 12,
      color: 'var(--fg)', width: '100%',
    }}>{children}</button>
  )
}

// ── Typography editor ──────────────────────────────────────────────────────
function TypographyEditor({ surface, brandDraft, setBrandDraft }: any) {
  const patch = (p: any) => setBrandDraft((d: any) => ({ ...d, ...p }))
  if (surface === 'display' || surface === 'body') {
    const key = surface === 'display' ? 'font_display' : 'font_family'
    const current = brandDraft[key] || (surface === 'display' ? 'Hanken Grotesk' : 'Manrope')
    return (
      <Row label={surface === 'display' ? 'Display font' : 'Body font'}
        hint={surface === 'display' ? 'Headings in hosted auth pages' : 'Form labels, body copy'}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <select style={{ ...CTL_INPUT, appearance: 'auto' }}
            value={current}
            onChange={(e) => patch({ [key]: e.target.value })}>
            {FONT_CURATED.map(f => <option key={f.value} value={f.value}>{f.label}</option>)}
          </select>
          <div style={{
            padding: 14, borderRadius: 4,
            background: 'var(--surface-1)', border: '1px solid var(--hairline)',
            fontFamily: current, color: 'var(--fg)',
            fontSize: surface === 'display' ? 22 : 14,
            fontWeight: surface === 'display' ? 600 : 400,
            letterSpacing: surface === 'display' ? '-0.01em' : 0,
          }}>
            The quick brown fox jumps over the lazy dog — 1234567890
          </div>
        </div>
      </Row>
    )
  }
  if (surface === 'mono') {
    const current = brandDraft.font_mono || 'Azeret Mono'
    return (
      <Row label="Mono font" hint="Codes, IDs, token values">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <select style={{ ...CTL_INPUT, appearance: 'auto' }}
            value={current}
            onChange={(e) => patch({ font_mono: e.target.value })}>
            {FONT_MONO.map(f => <option key={f.value} value={f.value}>{f.label}</option>)}
          </select>
          <div style={{
            padding: 14, borderRadius: 4,
            background: 'var(--surface-1)', border: '1px solid var(--hairline)',
            fontFamily: current, fontSize: 12, color: 'var(--fg)',
          }}>
            tok_1234567890abcdef · 0xDEADBEEF · GET /api/v1/auth
          </div>
        </div>
      </Row>
    )
  }
  if (surface === 'scale') {
    const base = Number(brandDraft.font_base_size || 13)
    return (
      <Row label="Base size" hint="Used as the body font-size. Display scales up from this.">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <input type="number" min={11} max={18} step={0.5}
            style={{ ...CTL_INPUT, width: 120 }}
            value={base}
            onChange={(e) => patch({ font_base_size: Number(e.target.value) || 13 })}/>
          <div style={{
            padding: 14, borderRadius: 4,
            background: 'var(--surface-1)', border: '1px solid var(--hairline)',
            fontSize: base, color: 'var(--fg)',
          }}>
            Body ({base}px) — the fastest dashboard interaction is one that doesn't require waiting.
          </div>
        </div>
      </Row>
    )
  }
  return null
}

// ── Auth copy META editor ──────────────────────────────────────────────────
function AuthCopyEditor({ surface, brandDraft, setBrandDraft }: any) {
  const overrides = brandDraft.auth_copy || {}
  const current = { ...(AUTH_COPY_DEFAULTS[surface] || { heading: '', subheading: '', cta: '', smallPrint: '' }), ...(overrides[surface] || {}) }
  const setField = (field: string, value: string) => {
    setBrandDraft((d: any) => ({
      ...d,
      auth_copy: { ...(d.auth_copy || {}), [surface]: { ...(d.auth_copy?.[surface] || {}), [field]: value } },
    }))
  }
  const resetSurface = () => {
    setBrandDraft((d: any) => {
      const next = { ...(d.auth_copy || {}) }
      delete next[surface]
      return { ...d, auth_copy: next }
    })
  }
  return (
    <div>
      <Row label="Heading" hint="Main H1 shown above the form">
        <input style={CTL_INPUT} value={current.heading}
          onChange={(e) => setField('heading', e.target.value)}/>
      </Row>
      <Row label="Subheading" hint="Secondary line under the heading">
        <input style={CTL_INPUT} value={current.subheading}
          onChange={(e) => setField('subheading', e.target.value)}/>
      </Row>
      <Row label="Primary button label" hint="CTA text on the main action">
        <input style={CTL_INPUT} value={current.cta}
          onChange={(e) => setField('cta', e.target.value)}/>
      </Row>
      <Row label="Small print" hint="Fine-print under the form. Terms, expiry, etc.">
        <textarea style={{ ...CTL_INPUT, height: 'auto', minHeight: 60, padding: '6px 10px', resize: 'vertical' }}
          value={current.smallPrint}
          onChange={(e) => setField('smallPrint', e.target.value)}/>
      </Row>
      <Row label="Trust chips" hint="Comma-separated labels shown as footer chips">
        <input style={CTL_INPUT}
          value={(current as any).trustChips || ''}
          onChange={(e) => setField('trustChips', e.target.value)}
          placeholder="SOC 2, GDPR, E2EE"/>
      </Row>
      <div style={{ marginTop: 12 }}>
        <button className="btn ghost sm" onClick={resetSurface}>Reset to defaults</button>
      </div>
    </div>
  )
}

// ── Email editor (preserves all old email_tab fields) ─────────────────────
function EmailEditor({ tplDraft, setTplDraft, onSendTest, onReset }: any) {
  if (!tplDraft) {
    return <div style={{ color: 'var(--fg-dim)', fontSize: 12 }}>Select a template from the left rail.</div>
  }
  const patch = (p: any) => setTplDraft({ ...tplDraft, ...p })
  const setParagraph = (idx: number, value: string) => {
    const next = [...(tplDraft.body_paragraphs || [])]; next[idx] = value
    setTplDraft({ ...tplDraft, body_paragraphs: next })
  }
  const addParagraph = () => setTplDraft({ ...tplDraft, body_paragraphs: [...(tplDraft.body_paragraphs || []), ''] })
  const removeParagraph = (idx: number) => {
    const next = [...(tplDraft.body_paragraphs || [])]; next.splice(idx, 1)
    setTplDraft({ ...tplDraft, body_paragraphs: next })
  }
  const paragraphs: string[] = tplDraft.body_paragraphs || []
  return (
    <div>
      <Row label="Subject">
        <input style={CTL_INPUT} value={tplDraft.subject || ''} onChange={(e) => patch({ subject: e.target.value })}/>
      </Row>
      <Row label="Preheader" hint="Preview snippet shown in the inbox">
        <input style={CTL_INPUT} value={tplDraft.preheader || ''} onChange={(e) => patch({ preheader: e.target.value })}/>
      </Row>
      <Row label="Header text" hint="Large heading at the top of the message">
        <input style={CTL_INPUT} value={tplDraft.header_text || ''} onChange={(e) => patch({ header_text: e.target.value })}/>
      </Row>
      <Row label="Body paragraphs" hint="Plain text — mustache tokens like {{.AppName}} are allowed">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {paragraphs.map((p, i) => (
            <div key={i} style={{ display: 'flex', gap: 6 }}>
              <textarea style={{ ...CTL_INPUT, flex: 1, height: 'auto', minHeight: 54, padding: '6px 10px', resize: 'vertical' }}
                value={p} onChange={(e) => setParagraph(i, e.target.value)}/>
              <button className="btn danger sm" style={{ width: 28, padding: 0 }} onClick={() => removeParagraph(i)}>
                <Icon.X width={12} height={12}/>
              </button>
            </div>
          ))}
          <button className="btn ghost sm" style={{ alignSelf: 'flex-start' }} onClick={addParagraph}>
            <Icon.Plus width={12} height={12}/> Add paragraph
          </button>
        </div>
      </Row>
      <Row label="Call-to-action text">
        <input style={CTL_INPUT} value={tplDraft.cta_text || ''} onChange={(e) => patch({ cta_text: e.target.value })}/>
      </Row>
      <Row label="Call-to-action URL template" hint="Usually {{.Link}}">
        <input style={{ ...CTL_INPUT, fontFamily: 'var(--font-mono)' }}
          spellCheck={false}
          value={tplDraft.cta_url_template || ''} onChange={(e) => patch({ cta_url_template: e.target.value })}/>
      </Row>
      <Row label="Footer" hint="Fine-print under the CTA">
        <textarea style={{ ...CTL_INPUT, height: 'auto', minHeight: 60, padding: '6px 10px', resize: 'vertical' }}
          value={tplDraft.footer_text || ''} onChange={(e) => patch({ footer_text: e.target.value })}/>
      </Row>
      <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
        <button className="btn ghost sm" onClick={onSendTest}>
          <Icon.Mail width={12} height={12}/> Send test
        </button>
        <div style={{ flex: 1 }}/>
        <button className="btn danger sm" onClick={onReset}>Reset to default</button>
      </div>
    </div>
  )
}

// ── Integrations editor ────────────────────────────────────────────────────
const INTEGRATION_MODES = [
  { value: 'hosted',     label: 'Hosted' },
  { value: 'components', label: 'Components' },
  { value: 'proxy',      label: 'Proxy' },
  { value: 'custom',     label: 'Custom' },
]
function IntegrationsEditor({ surface, apps, onAppUpdate }: any) {
  const app = (apps || []).find((a: any) => a.id === surface)
  const toast = useToast()
  const [snippetModal, setSnippetModal] = React.useState<any>(null)
  if (!app) {
    return <div style={{ color: 'var(--fg-dim)', fontSize: 12 }}>Select an application from the left rail.</div>
  }
  const openSnippet = async () => {
    try {
      const r = await API.get(`/admin/apps/${app.id}/snippet?framework=react`)
      setSnippetModal({ snippets: r?.snippets || [] })
    } catch (e: any) { toast.error(e?.message || 'Failed to load snippet') }
  }
  const mode = app.integration_mode || 'hosted'
  const slug = app.slug || app.client_id || app.id
  return (
    <div>
      <Row label="Application">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <div style={{ fontSize: 13, fontWeight: 500 }}>{app.name}</div>
          <div style={{ fontSize: 10.5, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>{app.client_id}</div>
        </div>
      </Row>
      <Row label="Integration mode" hint="How this app presents auth to its users">
        <select style={{ ...CTL_INPUT, appearance: 'auto' }}
          value={mode} onChange={(e) => onAppUpdate(app, { integration_mode: e.target.value })}>
          {INTEGRATION_MODES.map(m => <option key={m.value} value={m.value}>{m.label}</option>)}
        </select>
      </Row>
      {mode === 'hosted' && (
        <Row label="Hosted login URL">
          <a href={`/hosted/${slug}/login`} target="_blank" rel="noopener noreferrer" className="btn sm">
            <Icon.External width={12} height={12}/> Open hosted login
          </a>
        </Row>
      )}
      {mode === 'components' && (
        <Row label="Code snippet" hint="React quickstart for this application">
          <button className="btn primary sm" onClick={openSnippet}>
            <Icon.Copy width={12} height={12}/> Get snippet
          </button>
        </Row>
      )}
      {mode === 'proxy' && (
        <>
          <Row label="Fallback" hint="What the proxy should do on login when the upstream has no page">
            <select style={{ ...CTL_INPUT, appearance: 'auto' }}
              value={app.proxy_login_fallback || 'hosted'}
              onChange={(e) => onAppUpdate(app, { proxy_login_fallback: e.target.value })}>
              <option value="hosted">hosted</option>
              <option value="custom_url">custom_url</option>
            </select>
          </Row>
          {app.proxy_login_fallback === 'custom_url' && (
            <Row label="Custom URL">
              <input style={CTL_INPUT}
                defaultValue={app.proxy_login_fallback_url || ''}
                placeholder="https://app.example.com/login"
                onBlur={(e) => {
                  if (e.target.value !== (app.proxy_login_fallback_url || '')) {
                    onAppUpdate(app, { proxy_login_fallback_url: e.target.value })
                  }
                }}/>
            </Row>
          )}
        </>
      )}
      {mode === 'custom' && (
        <Row label="API-only" hint="This app uses the raw API at /api/v1/auth/*">
          <div style={{ fontSize: 12, color: 'var(--fg-muted)' }}>No UI integration — use the JSON auth endpoints directly.</div>
        </Row>
      )}
      {snippetModal && (
        <SnippetModal snippets={snippetModal.snippets} appName={app.name}
          onClose={() => setSnippetModal(null)}/>
      )}
    </div>
  )
}

function SnippetModal({ snippets, appName, onClose }: any) {
  const toast = useToast()
  const copy = async (c: string) => {
    try { await navigator.clipboard.writeText(c); toast.success('Copied') }
    catch { toast.error('Copy failed') }
  }
  return (
    <div onClick={onClose} style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50,
    }}>
      <div onClick={(e) => e.stopPropagation()} style={{
        background: 'var(--surface-1)', border: '1px solid var(--hairline-bright)',
        borderRadius: 6, padding: 20, width: 'min(720px, 92vw)', maxHeight: '88vh', overflow: 'auto',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 14 }}>
          <h2 style={{ margin: 0, fontSize: 15, fontWeight: 600, flex: 1 }}>Snippets: {appName}</h2>
          <button className="btn sm" onClick={onClose}>Close</button>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          {snippets.map((s: any, i: number) => (
            <div key={i}>
              <div style={{ display: 'flex', alignItems: 'center', marginBottom: 6 }}>
                <div style={{ flex: 1, fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', fontWeight: 500 }}>
                  {s.label}
                </div>
                <button className="btn sm" onClick={() => copy(s.code)}>Copy</button>
              </div>
              <pre style={{
                margin: 0, padding: 12, fontSize: 11.5, fontFamily: 'var(--font-mono)',
                background: 'var(--surface-0)', border: '1px solid var(--hairline)',
                borderRadius: 'var(--radius)', overflow: 'auto', whiteSpace: 'pre',
              }}>{s.code}</pre>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// ───────────────────────────────────────────────────────────────────────────
// Preview pane (user-resizable)
// ───────────────────────────────────────────────────────────────────────────

function PreviewPane(props: {
  width: number; onWidthChange: (n: number) => void
  pv: 'dark' | 'light'; onPvChange: (p: 'dark' | 'light') => void
  section: string; surface: string
  brandDraft: any
  onPreviewSurface: (s: string) => void
  tplDraft: any
  templateId: string | null
}) {
  const { width, onWidthChange, pv, onPvChange, section, surface, brandDraft } = props
  const dragging = React.useRef(false)

  React.useEffect(() => {
    const onMove = (e: MouseEvent) => {
      if (!dragging.current) return
      const next = Math.max(320, Math.min(720, window.innerWidth - e.clientX))
      onWidthChange(next)
    }
    const onUp = () => { dragging.current = false; document.body.style.cursor = '' }
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
    return () => { window.removeEventListener('mousemove', onMove); window.removeEventListener('mouseup', onUp) }
  }, [onWidthChange])

  // Build the theme-vars wrapper style from draft.
  const themeStyle = React.useMemo(() => computeThemeVars(brandDraft, pv), [brandDraft, pv])

  return (
    <div style={{ flex: `0 0 ${width}px`, width, position: 'relative', display: 'flex', flexDirection: 'column' }}>
      {/* Drag handle */}
      <div
        onMouseDown={() => { dragging.current = true; document.body.style.cursor = 'col-resize' }}
        style={{
          position: 'absolute', left: -3, top: 0, bottom: 0, width: 6,
          cursor: 'col-resize', zIndex: 5,
        }}
        title="Drag to resize"
      />
      {/* Preview toolbar */}
      <div style={{
        height: 40, padding: '0 12px',
        display: 'flex', alignItems: 'center', gap: 8,
        borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)',
      }}>
        <select
          value={section === 'auth' ? surface : '__none'}
          onChange={(e) => {
            if (e.target.value === '__none') return
            props.onPreviewSurface(e.target.value)
          }}
          style={{ ...CTL_INPUT, height: 26, width: 180, appearance: 'auto' }}
        >
          {section !== 'auth' && <option value="__none">Preview surface…</option>}
          {AUTH_SURFACES_FOR_PREVIEW.map(s => (
            <option key={s.id} value={s.id}>{s.label}</option>
          ))}
        </select>
        <div style={{ flex: 1 }}/>
        <div style={{
          display: 'inline-flex', border: '1px solid var(--hairline-strong)',
          borderRadius: 4, overflow: 'hidden',
        }}>
          {(['dark', 'light'] as const).map(opt => (
            <button key={opt}
              onClick={() => onPvChange(opt)}
              style={{
                all: 'unset', cursor: 'pointer',
                padding: '4px 10px', fontSize: 11,
                background: pv === opt ? 'var(--surface-2)' : 'transparent',
                color: pv === opt ? 'var(--fg)' : 'var(--fg-dim)',
                fontWeight: pv === opt ? 600 : 400,
              }}
            >{opt}</button>
          ))}
        </div>
      </div>

      {/* Preview body */}
      <div style={{ flex: 1, minHeight: 0, overflow: 'auto', background: 'var(--surface-0)' }}>
        <div style={themeStyle}>
          <PreviewBody section={section} surface={surface} brandDraft={brandDraft} templateId={props.templateId}/>
        </div>
      </div>
    </div>
  )
}

function computeThemeVars(brandDraft: any, pv: 'dark' | 'light'): React.CSSProperties {
  const primary = brandDraft?.primary_color || DEFAULT_PRIMARY
  const surfaceBase = brandDraft?.surface_color || (pv === 'light' ? '#ffffff' : '#000000')
  const surfaceCard = brandDraft?.secondary_color || (pv === 'light' ? '#f6f6f6' : '#0c0c0c')
  const text = brandDraft?.text_color || (pv === 'light' ? '#0a0a0a' : '#fafafa')
  const muted = brandDraft?.text_muted_color || (pv === 'light' ? '#5a5a5a' : '#a3a3a3')
  const hairline = pv === 'light' ? '#e5e5e5' : '#1e1e1e'
  const hairlineStrong = pv === 'light' ? '#cccccc' : '#2a2a2a'
  const display = brandDraft?.font_display || 'Hanken Grotesk'
  const body = brandDraft?.font_family || 'Manrope'
  const mono = brandDraft?.font_mono || 'Azeret Mono'
  return {
    // Shark CSS vars the composed auth components read.
    ['--bg' as any]: surfaceBase,
    ['--surface-0' as any]: surfaceBase,
    ['--surface-1' as any]: surfaceCard,
    ['--surface-2' as any]: surfaceCard,
    ['--surface-3' as any]: surfaceCard,
    ['--fg' as any]: text,
    ['--fg-muted' as any]: muted,
    ['--fg-dim' as any]: muted,
    ['--fg-faint' as any]: muted,
    ['--hairline' as any]: hairline,
    ['--hairline-strong' as any]: hairlineStrong,
    ['--hairline-bright' as any]: hairlineStrong,
    ['--accent' as any]: primary,
    ['--shark-primary' as any]: primary,
    ['--success' as any]: brandDraft?.success_color || 'oklch(0.74 0.14 150)',
    ['--warn' as any]:    brandDraft?.warn_color    || 'oklch(0.82 0.14 80)',
    ['--danger' as any]:  brandDraft?.danger_color  || 'oklch(0.7 0.2 25)',
    ['--font-sans' as any]: `'${body}', system-ui, sans-serif`,
    ['--font-display' as any]: `'${display}', system-ui, sans-serif`,
    ['--font-mono' as any]: `'${mono}', ui-monospace, Menlo, monospace`,
    color: text,
    minHeight: '100%',
  }
}

// Dispatch preview content by section.
function PreviewBody({ section, surface, brandDraft, templateId }: any) {
  if (section === 'auth') return <AuthSurfacePreview surface={surface} brandDraft={brandDraft}/>
  if (section === 'email' && templateId) return <EmailPreview templateId={templateId}/>
  return <CompositePreview brandDraft={brandDraft}/>
}

function AuthSurfacePreview({ surface, brandDraft }: { surface: string; brandDraft: any }) {
  const appName = brandDraft.app_name || 'Acme'
  const copy = { ...(AUTH_COPY_DEFAULTS[surface] || {}), ...((brandDraft.auth_copy || {})[surface] || {}) }
  const wrap = (node: React.ReactNode) => (
    <div style={{ transform: 'scale(0.85)', transformOrigin: 'top center', pointerEvents: 'none' }}>{node}</div>
  )
  try {
    switch (surface) {
      case 'sign-in':  return wrap(<SignInForm  appName={appName} authMethods={['password','magic_link','passkey','oauth']}
        oauthProviders={[{ id: 'google', name: 'Google' }]}
        signUpHref="#" forgotPasswordHref="#"/>)
      case 'sign-up':  return wrap(<SignUpForm  appName={appName} onSubmit={async () => {}} signInHref="#"/>)
      case 'mfa':      return wrap(<MFAForm     appName={appName} onSubmit={async () => {}} onCancel={() => {}}/>)
      case 'forgot-password': return wrap(<ForgotPasswordForm appName={appName} onSubmit={async () => {}} signInHref="#"/>)
      case 'reset-password':  return wrap(<ResetPasswordForm  appName={appName} onSubmit={async () => {}} signInHref="#"/>)
      case 'magic-link-sent': return wrap(<MagicLinkSent      appName={appName} email="user@example.com" signInHref="#"/>)
      case 'email-verify':    return wrap(<EmailVerify        appName={appName} state="pending" onResend={async () => {}}/>)
      case 'error-page':      return wrap(<ErrorPage          appName={appName} code="404" message={copy.heading || 'Not found'} homeHref="#"/>)
    }
  } catch (err) {
    return <PreviewError err={err as Error}/>
  }
  return <div style={{ padding: 20, color: 'var(--fg-dim)' }}>No preview for this surface.</div>
}

function PreviewError({ err }: { err: Error }) {
  return (
    <div style={{ padding: 20 }}>
      <div style={{ fontSize: 11, color: 'var(--danger)', marginBottom: 6 }}>Preview error</div>
      <div style={{ fontSize: 12, fontFamily: 'var(--font-mono)', color: 'var(--fg-muted)' }}>{err.message}</div>
    </div>
  )
}

function EmailPreview({ templateId }: { templateId: string }) {
  const [html, setHtml] = React.useState('')
  React.useEffect(() => {
    let cancelled = false
    const h = setTimeout(async () => {
      try {
        const r = await API.post(`/admin/email-templates/${templateId}/preview`, {})
        if (!cancelled) setHtml(r.html || '')
      } catch (_) { /* keep last-good */ }
    }, 120)
    return () => { cancelled = true; clearTimeout(h) }
  }, [templateId])
  return (
    <div style={{ padding: 16 }}>
      <iframe title="email preview" srcDoc={html} sandbox=""
        style={{ width: '100%', minHeight: 520, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: '#fff' }}/>
    </div>
  )
}

function CompositePreview({ brandDraft }: { brandDraft: any }) {
  const display = brandDraft?.font_display || 'Hanken Grotesk'
  const body = brandDraft?.font_family || 'Manrope'
  const mono = brandDraft?.font_mono || 'Azeret Mono'
  const swatches: { label: string; v?: string; fallback: string }[] = [
    { label: 'Primary',    v: brandDraft.primary_color,     fallback: DEFAULT_PRIMARY },
    { label: 'Surface',    v: brandDraft.surface_color,     fallback: '#000000' },
    { label: 'Card',       v: brandDraft.secondary_color,   fallback: '#0c0c0c' },
    { label: 'Text',       v: brandDraft.text_color,        fallback: '#fafafa' },
    { label: 'Success',    v: brandDraft.success_color,     fallback: 'oklch(0.74 0.14 150)' },
    { label: 'Warn',       v: brandDraft.warn_color,        fallback: 'oklch(0.82 0.14 80)' },
    { label: 'Danger',     v: brandDraft.danger_color,      fallback: 'oklch(0.7 0.2 25)' },
  ]
  return (
    <div style={{ padding: 20, display: 'flex', flexDirection: 'column', gap: 18 }}>
      {/* Logo stack */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
        {(['#000', '#fff'] as const).map((bg) => (
          <div key={bg} style={{
            height: 90, background: bg,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            border: '1px solid var(--hairline-strong)', borderRadius: 4,
          }}>
            {brandDraft.logo_url
              ? <img src={brandDraft.logo_url} alt="logo" style={{ maxHeight: 60, maxWidth: '80%' }}/>
              : <span style={{ color: bg === '#000' ? '#888' : '#999', fontSize: 11 }}>no logo</span>}
          </div>
        ))}
      </div>

      {/* Swatches */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8 }}>
        {swatches.map(s => {
          const hex = colorAsHex(s.v, s.fallback)
          return (
            <div key={s.label} style={{
              padding: 10, border: '1px solid var(--hairline)',
              borderRadius: 4, background: 'var(--surface-1)',
            }}>
              <div style={{ height: 36, background: hex, borderRadius: 3, border: '1px solid var(--hairline-strong)' }}/>
              <div style={{ marginTop: 6, fontSize: 10.5, color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
                {s.label}
              </div>
              <div style={{ fontSize: 10, color: 'var(--fg-dim)', fontFamily: mono }}>{hex}</div>
            </div>
          )
        })}
      </div>

      {/* Typography sample */}
      <div style={{ padding: 16, border: '1px solid var(--hairline)', borderRadius: 4, background: 'var(--surface-1)' }}>
        <div style={{ fontFamily: display, fontSize: 22, fontWeight: 600, letterSpacing: '-0.01em', color: 'var(--fg)' }}>
          The quick brown fox
        </div>
        <div style={{ fontFamily: body, fontSize: 14, color: 'var(--fg-muted)', marginTop: 4 }}>
          Jumps over the lazy dog — bold, elegant, fast.
        </div>
        <div style={{ fontFamily: mono, fontSize: 11, color: 'var(--fg-dim)', marginTop: 6 }}>
          tok_1234567890abcdef · GET /api/v1/auth
        </div>
      </div>
    </div>
  )
}
