import { useContext } from 'react'
import { AuthContext } from './context'

export function useOrganization() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useOrganization must be used within SharkProvider')
  return {
    isLoaded: ctx.isLoaded,
    organization: ctx.organization,
  }
}
