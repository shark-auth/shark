"""
F2 — Login hint default-open on first visit + recovery copy.

Static checks against login.tsx source (no live server needed).
"""
import os
import pytest

LOGIN_TSX = os.path.abspath(
    os.path.join(os.path.dirname(__file__), '..', '..', 'admin', 'src', 'components', 'login.tsx')
)


def _src() -> str:
    with open(LOGIN_TSX, 'r', encoding='utf-8') as f:
        return f.read()


def test_hint_default_open_uses_last_login_key():
    """hintOpen must default to true when shark.admin.lastLogin is absent."""
    src = _src()
    assert "shark.admin.lastLogin" in src, (
        "login.tsx must reference 'shark.admin.lastLogin' for first-visit detection."
    )
    # The initializer must check localStorage.getItem for that key
    assert "localStorage.getItem('shark.admin.lastLogin')" in src, (
        "hintOpen initializer must call localStorage.getItem('shark.admin.lastLogin')."
    )


def test_hint_text_references_firstboot_file():
    """Hint text must mention data/admin.key.firstboot."""
    src = _src()
    assert "data/admin.key.firstboot" in src, (
        "Hint text must reference 'data/admin.key.firstboot' so users know where to find their key."
    )


def test_hint_text_references_shark_serve():
    """Hint text must direct users to the terminal where shark serve runs."""
    src = _src()
    assert "shark serve" in src, (
        "Hint text must mention 'shark serve' to guide users to the right terminal."
    )


def test_error_message_references_firstboot_file():
    """401 error message must mention data/admin.key.firstboot."""
    src = _src()
    assert "data/admin.key.firstboot" in src, (
        "Error message on invalid key must reference 'data/admin.key.firstboot'."
    )
    # Specifically the error message variant
    assert "restart shark serve fresh" in src, (
        "Error message must include 'restart shark serve fresh' recovery hint."
    )


def test_successful_login_sets_last_login():
    """On successful login, localStorage.setItem('shark.admin.lastLogin', ...) must be called."""
    src = _src()
    assert "localStorage.setItem('shark.admin.lastLogin'" in src, (
        "After successful login, must call localStorage.setItem('shark.admin.lastLogin', ...) "
        "so the hint stays collapsed on return visits."
    )


def test_no_nonexistent_commands_in_hint():
    """Hint must NOT reference shark admin-key show or shark admin-key regenerate."""
    src = _src()
    assert "shark admin-key show" not in src, (
        "Hint must not reference 'shark admin-key show' — that command does not exist."
    )
    assert "shark admin-key regenerate" not in src, (
        "Hint must not reference 'shark admin-key regenerate' — that command does not exist."
    )
