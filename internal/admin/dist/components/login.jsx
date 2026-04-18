// Login screen — API key auth gate for SharkAuth admin

function Login({ onLogin }) {
  const [key, setKey] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState('');

  const submit = async () => {
    const trimmed = key.trim();
    if (!trimmed) return;
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/v1/admin/stats', {
        headers: { Authorization: 'Bearer ' + trimmed },
      });
      if (res.ok) {
        sessionStorage.setItem('shark_admin_key', trimmed);
        onLogin(trimmed);
      } else if (res.status === 401) {
        setError('Invalid API key.');
      } else {
        setError('Error ' + res.status + ': ' + res.statusText);
      }
    } catch (e) {
      setError(e.message || 'Network error.');
    } finally {
      setLoading(false);
    }
  };

  const onKeyDown = (e) => {
    if (e.key === 'Enter') submit();
  };

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100vh',
      background: 'var(--bg)',
    }}>
      <div style={{ width: 360, display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* Logo */}
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 8 }}>
          <SharkFullLogo width={160} />
        </div>

        {/* Input */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <input
            type="password"
            autoComplete="current-password"
            placeholder="sk_live_…"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            onKeyDown={onKeyDown}
            disabled={loading}
            style={{
              width: '100%',
              height: 36,
              padding: '0 10px',
              borderRadius: 5,
              border: '1px solid var(--hairline-strong)',
              background: 'var(--surface-1)',
              color: 'var(--fg)',
              fontFamily: 'var(--font-mono)',
              fontSize: 13,
              letterSpacing: '0.02em',
              outline: 'none',
              transition: 'border-color 80ms',
            }}
            onFocus={(e) => e.target.style.borderColor = 'var(--hairline-bright)'}
            onBlur={(e) => e.target.style.borderColor = 'var(--hairline-strong)'}
          />

          {/* Error */}
          {error && (
            <span style={{ color: 'var(--danger)', fontSize: 12, lineHeight: 1.4 }}>
              {error}
            </span>
          )}
        </div>

        {/* Submit */}
        <button
          className="btn primary"
          style={{ width: '100%', height: 34, justifyContent: 'center', fontSize: 13 }}
          onClick={submit}
          disabled={loading || !key.trim()}
        >
          {loading ? 'Signing in…' : 'Sign in'}
        </button>

        {/* Hint */}
        <p style={{
          margin: 0,
          fontSize: 11,
          color: 'var(--fg-dim)',
          textAlign: 'center',
          lineHeight: 1.5,
        }}>
          Paste your admin API key. Stored in session only — cleared on tab close.
        </p>
      </div>

      {/* Version */}
      <div style={{
        position: 'fixed',
        bottom: 16,
        fontSize: 10,
        color: 'var(--fg-faint)',
        fontFamily: 'var(--font-mono)',
      }}>
        SharkAuth v0.x
      </div>
    </div>
  );
}

Object.assign(window, { Login });
