// @ts-nocheck
import React from 'react'

export const ToastContext = React.createContext();

export function ToastProvider({ children }) {
  const [toasts, setToasts] = React.useState([]);
  const timersRef = React.useRef({});

  const removeToast = (id) => {
    clearTimeout(timersRef.current[id]);
    clearInterval(timersRef.current['cd_' + id]);
    delete timersRef.current[id];
    delete timersRef.current['cd_' + id];
    setToasts(prev => prev.filter(t => t.id !== id));
  };

  const addToast = (type, message, duration = 5000) => {
    const id = Date.now() + Math.random();
    setToasts(prev => [{ id, type, message }, ...prev].slice(0, 5));
    timersRef.current[id] = setTimeout(() => removeToast(id), duration);
  };

  const addUndoToast = (message, onAction, duration = 5000) => {
    const id = Date.now() + Math.random();
    const expiresAt = Date.now() + duration;
    setToasts(prev => [{ id, type: 'undo', message, remaining: Math.ceil(duration / 1000) }, ...prev].slice(0, 5));

    // Countdown ticker
    timersRef.current['cd_' + id] = setInterval(() => {
      const left = Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000));
      setToasts(prev => prev.map(t => t.id === id ? { ...t, remaining: left } : t));
    }, 500);

    // Execute action after duration
    timersRef.current[id] = setTimeout(() => {
      removeToast(id);
      onAction();
    }, duration);

    return id; // caller can use to cancel
  };

  const cancelUndo = (id) => {
    removeToast(id);
  };

  const toast = {
    success: (msg) => addToast('success', msg),
    error:   (msg) => addToast('error',   msg, 8000),
    info:    (msg) => addToast('info',    msg),
    warn:    (msg) => addToast('warn',    msg),
    undo:    (msg, onAction) => addUndoToast(msg, onAction),
    cancel:  cancelUndo,
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
                        t.type === 'warn'    ? 'var(--warn-bg)'    :
                                               'var(--surface-3)',
            color: t.type === 'error'   ? 'var(--danger)'  :
                   t.type === 'success' ? 'var(--success)' :
                   t.type === 'warn'    ? 'var(--warn)'    :
                                          'var(--fg-muted)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 'var(--radius)',
            fontSize: 12,
            fontWeight: 500,
            pointerEvents: 'auto',
            animation: 'slideIn 160ms ease-out',
            maxWidth: 340,
            display: 'flex', alignItems: 'center', gap: 10,
          }}>
            <span style={{ flex: 1 }}>{t.message}</span>
            {t.type === 'undo' && (
              <button
                onClick={() => cancelUndo(t.id)}
                style={{
                  fontSize: 12, fontWeight: 600, color: 'var(--fg)',
                  cursor: 'pointer', whiteSpace: 'nowrap',
                  background: 'none', border: 'none', padding: 0,
                }}
              >
                Undo ({t.remaining}s)
              </button>
            )}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() { return React.useContext(ToastContext); }
