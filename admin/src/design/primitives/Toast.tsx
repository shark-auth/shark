import React from 'react'
import { tokens } from '../tokens'

export type ToastVariant = 'info' | 'success' | 'warn' | 'danger'

export interface ToastItem {
  id: string
  variant: ToastVariant
  message: string
  duration?: number
}

export interface ToastContextValue {
  info: (message: string, duration?: number) => void
  success: (message: string, duration?: number) => void
  warn: (message: string, duration?: number) => void
  danger: (message: string, duration?: number) => void
  dismiss: (id: string) => void
}

export const ToastContext = React.createContext<ToastContextValue | null>(null)

const variantColors: Record<ToastVariant, { bg: string; border: string; color: string }> = {
  info: {
    bg: tokens.color.surface3,
    border: tokens.color.hairline,
    color: tokens.color.fgMuted,
  },
  success: {
    bg: 'oklch(18% 0.04 160)',
    border: 'oklch(28% 0.06 160)',
    color: tokens.color.success,
  },
  warn: {
    bg: 'oklch(18% 0.04 85)',
    border: 'oklch(28% 0.06 85)',
    color: tokens.color.warn,
  },
  danger: {
    bg: 'oklch(18% 0.06 25)',
    border: 'oklch(28% 0.08 25)',
    color: tokens.color.danger,
  },
}

const variantIcons: Record<ToastVariant, React.ReactNode> = {
  info: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.4"/>
      <path d="M8 7v4M8 5v.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
    </svg>
  ),
  success: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.4"/>
      <path d="M5 8l2 2 4-4" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round"/>
    </svg>
  ),
  warn: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path d="M8 2l6 11H2L8 2z" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round"/>
      <path d="M8 6.5v3M8 11.5v.5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
    </svg>
  ),
  danger: (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.4"/>
      <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round"/>
    </svg>
  ),
}

const SLIDE_IN_KEYFRAMES = `
@keyframes shark-toast-slide-in {
  from { transform: translateX(100%); opacity: 0; }
  to   { transform: translateX(0);   opacity: 1; }
}
`

interface ToastItemViewProps {
  item: ToastItem
  onDismiss: (id: string) => void
}

function ToastItemView({ item, onDismiss }: ToastItemViewProps) {
  const colors = variantColors[item.variant]

  const itemStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-start',
    gap: tokens.space[2],
    padding: `${tokens.space[2]}px ${tokens.space[3]}px`,
    background: colors.bg,
    border: `1px solid ${colors.border}`,
    borderRadius: tokens.radius.md,
    boxShadow: tokens.shadow.md,
    minWidth: 260,
    maxWidth: 360,
    animation: `shark-toast-slide-in ${tokens.motion.fast}`,
    color: colors.color,
    pointerEvents: 'auto',
  }

  const iconStyle: React.CSSProperties = {
    flexShrink: 0,
    marginTop: 1,
  }

  const messageStyle: React.CSSProperties = {
    flex: 1,
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    fontWeight: tokens.type.weight.medium,
    color: tokens.color.fg,
    lineHeight: 1.4,
  }

  const closeBtnStyle: React.CSSProperties = {
    flexShrink: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 18,
    height: 18,
    background: 'transparent',
    border: 'none',
    borderRadius: tokens.radius.sm,
    color: tokens.color.fgDim,
    cursor: 'pointer',
    padding: 0,
    marginTop: 1,
  }

  return (
    <div style={itemStyle} role="status" aria-live="polite" aria-atomic="true">
      <span style={iconStyle}>{variantIcons[item.variant]}</span>
      <span style={messageStyle}>{item.message}</span>
      <button
        style={closeBtnStyle}
        onClick={() => onDismiss(item.id)}
        aria-label="Dismiss notification"
        type="button"
      >
        <svg width="10" height="10" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <path d="M2 2l8 8M10 2l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
        </svg>
      </button>
    </div>
  )
}

export interface ToastProviderProps {
  children: React.ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = React.useState<ToastItem[]>([])
  const timers = React.useRef<Record<string, ReturnType<typeof setTimeout>>>({})

  const dismiss = React.useCallback((id: string) => {
    clearTimeout(timers.current[id])
    delete timers.current[id]
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const add = React.useCallback(
    (variant: ToastVariant, message: string, duration = 4000) => {
      const id = `${Date.now()}-${Math.random().toString(36).slice(2)}`
      setToasts((prev) => [...prev, { id, variant, message, duration }].slice(-5))
      timers.current[id] = setTimeout(() => dismiss(id), duration)
    },
    [dismiss],
  )

  // Cleanup on unmount
  React.useEffect(() => {
    const t = timers.current
    return () => { Object.values(t).forEach(clearTimeout) }
  }, [])

  const ctx: ToastContextValue = React.useMemo(
    () => ({
      info: (msg, dur) => add('info', msg, dur),
      success: (msg, dur) => add('success', msg, dur),
      warn: (msg, dur) => add('warn', msg, dur),
      danger: (msg, dur) => add('danger', msg, dur),
      dismiss,
    }),
    [add, dismiss],
  )

  const containerStyle: React.CSSProperties = {
    position: 'fixed',
    bottom: tokens.space[4],
    right: tokens.space[4],
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.space[2],
    zIndex: tokens.zIndex.toast,
    pointerEvents: 'none',
  }

  return (
    <ToastContext.Provider value={ctx}>
      <style>{SLIDE_IN_KEYFRAMES}</style>
      {children}
      <div style={containerStyle} aria-label="Notifications" aria-live="polite">
        {toasts.map((item) => (
          <ToastItemView key={item.id} item={item} onDismiss={dismiss} />
        ))}
      </div>
    </ToastContext.Provider>
  )
}

ToastProvider.displayName = 'ToastProvider'

export function useToast(): ToastContextValue {
  const ctx = React.useContext(ToastContext)
  if (!ctx) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return ctx
}
