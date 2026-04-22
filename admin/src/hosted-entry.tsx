import React from 'react'
import ReactDOM from 'react-dom/client'
import { HostedApp } from './hosted/App'

export interface HostedConfig {
  app: { slug: string; name: string; logo_url?: string }
  branding: {
    primary_color?: string
    secondary_color?: string
    font_family?: string
    logo_url?: string
  }
  authMethods: ('password' | 'magic_link' | 'passkey' | 'oauth')[]
  oauthProviders?: { id: string; name: string; iconUrl?: string }[]
  oauth: { client_id: string; redirect_uri: string; state: string; scope?: string }
}

const cfg = (window as unknown as { __SHARK_HOSTED: HostedConfig | null }).__SHARK_HOSTED
const root = document.getElementById('hosted-root')!
if (!cfg) {
  root.textContent = 'Hosted auth config missing'
} else {
  ReactDOM.createRoot(root).render(<HostedApp config={cfg} />)
}
