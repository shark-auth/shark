import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import { SharkProvider } from '../SharkProvider'
import { useAuth } from '../../hooks/useAuth'
import { useUser } from '../../hooks/useUser'
import * as storage from '../../core/storage'

// ---------- helpers ----------

function AuthStatus() {
  const { isLoaded, isAuthenticated } = useAuth()
  const { user } = useUser()
  return (
    <div>
      <span data-testid="loaded">{String(isLoaded)}</span>
      <span data-testid="authenticated">{String(isAuthenticated)}</span>
      <span data-testid="email">{user?.email ?? 'none'}</span>
    </div>
  )
}

const PUBLISHABLE_KEY = 'pk_test_abc'
const AUTH_URL = 'https://auth.example.com'

// ---------- tests ----------

beforeEach(() => {
  vi.restoreAllMocks()
  storage.clearAll()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SharkProvider — no token', () => {
  it('sets isLoaded=true and isAuthenticated=false when no token in storage', async () => {
    render(
      <SharkProvider publishableKey={PUBLISHABLE_KEY} authUrl={AUTH_URL}>
        <AuthStatus />
      </SharkProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('loaded').textContent).toBe('true')
    })
    expect(screen.getByTestId('authenticated').textContent).toBe('false')
    expect(screen.getByTestId('email').textContent).toBe('none')
  })
})

describe('SharkProvider — hydration path with /users/me', () => {
  it('fetches /api/v1/users/me and hydrates user when token present', async () => {
    // Put a fake (non-expired) token in storage
    storage.setAccessToken('fake.token.value', Date.now() + 3600_000)

    const mockUser = { id: 'u1', email: 'test@example.com', firstName: 'Test' }
    const mockSession = { id: 's1', userId: 'u1', expiresAt: 9999999999000 }

    // Mock global fetch used inside createClient
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ user: mockUser, session: mockSession }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )

    render(
      <SharkProvider publishableKey={PUBLISHABLE_KEY} authUrl={AUTH_URL}>
        <AuthStatus />
      </SharkProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('authenticated').textContent).toBe('true')
    })
    expect(screen.getByTestId('email').textContent).toBe('test@example.com')
    expect(fetchSpy).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/users/me'),
      expect.any(Object),
    )
  })
})

describe('SharkProvider — hydration fallback to JWT decode', () => {
  it('falls back to JWT claims when /users/me fails', async () => {
    // A real-ish JWT with sub + email claims (not signature-verified, just decoded)
    const header = btoa(JSON.stringify({ alg: 'RS256', typ: 'JWT' })).replace(/=/g, '')
    const payload = btoa(
      JSON.stringify({
        sub: 'u2',
        email: 'fallback@example.com',
        exp: Math.floor(Date.now() / 1000) + 3600,
        jti: 's2',
      }),
    ).replace(/=/g, '')
    const fakeJwt = `${header}.${payload}.fake-sig`

    storage.setAccessToken(fakeJwt, Date.now() + 3600_000)

    // Make the /users/me call fail
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network error'))

    render(
      <SharkProvider publishableKey={PUBLISHABLE_KEY} authUrl={AUTH_URL}>
        <AuthStatus />
      </SharkProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('authenticated').textContent).toBe('true')
    })
    expect(screen.getByTestId('email').textContent).toBe('fallback@example.com')
  })
})

describe('SharkProvider — clears on bad token', () => {
  it('sets isAuthenticated=false when token decode throws', async () => {
    storage.setAccessToken('not.a.valid.jwt.here', Date.now() + 3600_000)

    // /users/me fails and JWT decode also fails
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('network error'))

    render(
      <SharkProvider publishableKey={PUBLISHABLE_KEY} authUrl={AUTH_URL}>
        <AuthStatus />
      </SharkProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('loaded').textContent).toBe('true')
    })
    expect(screen.getByTestId('authenticated').textContent).toBe('false')
  })
})
