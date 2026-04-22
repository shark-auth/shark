import React from 'react'
import { AuthContext } from '../hooks/context'
import { useOrganization } from '../hooks/useOrganization'
import type { Organization } from '../core/types'

export interface OrganizationSwitcherProps {
  onOrganizationChange?: (org: Organization) => void
}

export function OrganizationSwitcher({ onOrganizationChange }: OrganizationSwitcherProps) {
  const ctx = React.useContext(AuthContext)
  const { isLoaded, organization } = useOrganization()
  const [orgs, setOrgs] = React.useState<Organization[]>([])
  const [loading, setLoading] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)

  React.useEffect(() => {
    if (!ctx || !isLoaded) return

    let cancelled = false
    setLoading(true)
    ctx.client
      .fetch('/api/v1/organizations')
      .then(async resp => {
        if (!resp.ok) throw new Error(`Failed to fetch organizations: ${resp.status}`)
        const data = await resp.json() as { organizations?: Organization[] } | Organization[]
        if (!cancelled) {
          const list = Array.isArray(data) ? data : (data.organizations ?? [])
          setOrgs(list)
        }
      })
      .catch(e => {
        if (!cancelled) setError(String(e))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => { cancelled = true }
  }, [ctx, isLoaded])

  if (!isLoaded) return null

  const handleChange = async (e: React.ChangeEvent<HTMLSelectElement>) => {
    const orgId = e.target.value
    const selected = orgs.find(o => o.id === orgId)
    if (!selected || !ctx) return

    try {
      // Notify server of active org switch
      await ctx.client.fetch('/api/v1/organizations/active', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ organizationId: orgId }),
      })
      onOrganizationChange?.(selected)
    } catch {
      // best-effort
    }
  }

  if (loading) return <span style={{ fontSize: 13, color: '#6b7280' }}>Loading organizations…</span>
  if (error) return <span style={{ fontSize: 13, color: '#ef4444' }}>{error}</span>
  if (orgs.length === 0) return null

  return (
    <select
      value={organization?.id ?? ''}
      onChange={handleChange}
      style={{
        padding: '6px 10px',
        border: '1px solid #d1d5db',
        borderRadius: 6,
        fontSize: 14,
        background: '#fff',
        cursor: 'pointer',
      }}
      aria-label="Switch organization"
    >
      {orgs.map(org => (
        <option key={org.id} value={org.id}>
          {org.name}
        </option>
      ))}
    </select>
  )
}
