import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { Card } from '../primitives/Card'

export interface EmailVerifyProps {
  state: 'pending' | 'success' | 'error'
  errorMessage?: string
  onResend?: () => Promise<void>
  onContinue?: () => void
}

function CheckIcon() {
  return (
    <svg
      width={48}
      height={48}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="10" />
      <path d="M8 12l3 3 5-6" />
    </svg>
  )
}

function ClockIcon() {
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
      <circle cx="12" cy="12" r="10" />
      <path d="M12 6v6l4 2" />
    </svg>
  )
}

function AlertIcon() {
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
      <circle cx="12" cy="12" r="10" />
      <path d="M12 8v4" />
      <path d="M12 16h.01" />
    </svg>
  )
}

const STATE_CONFIG = {
  pending: {
    icon: <ClockIcon />,
    iconColor: 'var(--shark-primary, #7c3aed)',
    title: 'Verify your email',
    description: "We've sent a verification link to your email. Please check your inbox and click the link to verify your account.",
  },
  success: {
    icon: <CheckIcon />,
    iconColor: 'oklch(68% 0.15 160)',
    title: 'Email verified!',
    description: 'Your email address has been successfully verified. You can now continue.',
  },
  error: {
    icon: <AlertIcon />,
    iconColor: 'oklch(62% 0.2 25)',
    title: 'Verification failed',
    description: 'We could not verify your email address.',
  },
}

export function EmailVerify({ state, errorMessage, onResend, onContinue }: EmailVerifyProps) {
  const [resendLoading, setResendLoading] = React.useState(false)
  const [resendError, setResendError] = React.useState('')
  const [resendSuccess, setResendSuccess] = React.useState(false)

  const config = STATE_CONFIG[state]

  async function handleResend() {
    if (resendLoading) return
    setResendError('')
    setResendSuccess(false)
    setResendLoading(true)
    try {
      await onResend?.()
      setResendSuccess(true)
    } catch (err) {
      setResendError(err instanceof Error ? err.message : 'Could not resend verification email.')
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

  const iconWrapStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'center',
    color: config.iconColor,
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

  const descStyle: React.CSSProperties = {
    fontSize: tokens.type.size.base,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgMuted,
    textAlign: 'center',
    lineHeight: 1.6,
    marginBottom: tokens.space[6],
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

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <Card bodyStyle={{ padding: tokens.space[8] }}>
          <div style={iconWrapStyle} aria-hidden="true">
            {config.icon}
          </div>

          <h1 style={titleStyle}>{config.title}</h1>

          <p style={descStyle}>
            {state === 'error' && errorMessage ? errorMessage : config.description}
          </p>

          <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[2] }}>
            {state === 'success' && onContinue && (
              <Button
                type="button"
                variant="primary"
                size="lg"
                onClick={onContinue}
                style={{ width: '100%' }}
              >
                Continue
              </Button>
            )}

            {(state === 'pending' || state === 'error') && onResend && (
              <Button
                type="button"
                variant={state === 'error' ? 'primary' : 'ghost'}
                size="lg"
                loading={resendLoading}
                disabled={resendLoading}
                onClick={handleResend}
                style={{ width: '100%' }}
              >
                Resend verification email
              </Button>
            )}
          </div>

          {resendError && (
            <div role="alert" style={errorStyle}>{resendError}</div>
          )}

          {resendSuccess && !resendError && (
            <div role="status" style={successStyle}>Verification email sent!</div>
          )}
        </Card>
      </div>
    </div>
  )
}

EmailVerify.displayName = 'EmailVerify'
