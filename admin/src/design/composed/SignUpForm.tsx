import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { FormField } from '../primitives/FormField'
import { Card } from '../primitives/Card'

export interface SignUpFormProps {
  appName: string
  requirePassword: boolean
  requireName?: boolean
  onSubmit: (data: { email: string; password?: string; name?: string }) => Promise<void>
  signInHref?: string
  termsHref?: string
}

export function SignUpForm({
  appName,
  requirePassword,
  requireName = false,
  onSubmit,
  signInHref,
  termsHref,
}: SignUpFormProps) {
  const [name, setName] = React.useState('')
  const [email, setEmail] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [confirm, setConfirm] = React.useState('')

  const [nameError, setNameError] = React.useState('')
  const [emailError, setEmailError] = React.useState('')
  const [passwordError, setPasswordError] = React.useState('')
  const [confirmError, setConfirmError] = React.useState('')
  const [formError, setFormError] = React.useState('')
  const [loading, setLoading] = React.useState(false)

  function validate(): boolean {
    setNameError('')
    setEmailError('')
    setPasswordError('')
    setConfirmError('')
    setFormError('')

    let firstId: string | null = null

    function mark(id: string) {
      if (!firstId) firstId = id
    }

    if (requireName && !name.trim()) {
      setNameError('Name is required')
      mark('signup-name')
    }

    if (!email.trim()) {
      setEmailError('Email is required')
      mark('signup-email')
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setEmailError('Enter a valid email address')
      mark('signup-email')
    }

    if (requirePassword) {
      if (!password) {
        setPasswordError('Password is required')
        mark('signup-password')
      } else if (password.length < 8) {
        setPasswordError('Password must be at least 8 characters')
        mark('signup-password')
      } else if (password !== confirm) {
        setConfirmError('Passwords do not match')
        mark('signup-confirm')
      }
    }

    if (firstId) {
      document.getElementById(firstId)?.focus()
      return false
    }
    return true
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (loading || !validate()) return

    setLoading(true)
    try {
      await onSubmit({
        email,
        ...(requirePassword ? { password } : {}),
        ...(requireName ? { name } : {}),
      })
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Sign up failed. Please try again.')
      document.getElementById('signup-email')?.focus()
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

  const termsStyle: React.CSSProperties = {
    fontSize: tokens.type.size.xs,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgDim,
    textAlign: 'center',
    marginTop: tokens.space[3],
  }

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <h1 style={titleStyle}>{appName}</h1>
        <p style={subtitleStyle}>Create your account</p>

        <Card bodyStyle={{ padding: tokens.space[6] }}>
          {formError && (
            <div role="alert" style={errorBannerStyle}>{formError}</div>
          )}

          <form onSubmit={handleSubmit} noValidate>
            <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[3] }}>
              {requireName && (
                <FormField
                  label="Full name"
                  required
                  htmlFor="signup-name"
                  errorText={nameError}
                  inputProps={{
                    id: 'signup-name',
                    type: 'text',
                    autoComplete: 'name',
                    value: name,
                    onChange: (e) => setName(e.target.value),
                    disabled: loading,
                  }}
                />
              )}

              <FormField
                label="Email"
                required
                htmlFor="signup-email"
                errorText={emailError}
                inputProps={{
                  id: 'signup-email',
                  type: 'email',
                  autoComplete: 'email',
                  value: email,
                  onChange: (e) => setEmail(e.target.value),
                  disabled: loading,
                }}
              />

              {requirePassword && (
                <>
                  <FormField
                    label="Password"
                    required
                    htmlFor="signup-password"
                    errorText={passwordError}
                    helperText={!passwordError ? 'At least 8 characters' : undefined}
                    inputProps={{
                      id: 'signup-password',
                      type: 'password',
                      autoComplete: 'new-password',
                      value: password,
                      onChange: (e) => setPassword(e.target.value),
                      disabled: loading,
                    }}
                  />

                  <FormField
                    label="Confirm password"
                    required
                    htmlFor="signup-confirm"
                    errorText={confirmError}
                    inputProps={{
                      id: 'signup-confirm',
                      type: 'password',
                      autoComplete: 'new-password',
                      value: confirm,
                      onChange: (e) => setConfirm(e.target.value),
                      disabled: loading,
                    }}
                  />
                </>
              )}

              <Button
                type="submit"
                variant="primary"
                size="lg"
                loading={loading}
                disabled={loading}
                style={{ width: '100%' }}
              >
                Create account
              </Button>
            </div>
          </form>

          {termsHref && (
            <p style={termsStyle}>
              By creating an account you agree to our{' '}
              <a href={termsHref} style={linkStyle} target="_blank" rel="noreferrer">
                Terms of Service
              </a>
            </p>
          )}
        </Card>

        {signInHref && (
          <div style={footerStyle}>
            <a href={signInHref} style={linkStyle}>Already have an account? Sign in</a>
          </div>
        )}
      </div>
    </div>
  )
}

SignUpForm.displayName = 'SignUpForm'
