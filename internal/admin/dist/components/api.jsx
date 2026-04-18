// api.jsx — fetch wrapper with Bearer auth, loaded before page components

const API = {
  _key: () => sessionStorage.getItem('shark_admin_key'),

  async request(method, path, body) {
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
    const res = await fetch(`/api/v1${path}`, opts);
    if (res.status === 401) {
      sessionStorage.removeItem('shark_admin_key');
      window.location.reload();
      throw new Error('Session expired');
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({ message: `HTTP ${res.status}` }));
      throw new Error(err.message || err.error || `HTTP ${res.status}`);
    }
    if (res.status === 204) return null;
    return res.json();
  },

  get(path) { return this.request('GET', path); },
  post(path, body) { return this.request('POST', path, body); },
  patch(path, body) { return this.request('PATCH', path, body); },
  put(path, body) { return this.request('PUT', path, body); },
  del(path) { return this.request('DELETE', path); },
};

// Reusable data-fetching hook
function useAPI(path, deps) {
  const [data, setData] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);

  const refresh = React.useCallback(() => {
    if (!path) { setLoading(false); return; }
    setLoading(true);
    setError(null);
    API.get(path)
      .then(d => { setData(d); setError(null); })
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [path, ...(deps || [])]);

  React.useEffect(() => { refresh(); }, [refresh]);

  return { data, loading, error, refresh, setData };
}
