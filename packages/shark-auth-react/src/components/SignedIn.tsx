import React from 'react'
import { useAuth } from '../hooks/useAuth'

export interface SignedInProps {
  children: React.ReactNode
}

export function SignedIn({ children }: SignedInProps) {
  const { isLoaded, isAuthenticated } = useAuth()
  if (!isLoaded) return null
  return isAuthenticated ? <>{children}</> : null
}
