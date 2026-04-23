import React, { useState } from 'react'
import { ErrorPage as ErrorPageDesign } from '../../design/composed/ErrorPage'
import type { HostedConfig } from '../../hosted-entry'

interface ErrorPageProps {
  /** Static error code passed directly (e.g. 404 from the fallback route). */
  code?: number | string
  /** Static message passed directly. Takes precedence over query string. */
  message?: string
  config?: HostedConfig
}

/**
 * Route-level error page.
 *
 * Two modes:
 *  1. Static: <ErrorPage code={404} message="Page not found" /> — used by App.tsx fallback route.
 *  2. Dynamic: <ErrorPage config={config} /> — reads ?code=...&msg=... from the URL.
 */
export function ErrorPage({ code: propCode, message: propMessage, config }: ErrorPageProps) {
  const [queryCode] = useState<string>(() => {
    const params = new URLSearchParams(window.location.search)
    return params.get('code') ?? ''
  })

  const [queryMsg] = useState<string>(() => {
    const params = new URLSearchParams(window.location.search)
    return params.get('msg') ?? ''
  })

  const displayCode = propCode != null ? String(propCode) : queryCode || undefined
  const displayMessage = propMessage ?? (queryMsg || 'An unexpected error occurred.')
  const slug = config?.app?.slug ?? ''

  const actions = slug
    ? [{ label: 'Back to sign in', href: `/hosted/${slug}/login` }]
    : undefined

  return (
    <ErrorPageDesign
      code={displayCode}
      title="Something went wrong"
      message={displayMessage}
      actions={actions}
    />
  )
}
