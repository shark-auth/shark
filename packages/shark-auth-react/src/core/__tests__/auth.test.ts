import { describe, it, expect } from 'vitest'
import { generateCodeVerifier, generateCodeChallenge } from '../auth'

describe('generateCodeVerifier', () => {
  it('produces a string of exactly 64 characters', () => {
    const v = generateCodeVerifier()
    expect(typeof v).toBe('string')
    expect(v).toHaveLength(64)
  })

  it('produces different values on successive calls', () => {
    const a = generateCodeVerifier()
    const b = generateCodeVerifier()
    expect(a).not.toBe(b)
  })

  it('only contains base64url-safe characters', () => {
    const v = generateCodeVerifier()
    expect(v).toMatch(/^[A-Za-z0-9\-_]+$/)
  })
})

describe('generateCodeChallenge', () => {
  it('is deterministic for a known verifier', async () => {
    // SHA-256 of "abc" base64url = "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"
    const challenge = await generateCodeChallenge('abc')
    expect(challenge).toBe('ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0')
  })

  it('returns a non-empty string without padding characters', async () => {
    const challenge = await generateCodeChallenge(generateCodeVerifier())
    expect(challenge.length).toBeGreaterThan(0)
    expect(challenge).not.toContain('=')
    expect(challenge).not.toContain('+')
    expect(challenge).not.toContain('/')
  })
})
