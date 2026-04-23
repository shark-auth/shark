import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { Card } from '../primitives/Card'

export interface MFAFormProps {
  method: 'totp' | 'sms' | 'email' | 'webauthn'
  onSubmit: (code: string) => Promise<void>
  onResend?: () => Promise<void>
  onUseBackup?: () => void
}

const METHOD_LABELS: Record<MFAFormProps['method'], string> = {
  totp: 'Enter the 6-digit code from your authenticator app.',
  sms: 'Enter the 6-digit code sent to your phone.',
  email: 'Enter the 6-digit code sent to your email.',
  webauthn: 'Follow the prompt on your security key or device.',
}

const CODE_LENGTH = 6

export function MFAForm({ method, onSubmit, onResend, onUseBackup }: MFAFormProps) {
  const [digits, setDigits] = React.useState<string[]>(Array(CODE_LENGTH).fill(''))
  const [error, setError] = React.useState('')
  const [loading, setLoading] = React.useState(false)
  const [resendLoading, setResendLoading] = React.useState(false)
  const inputRefs = React.useRef<(HTMLInputElement | null)[]>([])

  const code = digits.join('')
  const isWebAuthn = method === 'webauthn'

  function handleDigitChange(index: number, value: string) {
    const digit = value.replace(/\D/g, '').slice(-1)
    const next = [...digits]
    next[index] = digit
    setDigits(next)
    setError('')

    if (digit && index < CODE_LENGTH - 1) {
      inputRefs.current[index + 1]?.focus()
    }
  }

  function handleDigitKeyDown(index: number, e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Backspace' && !digits[index] && index > 0) {
      inputRefs.current[index - 1]?.focus()
    }
  }

  function handlePaste(e: React.ClipboardEvent) {
    e.preventDefault()
    const pasted = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, CODE_LENGTH)
    if (!pasted) return
    const next = Array(CODE_LENGTH).fill('')
    for (let i = 0; i < pasted.length; i++) next[i] = pasted[i]
    setDigits(next)
    const focusIdx = Math.min(pasted.length, CODE_LENGTH - 1)
    inputRefs.current[focusIdx]?.focus()
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (loading) return
    if (code.length < CODE_LENGTH) {
      setError('Please enter all 6 digits.')
      inputRefs.current[code.length]?.focus()
      return
    }
    setLoading(true)
    setError('')
    try {
      await onSubmit(code)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid code. Please try again.')
      setDigits(Array(CODE_LENGTH).fill(''))
      inputRefs.current[0]?.focus()
    } finally {
      setLoading(false)
    }
  }

  async function handleResend() {
    if (resendLoading) return
    setResendLoading(true)
    try {
      await onResend?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not resend code.')
    } finally {
      setResendLoading(false)
    }
  }

  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: `${tokens.space[8]}px ${tokens.space[6]}px`,
    minHeight: '100vh',
    background: tokens.color.surface0,
  }

  const innerStyle: React.CSSProperties = {
    width: '100%',
    maxWidth: 400,
  }

  const titleStyle: React.CSSProperties = {
    fontSize: tokens.type.size.xl,
    fontFamily: tokens.type.display.family,
    fontWeight: tokens.type.weight.bold,
    color: tokens.color.fg,
    textAlign: 'center',
    marginBottom: tokens.space[2],
  }

  const descStyle: React.CSSProperties = {
    fontSize: tokens.type.size.base,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgMuted,
    textAlign: 'center',
    marginBottom: tokens.space[6],
    lineHeight: 1.5,
  }

  const boxesStyle: React.CSSProperties = {
    display: 'flex',
    gap: tokens.space[2],
    justifyContent: 'center',
    marginBottom: tokens.space[4],
  }

  const digitInputStyle: React.CSSProperties = {
    width: 44,
    height: 52,
    textAlign: 'center',
    fontSize: tokens.type.size.lg,
    fontFamily: tokens.type.mono.family,
    fontWeight: tokens.type.weight.semibold,
    color: tokens.color.fg,
    background: tokens.color.surface2,
    border: `1px solid ${error ? tokens.color.danger : tokens.color.hairline}`,
    borderRadius: tokens.radius.md,
    outline: 'none',
    caretColor: tokens.color.primary,
    transition: `border-color ${tokens.motion.fast}`,
  }

  const errorStyle: React.CSSProperties = {
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    color: tokens.color.danger,
    textAlign: 'center',
    marginBottom: tokens.space[3],
  }

  const actionsStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.space[2],
    marginTop: tokens.space[4],
  }

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <h1 style={titleStyle}>Two-factor authentication</h1>
        <p style={descStyle}>{METHOD_LABELS[method]}</p>

        <Card bodyStyle={{ padding: tokens.space[6] }}>
          <form onSubmit={handleSubmit} noValidate>
            {!isWebAuthn && (
              <>
                <div style={boxesStyle} onPaste={handlePaste}>
                  {digits.map((d, i) => (
                    <input
                      key={i}
                      ref={(el) => { inputRefs.current[i] = el }}
                      type="text"
                      inputMode="numeric"
                      autoComplete={i === 0 ? 'one-time-code' : 'off'}
                      maxLength={1}
                      value={d}
                      aria-label={`Digit ${i + 1} of ${CODE_LENGTH}`}
                      disabled={loading}
                      style={digitInputStyle}
                      onChange={(e) => handleDigitChange(i, e.target.value)}
                      onKeyDown={(e) => handleDigitKeyDown(i, e)}
                      onFocus={(e) => {
                        e.target.style.borderColor = tokens.color.primary
                        e.target.style.boxShadow = `0 0 0 2px ${tokens.color.focusRing}33`
                      }}
                      onBlur={(e) => {
                        e.target.style.borderColor = error ? tokens.color.danger : tokens.color.hairline
                        e.target.style.boxShadow = ''
                      }}
                    />
                  ))}
                </div>

                {error && (
                  <div role="alert" style={errorStyle}>{error}</div>
                )}

                <Button
                  type="submit"
                  variant="primary"
                  size="lg"
                  loading={loading}
                  disabled={loading || code.length < CODE_LENGTH}
                  style={{ width: '100%' }}
                >
                  Verify
                </Button>
              </>
            )}

            {isWebAuthn && (
              <>
                {error && (
                  <div role="alert" style={{ ...errorStyle, marginBottom: tokens.space[3] }}>{error}</div>
                )}
                <Button
                  type="submit"
                  variant="primary"
                  size="lg"
                  loading={loading}
                  disabled={loading}
                  style={{ width: '100%' }}
                >
                  Use security key
                </Button>
              </>
            )}

            <div style={actionsStyle}>
              {onResend && !isWebAuthn && (
                <Button
                  type="button"
                  variant="ghost"
                  size="md"
                  loading={resendLoading}
                  disabled={loading || resendLoading}
                  onClick={handleResend}
                  style={{ width: '100%' }}
                >
                  Resend code
                </Button>
              )}

              {onUseBackup && (
                <Button
                  type="button"
                  variant="ghost"
                  size="md"
                  disabled={loading}
                  onClick={onUseBackup}
                  style={{ width: '100%' }}
                >
                  Use backup code
                </Button>
              )}
            </div>
          </form>
        </Card>
      </div>
    </div>
  )
}

MFAForm.displayName = 'MFAForm'
