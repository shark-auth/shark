import { describe, it, expect } from 'vitest'
import React from 'react'
import { render, screen } from '@testing-library/react'
import { AuthContext } from '../../hooks/context'
import type { AuthContextValue } from '../../hooks/context'
import type { SharkClient } from '../../core/client'
import { SignedIn } from '../SignedIn'

const stubClient: SharkClient = { fetch: async () => new Response(null, { status: 200 }) }

function makeValue(overrides: Partial<AuthContextValue>): AuthContextValue {
  return {
    isLoaded: true,
    isAuthenticated: false,
    user: null,
    session: null,
    organization: null,
    client: stubClient,
    signOut: async () => {},
    ...overrides,
  }
}

function Wrapper({ value, children }: { value: AuthContextValue; children: React.ReactNode }) {
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

describe('SignedIn', () => {
  it('renders children when authenticated and loaded', () => {
    render(
      <Wrapper value={makeValue({ isAuthenticated: true })}>
        <SignedIn>
          <span>protected content</span>
        </SignedIn>
      </Wrapper>,
    )
    expect(screen.getByText('protected content')).toBeTruthy()
  })

  it('renders nothing when not authenticated', () => {
    const { container } = render(
      <Wrapper value={makeValue({ isAuthenticated: false })}>
        <SignedIn>
          <span>should be hidden</span>
        </SignedIn>
      </Wrapper>,
    )
    expect(container.textContent).toBe('')
  })

  it('renders nothing while not loaded', () => {
    const { container } = render(
      <Wrapper value={makeValue({ isLoaded: false, isAuthenticated: true })}>
        <SignedIn>
          <span>loading</span>
        </SignedIn>
      </Wrapper>,
    )
    expect(container.textContent).toBe('')
  })
})
