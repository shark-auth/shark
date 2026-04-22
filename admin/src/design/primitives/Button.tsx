import React from 'react'
import { tokens } from '../tokens'

export type ButtonVariant = 'primary' | 'ghost' | 'danger' | 'icon'
export type ButtonSize = 'sm' | 'md' | 'lg'

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
  children?: React.ReactNode
}

const heightMap: Record<ButtonSize, number> = { sm: 28, md: 32, lg: 40 }
const fontSizeMap: Record<ButtonSize, number> = { sm: 12, md: 13, lg: 14 }
const paddingMap: Record<ButtonSize, string> = { sm: '0 10px', md: '0 14px', lg: '0 18px' }

function Spinner({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      aria-hidden="true"
      style={{
        animation: `shark-spin 700ms linear infinite`,
        flexShrink: 0,
      }}
    >
      <style>{`@keyframes shark-spin { to { transform: rotate(360deg); } }`}</style>
      <circle
        cx="8" cy="8" r="6"
        stroke="currentColor"
        strokeWidth="2"
        strokeOpacity="0.25"
      />
      <path
        d="M14 8a6 6 0 00-6-6"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
      />
    </svg>
  )
}

function getVariantStyles(variant: ButtonVariant, disabled: boolean): React.CSSProperties {
  const base: React.CSSProperties = {
    transition: `transform ${tokens.motion.fast}, box-shadow ${tokens.motion.fast}, background ${tokens.motion.fast}, opacity ${tokens.motion.fast}`,
    cursor: disabled ? 'not-allowed' : 'pointer',
    opacity: disabled ? 0.5 : 1,
    border: '1px solid transparent',
    outline: 'none',
  }

  switch (variant) {
    case 'primary':
      return {
        ...base,
        background: tokens.color.primary,
        color: tokens.color.primaryFg,
        borderColor: 'transparent',
      }
    case 'ghost':
      return {
        ...base,
        background: 'transparent',
        color: tokens.color.fg,
        borderColor: tokens.color.hairline,
      }
    case 'danger':
      return {
        ...base,
        background: tokens.color.danger,
        color: tokens.color.dangerFg,
        borderColor: 'transparent',
      }
    case 'icon':
      return {
        ...base,
        background: 'transparent',
        color: tokens.color.fgMuted,
        borderColor: 'transparent',
        padding: '0',
      }
  }
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  function Button(
    {
      variant = 'ghost',
      size = 'md',
      loading = false,
      disabled = false,
      children,
      style,
      onMouseEnter,
      onMouseLeave,
      onMouseDown,
      onMouseUp,
      onFocus,
      onBlur,
      ...rest
    },
    ref,
  ) {
    const [hovered, setHovered] = React.useState(false)
    const [pressed, setPressed] = React.useState(false)
    const [focused, setFocused] = React.useState(false)

    const h = heightMap[size]
    const isDisabledOrLoading = disabled || loading
    const isIcon = variant === 'icon'

    const variantStyles = getVariantStyles(variant, isDisabledOrLoading)

    const hoverStyles: React.CSSProperties =
      hovered && !isDisabledOrLoading
        ? {
            transform: 'translateY(-1px)',
            boxShadow: tokens.shadow.sm,
            ...(variant === 'ghost' ? { borderColor: tokens.color.fgDim } : {}),
          }
        : {}

    const pressedStyles: React.CSSProperties =
      pressed && !isDisabledOrLoading
        ? { transform: 'translateY(0)', boxShadow: 'none' }
        : {}

    const focusStyles: React.CSSProperties = focused
      ? {
          outline: `2px solid ${tokens.color.focusRing}`,
          outlineOffset: '2px',
        }
      : {}

    const minWidth = loading ? `${isIcon ? h : h * 2.5}px` : undefined

    const computed: React.CSSProperties = {
      display: 'inline-flex',
      alignItems: 'center',
      justifyContent: 'center',
      gap: 6,
      height: h,
      width: isIcon ? h : undefined,
      minWidth,
      padding: isIcon ? '0' : paddingMap[size],
      borderRadius: tokens.radius.md,
      fontSize: fontSizeMap[size],
      fontFamily: tokens.type.body.family,
      fontWeight: tokens.type.weight.medium,
      lineHeight: 1,
      userSelect: 'none',
      whiteSpace: 'nowrap',
      ...variantStyles,
      ...hoverStyles,
      ...pressedStyles,
      ...focusStyles,
      ...style,
    }

    return (
      <button
        ref={ref}
        disabled={isDisabledOrLoading}
        aria-busy={loading || undefined}
        aria-disabled={isDisabledOrLoading || undefined}
        style={computed}
        onMouseEnter={(e) => { setHovered(true); onMouseEnter?.(e) }}
        onMouseLeave={(e) => { setHovered(false); setPressed(false); onMouseLeave?.(e) }}
        onMouseDown={(e) => { setPressed(true); onMouseDown?.(e) }}
        onMouseUp={(e) => { setPressed(false); onMouseUp?.(e) }}
        onFocus={(e) => { setFocused(true); onFocus?.(e) }}
        onBlur={(e) => { setFocused(false); onBlur?.(e) }}
        {...rest}
      >
        {loading ? <Spinner size={size === 'lg' ? 16 : 14} /> : children}
      </button>
    )
  },
)

Button.displayName = 'Button'
