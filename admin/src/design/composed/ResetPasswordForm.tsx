import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { FormField } from '../primitives/FormField'
import { Card } from '../primitives/Card'

export interface ResetPasswordFormProps {
  appName: string
  onSubmit: (password: string) => Promise<void>
  signInHref?: string
}

export function ResetPasswordForm({
  appName,
  onSubmit,
  signInHref,
}: ResetPasswordFormProps) {
  const [password, setPassword] = React.useState('')
  const [confirm, setConfirm] = React.useState('')
  const [passwordError, setPasswordError] = React.useState('')
  const [confirmError, setConfirmError] = React.useState('')
  const [formError, setFormError] = React.useState('')
  const [loading, setLoading] = React.useState(false)
  const [success, setSuccess] = React.useState(false)

  function validate(): boolean {
    setPasswordError('')
    setConfirmError('')
    setFormError('')

    let ok = true

    if (!password) {
      setPasswordError('Password is required')
      ok = false
    } else if (password.length < 8) {
      setPasswordError('Password must be at least 8 characters')
      ok = false
    }

    if (password !== confirm) {
      setConfirmError('Passwords do not match')
      ok = false
    }

    return ok
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (loading || !validate()) return

    setLoading(true)
    try {
      await onSubmit(password)
      setSuccess(true)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Could not reset password. Please try again.')
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
        <p style={subtitleStyle}>Set new password</p>

        <Card bodyStyle={{ padding: tokens.space[6] }}>
          {success ? (
            <div style={successBannerStyle}>
              <p style={{ marginBottom: tokens.space[4] }}>
                Your password has been reset successfully.
              </p>
              {signInHref && (
                <Button
                  type="button"
                  variant="primary"
                  size="md"
                  onClick={() => window.location.href = signInHref!}
                  style={{ width: '100%' }}
                >
                  Sign in
                </Button>
              )}
            </div>
          ) : (
            <>
              {formError && (
                <div role="alert" style={errorBannerStyle}>{formError}</div>
              )}

              <form onSubmit={handleSubmit} noValidate>
                <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[3] }}>
                  <FormField
                    label="New password"
                    required
                    htmlFor="reset-password"
                    errorText={passwordError}
                    helperText={!passwordError ? 'At least 8 characters' : undefined}
                    inputProps={{
                      id: 'reset-password',
                      type: 'password',
                      autoComplete: 'new-password',
                      value: password,
                      onChange: (e) => setPassword(e.target.value),
                      disabled: loading,
                    }}
                  />

                  <FormField
                    label="Confirm new password"
                    required
                    htmlFor="reset-confirm"
                    errorText={confirmError}
                    inputProps={{
                      id: 'reset-confirm',
                      type: 'password',
                      autoComplete: 'new-password',
                      value: confirm,
                      onChange: (e) => setConfirm(e.target.value),
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
                    Reset password
                  </Button>
                </div>
              </form>
            </>
          )}
        </Card>

        {signInHref && !success && (
          <div style={footerStyle}>
            <a href={signInHref} style={linkStyle}>Back to sign in</a>
          </div>
        )}
      </div>
    </div>
  )
}

ResetPasswordForm.displayName = 'ResetPasswordForm'
