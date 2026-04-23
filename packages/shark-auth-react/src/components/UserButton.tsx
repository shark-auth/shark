import React from 'react'
import { useAuth } from '../hooks/useAuth'
import { useUser } from '../hooks/useUser'

export interface UserButtonProps {
  profileUrl?: string
  manageAccountUrl?: string
  afterSignOutUrl?: string
}

const styles = {
  wrapper: { position: 'relative' as const, display: 'inline-block' },
  avatar: {
    width: 36,
    height: 36,
    borderRadius: '50%',
    cursor: 'pointer',
    border: '2px solid #e5e7eb',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden' as const,
    background: '#6366f1',
    color: '#fff',
    fontSize: 14,
    fontWeight: 600,
    userSelect: 'none' as const,
  },
  dropdown: {
    position: 'absolute' as const,
    top: 44,
    right: 0,
    background: '#fff',
    border: '1px solid #e5e7eb',
    borderRadius: 8,
    boxShadow: '0 4px 16px rgba(0,0,0,0.12)',
    minWidth: 200,
    zIndex: 9999,
    overflow: 'hidden' as const,
  },
  menuItem: {
    display: 'block',
    width: '100%',
    padding: '10px 16px',
    textAlign: 'left' as const,
    background: 'none',
    border: 'none',
    cursor: 'pointer',
    fontSize: 14,
    color: '#111827',
    textDecoration: 'none',
  },
  menuItemHover: {
    background: '#f9fafb',
  },
  divider: {
    height: 1,
    background: '#e5e7eb',
    margin: '4px 0',
  },
} as const

export function UserButton({
  profileUrl = '/profile',
  manageAccountUrl = '/account',
  afterSignOutUrl = '/',
}: UserButtonProps) {
  const { isLoaded, isAuthenticated, signOut } = useAuth()
  const { user } = useUser()
  const [open, setOpen] = React.useState(false)
  const wrapperRef = React.useRef<HTMLDivElement>(null)

  // Close dropdown on outside click
  React.useEffect(() => {
    if (!open) return
    function handleOutside(e: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleOutside)
    return () => document.removeEventListener('mousedown', handleOutside)
  }, [open])

  if (!isLoaded || !isAuthenticated || !user) return null

  const initials = [user.firstName, user.lastName]
    .filter(Boolean)
    .map(s => s![0].toUpperCase())
    .join('') || user.email[0].toUpperCase()

  const handleSignOut = async () => {
    setOpen(false)
    await signOut()
    if (typeof window !== 'undefined') {
      window.location.href = afterSignOutUrl
    }
  }

  return (
    <div ref={wrapperRef} style={styles.wrapper}>
      <div
        role="button"
        tabIndex={0}
        aria-label="User menu"
        aria-expanded={open}
        style={styles.avatar}
        onClick={() => setOpen(o => !o)}
        onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') setOpen(o => !o) }}
      >
        {user.imageUrl ? (
          <img
            src={user.imageUrl}
            alt={user.email}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        ) : (
          initials
        )}
      </div>

      {open && (
        <div style={styles.dropdown} role="menu">
          <div style={{ padding: '12px 16px', borderBottom: '1px solid #e5e7eb' }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: '#111827' }}>
              {user.firstName || user.email}
            </div>
            <div style={{ fontSize: 12, color: '#6b7280', marginTop: 2 }}>{user.email}</div>
          </div>

          <a
            href={profileUrl}
            style={styles.menuItem}
            role="menuitem"
            onClick={() => setOpen(false)}
          >
            Profile
          </a>
          <a
            href={manageAccountUrl}
            style={styles.menuItem}
            role="menuitem"
            onClick={() => setOpen(false)}
          >
            Manage account
          </a>
          <div style={styles.divider} />
          <button
            type="button"
            style={{ ...styles.menuItem, color: '#ef4444' }}
            role="menuitem"
            onClick={handleSignOut}
          >
            Sign out
          </button>
        </div>
      )}
    </div>
  )
}
