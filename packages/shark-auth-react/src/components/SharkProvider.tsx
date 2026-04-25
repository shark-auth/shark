import React from 'react'
import { createClient } from '../core/client'
import { getAccessToken, getRefreshToken, clearAll } from '../core/storage'
import { decodeClaims } from '../core/jwt'
import { exchangeToken } from '../core/auth'
import { generateDPoPProver, type DPoPProver } from '../core/dpop'
import { AuthContext, type GetTokenOptions, type GetTokenResult } from '../hooks/context'
import type { User, Session, Organization } from '../core/types'

export interface SharkProviderProps {
  publishableKey: string
  authUrl: string
  dpop?: boolean
  children: React.ReactNode
}

interface AuthState {
  isLoaded: boolean
  isAuthenticated: boolean
  user: User | null
  session: Session | null
  organization: Organization | null
}

export function SharkProvider({ publishableKey, authUrl, dpop, children }: SharkProviderProps) {
  const [state, setState] = React.useState<AuthState>({
    isLoaded: false,
    isAuthenticated: false,
    user: null,
    session: null,
    organization: null,
  })

  const [prover, setProver] = React.useState<DPoPProver | null>(null)

  React.useEffect(() => {
    if (dpop && !prover) {
      generateDPoPProver().then(setProver)
    }
  }, [dpop, prover])

  const client = React.useMemo(
    () => createClient(authUrl, publishableKey),
    [authUrl, publishableKey],
  )

  const getToken = React.useCallback(async (opts?: GetTokenOptions): Promise<string | GetTokenResult | null> => {
    let token = getAccessToken()

    // Auto-refresh logic
    if (!token) {
      const rt = getRefreshToken()
      if (rt) {
        try {
          let dpopProof: string | undefined
          if (prover && opts?.dpop) {
            dpopProof = await prover.createProof('POST', `${authUrl}/oauth/token`)
          }

          const resp = await exchangeToken(authUrl, publishableKey, {
            refreshToken: rt,
            dpopProof,
          })
          token = resp.access_token
        } catch {
          clearAll()
          setState(s => ({ ...s, isAuthenticated: false, user: null, session: null, organization: null }))
          return null
        }
      }
    }

    if (!token) return null

    if (opts?.dpop) {
      if (!prover) throw new Error('DPoP is not enabled on SharkProvider')
      if (!opts.method || !opts.url) throw new Error('method and url are required for DPoP tokens')

      const proof = await prover.createProof(opts.method, opts.url, token)
      return { token, dpop: proof }
    }

    return token
  }, [authUrl, publishableKey, prover])

  React.useEffect(() => {
    let cancelled = false

    async function hydrate() {
      const token = getAccessToken()
      if (!token) {
        if (!cancelled) setState(s => ({ ...s, isLoaded: true }))
        return
      }

      try {
        // Try /api/v1/users/me for live user data
        const resp = await client.fetch('/api/v1/users/me')
        if (resp.ok) {
          const data = await resp.json() as { user?: User; session?: Session; organization?: Organization }
          if (!cancelled) {
            setState({
              isLoaded: true,
              isAuthenticated: true,
              user: data.user ?? null,
              session: data.session ?? null,
              organization: data.organization ?? null,
            })
          }
          return
        }
      } catch {
        // network error — fall through to local decode
      }

      // Fallback: decode claims from JWT (offline / dev)
      try {
        const claims = decodeClaims(token)
        const user: User = {
          id: (claims.sub as string) ?? '',
          email: (claims.email as string) ?? '',
          firstName: claims.firstName as string | undefined,
          lastName: claims.lastName as string | undefined,
          imageUrl: claims.imageUrl as string | undefined,
        }
        const session: Session = {
          id: (claims.jti as string) ?? '',
          userId: user.id,
          expiresAt: typeof claims.exp === 'number' ? claims.exp * 1000 : Date.now() + 3600_000,
        }
        if (!cancelled) {
          setState({
            isLoaded: true,
            isAuthenticated: true,
            user,
            session,
            organization: null,
          })
        }
      } catch {
        clearAll()
        if (!cancelled) setState(s => ({ ...s, isLoaded: true }))
      }
    }

    hydrate()
    return () => { cancelled = true }
  }, [authUrl, publishableKey, client])

  const signOut = React.useCallback(async () => {
    const token = getAccessToken()
    if (token) {
      try {
        await client.fetch('/oauth/revoke', {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: `token=${encodeURIComponent(token)}`,
        })
      } catch {
        // best-effort revoke
      }
    }
    clearAll()
    setState(s => ({ ...s, isAuthenticated: false, user: null, session: null, organization: null }))
  }, [client])

  const value = React.useMemo(
    () => ({ ...state, client, getToken, signOut, authUrl, publishableKey }),
    [state, client, getToken, signOut, authUrl, publishableKey],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
