import { describe, it, expect } from 'vitest'
import React from 'react'
import { render, screen } from '@testing-library/react'
import { AuthContext } from '../../hooks/context'
import type { AuthContextValue } from '../../hooks/context'
import type { SharkClient } from '../../core/client'
import { SignedOut } from '../SignedOut'

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

describe('SignedOut', () => {
  it('renders children when NOT authenticated and loaded', () => {
    render(
      <Wrapper value={makeValue({ isAuthenticated: false })}>
        <SignedOut>
          <span>public content</span>
        </SignedOut>
      </Wrapper>,
    )
    expect(screen.getByText('public content')).toBeTruthy()
  })

  it('renders nothing when authenticated', () => {
    const { container } = render(
      <Wrapper value={makeValue({ isAuthenticated: true })}>
        <SignedOut>
          <span>should be hidden</span>
        </SignedOut>
      </Wrapper>,
    )
    expect(container.textContent).toBe('')
  })

  it('renders nothing while not loaded', () => {
    const { container } = render(
      <Wrapper value={makeValue({ isLoaded: false, isAuthenticated: false })}>
        <SignedOut>
          <span>loading</span>
        </SignedOut>
      </Wrapper>,
    )
    expect(container.textContent).toBe('')
  })
})
