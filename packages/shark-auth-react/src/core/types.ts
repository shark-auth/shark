export interface User {
  id: string
  email: string
  firstName?: string
  lastName?: string
  imageUrl?: string
}

export interface Session {
  id: string
  userId: string
  expiresAt: number
}

export interface Organization {
  id: string
  name: string
  slug: string
}

export interface AuthConfig {
  authUrl: string
  publishableKey: string
}

export interface TokenPair {
  accessToken: string
  refreshToken?: string
  expiresAt: number
}
