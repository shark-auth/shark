import { describe, it, expect } from 'vitest'
import React from 'react'
import { renderHook } from '@testing-library/react'
import { AuthContext } from '../context'
import type { AuthContextValue } from '../context'
import { useAuth } from '../useAuth'
import { useUser } from '../useUser'
import { useSession } from '../useSession'
import { useOrganization } from '../useOrganization'
import type { SharkClient } from '../../core/client'

// Minimal stub that satisfies SharkClient
const stubClient: SharkClient = {
  fetch: async () => new Response(null, { status: 200 }),
}

const stubValue: AuthContextValue = {
  isLoaded: true,
  isAuthenticated: true,
  user: { id: 'u1', email: 'test@example.com' },
  session: { id: 's1', userId: 'u1', expiresAt: 9999999999 },
  organization: { id: 'o1', name: 'Acme', slug: 'acme' },
  client: stubClient,
  signOut: async () => {},
}

function makeWrapper(value: AuthContextValue | null) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <AuthContext.Provider value={value}>
        {children}
      </AuthContext.Provider>
    )
  }
}

// ── Throw-without-provider tests ──────────────────────────────────────────

describe('useAuth — outside provider', () => {
  it('throws with descriptive message', () => {
    expect(() => renderHook(() => useAuth())).toThrow(
      'useAuth must be used within SharkProvider',
    )
  })
})

describe('useUser — outside provider', () => {
  it('throws with descriptive message', () => {
    expect(() => renderHook(() => useUser())).toThrow(
      'useUser must be used within SharkProvider',
    )
  })
})

describe('useSession — outside provider', () => {
  it('throws with descriptive message', () => {
    expect(() => renderHook(() => useSession())).toThrow(
      'useSession must be used within SharkProvider',
    )
  })
})

describe('useOrganization — outside provider', () => {
  it('throws with descriptive message', () => {
    expect(() => renderHook(() => useOrganization())).toThrow(
      'useOrganization must be used within SharkProvider',
    )
  })
})

// ── Inside-provider shape tests ────────────────────────────────────────────

describe('useAuth — inside provider', () => {
  const wrapper = makeWrapper(stubValue)

  it('returns isLoaded', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isLoaded).toBe(true)
  })

  it('returns isAuthenticated', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated).toBe(true)
  })

  it('returns signOut function', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(typeof result.current.signOut).toBe('function')
  })

  it('does not expose user/session/organization/client', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect((result.current as Record<string, unknown>).user).toBeUndefined()
    expect((result.current as Record<string, unknown>).session).toBeUndefined()
    expect((result.current as Record<string, unknown>).organization).toBeUndefined()
    expect((result.current as Record<string, unknown>).client).toBeUndefined()
  })
})

describe('useUser — inside provider', () => {
  const wrapper = makeWrapper(stubValue)

  it('returns isLoaded', () => {
    const { result } = renderHook(() => useUser(), { wrapper })
    expect(result.current.isLoaded).toBe(true)
  })

  it('returns user object', () => {
    const { result } = renderHook(() => useUser(), { wrapper })
    expect(result.current.user).toEqual({ id: 'u1', email: 'test@example.com' })
  })
})

describe('useUser — null user', () => {
  const wrapper = makeWrapper({ ...stubValue, user: null, isAuthenticated: false })

  it('returns null user when unauthenticated', () => {
    const { result } = renderHook(() => useUser(), { wrapper })
    expect(result.current.user).toBeNull()
  })
})

describe('useSession — inside provider', () => {
  const wrapper = makeWrapper(stubValue)

  it('returns isLoaded', () => {
    const { result } = renderHook(() => useSession(), { wrapper })
    expect(result.current.isLoaded).toBe(true)
  })

  it('returns session object', () => {
    const { result } = renderHook(() => useSession(), { wrapper })
    expect(result.current.session).toEqual({ id: 's1', userId: 'u1', expiresAt: 9999999999 })
  })
})

describe('useOrganization — inside provider', () => {
  const wrapper = makeWrapper(stubValue)

  it('returns isLoaded', () => {
    const { result } = renderHook(() => useOrganization(), { wrapper })
    expect(result.current.isLoaded).toBe(true)
  })

  it('returns organization object', () => {
    const { result } = renderHook(() => useOrganization(), { wrapper })
    expect(result.current.organization).toEqual({ id: 'o1', name: 'Acme', slug: 'acme' })
  })
})

describe('useOrganization — null org', () => {
  const wrapper = makeWrapper({ ...stubValue, organization: null })

  it('returns null organization when not set', () => {
    const { result } = renderHook(() => useOrganization(), { wrapper })
    expect(result.current.organization).toBeNull()
  })
})
