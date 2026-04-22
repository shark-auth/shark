import React from 'react'
import { tokens } from '../tokens'
import { Button } from '../primitives/Button'
import { Card } from '../primitives/Card'

export interface ErrorPageAction {
  label: string
  href?: string
  onClick?: () => void
}

export interface ErrorPageProps {
  code?: string
  title: string
  message: string
  actions?: ErrorPageAction[]
}

function WarningIcon() {
  return (
    <svg
      width={56}
      height={56}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
      <path d="M12 9v4" />
      <path d="M12 17h.01" />
    </svg>
  )
}

export function ErrorPage({ code, title, message, actions = [] }: ErrorPageProps) {
  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    padding: `${tokens.space[8]}px ${tokens.space[6]}px`,
    minHeight: '100vh',
    background: tokens.color.surface0,
  }

  const innerStyle: React.CSSProperties = {
    width: '100%',
    maxWidth: 480,
    textAlign: 'center',
  }

  const iconWrapStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'center',
    color: tokens.color.danger,
    marginBottom: tokens.space[6],
  }

  const codeStyle: React.CSSProperties = {
    fontSize: tokens.type.size.xs,
    fontFamily: tokens.type.mono.family,
    fontWeight: tokens.type.weight.medium,
    color: tokens.color.fgDim,
    textTransform: 'uppercase',
    letterSpacing: '0.08em',
    marginBottom: tokens.space[2],
  }

  const titleStyle: React.CSSProperties = {
    fontSize: tokens.type.size['2xl'],
    fontFamily: tokens.type.display.family,
    fontWeight: tokens.type.weight.bold,
    color: tokens.color.fg,
    marginBottom: tokens.space[3],
  }

  const messageStyle: React.CSSProperties = {
    fontSize: tokens.type.size.base,
    fontFamily: tokens.type.body.family,
    color: tokens.color.fgMuted,
    lineHeight: 1.6,
    marginBottom: tokens.space[8],
  }

  const actionsStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    gap: tokens.space[2],
    alignItems: 'center',
  }

  const linkBtnStyle: React.CSSProperties = {
    width: '100%',
    maxWidth: 280,
  }

  return (
    <div style={containerStyle}>
      <div style={innerStyle}>
        <Card bodyStyle={{ padding: `${tokens.space[8]}px ${tokens.space[6]}px` }}>
          <div style={iconWrapStyle}>
            <WarningIcon />
          </div>

          {code && (
            <p style={codeStyle} aria-label={`Error code: ${code}`}>{code}</p>
          )}

          <h1 style={titleStyle}>{title}</h1>
          <p style={messageStyle}>{message}</p>

          {actions.length > 0 && (
            <div style={actionsStyle}>
              {actions.map((action, i) => (
                action.href ? (
                  <a
                    key={i}
                    href={action.href}
                    style={{ textDecoration: 'none', ...linkBtnStyle }}
                  >
                    <Button
                      type="button"
                      variant={i === 0 ? 'primary' : 'ghost'}
                      size="lg"
                      style={{ width: '100%' }}
                    >
                      {action.label}
                    </Button>
                  </a>
                ) : (
                  <Button
                    key={i}
                    type="button"
                    variant={i === 0 ? 'primary' : 'ghost'}
                    size="lg"
                    onClick={action.onClick}
                    style={linkBtnStyle}
                  >
                    {action.label}
                  </Button>
                )
              ))}
            </div>
          )}
        </Card>
      </div>
    </div>
  )
}

ErrorPage.displayName = 'ErrorPage'
