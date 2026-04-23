import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'

export interface OAuthProviderButtonProps {
  providerID: string
  providerName: string
  iconUrl?: string
  onClick: () => void
  loading?: boolean
}

const ICON_SIZE = 20

function ProviderIcon({ iconUrl, providerName }: { iconUrl?: string; providerName: string }) {
  const fallbackStyle: React.CSSProperties = {
    width: ICON_SIZE,
    height: ICON_SIZE,
    borderRadius: tokens.radius.sm,
    background: tokens.color.surface3,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: tokens.type.size.xs,
    fontFamily: tokens.type.body.family,
    fontWeight: tokens.type.weight.semibold,
    color: tokens.color.fgMuted,
    flexShrink: 0,
    userSelect: 'none',
  }

  if (iconUrl) {
    return (
      <img
        src={iconUrl}
        alt=""
        aria-hidden="true"
        width={ICON_SIZE}
        height={ICON_SIZE}
        style={{ borderRadius: tokens.radius.sm, flexShrink: 0, objectFit: 'contain' }}
        onError={(e) => {
          const target = e.currentTarget
          target.style.display = 'none'
          const fallback = target.nextElementSibling as HTMLElement | null
          if (fallback) fallback.style.display = 'flex'
        }}
      />
    )
  }

  return (
    <span style={fallbackStyle} aria-hidden="true">
      {providerName.charAt(0).toUpperCase()}
    </span>
  )
}

export function OAuthProviderButton({
  providerName,
  iconUrl,
  onClick,
  loading = false,
}: OAuthProviderButtonProps) {
  const [internalLoading, setInternalLoading] = React.useState(false)
  const isLoading = loading || internalLoading

  function handleClick() {
    if (isLoading) return
    setInternalLoading(true)
    try {
      onClick()
    } finally {
      // External navigation typically — reset after a tick to avoid stuck state
      setTimeout(() => setInternalLoading(false), 3000)
    }
  }

  const buttonInnerStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: tokens.space[2],
    width: '100%',
    justifyContent: 'center',
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="lg"
      loading={isLoading}
      disabled={isLoading}
      onClick={handleClick}
      style={{ width: '100%' }}
    >
      {!isLoading && (
        <span style={buttonInnerStyle}>
          <ProviderIcon iconUrl={iconUrl} providerName={providerName} />
          Continue with {providerName}
        </span>
      )}
      {isLoading && `Continue with ${providerName}`}
    </Button>
  )
}

OAuthProviderButton.displayName = 'OAuthProviderButton'
