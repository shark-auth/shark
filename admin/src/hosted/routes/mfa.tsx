import React, { useState } from 'react'
import { useLocation } from 'wouter'
import { MFAForm } from '../../design/composed/MFAForm'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

type MFAMethod = 'totp' | 'sms' | 'email' | 'webauthn'

const VALID_METHODS: MFAMethod[] = ['totp', 'sms', 'email', 'webauthn']

function parseMFAMethod(raw: string | null): MFAMethod {
  if (raw && (VALID_METHODS as string[]).includes(raw)) return raw as MFAMethod
  return 'totp'
}

interface MFAPageProps {
  config: HostedConfig
}

export function MFAPage({ config }: MFAPageProps) {
  const [, navigate] = useLocation()
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  const [method] = useState<MFAMethod>(() => {
    const params = new URLSearchParams(window.location.search)
    return parseMFAMethod(params.get('method'))
  })

  const [session] = useState<string>(() => {
    const params = new URLSearchParams(window.location.search)
    return params.get('session') ?? ''
  })

  async function onSubmit(code: string): Promise<void> {
    try {
      const body: Record<string, string> = { code }
      if (session) body.session = session

      const res = await fetch('/api/v1/auth/mfa/challenge', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new Error(data.message || 'Something went wrong — try again')
      }

      // MFA passed — redirect to oauth redirect_uri
      const redirectUri = config.oauth.redirect_uri
      const state = config.oauth.state
      if (redirectUri) {
        const url = new URL(redirectUri, window.location.origin)
        if (state) url.searchParams.set('state', state)
        window.location.href = url.toString()
      }
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  async function onResend(): Promise<void> {
    try {
      const res = await fetch('/api/v1/auth/mfa/challenge', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ resend: true, ...(session ? { session } : {}) }),
      })

      if (!res.ok) {
        const data = await res.json().catch(() => ({}))
        throw new Error(data.message || 'Something went wrong — try again')
      }

      toast.success('Code resent!')
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  function onUseBackup(): void {
    navigate(`${base}/mfa?method=totp&backup=1${session ? `&session=${encodeURIComponent(session)}` : ''}`)
  }

  // Only sms/email methods support resend
  const supportsResend = method === 'sms' || method === 'email'

  return (
    <MFAForm
      method={method}
      onSubmit={onSubmit}
      onResend={supportsResend ? onResend : undefined}
      onUseBackup={method !== 'totp' ? undefined : onUseBackup}
    />
  )
}
