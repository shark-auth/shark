import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { Card } from '../primitives/Card'

export interface MagicLinkSentProps {
  email: string
  onResend?: () => Promise<void>
  resendCooldownSeconds?: number
}

function MailIcon() {
  return (
    <svg
      width={48}
      height={48}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <path d="M2 7l10 7 10-7" />
    </svg>
  )
}

export function MagicLinkSent({
  email,
  onResend,
  resendCooldownSeconds = 60,
}: MagicLinkSentProps) {
  const [cooldown, setCooldown] = React.useState(onResend ? (resendCooldownSeconds > 0 ? resendCooldownSeconds : 0) : 0)
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState('')
  const [success, setSuccess] = React.useState(false)
  const timerRef = React.useRef<ReturnType<typeof setInterval> | null>(null)

  function startCooldown(seconds: number = resendCooldownSeconds) {
    if (seconds <= 0) return
    setCooldown(seconds)
    if (timerRef.current) clearInterval(timerRef.current)
    timerRef.current = setInterval(() => {
      setCooldown((prev) => {
        if (prev <= 1) {
          clearInterval(timerRef.current!)
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }

  React.useEffect(() => {
    if (onResend && resendCooldownSeconds > 0) {
      startCooldown(resendCooldownSeconds)
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  async function handleResend() {
    if (loading || cooldown > 0) return
    setError('')
    setSuccess(false)
    setLoading(true)
    try {
      await onResend?.()
      setSuccess(true)
      startCooldown()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not resend link. Please try again.')
    } finally {
      setLoading(false)
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

  const iconWrapStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'center',
    color: tokens.color.primary,
    marginBottom: tokens.space[6],
  }

  const titleStyle: React.CSSProperties = {
    fontSize: tokens.type.size.xl,
    fontFamily: tokens.type.display.family,
    fontWeight: tokens.type.weight.bold,
    color: tokens.color.fg,
    textAlign: 'center',
    marginBottom: tokens.space[3],
  }

  const bodyStyle: React.CSSProperties = {
    fontSize: tokens.type.size.base,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgMuted,
    textAlign: 'center',
    lineHeight: 1.6,
    marginBottom: tokens.space[6],
  }

  const emailEmphasisStyle: React.CSSProperties = {
    color: tokens.color.fg,
    fontWeight: tokens.type.weight.semibold,
  }

  const errorStyle: React.CSSProperties = {
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    color: tokens.color.danger,
    textAlign: 'center',
    marginTop: tokens.space[3],
  }

  const successStyle: React.CSSProperties = {
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    color: tokens.color.success,
    textAlign: 'center',
    marginTop: tokens.space[3],
  }

  const cooldownLabel = cooldown > 0 ? `Resend in ${cooldown}s` : 'Resend email'

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <Card bodyStyle={{ padding: tokens.space[8] }}>
          <div style={iconWrapStyle}>
            <MailIcon />
          </div>

          <h1 style={titleStyle}>Check your email</h1>

          <p style={bodyStyle}>
            We sent a magic link to{' '}
            <span style={emailEmphasisStyle}>{email}</span>.
            Click the link in the email to sign in.
          </p>

          {onResend && (
            <Button
              type="button"
              variant="ghost"
              size="md"
              loading={loading}
              disabled={loading || cooldown > 0}
              onClick={handleResend}
              style={{ width: '100%' }}
            >
              {cooldownLabel}
            </Button>
          )}

          {error && (
            <div role="alert" style={errorStyle}>{error}</div>
          )}

          {success && !error && (
            <div role="status" style={successStyle}>Email sent!</div>
          )}
        </Card>
      </div>
    </div>
  )
}

MagicLinkSent.displayName = 'MagicLinkSent'
