import React from 'react'
import { tokens } from '../tokens'

export interface Tab {
  id: string
  label: string
  /** Optional badge count */
  badge?: number
  disabled?: boolean
}

export interface TabsProps {
  tabs: Tab[]
  activeTab: string
  onTabChange: (tabId: string) => void
  /** Renders content for the active tab panel */
  children?: React.ReactNode
  style?: React.CSSProperties
  /** Applied to the tab list container */
  listStyle?: React.CSSProperties
}

export function Tabs({ tabs, activeTab, onTabChange, children, style, listStyle }: TabsProps) {
  const tabListStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'flex-end',
    gap: 0,
    borderBottom: `1px solid ${tokens.color.hairline}`,
    ...listStyle,
  }

  const panelStyle: React.CSSProperties = {
    ...style,
  }

  function handleKeyDown(e: React.KeyboardEvent, currentId: string) {
    const enabledTabs = tabs.filter((t) => !t.disabled)
    const currentIndex = enabledTabs.findIndex((t) => t.id === currentId)

    if (e.key === 'ArrowRight') {
      e.preventDefault()
      const next = enabledTabs[(currentIndex + 1) % enabledTabs.length]
      onTabChange(next.id)
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault()
      const prev = enabledTabs[(currentIndex - 1 + enabledTabs.length) % enabledTabs.length]
      onTabChange(prev.id)
    } else if (e.key === 'Home') {
      e.preventDefault()
      onTabChange(enabledTabs[0].id)
    } else if (e.key === 'End') {
      e.preventDefault()
      onTabChange(enabledTabs[enabledTabs.length - 1].id)
    }
  }

  return (
    <div>
      <div role="tablist" style={tabListStyle} aria-label="Navigation tabs">
        {tabs.map((tab) => {
          const isActive = tab.id === activeTab

          const tabStyle: React.CSSProperties = {
            position: 'relative',
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            height: 36,
            padding: `0 ${tokens.space[3]}px`,
            background: 'transparent',
            border: 'none',
            borderBottom: isActive
              ? `2px solid ${tokens.color.primary}`
              : '2px solid transparent',
            borderRadius: 0,
            marginBottom: -1,
            color: isActive ? tokens.color.fg : tokens.color.fgMuted,
            fontSize: tokens.type.size.base,
            fontFamily: tokens.type.body.family,
            fontWeight: isActive ? tokens.type.weight.medium : tokens.type.weight.regular,
            cursor: tab.disabled ? 'not-allowed' : 'pointer',
            opacity: tab.disabled ? 0.4 : 1,
            outline: 'none',
            transition: `color ${tokens.motion.fast}, border-color ${tokens.motion.fast}`,
            whiteSpace: 'nowrap',
          }

          const badgeStyle: React.CSSProperties = {
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            minWidth: 18,
            height: 16,
            padding: '0 5px',
            background: isActive ? tokens.color.primary : tokens.color.surface3,
            color: isActive ? tokens.color.primaryFg : tokens.color.fgMuted,
            borderRadius: tokens.radius.sm,
            fontSize: tokens.type.size.xs,
            fontWeight: tokens.type.weight.medium,
            lineHeight: 1,
          }

          return (
            <button
              key={tab.id}
              role="tab"
              id={`tab-${tab.id}`}
              aria-selected={isActive}
              aria-controls={`panel-${tab.id}`}
              aria-disabled={tab.disabled || undefined}
              tabIndex={isActive ? 0 : -1}
              style={tabStyle}
              onClick={() => !tab.disabled && onTabChange(tab.id)}
              onKeyDown={(e) => handleKeyDown(e, tab.id)}
            >
              {tab.label}
              {tab.badge != null && (
                <span style={badgeStyle}>{tab.badge}</span>
              )}
            </button>
          )
        })}
      </div>
      {children && (
        <div
          role="tabpanel"
          id={`panel-${activeTab}`}
          aria-labelledby={`tab-${activeTab}`}
          style={panelStyle}
          tabIndex={0}
        >
          {children}
        </div>
      )}
    </div>
  )
}

Tabs.displayName = 'Tabs'
