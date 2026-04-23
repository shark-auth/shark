import React from 'react'
import { useLocation } from 'wouter'
import { PasskeyButton } from '../../design/composed/PasskeyButton'
import { useToast } from '../../design/primitives/Toast'
import type { HostedConfig } from '../../hosted-entry'

interface PasskeyPageProps {
  config: HostedConfig
}

export function PasskeyPage({ config }: PasskeyPageProps) {
  const [, navigate] = useLocation()
  const toast = useToast()
  const base = `/hosted/${config.app.slug}`

  async function onClick(): Promise<void> {
    if (!window.PublicKeyCredential) {
      toast.danger('Passkeys are not supported in this browser')
      return
    }
    try {
      // Begin login
      const beginRes = await fetch('/api/v1/auth/passkey/login/begin', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      })

      if (!beginRes.ok) {
        const body = await beginRes.json().catch(() => ({}))
        throw new Error(body.message || 'Something went wrong — try again')
      }

      const options = await beginRes.json()

      // Decode base64url fields
      const challenge = base64urlDecode(options.publicKey?.challenge ?? options.challenge)
      const allowCredentials = (
        options.publicKey?.allowCredentials ?? options.allowCredentials ?? []
      ).map((c: { type: string; id: string; transports?: string[] }) => ({
        ...c,
        id: base64urlDecode(c.id),
      }))

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

      const data = await finishRes.json().catch(() => ({}))
      if (data.mfaRequired) {
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

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '100vh' }}>
      <div style={{ width: '100%', maxWidth: 400, padding: '0 24px' }}>
        <h1 style={{ textAlign: 'center', marginBottom: 8 }}>{config.app.name}</h1>
        <p style={{ textAlign: 'center', marginBottom: 24, color: '#888' }}>
          Sign in with a passkey — use your device biometrics or security key.
        </p>
        <PasskeyButton onClick={onClick} />
        <div style={{ textAlign: 'center', marginTop: 16 }}>
          <a href={`${base}/login`} style={{ color: 'var(--shark-primary, #7c3aed)', textDecoration: 'none', fontSize: 14 }}>
            Back to sign in
          </a>
        </div>
      </div>
    </div>
  )
}

// --- WebAuthn helpers ---

function base64urlDecode(input: string): ArrayBuffer {
  const padded = input
    .replace(/-/g, '+')
    .replace(/_/g, '/')
    .padEnd(input.length + ((4 - (input.length % 4)) % 4), '=')
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
