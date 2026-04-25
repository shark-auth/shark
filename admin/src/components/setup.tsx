// @ts-nocheck
import React from 'react'

// Setup page — first-boot admin bootstrapping flow.
// Mounted at /admin/setup?token=<setup-token>.
// This page is intentionally standalone: no Sidebar, no TopBar, no auth session.
//
// Steps:
//   1. Show the one-time admin API key (fetched from GET /api/v1/admin/setup/info).
//   2. Form to enter the admin email address.
//   3. Confirmation that a magic-link was sent.

// ---------------------------------------------------------------------------
// Styles (inline — no Tailwind dependency, matches monochrome/square design)
// ---------------------------------------------------------------------------

const S = {
  page: {
    minHeight: '100vh',
    background: 'var(--bg, #fafafa)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '32px 16px',
    fontFamily: 'var(--font, system-ui, sans-serif)',
  } as React.CSSProperties,
  card: {
    background: 'var(--surface, #fff)',
    border: '1px solid var(--border, #e5e5e5)',
    borderRadius: 4,
    padding: '32px 28px',
    width: '100%',
    maxWidth: 480,
    boxShadow: '0 1px 3px rgba(0,0,0,.06)',
  } as React.CSSProperties,
  logo: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    marginBottom: 24,
  } as React.CSSProperties,
  logoText: {
    fontSize: 18,
    fontWeight: 700,
    letterSpacing: '-0.02em',
    color: 'var(--fg, #111)',
  } as React.CSSProperties,
  h1: {
    fontSize: 20,
    fontWeight: 700,
    color: 'var(--fg, #111)',
    margin: '0 0 6px',
  } as React.CSSProperties,
  subtitle: {
    fontSize: 13,
    color: 'var(--muted, #666)',
    margin: '0 0 24px',
  } as React.CSSProperties,
  label: {
    display: 'block',
    fontSize: 12,
    fontWeight: 600,
    color: 'var(--fg, #111)',
    marginBottom: 6,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.04em',
  } as React.CSSProperties,
  keyBox: {
    background: 'var(--code-bg, #f4f4f4)',
    border: '1px solid var(--border, #e5e5e5)',
    borderRadius: 4,
    padding: '12px 14px',
    fontFamily: 'monospace',
    fontSize: 13,
    wordBreak: 'break-all' as const,
    color: 'var(--fg, #111)',
    lineHeight: 1.5,
    marginBottom: 6,
  } as React.CSSProperties,
  warning: {
    fontSize: 12,
    color: 'var(--muted, #666)',
    background: 'var(--warn-bg, #fff8e1)',
    border: '1px solid var(--warn-border, #f0c040)',
    borderRadius: 4,
    padding: '8px 12px',
    marginBottom: 24,
  } as React.CSSProperties,
  copyBtn: {
    fontSize: 12,
    padding: '4px 10px',
    background: 'none',
    border: '1px solid var(--border, #e5e5e5)',
    borderRadius: 4,
    cursor: 'pointer',
    color: 'var(--fg, #111)',
    marginBottom: 8,
    fontFamily: 'inherit',
  } as React.CSSProperties,
  input: {
    width: '100%',
    padding: '8px 10px',
    fontSize: 13,
    border: '1px solid var(--border, #e5e5e5)',
    borderRadius: 4,
    background: 'var(--bg, #fff)',
    color: 'var(--fg, #111)',
    outline: 'none',
    boxSizing: 'border-box' as const,
    fontFamily: 'inherit',
  } as React.CSSProperties,
  btn: {
    width: '100%',
    padding: '9px 16px',
    fontSize: 13,
    fontWeight: 600,
    background: 'var(--fg, #111)',
    color: 'var(--bg, #fff)',
    border: '1px solid transparent',
    borderRadius: 4,
    cursor: 'pointer',
    marginTop: 14,
    fontFamily: 'inherit',
  } as React.CSSProperties,
  btnDisabled: {
    opacity: 0.5,
    cursor: 'not-allowed',
  } as React.CSSProperties,
  error: {
    fontSize: 12,
    color: '#c0392b',
    marginTop: 8,
  } as React.CSSProperties,
  stepDone: {
    textAlign: 'center' as const,
    padding: '8px 0 4px',
  } as React.CSSProperties,
  checkmark: {
    fontSize: 36,
    marginBottom: 12,
    display: 'block',
  } as React.CSSProperties,
  doneTitle: {
    fontSize: 18,
    fontWeight: 700,
    color: 'var(--fg, #111)',
    margin: '0 0 8px',
  } as React.CSSProperties,
  doneText: {
    fontSize: 13,
    color: 'var(--muted, #666)',
    lineHeight: 1.6,
  } as React.CSSProperties,
  divider: {
    borderTop: '1px solid var(--border, #e5e5e5)',
    margin: '20px 0',
  } as React.CSSProperties,
  stepIndicator: {
    display: 'flex',
    gap: 6,
    marginBottom: 20,
    alignItems: 'center',
  } as React.CSSProperties,
  stepDot: (active: boolean, done: boolean) => ({
    width: 6,
    height: 6,
    borderRadius: '50%',
    background: done ? 'var(--fg, #111)' : active ? 'var(--fg, #111)' : 'var(--border, #e5e5e5)',
    opacity: active ? 1 : done ? 0.5 : 1,
  } as React.CSSProperties),
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function Setup() {
  // Read ?token= from URL query param
  const token = React.useMemo(() => {
    const sp = new URLSearchParams(window.location.search);
    return sp.get('token') || '';
  }, []);

  // step: 'loading' | 'key' | 'email' | 'done' | 'error'
  const [step, setStep] = React.useState<'loading' | 'key' | 'email' | 'done' | 'error'>(
    token ? 'loading' : 'error'
  );
  const [adminKey, setAdminKey] = React.useState('');
  const [fetchError, setFetchError] = React.useState('');
  const [email, setEmail] = React.useState('');
  const [submitting, setSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState('');
  const [copied, setCopied] = React.useState(false);
  const [devInboxURL, setDevInboxURL] = React.useState('');

  // On mount: fetch the one-time admin key from the setup/info endpoint.
  React.useEffect(() => {
    if (!token) return;
    fetch('/api/v1/admin/setup/info', {
      headers: { Authorization: 'Setup ' + token },
    })
      .then(async (r) => {
        if (r.status === 410) {
          setFetchError('This setup link has already been used or has expired.');
          setStep('error');
          return;
        }
        if (!r.ok) {
          const body = await r.json().catch(() => ({}));
          setFetchError(body?.error_description || body?.message || 'Failed to load setup info.');
          setStep('error');
          return;
        }
        const d = await r.json();
        setAdminKey(d.api_key || '');
        setStep('key');
      })
      .catch((err) => {
        setFetchError('Network error: ' + String(err));
        setStep('error');
      });
  }, [token]);

  const handleCopy = () => {
    if (!adminKey) return;
    navigator.clipboard.writeText(adminKey).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }).catch(() => {
      // Fallback: select the text
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!email.trim() || submitting) return;
    setSubmitting(true);
    setSubmitError('');
    try {
      const r = await fetch('/api/v1/admin/setup/admin-user', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: 'Setup ' + token,
        },
        body: JSON.stringify({ email: email.trim().toLowerCase() }),
      });
      const data = await r.json().catch(() => ({}));
      if (r.status === 410) {
        setSubmitError('Setup has already been completed. Refresh and log in normally.');
        setSubmitting(false);
        return;
      }
      if (!r.ok) {
        setSubmitError(data?.error_description || data?.message || 'Failed to create admin user.');
        setSubmitting(false);
        return;
      }
      if (data.dev_inbox_url) setDevInboxURL(data.dev_inbox_url);
      setStep('done');
    } catch (err) {
      setSubmitError('Network error: ' + String(err));
      setSubmitting(false);
    }
  };

  // Loading spinner
  if (step === 'loading') {
    return (
      <div style={S.page}>
        <div style={S.card}>
          <Logo />
          <p style={{ color: 'var(--muted, #666)', fontSize: 13 }}>Loading setup…</p>
        </div>
      </div>
    );
  }

  // Error state
  if (step === 'error') {
    return (
      <div style={S.page}>
        <div style={S.card}>
          <Logo />
          <h1 style={S.h1}>Setup unavailable</h1>
          <p style={{ ...S.doneText, marginTop: 8 }}>
            {fetchError || 'No setup token found. Start the server to generate a new setup link.'}
          </p>
        </div>
      </div>
    );
  }

  // Done state
  if (step === 'done') {
    return (
      <div style={S.page}>
        <div style={S.card}>
          <Logo />
          <div style={S.stepDone}>
            <span style={S.checkmark}>✓</span>
            <h1 style={S.doneTitle}>Magic link sent</h1>
            <p style={S.doneText}>
              Check your email at <strong>{email}</strong> for the sign-in link.
            </p>
            {devInboxURL && (
              <p style={{ ...S.doneText, marginTop: 12 }}>
                Running in dev mode?{' '}
                <a href={devInboxURL} style={{ color: 'var(--fg, #111)', fontWeight: 600 }}>
                  View in dev inbox
                </a>
              </p>
            )}
            <p style={{ ...S.doneText, marginTop: 16 }}>
              Once you sign in, your admin API key above is the only way to access the API
              programmatically — make sure you've saved it.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Step 1: show key (step === 'key')
  // Step 2: enter email (step === 'email')
  return (
    <div style={S.page}>
      <div style={S.card}>
        <Logo />

        {/* Step indicators */}
        <div style={S.stepIndicator}>
          <div style={S.stepDot(step === 'key', step === 'email')} />
          <div style={S.stepDot(step === 'email', false)} />
        </div>

        {step === 'key' && (
          <>
            <h1 style={S.h1}>Welcome to SharkAuth</h1>
            <p style={S.subtitle}>Your admin API key has been generated. Copy it now — you won't see it again.</p>

            <label style={S.label}>Admin API Key</label>
            <div style={S.keyBox}>{adminKey}</div>
            <button
              style={S.copyBtn}
              onClick={handleCopy}
              type="button"
            >
              {copied ? 'Copied!' : 'Copy key'}
            </button>

            <div style={S.warning}>
              <strong>Save this key.</strong> It is shown only once and cannot be recovered.
              Store it in a secrets manager or password vault before continuing.
            </div>

            <button
              style={S.btn}
              type="button"
              onClick={() => setStep('email')}
            >
              I've saved it — continue
            </button>
          </>
        )}

        {step === 'email' && (
          <>
            <h1 style={S.h1}>Create admin account</h1>
            <p style={S.subtitle}>Enter your email. We'll send a magic link to sign in.</p>

            <form onSubmit={handleSubmit}>
              <label style={S.label} htmlFor="setup-email">Admin email</label>
              <input
                id="setup-email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                style={S.input}
                autoFocus
                required
              />
              {submitError && <p style={S.error}>{submitError}</p>}
              <button
                type="submit"
                style={{ ...S.btn, ...(submitting ? S.btnDisabled : {}) }}
                disabled={submitting}
              >
                {submitting ? 'Sending…' : 'Send magic link'}
              </button>
            </form>

            <div style={S.divider} />
            <button
              style={{ ...S.copyBtn, fontSize: 12 }}
              type="button"
              onClick={() => setStep('key')}
            >
              ← Back
            </button>
          </>
        )}
      </div>
    </div>
  );
}

function Logo() {
  return (
    <div style={S.logo}>
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2L2 7l10 5 10-5-10-5z"/>
        <path d="M2 17l10 5 10-5"/>
        <path d="M2 12l10 5 10-5"/>
      </svg>
      <span style={S.logoText}>SharkAuth</span>
    </div>
  );
}
