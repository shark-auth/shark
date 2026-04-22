import { describe, it, expect, beforeEach } from 'vitest'
import {
  getAccessToken,
  setAccessToken,
  clearAll,
  getCodeVerifier,
  setCodeVerifier,
  getRedirectAfter,
  setRedirectAfter,
} from '../storage'

describe('storage (browser/jsdom)', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('returns null when no token set', () => {
    expect(getAccessToken()).toBeNull()
  })

  it('stores and retrieves an access token that has not expired', () => {
    const future = Date.now() + 60_000
    setAccessToken('tok_abc', future)
    expect(getAccessToken()).toBe('tok_abc')
  })

  it('returns null and clears when token is expired', () => {
    const past = Date.now() - 1000
    setAccessToken('tok_expired', past)
    expect(getAccessToken()).toBeNull()
    // clearAll was called; token key should be gone
    expect(sessionStorage.getItem('shark_access_token')).toBeNull()
  })

  it('stores and retrieves code verifier', () => {
    setCodeVerifier('verifier_xyz')
    expect(getCodeVerifier()).toBe('verifier_xyz')
  })

  it('stores and retrieves redirect-after URL', () => {
    setRedirectAfter('/dashboard')
    expect(getRedirectAfter()).toBe('/dashboard')
  })

  it('clearAll removes all shark keys', () => {
    setAccessToken('tok', Date.now() + 99999)
    setCodeVerifier('ver')
    setRedirectAfter('/x')
    clearAll()
    expect(getAccessToken()).toBeNull()
    expect(getCodeVerifier()).toBeNull()
    expect(getRedirectAfter()).toBeNull()
  })
})
