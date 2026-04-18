// @ts-nocheck
import React from 'react'
import ReactDOM from 'react-dom/client'
import { ToastProvider } from './components/toast'
import { App } from './components/App'
import './styles.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <ToastProvider><App /></ToastProvider>
)
