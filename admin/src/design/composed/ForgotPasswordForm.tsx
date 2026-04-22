import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { FormField } from '../primitives/FormField'
import { Card } from '../primitives/Card'

export interface ForgotPasswordFormProps {
  appName: string
  onSubmit: (email: string) => Promise<void>
  signInHref?: string
}

export function ForgotPasswordForm({
  appName,
  onSubmit,
  signInHref,
}: ForgotPasswordFormProps) {
  const [email, setEmail] = React.useState('')
  const [emailError, setEmailError] = React.useState('')
  const [formError, setFormError] = React.useState('')
  const [loading, setLoading] = React.useState(false)
  const [success, setSuccess] = React.useState(false)

  function validate(): boolean {
    setEmailError('')
    setFormError('')

    if (!email.trim()) {
      setEmailError('Email is required')
      return false
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setEmailError('Enter a valid email address')
      return false
    }
    return true
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (loading || !validate()) return

    setLoading(true)
    try {
      await onSubmit(email)
      setSuccess(true)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Could not send reset link. Please try again.')
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

  const titleStyle: React.CSSProperties = {
    fontSize: tokens.type.size['2xl'],
    fontFamily: tokens.type.display.family,
    fontWeight: tokens.type.weight.bold,
    color: tokens.color.fg,
    textAlign: 'center',
    marginBottom: tokens.space[1],
  }

  const subtitleStyle: React.CSSProperties = {
    fontSize: tokens.type.size.base,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgMuted,
    textAlign: 'center',
    marginBottom: tokens.space[6],
  }

  const footerStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'center',
    gap: tokens.space[6],
    marginTop: tokens.space[6],
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
  }

  const linkStyle: React.CSSProperties = {
    color: tokens.color.primary,
    textDecoration: 'none',
  }

  const errorBannerStyle: React.CSSProperties = {
    padding: `${tokens.space[2]}px ${tokens.space[3]}px`,
    background: `${tokens.color.danger}22`,
    border: `1px solid ${tokens.color.danger}44`,
    borderRadius: tokens.radius.md,
    color: tokens.color.danger,
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    marginBottom: tokens.space[3],
  }

  const successBannerStyle: React.CSSProperties = {
    padding: `${tokens.space[4]}px ${tokens.space[3]}px`,
    background: `${tokens.color.success}22`,
    border: `1px solid ${tokens.color.success}44`,
    borderRadius: tokens.radius.md,
    color: tokens.color.success,
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    textAlign: 'center',
  }

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <h1 style={titleStyle}>{appName}</h1>
        <p style={subtitleStyle}>Reset your password</p>

        <Card bodyStyle={{ padding: tokens.space[6] }}>
          {success ? (
            <div style={successBannerStyle}>
              <p style={{ marginBottom: 0 }}>
                If an account exists for <strong>{email}</strong>, you will receive a password reset link shortly.
              </p>
            </div>
          ) : (
            <>
              {formError && (
                <div role="alert" style={errorBannerStyle}>{formError}</div>
              )}

              <p style={{ ...subtitleStyle, textAlign: 'left', fontSize: 13, marginBottom: tokens.space[4] }}>
                Enter your email address and we'll send you a link to reset your password.
              </p>

              <form onSubmit={handleSubmit} noValidate>
                <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[3] }}>
                  <FormField
                    label="Email"
                    required
                    htmlFor="forgot-email"
                    errorText={emailError}
                    inputProps={{
                      id: 'forgot-email',
                      type: 'email',
                      autoComplete: 'email',
                      value: email,
                      onChange: (e) => setEmail(e.target.value),
                      disabled: loading,
                    }}
                  />

                  <Button
                    type="submit"
                    variant="primary"
                    size="lg"
                    loading={loading}
                    disabled={loading}
                    style={{ width: '100%' }}
                  >
                    Send reset link
                  </Button>
                </div>
              </form>
            </>
          )}
        </Card>

        {signInHref && (
          <div style={footerStyle}>
            <a href={signInHref} style={linkStyle}>Back to sign in</a>
          </div>
        )}
      </div>
    </div>
  )
}

ForgotPasswordForm.displayName = 'ForgotPasswordForm'
