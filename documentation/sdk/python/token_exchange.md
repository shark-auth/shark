# `OAuthClient.token_exchange()` — RFC 8693 Token Exchange

Cross-reference: [get_token_with_dpop](./get_token_with_dpop.md)

---

## Overview

`token_exchange()` implements [RFC 8693](https://www.rfc-editor.org/rfc/rfc8693) OAuth 2.0 Token Exchange. It POSTs to the same `/oauth/token` endpoint as `get_token_with_dpop()`, but with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`.

Use this for **delegation chains**: agent A receives a wide-scope token, narrows it to a sub-scope, and hands the child token to agent B. The child token carries an `act`-claim chain that proves the original human's authorization. Both tokens share the same DPoP keypair (`cnf.jkt` is unchanged), so the holder still needs the private key.

---

## Method Signature

```python
def token_exchange(
    self,
    *,
    subject_token: str,
    dpop_prover: DPoPProver,
    scope: str | None = None,
    audience: str | None = None,
    actor_token: str | None = None,
    subject_token_type: str = "urn:ietf:params:oauth:token-type:access_token",
    requested_token_type: str = "urn:ietf:params:oauth:token-type:access_token",
    **extra: Any,
) -> Token:
```

---

## Parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `subject_token` | `str` | required | The existing access token to exchange (e.g. `token.access_token`). |
| `dpop_prover` | `DPoPProver` | required | Same prover used for the original token. Key binding is preserved — `cnf.jkt` will match in the child token. |
| `scope` | `str \| None` | `None` | Narrower space-separated scopes to downscope to (e.g. `"mcp:read"`). Omitting leaves scope up to the server. |
| `audience` | `str \| None` | `None` | Restrict the new token to a specific resource server URL. |
| `actor_token` | `str \| None` | `None` | Optional `act`-claim parent token (the agent performing the delegation). When provided, `actor_token_type` is automatically added. |
| `subject_token_type` | `str` | `"urn:ietf:params:oauth:token-type:access_token"` | RFC 8693 type URI for `subject_token`. |
| `requested_token_type` | `str` | `"urn:ietf:params:oauth:token-type:access_token"` | RFC 8693 type URI for the token to be issued. |
| `**extra` | `Any` | — | Any additional form fields forwarded to the token endpoint. |

---

## Returns

A `Token` dataclass with:

| Field | Notes |
|---|---|
| `access_token` | New child access token string. |
| `token_type` | Usually `"DPoP"`. |
| `expires_in` | Lifetime in seconds, or `None`. |
| `scope` | Granted scope (may be narrower than `subject_token`'s scope). |
| `cnf_jkt` | **Unchanged from parent** — same keypair is bound. Token theft alone is still useless. |
| `raw` | Full server JSON response. |

---

## Raises

| Error | Status | Cause | Fix |
|---|---|---|---|
| `OAuthError(error="invalid_token")` | 401 | `subject_token` is expired or revoked. | Obtain a fresh parent token via `get_token_with_dpop()` before exchanging. |
| `OAuthError(error="invalid_scope")` | 400 | Requested scope exceeds the parent token's grant. | Use a scope that is a strict subset of the parent's granted scope. |
| `OAuthError(error="invalid_request")` | 400 | Missing required field or malformed body. | Check that `subject_token` and `dpop_prover` are non-empty and valid. |

---

## Example — Full Delegation Chain (10-liner)

```python
from shark_auth import OAuthClient
from shark_auth.dpop import DPoPProver

prover = DPoPProver.generate()
client = OAuthClient(base_url="https://auth.example.com")

# Step 1: Agent A gets a wide-scope parent token.
parent = client.get_token_with_dpop(
    grant_type="client_credentials",
    dpop_prover=prover,
    client_id="agent-a",
    client_secret="s3cr3t",
    scope="mcp:read mcp:write",
)

# Step 2: Agent A narrows it before handing off to Agent B.
child = client.token_exchange(
    subject_token=parent.access_token,
    dpop_prover=prover,          # same key — cnf.jkt preserved
    scope="mcp:read",            # downscoped
    audience="https://mcp.example.com",
)

assert child.cnf_jkt == parent.cnf_jkt   # True — same keypair
print(child.scope)                        # "mcp:read"
# Agent B uses child.access_token with the same prover.
```

### With Actor Token (explicit delegation chain)

```python
child = client.token_exchange(
    subject_token=parent.access_token,
    dpop_prover=prover,
    scope="mcp:read",
    actor_token=orchestrator_token.access_token,  # proves who delegated
)
```

---

## Implementation Notes

- POSTs to `/oauth/token` — the same endpoint as `get_token_with_dpop()`. Both methods share the private `_post_token_request(form_body, dpop_proof)` helper.
- DPoP proof is generated fresh per call (`dpop_prover.make_proof(htm="POST", htu=token_endpoint)`).
- `subject_token` is sent as a form field only (no `Authorization: Bearer` header) — form-only is the supported v0.1 mode for the shark backend.
- `actor_token_type` is automatically set to `"urn:ietf:params:oauth:token-type:access_token"` whenever `actor_token` is provided.

---

## See Also

- [get_token_with_dpop](./get_token_with_dpop.md) — Method 1, obtain the parent token.
- [RFC 8693](https://www.rfc-editor.org/rfc/rfc8693) — OAuth 2.0 Token Exchange specification.
- [RFC 9449](https://www.rfc-editor.org/rfc/rfc9449) — DPoP (Demonstrating Proof of Possession).
