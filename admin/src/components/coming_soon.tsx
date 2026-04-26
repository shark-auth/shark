// @ts-nocheck
import React from 'react'

interface ComingSoonProps {
  title: string;
  message: string;
  hint?: string;
  githubUrl?: string;
}

export function ComingSoon({ title, message, hint, githubUrl = 'https://github.com/sharkauth/sharkauth/discussions' }: ComingSoonProps) {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100%',
      padding: 32,
    }}>
      <div style={{
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 0,
        padding: '32px 36px',
        maxWidth: 480,
        width: '100%',
        textAlign: 'center',
      }}>
        <div style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          width: 40,
          height: 40,
          background: 'var(--surface-3)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 0,
          marginBottom: 20,
        }}>
          <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
            <circle cx="9" cy="9" r="8" stroke="currentColor" strokeWidth="1.5"/>
            <path d="M9 5v4.5M9 13v.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="square"/>
          </svg>
        </div>

        <h2 style={{
          fontSize: 15,
          fontWeight: 600,
          margin: '0 0 8px',
          color: 'var(--fg)',
          letterSpacing: '-0.01em',
        }}>{title}</h2>

        <p style={{
          fontSize: 12.5,
          color: 'var(--fg-muted)',
          margin: '0 0 16px',
          lineHeight: 1.6,
        }}>{message}</p>

        {hint && (
          <p style={{
            fontSize: 11.5,
            color: 'var(--fg-dim)',
            margin: '0 0 20px',
            padding: '10px 12px',
            background: 'var(--surface-0)',
            border: '1px solid var(--hairline)',
            borderRadius: 0,
            lineHeight: 1.6,
            textAlign: 'left',
            fontFamily: 'var(--font-mono, monospace)',
          }}>{hint}</p>
        )}

        <a
          href={githubUrl}
          target="_blank"
          rel="noopener noreferrer"
          style={{
            fontSize: 12,
            color: 'var(--fg-muted)',
            textDecoration: 'none',
            borderBottom: '1px solid var(--hairline-strong)',
            paddingBottom: 1,
          }}
          onMouseEnter={e => (e.currentTarget.style.color = 'var(--fg)')}
          onMouseLeave={e => (e.currentTarget.style.color = 'var(--fg-muted)')}
        >
          Track this on GitHub →
        </a>
      </div>
    </div>
  );
}
