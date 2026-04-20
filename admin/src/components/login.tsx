// @ts-nocheck
import React from 'react'
import { SharkFullLogo } from './shared'



export function Login({ onLogin }) {
  const [key, setKey] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState('');
  const [hintOpen, setHintOpen] = React.useState(false);
  // Bootstrap-consume attempted? Used to suppress the "Invalid token"
  // error turning into a re-try loop on re-renders.
  const bootstrapTried = React.useRef(false);

  // T15: if the URL carries ?bootstrap=<tok>, auto-POST to consume and log
  // the operator in without them ever seeing the key. On success we drop
  // the param, stash the minted key, and hand off to onLogin. On failure
  // we show the error inline and keep the password input visible so the
  // operator can fall back to pasting their key manually.
  React.useEffect(() => {
    if (bootstrapTried.current) return;
    const params = new URLSearchParams(window.location.search);
    const tok = params.get('bootstrap');
    if (!tok) return;
    bootstrapTried.current = true;
    setLoading(true);
    (async () => {
      try {
        const res = await fetch('/api/v1/admin/bootstrap/consume', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token: tok }),
        });
        // Strip the token from the URL regardless of outcome so it isn't
        // replayed on reload and doesn't leak into referrer headers.
        params.delete('bootstrap');
        const clean = window.location.pathname + (params.toString() ? '?' + params.toString() : '');
        window.history.replaceState(null, '', clean);
        if (res.ok) {
          const body = await res.json();
          const minted = body?.api_key;
          if (minted) {
            sessionStorage.setItem('shark_admin_key', minted);
            onLogin(minted);
            return;
          }
          setError('Bootstrap response missing api_key.');
        } else if (res.status === 401) {
          setError('Bootstrap link is invalid, expired, or already used. Paste your admin key instead.');
        } else {
          setError('Bootstrap failed: ' + res.status + ' ' + res.statusText);
        }
      } catch (e) {
        setError(e.message || 'Network error during bootstrap.');
      } finally {
        setLoading(false);
      }
    })();
  }, [onLogin]);

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

          {/* T15: always-visible "Where is my key?" expander. Teaches new
              operators the exact CLI to run on the host that's running
              `shark serve`, so a forgotten key never blocks login. */}
          <button
            type="button"
            onClick={() => setHintOpen(v => !v)}
            style={{
              alignSelf: 'flex-start',
              padding: 0,
              marginTop: 2,
              background: 'transparent',
              border: 'none',
              color: 'var(--fg-dim)',
              fontSize: 11,
              fontFamily: 'var(--font-sans)',
              cursor: 'pointer',
              textDecoration: 'underline',
              textUnderlineOffset: 2,
            }}
          >
            [?] Where is my key?
          </button>
          {hintOpen && (
            <div style={{
              marginTop: 4,
              padding: '8px 10px',
              borderRadius: 5,
              border: '1px solid var(--hairline)',
              background: 'var(--surface-1)',
              fontSize: 11,
              color: 'var(--fg-dim)',
              lineHeight: 1.5,
            }}>
              <div style={{ marginBottom: 4 }}>
                In the terminal running <code>shark serve</code>, run:
              </div>
              <code style={{
                display: 'block',
                padding: '6px 8px',
                borderRadius: 4,
                background: 'var(--bg)',
                color: 'var(--fg)',
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                userSelect: 'all',
              }}>
                shark admin-key show
              </code>
              <div style={{ marginTop: 6, color: 'var(--fg-faint)' }}>
                Or restart the server to get a one-time bootstrap URL.
              </div>
            </div>
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

