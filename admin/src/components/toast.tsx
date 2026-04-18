// @ts-nocheck
import React from 'react'

export const ToastContext = React.createContext();

export function ToastProvider({ children }) {
  const [toasts, setToasts] = React.useState([]);

  const addToast = (type, message, duration = 5000) => {
    const id = Date.now() + Math.random();
    setToasts(prev => [{ id, type, message }, ...prev].slice(0, 5));
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== id));
    }, duration);
  };

  const toast = {
    success: (msg) => addToast('success', msg),
    error:   (msg) => addToast('error',   msg),
    info:    (msg) => addToast('info',    msg),
  };

  return (
    <ToastContext.Provider value={toast}>
      {children}
      <div style={{
        position: 'fixed', bottom: 16, right: 16, zIndex: 100,
        display: 'flex', flexDirection: 'column', gap: 6,
        pointerEvents: 'none',
      }}>
        {toasts.slice(0, 3).map(t => (
          <div key={t.id} style={{
            padding: '8px 14px',
            background: t.type === 'error'   ? 'var(--danger-bg)'  :
                        t.type === 'success' ? 'var(--success-bg)' :
                                               'var(--surface-3)',
            color: t.type === 'error'   ? 'var(--danger)'  :
                   t.type === 'success' ? 'var(--success)' :
                                          'var(--fg-muted)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 'var(--radius)',
            fontSize: 12,
            fontWeight: 500,
            pointerEvents: 'auto',
            animation: 'slideIn 160ms ease-out',
            maxWidth: 340,
          }}>
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() { return React.useContext(ToastContext); }
