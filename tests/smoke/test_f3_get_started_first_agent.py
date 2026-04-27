"""
F3 — Get-Started rewrite: drop install talk, focus first user/agent/delegation chain.

Static checks against get_started.tsx source (no live server needed).
Optional live tests skipped if shark is not reachable.
"""
import os
import pytest

GET_STARTED_TSX = os.path.abspath(
    os.path.join(os.path.dirname(__file__), '..', '..', 'admin', 'src', 'components', 'get_started.tsx')
)

# Try to import requests for optional live tests
try:
    import requests as _requests
    _HAS_REQUESTS = True
except ImportError:
    _HAS_REQUESTS = False

SHARK_BASE = os.environ.get('SHARK_BASE_URL', 'http://localhost:8080')
SHARK_ADMIN_KEY = os.environ.get('SHARK_ADMIN_KEY', '')


def _src() -> str:
    with open(GET_STARTED_TSX, 'r', encoding='utf-8') as f:
        return f.read()


# ── Banned phrases (must NOT appear) ─────────────────────────────────────────

BANNED_PHRASES = [
    'install.sh',
    'npm install @sharkauth',
    'pip install shark-auth',    # bare pip install without git+
    'shark init',
]


@pytest.mark.parametrize('phrase', BANNED_PHRASES)
def test_banned_phrase_absent(phrase):
    """Removed install / legacy phrases must not appear in get_started.tsx."""
    src = _src()
    # Special case: allow 'pip install git+...' but reject bare 'pip install shark-auth'
    if phrase == 'pip install shark-auth':
        # Allow git+ variant
        lines_with_phrase = [
            line for line in src.splitlines()
            if phrase in line and 'git+' not in line
        ]
        assert not lines_with_phrase, (
            f"Bare '{phrase}' (without git+) found in get_started.tsx:\n"
            + '\n'.join(lines_with_phrase)
        )
    else:
        assert phrase not in src, (
            f"Banned phrase '{phrase}' still present in get_started.tsx. "
            "Remove all install-talk — user is already running shark."
        )


# ── Required phrases (must appear) ───────────────────────────────────────────

REQUIRED_PHRASES = [
    'human user',
    'agent',
    'delegation chain',
    'Python SDK',
]


@pytest.mark.parametrize('phrase', REQUIRED_PHRASES)
def test_required_phrase_present(phrase):
    """Key copy phrases for each new section must appear in get_started.tsx."""
    src = _src()
    assert phrase.lower() in src.lower(), (
        f"Required phrase '{phrase}' not found in get_started.tsx. "
        "Each new section must contain its key copy."
    )


# ── Section structure ─────────────────────────────────────────────────────────

def test_user_create_task_present():
    """A task for creating the first human user must be present."""
    src = _src()
    assert 'user.create' in src or 'user create' in src.lower(), (
        "get_started.tsx must include a task for creating the first human user."
    )


def test_agent_register_task_present():
    """A task for registering the first agent must be present."""
    src = _src()
    assert 'agent.register' in src or 'agent register' in src.lower(), (
        "get_started.tsx must include a task for registering the first agent."
    )


def test_delegation_demo_task_present():
    """Delegation chain demo task must be present."""
    src = _src()
    assert 'delegation.run' in src or 'delegation-chain-demo' in src, (
        "get_started.tsx must include a delegation chain demo task."
    )


def test_python_sdk_dogfood_install_uses_git_plus():
    """Python SDK install must use git+ form, not bare PyPI."""
    src = _src()
    assert 'git+https://github.com/sharkauth/shark' in src, (
        "Python SDK install must use git+ form: "
        "pip install git+https://github.com/sharkauth/shark#subdirectory=sdk/python"
    )


def test_no_shark_init_reference():
    """shark init must not appear — yaml config is dead per W17."""
    src = _src()
    assert 'shark init' not in src, (
        "'shark init' must not appear in get_started.tsx — yaml config removed in W17."
    )


# ── Optional live tests ───────────────────────────────────────────────────────

def _shark_reachable() -> bool:
    if not _HAS_REQUESTS or not SHARK_ADMIN_KEY:
        return False
    try:
        r = _requests.get(f'{SHARK_BASE}/api/v1/admin/stats',
                          headers={'Authorization': f'Bearer {SHARK_ADMIN_KEY}'},
                          timeout=3)
        return r.status_code == 200
    except Exception:
        return False


@pytest.mark.skipif(not _HAS_REQUESTS, reason='requests not installed')
def test_users_endpoint_exists():
    """GET /api/v1/users (or /api/v1/admin/users) must return 200 or 401, not 404."""
    if not _shark_reachable():
        pytest.skip('shark not reachable or SHARK_ADMIN_KEY not set')
    r = _requests.get(f'{SHARK_BASE}/api/v1/admin/users?limit=1',
                      headers={'Authorization': f'Bearer {SHARK_ADMIN_KEY}'},
                      timeout=5)
    assert r.status_code != 404, (
        f"GET /api/v1/admin/users returned 404 — endpoint must exist for auto-checkoff."
    )


@pytest.mark.skipif(not _HAS_REQUESTS, reason='requests not installed')
def test_agents_endpoint_exists():
    """GET /api/v1/admin/agents must return 200 or 401, not 404."""
    if not _shark_reachable():
        pytest.skip('shark not reachable or SHARK_ADMIN_KEY not set')
    r = _requests.get(f'{SHARK_BASE}/api/v1/admin/agents?limit=1',
                      headers={'Authorization': f'Bearer {SHARK_ADMIN_KEY}'},
                      timeout=5)
    assert r.status_code != 404, (
        f"GET /api/v1/admin/agents returned 404 — endpoint must exist for auto-checkoff."
    )
