import React from 'react'
import { tokens } from '../tokens'

export interface ModalProps {
  /** Controls visibility */
  open: boolean
  /** Called when modal should close (Esc, backdrop click, close button) */
  onClose: () => void
  /** aria-labelledby target — should match the id of the modal title element */
  titleId?: string
  /** Optional title rendered in default header */
  title?: string
  children?: React.ReactNode
  style?: React.CSSProperties
  /** Override max-width (default 480) */
  maxWidth?: number
}

/** Minimal inline focus trap: cycles focus through focusable elements on Tab */
function useFocusTrap(ref: React.RefObject<HTMLElement | null>, active: boolean) {
  React.useEffect(() => {
    if (!active || !ref.current) return

    const el = ref.current
    const FOCUSABLE = [
      'a[href]',
      'button:not([disabled])',
      'input:not([disabled])',
      'select:not([disabled])',
      'textarea:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
    ].join(',')

    // Focus first focusable on mount
    const first = el.querySelector<HTMLElement>(FOCUSABLE)
    first?.focus()

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Tab') return
      const focusable = Array.from(el.querySelectorAll<HTMLElement>(FOCUSABLE))
      if (focusable.length === 0) return

      const firstEl = focusable[0]
      const lastEl = focusable[focusable.length - 1]

      if (e.shiftKey) {
        if (document.activeElement === firstEl) {
          e.preventDefault()
          lastEl.focus()
        }
      } else {
        if (document.activeElement === lastEl) {
          e.preventDefault()
          firstEl.focus()
        }
      }
    }

    el.addEventListener('keydown', handleKeyDown)
    return () => el.removeEventListener('keydown', handleKeyDown)
  }, [active, ref])
}

const DEFAULT_TITLE_ID = 'shark-modal-title'

export function Modal({
  open,
  onClose,
  titleId = DEFAULT_TITLE_ID,
  title,
  children,
  style,
  maxWidth = 480,
}: ModalProps) {
  const dialogRef = React.useRef<HTMLDivElement>(null)

  useFocusTrap(dialogRef, open)

  // Escape key handler
  React.useEffect(() => {
    if (!open) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose])

  // Prevent body scroll while open
  React.useEffect(() => {
    if (!open) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = prev }
  }, [open])

  if (!open) return null

  const backdropStyle: React.CSSProperties = {
    position: 'fixed',
    inset: 0,
    background: 'oklch(0% 0 0 / 60%)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: tokens.zIndex.modal,
    padding: tokens.space[4],
  }

  const dialogStyle: React.CSSProperties = {
    position: 'relative',
    background: tokens.color.surface2,
    border: `1px solid ${tokens.color.hairline}`,
    borderRadius: tokens.radius.xl,
    padding: tokens.space[8],
    width: '100%',
    maxWidth,
    boxShadow: tokens.shadow.md,
    outline: 'none',
    ...style,
  }

  const closeBtnStyle: React.CSSProperties = {
    position: 'absolute',
    top: tokens.space[4],
    right: tokens.space[4],
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 28,
    height: 28,
    background: 'transparent',
    border: `1px solid ${tokens.color.hairline}`,
    borderRadius: tokens.radius.md,
    color: tokens.color.fgMuted,
    cursor: 'pointer',
    transition: `border-color ${tokens.motion.fast}, color ${tokens.motion.fast}`,
    outline: 'none',
  }

  const titleStyle: React.CSSProperties = {
    margin: 0,
    marginBottom: title ? tokens.space[4] : 0,
    fontSize: tokens.type.size.md,
    fontFamily: tokens.type.display.family,
    fontWeight: tokens.type.weight.semibold,
    color: tokens.color.fg,
    lineHeight: 1.2,
    paddingRight: 32,
  }

  function handleBackdropClick(e: React.MouseEvent) {
    if (e.target === e.currentTarget) onClose()
  }

  return (
    <div style={backdropStyle} onClick={handleBackdropClick} aria-hidden="false">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        tabIndex={-1}
        style={dialogStyle}
      >
        <button
          style={closeBtnStyle}
          onClick={onClose}
          aria-label="Close modal"
          type="button"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
            <path d="M2 2l8 8M10 2l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
          </svg>
        </button>
        {title && (
          <h2 id={titleId} style={titleStyle}>
            {title}
          </h2>
        )}
        {children}
      </div>
    </div>
  )
}

Modal.displayName = 'Modal'
