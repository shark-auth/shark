# pyproject.toml

**Path:** `sdk/python/pyproject.toml`
**LOC:** 60

## Package metadata
- **name:** `shark-auth`
- **version:** `0.1.0`
- **description:** Python SDK for SharkAuth agent-auth primitives (DPoP, device flow, token vault, agent tokens).
- **license:** MIT (text-form)
- **readme:** `README.md`
- **requires-python:** `>=3.9`
- **authors:** SharkAuth
- **keywords:** oauth, oauth2, dpop, agent, auth, device-flow, jwt, rfc9449, rfc8628
- **classifiers:** Alpha, Python 3.9–3.12, Topic Security, `Typing :: Typed` (PEP 561)

## Build system
- **build-backend:** `hatchling.build`
- **requires:** `hatchling>=1.24`
- **wheel package:** `shark_auth/`
- **sdist include:** `/shark_auth`, `/tests`, `/README.md`, `/LICENSE`

## Runtime dependencies
- `cryptography>=42.0` — P-256 keygen + PEM IO for DPoP
- `PyJWT[crypto]>=2.8` — ES256 sign + RS/ES verify, `PyJWKClient`/`PyJWKSet`
- `requests>=2.31` — synchronous HTTP layer
- `typing_extensions>=4.0` — backport for `Literal`, `TypedDict` on 3.9

## Dev dependencies (`[project.optional-dependencies] dev`)
- `pytest>=7.4`
- `pytest-httpserver>=1.0`
- `responses>=0.24`
- `build>=1.0`

Install via `pip install -e ".[dev]"`.

## URLs
- Homepage / Repository: `https://github.com/shark-auth/shark`
- Issues: `https://github.com/shark-auth/shark/issues`

## PyPI publish status
**No publish automation today** (per audit). The package is buildable (`python -m build`) and the metadata is PyPI-ready, but there is no GitHub Action / CI workflow that uploads to PyPI on tag push, and no record of a published `0.1.0` release. Publishing today is a manual `twine upload` from a local checkout.

## Notes
- Single source of truth for version is this file (`__init__.py` re-declares `__version__ = "0.1.0"` manually — keep in sync on bumps).
- No `tool.ruff` / `tool.mypy` / `tool.black` config in this file — linting/typing config (if any) lives elsewhere or is project-default.
