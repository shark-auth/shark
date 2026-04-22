import React from 'react'
import { Button } from '../primitives/Button'

export interface PasskeyButtonProps {
  onClick: () => Promise<void>
  label?: string
  loading?: boolean
  disabled?: boolean
}

function KeyIcon() {
  return (
    <svg
      width={16}
      height={16}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      style={{ flexShrink: 0 }}
    >
      <circle cx="7.5" cy="15.5" r="5.5" />
      <path d="M21 2l-9.6 9.6" />
      <path d="M15.5 7.5l3 3L22 7l-3-3" />
    </svg>
  )
}

export function PasskeyButton({
  onClick,
  label = 'Continue with passkey',
  loading = false,
  disabled = false,
}: PasskeyButtonProps) {
  const [internalLoading, setInternalLoading] = React.useState(false)
  const isLoading = loading || internalLoading

  async function handleClick() {
    if (isLoading || disabled) return
    setInternalLoading(true)
    try {
      await onClick()
    } finally {
      setInternalLoading(false)
    }
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="lg"
      loading={isLoading}
      disabled={disabled || isLoading}
      onClick={handleClick}
      style={{ width: '100%', gap: 8 }}
    >
      {!isLoading && <KeyIcon />}
      {label}
    </Button>
  )
}

PasskeyButton.displayName = 'PasskeyButton'
