import React from 'react'
import { startAuthFlow } from '../core/auth'
import { AuthContext } from '../hooks/context'

export interface SignInProps {
  redirectUrl?: string
  children?: React.ReactNode
  className?: string
}

export function SignIn({ redirectUrl, children, className }: SignInProps) {
  const ctx = React.useContext(AuthContext)

  const handleClick = React.useCallback(async () => {
    const authUrl = ctx?.authUrl
    const publishableKey = ctx?.publishableKey
    if (!authUrl || !publishableKey) {
      console.error('[SharkAuth] SignIn must be rendered inside <SharkProvider>')
      return
    }
    const { url } = await startAuthFlow(
      redirectUrl ?? (typeof window !== 'undefined' ? window.location.href : '/'),
      authUrl,
      publishableKey,
    )
    if (typeof window !== 'undefined') {
      window.location.href = url
    }
  }, [ctx, redirectUrl])

  if (children) {
    return (
      <span onClick={handleClick} className={className} style={{ cursor: 'pointer' }}>
        {children}
      </span>
    )
  }

  return (
    <button type="button" onClick={handleClick} className={className}>
      Sign in
    </button>
  )
}
