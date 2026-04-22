import React from 'react'
import { useLocation } from 'wouter'
import { SignInForm } from '../../design/composed/SignInForm'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface LoginPageProps {
  config: HostedConfig
}

export function LoginPage({ config }: LoginPageProps) {
  const [, navigate] = useLocation()
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  async function onPasswordSubmit(email: string, password: string): Promise<void> {
    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      const data = await res.json().catch(() => ({}))

      if (data.mfaRequired) {
        navigate("/mfa?method=totp")
        return
      }

      // Redirect to OAuth redirect_uri with state
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

  async function onMagicLinkRequest(email: string): Promise<void> {
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

      navigate(`/magic?email=${encodeURIComponent(email)}`)
    } catch (err) {
      toast.danger(err instanceof Error ? err.message : 'Something went wrong — try again')
      throw err
    }
  }

  async function onPasskeyStart(): Promise<void> {
    if (!window.PublicKeyCredential) {
      navigate("/passkey")
      return
    }
    try {
      const res = await fetch('/api/v1/auth/passkey/login/begin', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      const options = await res.json()

      // Decode base64url fields required by WebAuthn
      const challenge = base64urlDecode(options.publicKey?.challenge ?? options.challenge)
      const allowCredentials = (options.publicKey?.allowCredentials ?? options.allowCredentials ?? []).map(
        (c: { type: string; id: string; transports?: string[] }) => ({
          ...c,
          id: base64urlDecode(c.id),
        })
      )

      const publicKeyOptions: PublicKeyCredentialRequestOptions = {
        ...(options.publicKey ?? options),
        challenge,
        allowCredentials,
      }

      const assertion = await navigator.credentials.get({ publicKey: publicKeyOptions })
      if (!assertion) throw new Error('Passkey authentication cancelled')

      const finishRes = await fetch('/api/v1/auth/passkey/login/finish', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(serializeAssertion(assertion as PublicKeyCredential)),
      })

      if (!finishRes.ok) {
        const body = await finishRes.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      const finishData = await finishRes.json().catch(() => ({}))
      if (finishData.mfaRequired) {
        navigate("/mfa?method=totp")
        return
      }

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

  function onOAuthStart(providerID: string): void {
    const params = new URLSearchParams({
      client_id: config.oauth.client_id,
      state: config.oauth.state,
      redirect_uri: config.oauth.redirect_uri,
    })
    if (config.oauth.scope) params.set('scope', config.oauth.scope)
    window.location.href = `/api/v1/auth/oauth/${providerID}?${params.toString()}`
  }

  return (
    <SignInForm
      appName={config.app.name}
      authMethods={config.authMethods}
      oauthProviders={config.oauthProviders}
      onPasswordSubmit={onPasswordSubmit}
      onMagicLinkRequest={onMagicLinkRequest}
      onPasskeyStart={onPasskeyStart}
      onOAuthStart={onOAuthStart}
      signUpHref={`${base}/signup`}
      forgotPasswordHref={`${base}/forgot-password`}
    />
  )
}

// --- WebAuthn helpers ---

function base64urlDecode(input: string): ArrayBuffer {
  const padded = input.replace(/-/g, '+').replace(/_/g, '/').padEnd(
    input.length + ((4 - (input.length % 4)) % 4),
    '='
  )
  const binary = atob(padded)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes.buffer
}

function arrayBufferToBase64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i])
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

function serializeAssertion(cred: PublicKeyCredential): Record<string, unknown> {
  const response = cred.response as AuthenticatorAssertionResponse
  return {
    id: cred.id,
    rawId: arrayBufferToBase64url(cred.rawId),
    type: cred.type,
    response: {
      clientDataJSON: arrayBufferToBase64url(response.clientDataJSON),
      authenticatorData: arrayBufferToBase64url(response.authenticatorData),
      signature: arrayBufferToBase64url(response.signature),
      userHandle: response.userHandle ? arrayBufferToBase64url(response.userHandle) : null,
    },
  }
}
