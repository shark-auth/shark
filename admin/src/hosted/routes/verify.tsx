import React, { useEffect, useState } from 'react'
import { EmailVerify } from '../../design/composed/EmailVerify'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface VerifyPageProps {
  config: HostedConfig
}

type VerifyState = 'pending' | 'success' | 'error'

export function VerifyPage({ config }: VerifyPageProps) {
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  const [token] = useState<string | null>(() => {
    const params = new URLSearchParams(window.location.search)
    return params.get('token')
  })

  const [state, setState] = useState<VerifyState>('pending')
  const [errorMessage, setErrorMessage] = useState<string>('')

  // Run verification once on mount if token is present
  useEffect(() => {
    if (!token) return

    let cancelled = false

    async function verify() {
      try {
        const res = await fetch(`/api/v1/auth/email/verify?token=${encodeURIComponent(token!)}`, {
          credentials: 'include',
        })

        if (cancelled) return

        if (!res.ok) {
          const body = await res.json().catch(() => ({}))
          setErrorMessage(body.message || 'Verification failed — the link may have expired.')
          setState('error')
          return
        }

        setState('success')
      } catch {
        if (!cancelled) {
          setErrorMessage('Something went wrong — try again')
          setState('error')
        }
      }
    }

    verify()

    return () => {
      cancelled = true
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  async function onResend(): Promise<void> {
    try {
      const res = await fetch('/api/v1/auth/email/verify/send', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      toast.success('Verification email sent!')
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  function onContinue(): void {
    const redirectUri = config.oauth.redirect_uri
    const oauthState = config.oauth.state
    if (redirectUri) {
      const url = new URL(redirectUri, window.location.origin)
      if (oauthState) url.searchParams.set('state', oauthState)
      window.location.href = url.toString()
    } else {
      window.location.href = `${base}/login`
    }
  }

  return (
    <EmailVerify
      state={state}
      errorMessage={errorMessage || undefined}
      onResend={state !== 'success' ? onResend : undefined}
      onContinue={state === 'success' ? onContinue : undefined}
    />
  )
}
