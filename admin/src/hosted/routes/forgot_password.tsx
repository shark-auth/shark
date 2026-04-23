import React from 'react'
import { ForgotPasswordForm } from '../../design/composed/ForgotPasswordForm'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface ForgotPasswordPageProps {
  config: HostedConfig
}

export function ForgotPasswordPage({ config }: ForgotPasswordPageProps) {
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  async function onSubmit(email: string): Promise<void> {
    try {
      const res = await fetch('/api/v1/auth/password/send-reset-link', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      toast.success('Reset link sent!')
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  return (
    <ForgotPasswordForm
      appName={config.app.name}
      onSubmit={onSubmit}
      signInHref={`${base}/login`}
    />
  )
}
