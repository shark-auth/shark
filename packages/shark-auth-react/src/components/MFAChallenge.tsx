import React from 'react'
import { AuthContext } from '../hooks/context'

export interface MFAChallengeProps {
  onSuccess?: () => void
  onError?: (err: Error) => void
}

const inputStyle: React.CSSProperties = {
  display: 'block',
  width: '100%',
  padding: '8px 12px',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  fontSize: 16,
  letterSpacing: '0.25em',
  textAlign: 'center',
  marginBottom: 12,
}

const buttonStyle: React.CSSProperties = {
  display: 'block',
  width: '100%',
  padding: '10px 16px',
  background: '#6366f1',
  color: '#fff',
  border: 'none',
  borderRadius: 6,
  fontSize: 15,
  fontWeight: 600,
  cursor: 'pointer',
}

export function MFAChallenge({ onSuccess, onError }: MFAChallengeProps) {
  const ctx = React.useContext(AuthContext)
  const [code, setCode] = React.useState('')
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!ctx) {
      setError('Not inside SharkProvider')
      return
    }
    setLoading(true)
    setError(null)
    try {
      const resp = await ctx.client.fetch('/auth/mfa/challenge', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code }),
      })
      if (!resp.ok) {
        const text = await resp.text()
        throw new Error(`MFA challenge failed ${resp.status}: ${text}`)
      }
      setCode('')
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
    <form onSubmit={handleSubmit} style={{ maxWidth: 320 }}>
      <label style={{ display: 'block', marginBottom: 6, fontSize: 14, fontWeight: 500 }}>
        Authenticator code
      </label>
      <input
        type="text"
        inputMode="numeric"
        autoComplete="one-time-code"
        maxLength={8}
        value={code}
        onChange={e => setCode(e.target.value.replace(/\D/g, ''))}
        placeholder="123456"
        required
        style={inputStyle}
      />
      {error && (
        <p style={{ color: '#ef4444', fontSize: 13, marginBottom: 10 }}>{error}</p>
      )}
      <button type="submit" disabled={loading || code.length < 6} style={buttonStyle}>
        {loading ? 'Verifying…' : 'Verify'}
      </button>
    </form>
  )
}
