import React from 'react'
import { startAuthFlow } from '../core/auth'
import { AuthContext } from '../hooks/context'

export interface SignUpProps {
  redirectUrl?: string
  children?: React.ReactNode
  className?: string
}

export function SignUp({ redirectUrl, children, className }: SignUpProps) {
  const ctx = React.useContext(AuthContext)

  const handleClick = React.useCallback(async () => {
    const authUrl = ctx?.authUrl
    const publishableKey = ctx?.publishableKey
    if (!authUrl || !publishableKey) {
      console.error('[SharkAuth] SignUp must be rendered inside <SharkProvider>')
      return
    }
    // startAuthFlow builds the /oauth/authorize URL; we then append screen_hint=signup
    const { url } = await startAuthFlow(
      redirectUrl ?? (typeof window !== 'undefined' ? window.location.href : '/'),
      authUrl,
      publishableKey,
    )
    const finalUrl = `${url}&screen_hint=signup`
    if (typeof window !== 'undefined') {
      window.location.href = finalUrl
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
      Sign up
    </button>
  )
}
