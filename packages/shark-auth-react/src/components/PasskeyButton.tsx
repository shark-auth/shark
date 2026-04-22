import React from 'react'
import { AuthContext } from '../hooks/context'

export interface PasskeyButtonProps {
  onSuccess?: () => void
  onError?: (err: Error) => void
  children?: React.ReactNode
  className?: string
}

export function PasskeyButton({ onSuccess, onError, children, className }: PasskeyButtonProps) {
  const ctx = React.useContext(AuthContext)
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)

  const handleClick = async () => {
    if (!ctx) {
      setError('Not inside SharkProvider')
      return
    }
    setLoading(true)
    setError(null)

    try {
      // Step 1: get challenge from server
      const beginResp = await ctx.client.fetch('/auth/passkey/login/begin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      })
      if (!beginResp.ok) {
        throw new Error(`Passkey begin failed ${beginResp.status}`)
      }
      // Server sends a JSON object with base64url-encoded challenge
      interface PasskeyBeginOptions {
        challenge: string
        allowCredentials?: Array<{ id: string; type: string }>
        userVerification?: UserVerificationRequirement
        timeout?: number
      }
      const options = await beginResp.json() as PasskeyBeginOptions

      function fromBase64url(s: string): ArrayBuffer {
        const padded = s.replace(/-/g, '+').replace(/_/g, '/') + '=='.slice(0, (4 - s.length % 4) % 4)
        const bytes = Uint8Array.from(atob(padded), c => c.charCodeAt(0))
        return bytes.buffer as ArrayBuffer
      }

      const challengeBytes = fromBase64url(options.challenge)
      const allowCredentials: PublicKeyCredentialDescriptor[] | undefined =
        options.allowCredentials?.map(cred => ({
          id: fromBase64url(cred.id),
          type: cred.type as PublicKeyCredentialType,
        }))

      // Step 2: invoke WebAuthn
      const assertion = await navigator.credentials.get({
        publicKey: {
          challenge: challengeBytes,
          allowCredentials,
          userVerification: 'preferred',
        },
      }) as PublicKeyCredential | null

      if (!assertion) throw new Error('No credential returned from authenticator')

      const response = assertion.response as AuthenticatorAssertionResponse
      function toBase64url(buf: ArrayBuffer): string {
        return btoa(String.fromCharCode(...new Uint8Array(buf)))
          .replace(/\+/g, '-')
          .replace(/\//g, '_')
          .replace(/=/g, '')
      }

      // Step 3: send assertion to server
      const finishResp = await ctx.client.fetch('/auth/passkey/login/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: assertion.id,
          rawId: toBase64url(assertion.rawId),
          type: assertion.type,
          response: {
            authenticatorData: toBase64url(response.authenticatorData),
            clientDataJSON: toBase64url(response.clientDataJSON),
            signature: toBase64url(response.signature),
            userHandle: response.userHandle ? toBase64url(response.userHandle) : null,
          },
        }),
      })

      if (!finishResp.ok) {
        const text = await finishResp.text()
        throw new Error(`Passkey finish failed ${finishResp.status}: ${text}`)
      }

      onSuccess?.()
    } catch (err) {
      const e = err instanceof Error ? err : new Error(String(err))
      setError(e.message)
      onError?.(e)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <button
        type="button"
        onClick={handleClick}
        disabled={loading}
        className={className}
        style={{
          padding: '10px 20px',
          background: '#111827',
          color: '#fff',
          border: 'none',
          borderRadius: 6,
          fontSize: 15,
          fontWeight: 600,
          cursor: loading ? 'not-allowed' : 'pointer',
          opacity: loading ? 0.7 : 1,
        }}
      >
        {loading ? 'Authenticating…' : (children ?? 'Sign in with passkey')}
      </button>
      {error && (
        <p style={{ color: '#ef4444', fontSize: 13, marginTop: 8 }}>{error}</p>
      )}
    </div>
  )
}
