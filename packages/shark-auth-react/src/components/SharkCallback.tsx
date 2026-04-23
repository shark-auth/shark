import React from 'react'
import { exchangeCodeForToken } from '../core/auth'
import { getRedirectAfter } from '../core/storage'
import { AuthContext } from '../hooks/context'

export interface SharkCallbackProps {
  /** Override where to redirect after successful exchange. Defaults to stored shark_redirect_after or '/'. */
  fallbackRedirectUrl?: string
  /** Custom loading UI */
  loading?: React.ReactNode
  /** Called when exchange fails */
  onError?: (err: Error) => void
}

export function SharkCallback({
  fallbackRedirectUrl = '/',
  loading,
  onError,
}: SharkCallbackProps) {
  const ctx = React.useContext(AuthContext)
  const [error, setError] = React.useState<string | null>(null)

  React.useEffect(() => {
    const authUrl = ctx?.authUrl
    const publishableKey = ctx?.publishableKey
    if (!authUrl || !publishableKey) return

    const params = new URLSearchParams(
      typeof window !== 'undefined' ? window.location.search : '',
    )
    const code = params.get('code')
    if (!code) {
      const err = new Error('No authorization code in URL')
      setError(err.message)
      onError?.(err)
      return
    }

    exchangeCodeForToken(code, authUrl, publishableKey)
      .then(() => {
        const redirectTo = getRedirectAfter() ?? fallbackRedirectUrl
        if (typeof window !== 'undefined') {
          window.location.replace(redirectTo)
        }
      })
      .catch((err: unknown) => {
        const e = err instanceof Error ? err : new Error(String(err))
        setError(e.message)
        onError?.(e)
      })
  }, [ctx, fallbackRedirectUrl, onError])

  if (error) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <p style={{ color: '#ef4444', fontSize: 15 }}>Authentication failed: {error}</p>
      </div>
    )
  }

  return (
    <div style={{ padding: 24, textAlign: 'center' }}>
      {loading ?? (
        <p style={{ color: '#6b7280', fontSize: 15 }}>Signing in…</p>
      )}
    </div>
  )
}
