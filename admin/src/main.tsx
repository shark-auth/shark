// @ts-nocheck
import React from 'react'
import ReactDOM from 'react-dom/client'
import { ToastProvider } from './components/toast'
import { App } from './components/App'
import { ErrorBoundary } from './components/ErrorBoundary'
import './styles.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <ErrorBoundary>
    <ToastProvider><App /></ToastProvider>
  </ErrorBoundary>
)
