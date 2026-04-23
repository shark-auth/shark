import React from 'react'
import { MagicLinkSent } from '../../design/composed/MagicLinkSent'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface MagicPageProps {
  config: HostedConfig
}

const INITIAL_COOLDOWN_SECONDS = 30

export function MagicPage(_props: MagicPageProps) {
  const toast = useToast()

  // Read email from query string once on mount
  const [email] = React.useState<string>(() => {
    const params = new URLSearchParams(window.location.search)
    return decodeURIComponent(params.get('email') ?? '')
  })

  async function onResend(): Promise<void> {
    if (!email) return
    try {
      const res = await fetch('/api/v1/auth/magic-link/send', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  return (
    <MagicLinkSent
      email={email}
      onResend={onResend}
      resendCooldownSeconds={INITIAL_COOLDOWN_SECONDS}
    />
  )
}
