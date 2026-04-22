import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { FormField } from '../primitives/FormField'
import { Card } from '../primitives/Card'
import { PasskeyButton } from './PasskeyButton'
import { OAuthProviderButton } from './OAuthProviderButton'
import sharkGlyph from '../../assets/sharky-glyph.png'
import sharkWordmark from '../../assets/sharky-wordmark.png'

export interface SignInFormProps {
  appName: string
  authMethods: ('password' | 'magic_link' | 'passkey' | 'oauth')[]
  oauthProviders?: { id: string; name: string; iconUrl?: string }[]
  onPasswordSubmit?: (email: string, password: string) => Promise<void>
  onMagicLinkRequest?: (email: string) => Promise<void>
  onPasskeyStart?: () => Promise<void>
  onOAuthStart?: (providerID: string) => void
  signUpHref?: string
  forgotPasswordHref?: string
}

export function SignInForm({
  appName,
  authMethods,
  oauthProviders = [],
  onPasswordSubmit,
  onMagicLinkRequest,
  onPasskeyStart,
  onOAuthStart,
  signUpHref,
  forgotPasswordHref,
}: SignInFormProps) {
  const [email, setEmail] = React.useState('')
  const [password, setPassword] = React.useState('')
  const [emailError, setEmailError] = React.useState('')
  const [passwordError, setPasswordError] = React.useState('')
  const [formError, setFormError] = React.useState('')
  const [loading, setLoading] = React.useState(false)
  const [magicLoading, setMagicLoading] = React.useState(false)
  const [passkeyLoading, setPasskeyLoading] = React.useState(false)

  const hasPassword = authMethods.includes('password')
  const hasMagicLink = authMethods.includes('magic_link')
  const hasPasskey = authMethods.includes('passkey')
  const hasOAuth = authMethods.includes('oauth')
  const hasAlt = hasMagicLink || hasPasskey

  function validate(): boolean {
    let ok = true
    setEmailError('')
    setPasswordError('')
    setFormError('')

    if (!email.trim()) {
      setEmailError('Email is required')
      ok = false
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setEmailError('Enter a valid email address')
      ok = false
    }

    if (hasPassword && !password) {
      setPasswordError('Password is required')
      ok = false
    }

    if (!ok) {
      if (!email.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
        document.getElementById('signin-email')?.focus()
      } else {
        document.getElementById('signin-password')?.focus()
      }
    }
    return ok
  }

  async function handlePasswordSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (loading || !validate()) return
    setLoading(true)
    try {
      await onPasswordSubmit?.(email, password)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Sign in failed. Please try again.')
      document.getElementById('signin-email')?.focus()
    } finally {
      setLoading(false)
    }
  }

  async function handleMagicLink() {
    setEmailError('')
    setFormError('')
    if (!email.trim()) {
      setEmailError('Email is required')
      document.getElementById('signin-email')?.focus()
      return
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      setEmailError('Enter a valid email address')
      document.getElementById('signin-email')?.focus()
      return
    }
    setMagicLoading(true)
    try {
      await onMagicLinkRequest?.(email)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Could not send magic link.')
    } finally {
      setMagicLoading(false)
    }
  }

  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: `80px 24px`,
    minHeight: '100vh',
    background: 'var(--bg)',
  }

  const innerStyle: React.CSSProperties = {
    width: '100%',
    maxWidth: 400,
  }

  const titleStyle: React.CSSProperties = {
    fontSize: tokens.type.size.xl,
    fontFamily: 'Hanken Grotesk, var(--font-sans)',
    fontWeight: 600,
    color: tokens.color.fg,
    textAlign: 'center',
    marginBottom: tokens.space[1],
    letterSpacing: '-0.01em'
  }

  const subtitleStyle: React.CSSProperties = {
    fontSize: 13,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgDim,
    textAlign: 'center',
    marginBottom: tokens.space[6],
  }

  const dividerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
    margin: `20px 0`,
  }

  const dividerLineStyle: React.CSSProperties = {
    flex: 1,
    height: 1,
    background: tokens.color.hairline,
  }

  const dividerTextStyle: React.CSSProperties = {
    fontSize: 10,
    fontFamily: 'Azeret Mono, var(--font-mono)',
    color: 'var(--fg-dim)',
    whiteSpace: 'nowrap',
    textTransform: 'uppercase',
    letterSpacing: '0.05em'
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
    color: 'var(--fg-dim)',
    textDecoration: 'none',
    transition: 'color 60ms ease-out'
  }

  const errorBannerStyle: React.CSSProperties = {
    padding: `${tokens.space[2]}px ${tokens.space[3]}px`,
    background: 'var(--bg)',
    border: `1px solid ${tokens.color.danger}`,
    borderRadius: 2,
    color: tokens.color.danger,
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    marginBottom: tokens.space[3],
  }

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <div 
          className="premium-branding"
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 8,
            marginBottom: 24,
            opacity: 0.8,
            filter: 'grayscale(1)',
            cursor: 'default',
            userSelect: 'none'
          }}
        >
          <span style={{ 
            fontSize: 9, 
            color: 'var(--fg-dim)', 
            letterSpacing: '0.08em', 
            fontWeight: 600, 
            textTransform: 'uppercase',
            fontFamily: 'Azeret Mono, var(--font-mono)'
          }}>
            Secured by
          </span>
          <img src={sharkGlyph} alt="Shark Auth" style={{ height: 16, width: 'auto', opacity: 0.8 }} />
          <img src={sharkWordmark} alt="Shark Auth" style={{ height: 12, width: 'auto', opacity: 0.8 }} />
        </div>

        <h1 style={titleStyle}>{appName}</h1>
        <p style={subtitleStyle}>Sign in to your account</p>

        <Card bodyStyle={{ padding: 32, background: 'var(--surface-0)', border: '1px solid var(--hairline)', borderRadius: 2 }}>
          {formError && (
            <div role="alert" style={errorBannerStyle}>{formError}</div>
          )}

          {hasOAuth && oauthProviders.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[2], marginBottom: tokens.space[4] }}>
              {oauthProviders.map((p) => (
                <OAuthProviderButton
                  key={p.id}
                  providerID={p.id}
                  providerName={p.name}
                  iconUrl={p.iconUrl}
                  onClick={() => onOAuthStart?.(p.id)}
                />
              ))}
            </div>
          )}

          {hasOAuth && oauthProviders.length > 0 && (hasPassword || hasAlt) && (
            <div style={dividerStyle}>
              <div style={dividerLineStyle} />
              <span style={dividerTextStyle}>or</span>
              <div style={dividerLineStyle} />
            </div>
          )}

          <form onSubmit={handlePasswordSubmit} noValidate>
            <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[3] }}>
              <FormField
                label="Email"
                required
                htmlFor="signin-email"
                errorText={emailError}
                inputProps={{
                  id: 'signin-email',
                  type: 'email',
                  autoComplete: 'email',
                  value: email,
                  onChange: (e) => setEmail(e.target.value),
                  disabled: loading,
                }}
              />

              {hasPassword && (
                <FormField
                  label="Password"
                  required
                  htmlFor="signin-password"
                  errorText={passwordError}
                  inputProps={{
                    id: 'signin-password',
                    type: 'password',
                    autoComplete: 'current-password',
                    value: password,
                    onChange: (e) => setPassword(e.target.value),
                    disabled: loading,
                  }}
                />
              )}

              {hasPassword && (
                <Button
                  type="submit"
                  variant="primary"
                  size="lg"
                  loading={loading}
                  disabled={loading}
                  style={{ width: '100%', height: 40, fontSize: 13, fontWeight: 600, borderRadius: 2 }}
                >
                  Sign in
                </Button>
              )}
            </div>
          </form>

          {hasAlt && (
            <>
              {hasPassword && (
                <div style={dividerStyle}>
                  <div style={dividerLineStyle} />
                  <span style={dividerTextStyle}>or</span>
                  <div style={dividerLineStyle} />
                </div>
              )}

              <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space[2] }}>
                {hasMagicLink && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="lg"
                    loading={magicLoading}
                    disabled={loading || magicLoading || passkeyLoading}
                    onClick={handleMagicLink}
                    style={{ width: '100%', height: 40, fontSize: 13, borderRadius: 2 }}
                  >
                    Continue with magic link
                  </Button>
                )}

                {hasPasskey && (
                  <PasskeyButton
                    loading={passkeyLoading}
                    disabled={loading || magicLoading || passkeyLoading}
                    onClick={async () => {
                      setPasskeyLoading(true)
                      try {
                        await onPasskeyStart?.()
                      } catch (err) {
                        setFormError(err instanceof Error ? err.message : 'Passkey sign in failed.')
                      } finally {
                        setPasskeyLoading(false)
                      }
                    }}
                  />
                )}
              </div>
            </>
          )}
        </Card>

        {/* Branding moved to top */}

        {(signUpHref || forgotPasswordHref) && (
          <div style={footerStyle}>
            {forgotPasswordHref && (
              <a href={forgotPasswordHref} style={linkStyle}>Forgot password?</a>
            )}
            {signUpHref && (
              <a href={signUpHref} style={linkStyle}>Create an account</a>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

SignInForm.displayName = 'SignInForm'
