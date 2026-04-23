import React from 'react'
import { tokens } from '../tokens'

export interface CardProps {
  /** Optional header content rendered in a bordered header region */
  header?: React.ReactNode
  children?: React.ReactNode
  style?: React.CSSProperties
  /** Applied to the inner body padding container */
  bodyStyle?: React.CSSProperties
  className?: string
}

export function Card({ header, children, style, bodyStyle, className }: CardProps) {
  const cardStyle: React.CSSProperties = {
    background: tokens.color.surface2,
    border: `1px solid ${tokens.color.hairline}`,
    borderRadius: tokens.radius.lg,
    overflow: 'hidden',
    ...style,
  }

  const headerStyle: React.CSSProperties = {
    padding: `${tokens.space[3]}px ${tokens.space[6]}px`,
    borderBottom: `1px solid ${tokens.color.hairline}`,
    fontSize: tokens.type.size.sm,
    fontFamily: tokens.type.body.family,
    fontWeight: tokens.type.weight.semibold,
    color: tokens.color.fg,
    display: 'flex',
    alignItems: 'center',
    gap: tokens.space[2],
  }

  const bodyStyleComputed: React.CSSProperties = {
    padding: tokens.space[6],
    ...bodyStyle,
  }

  return (
    <div style={cardStyle} className={className}>
      {header && <div style={headerStyle}>{header}</div>}
      <div style={bodyStyleComputed}>{children}</div>
    </div>
  )
}

Card.displayName = 'Card'
