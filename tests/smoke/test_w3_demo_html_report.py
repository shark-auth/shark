"""
Wave 3 smoke — HTML report structure validation.

Verifies the generated report contains required structural elements
and is self-contained (no external CDN references).
"""
import os
import subprocess
import tempfile

import pytest


SHARK_EXE = os.path.join(os.path.dirname(__file__), "shark.exe")
BASE_URL = os.environ.get("SHARK_BASE_URL", "")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")


@pytest.mark.skipif(
    not BASE_URL or not ADMIN_KEY,
    reason="SHARK_BASE_URL and SHARK_ADMIN_KEY required for demo smoke tests",
)
def test_demo_html_report_custom_output():
    """--output flag writes report to the specified path."""
    with tempfile.NamedTemporaryFile(suffix=".html", delete=False) as f:
        out_path = f.name

    try:
        result = subprocess.run(
            [
                SHARK_EXE,
                "demo",
                "delegation-with-trace",
                "--base-url", BASE_URL,
                "--admin-key", ADMIN_KEY,
                "--output", out_path,
                "--no-open",
            ],
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )

        assert result.returncode == 0, (
            f"demo exited {result.returncode}\n"
            f"stdout: {result.stdout}\n"
            f"stderr: {result.stderr}"
        )

        assert os.path.exists(out_path), f"Report not written to {out_path}"

        content = open(out_path, encoding="utf-8").read()

        # Must have a proper HTML title
        assert "<title>" in content, "Missing <title> in report"
        assert "SharkAuth" in content, "Missing 'SharkAuth' in report title area"
        assert "Delegation Trace" in content, "Missing 'Delegation Trace' in report"

        # Self-contained: no CDN links
        assert "cdn.jsdelivr.net" not in content, "Report must not reference CDN"
        assert "unpkg.com" not in content, "Report must not reference CDN"
        assert "googleapis.com" not in content, "Report must not reference CDN"

        # Style and script inlined
        assert "<style>" in content, "Missing inline <style>"
        assert "<script>" in content, "Missing inline <script>"

        # Token section present
        assert "Token" in content, "Missing token section"

        # Audit section present
        assert "Audit" in content, "Missing audit section"

    finally:
        try:
            os.remove(out_path)
        except OSError:
            pass


@pytest.mark.skipif(
    not BASE_URL or not ADMIN_KEY,
    reason="SHARK_BASE_URL and SHARK_ADMIN_KEY required for demo smoke tests",
)
def test_demo_html_report_default_output():
    """Without --output, report lands at ./demo-report.html."""
    report = "./demo-report.html"

    # Remove any stale file
    try:
        os.remove(report)
    except OSError:
        pass

    result = subprocess.run(
        [
            SHARK_EXE,
            "demo",
            "delegation-with-trace",
            "--base-url", BASE_URL,
            "--admin-key", ADMIN_KEY,
            "--no-open",
        ],
        capture_output=True,
        text=True,
        timeout=60,
        check=False,
    )

    assert result.returncode == 0
    assert os.path.exists(report), f"Default report not created at {report}"

    content = open(report, encoding="utf-8").read()
    assert "SharkAuth" in content
    assert "Delegation Trace" in content

    try:
        os.remove(report)
    except OSError:
        pass
