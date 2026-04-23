import React, { useEffect } from 'react'
import { Router, Route, Switch } from 'wouter'
import { ToastProvider } from '../design/primitives/Toast'
import type { HostedConfig } from '../hosted-entry'
import { LoginPage } from './routes/login'
import { SignupPage } from './routes/signup'
import { MagicPage } from './routes/magic'
import { PasskeyPage } from './routes/passkey'
import { MFAPage } from './routes/mfa'
import { VerifyPage } from './routes/verify'
import { ForgotPasswordPage } from './routes/forgot_password'
import { ResetPasswordPage } from './routes/reset_password'
import { ErrorPage } from './routes/error'

interface HostedAppProps {
  config: HostedConfig
}

export function HostedApp({ config }: HostedAppProps) {
  useEffect(() => {
    const root = document.documentElement
    const { branding } = config
    if (branding.primary_color) root.style.setProperty('--color-primary', branding.primary_color)
    if (branding.secondary_color) root.style.setProperty('--color-secondary', branding.secondary_color)
    if (branding.font_family) root.style.setProperty('--font-family', branding.font_family)
  }, [config])

  const base = `/hosted/${config.app.slug}`

  return (
    <ToastProvider>
      <Router base={base}>
        <Switch>
          <Route path="/login" component={() => <LoginPage config={config} />} />
          <Route path="/signup" component={() => <SignupPage config={config} />} />
          <Route path="/magic" component={() => <MagicPage config={config} />} />
          <Route path="/passkey" component={() => <PasskeyPage config={config} />} />
          <Route path="/mfa" component={() => <MFAPage config={config} />} />
          <Route path="/verify" component={() => <VerifyPage config={config} />} />
          <Route path="/forgot-password" component={() => <ForgotPasswordPage config={config} />} />
          <Route path="/reset-password" component={() => <ResetPasswordPage config={config} />} />
          <Route path="/error" component={() => <ErrorPage config={config} />} />
          <Route component={() => <ErrorPage code={404} message="Page not found" config={config} />} />
        </Switch>
      </Router>
    </ToastProvider>
  )
}
