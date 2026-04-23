import { createContext } from 'react'
import type { User, Session, Organization } from '../core/types'
import type { createClient } from '../core/client'

export interface GetTokenOptions {
  dpop?: boolean
  method?: string
  url?: string
}

export interface GetTokenResult {
  token: string
  dpop?: string
}

export interface AuthContextValue {
  isLoaded: boolean
  isAuthenticated: boolean
  user: User | null
  session: Session | null
  organization: Organization | null
  client: ReturnType<typeof createClient>
  getToken: (opts?: GetTokenOptions) => Promise<string | GetTokenResult | null>
  signOut: () => Promise<void>
  /** Available for internal use by SignIn/SignUp components */
  authUrl?: string
  publishableKey?: string
}

export const AuthContext = createContext<AuthContextValue | null>(null)
