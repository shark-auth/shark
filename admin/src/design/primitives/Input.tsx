import React from 'react'
import { tokens } from '../tokens'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  error?: boolean
  /** Optional leading icon rendered inside the input */
  leadingIcon?: React.ReactNode
  /** Optional trailing adornment (icon or node) */
  trailingAdornment?: React.ReactNode
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  function Input(
    {
      error = false,
      leadingIcon,
      trailingAdornment,
      disabled,
      style,
      onFocus,
      onBlur,
      ...rest
    },
    ref,
  ) {
    const [focused, setFocused] = React.useState(false)

    const borderColor = error
      ? tokens.color.danger
      : focused
      ? tokens.color.primary
      : tokens.color.hairline

    const wrapperStyle: React.CSSProperties = {
      position: 'relative',
      display: 'inline-flex',
      alignItems: 'center',
      width: '100%',
      height: 34,
      background: tokens.color.surface2,
      border: `1px solid ${borderColor}`,
      borderRadius: tokens.radius.md,
      transition: `border-color ${tokens.motion.fast}`,
      opacity: disabled ? 0.5 : 1,
      boxSizing: 'border-box',
    }

    const inputStyle: React.CSSProperties = {
      flex: 1,
      height: '100%',
      padding: `0 ${tokens.space[3]}px`,
      paddingLeft: leadingIcon ? 32 : tokens.space[3],
      paddingRight: trailingAdornment ? 32 : tokens.space[3],
      background: 'transparent',
      border: 'none',
      outline: 'none',
      color: tokens.color.fg,
      fontSize: tokens.type.size.base,
      fontFamily: tokens.type.body.family,
      fontWeight: tokens.type.weight.regular,
      lineHeight: 1,
      cursor: disabled ? 'not-allowed' : 'text',
      ...style,
    }

    const iconWrapStyle: React.CSSProperties = {
      position: 'absolute',
      left: tokens.space[2],
      display: 'flex',
      alignItems: 'center',
      color: tokens.color.fgDim,
      pointerEvents: 'none',
    }

    const trailingWrapStyle: React.CSSProperties = {
      position: 'absolute',
      right: tokens.space[2],
      display: 'flex',
      alignItems: 'center',
      color: tokens.color.fgDim,
    }

    const focusRingStyle: React.CSSProperties = focused
      ? {
          boxShadow: `0 0 0 2px ${tokens.color.focusRing}33`,
        }
      : {}

    return (
      <div style={{ ...wrapperStyle, ...focusRingStyle }}>
        {leadingIcon && <span style={iconWrapStyle}>{leadingIcon}</span>}
        <input
          ref={ref}
          disabled={disabled}
          aria-invalid={error || undefined}
          style={inputStyle}
          onFocus={(e) => { setFocused(true); onFocus?.(e) }}
          onBlur={(e) => { setFocused(false); onBlur?.(e) }}
          {...rest}
        />
        {trailingAdornment && <span style={trailingWrapStyle}>{trailingAdornment}</span>}
      </div>
    )
  },
)

Input.displayName = 'Input'
