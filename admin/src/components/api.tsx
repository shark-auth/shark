// @ts-nocheck
import React from 'react'

export const API = {
  _key: () => sessionStorage.getItem('shark_admin_key'),

  async request(method, path, body, signal) {
    const key = this._key();
    if (!key) throw new Error('Not authenticated');
    const opts = {
      method,
      headers: {
        'Authorization': `Bearer ${key}`,
        'Content-Type': 'application/json',
      },
    };
    if (body && method !== 'GET') opts.body = JSON.stringify(body);
    if (signal) opts.signal = signal;
    const res = await fetch(`/api/v1${path}`, opts);
    if (res.status === 401) {
      if (path.startsWith('/admin/') || path.startsWith('/agents') || path.startsWith('/api-keys') || path.startsWith('/users') || path.startsWith('/roles') || path.startsWith('/audit-logs')) {
        sessionStorage.removeItem('shark_admin_key');
        window.dispatchEvent(new Event('shark-auth-expired'));
      }
      throw new Error('Unauthorized');
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({ message: `HTTP ${res.status}` }));
      throw new Error(err.message || err.error || `HTTP ${res.status}`);
    }
    if (res.status === 204) return null;
    return res.json();
  },

  get(path, signal) { return this.request('GET', path, null, signal); },
  post(path, body) { return this.request('POST', path, body); },
  patch(path, body) { return this.request('PATCH', path, body); },
  put(path, body) { return this.request('PUT', path, body); },
  del(path) { return this.request('DELETE', path); },

  async postFormData(path, form) {
    const key = this._key();
    if (!key) throw new Error('Not authenticated');
    const res = await fetch(`/api/v1${path}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${key}`,
      },
      body: form,
    });
    if (res.status === 401) {
      sessionStorage.removeItem('shark_admin_key');
      window.dispatchEvent(new Event('shark-auth-expired'));
      throw new Error('Unauthorized');
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({ message: `HTTP ${res.status}` }));
      throw new Error(err.message || err.error || `HTTP ${res.status}`);
    }
    return res.json();
  },
};

export function useAPI(path, deps) {
  const [data, setData] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);
  const abortRef = React.useRef(null);

  const refresh = React.useCallback((opts?: { silent?: boolean }) => {
    if (!path) { setLoading(false); return; }
    if (abortRef.current) abortRef.current.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    // silent=true (background poll) skips the loading flash when data is
    // already present — prevents the table from briefly clearing on every
    // 1.5 s tick and closes the race window where allEmails momentarily
    // reads [] while the next response is in-flight.
    if (!opts?.silent) {
      setLoading(true);
    }
    setError(null);
    API.get(path, controller.signal)
      .then(d => {
        if (controller.signal.aborted) return;
        setData(d);
        setError(null);
      })
      .catch(e => {
        if (e.name === 'AbortError' || controller.signal.aborted) return;
        setError(e.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
  }, [path, ...(deps || [])]);

  React.useEffect(() => {
    refresh();
    return () => { if (abortRef.current) abortRef.current.abort(); };
  }, [refresh]);

  return { data, loading, error, refresh, setData };
}
