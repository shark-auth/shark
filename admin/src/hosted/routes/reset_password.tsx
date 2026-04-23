import React, { useState } from 'react'
import { ResetPasswordForm } from '../../design/composed/ResetPasswordForm'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface ResetPasswordPageProps {
  config: HostedConfig
}

export function ResetPasswordPage({ config }: ResetPasswordPageProps) {
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  const [token] = useState<string | null>(() => {
    const params = new URLSearchParams(window.location.search)
    return params.get('token')
  })

  async function onSubmit(password: string): Promise<void> {
    if (!token) {
      toast.danger('Reset token is missing — check your email link.')
      return
    }

    try {
      const res = await fetch('/api/v1/auth/password/reset', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, password }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      toast.success('Password reset successfully!')
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  if (!token) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', padding: 20 }}>
        <p style={{ color: 'var(--color-danger)', textAlign: 'center' }}>
          Invalid reset link. Please request a new one from the login page.
        </p>
        <a href={`${base}/login`} style={{ color: 'var(--color-primary)', marginTop: 12 }}>Back to login</a>
      </div>
    )
  }

  return (
    <ResetPasswordForm
      appName={config.app.name}
      onSubmit={onSubmit}
      signInHref={`${base}/login`}
    />
  )
}
