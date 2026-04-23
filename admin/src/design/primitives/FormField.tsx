import React from 'react'
import { tokens } from '../tokens'
import { Input, InputProps } from './Input'

export interface FormFieldProps {
  /** Field label — displayed above the input */
  label?: string
  /** Helper text shown below the input in muted color */
  helperText?: string
  /** Error message — overrides helperText and sets input error state */
  errorText?: string
  /** Marks field as required (adds asterisk to label) */
  required?: boolean
  /** htmlFor on the label — defaults to input id if provided */
  htmlFor?: string
  /** All InputProps are spread onto the underlying Input */
  inputProps?: InputProps
  /** Render custom child instead of default Input */
  children?: React.ReactNode
  style?: React.CSSProperties
}

export function FormField({
  label,
  helperText,
  errorText,
  required,
  htmlFor,
  inputProps,
  children,
  style,
}: FormFieldProps) {
  const hasError = Boolean(errorText)
  const inputId = htmlFor ?? inputProps?.id

  const wrapperStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.space[2],
    ...style,
  }

  const labelStyle: React.CSSProperties = {
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    fontWeight: tokens.type.weight.medium,
    color: tokens.color.fgMuted,
    lineHeight: 1,
    display: 'flex',
    alignItems: 'center',
    gap: 3,
  }

  const requiredStyle: React.CSSProperties = {
    color: tokens.color.danger,
    lineHeight: 1,
  }

  const helperStyle: React.CSSProperties = {
    fontSize: tokens.type.size.xs,
    fontFamily: tokens.type.body.family,
    fontWeight: tokens.type.weight.regular,
    color: hasError ? tokens.color.danger : tokens.color.fgDim,
    lineHeight: 1.4,
  }

  return (
    <div style={wrapperStyle}>
      {label && (
        <label htmlFor={inputId} style={labelStyle}>
          {label}
          {required && <span style={requiredStyle} aria-hidden="true">*</span>}
        </label>
      )}
      {children ?? (
        <Input
          id={inputId}
          error={hasError}
          aria-describedby={
            (helperText || errorText) ? `${inputId}-help` : undefined
          }
          aria-required={required || undefined}
          {...inputProps}
        />
      )}
      {(helperText || errorText) && (
        <span
          id={inputId ? `${inputId}-help` : undefined}
          role={hasError ? 'alert' : undefined}
          style={helperStyle}
        >
          {errorText ?? helperText}
        </span>
      )}
    </div>
  )
}

FormField.displayName = 'FormField'
