import React from 'react'
import { useAuth } from '../hooks/useAuth'

export interface SignedOutProps {
  children: React.ReactNode
}

export function SignedOut({ children }: SignedOutProps) {
  const { isLoaded, isAuthenticated } = useAuth()
  if (!isLoaded) return null
  return isAuthenticated ? null : <>{children}</>
}
