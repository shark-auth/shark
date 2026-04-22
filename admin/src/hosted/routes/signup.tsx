import React from 'react'
import { useLocation } from 'wouter'
import { SignUpForm } from '../../design/composed/SignUpForm'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface SignupPageProps {
  config: HostedConfig
}

export function SignupPage({ config }: SignupPageProps) {
  const [, navigate] = useLocation()
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  async function onSubmit(data: { email: string; password?: string; name?: string }): Promise<void> {
    try {
      const res = await fetch('/api/v1/auth/signup', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: data.email,
          ...(data.password ? { password: data.password } : {}),
          ...(data.name ? { name: data.name } : {}),
        }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      const result = await res.json().catch(() => ({}))

      // If email not verified, prompt verification
      if (result.emailVerified === false) {
        navigate(`${base}/verify`)
        return
      }

      // Otherwise redirect to OAuth redirect_uri
      const redirectUri = config.oauth.redirect_uri
      const state = config.oauth.state
      if (redirectUri) {
        const url = new URL(redirectUri, window.location.origin)
        if (state) url.searchParams.set('state', state)
        window.location.href = url.toString()
      } else {
        navigate(`${base}/login`)
      }
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  const hasPassword = config.authMethods.includes('password')

  return (
    <SignUpForm
      appName={config.app.name}
      requirePassword={hasPassword}
      requireName={false}
      onSubmit={onSubmit}
      signInHref={`${base}/login`}
    />
  )
}
