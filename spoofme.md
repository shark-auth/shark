# SharkAuth: Identity Infrastructure for Humans and Agents

SharkAuth is a self-hosted, high-density Identity and Access Management (IAM) platform designed to unify traditional user authentication with the emerging requirements of autonomous AI agents. Built in Go, it delivers a complete auth stack in a single, zero-dependency binary.

## The Core Proposition

Most auth providers treat "Machine-to-Machine" (M2M) as an afterthought or a high-priced enterprise add-on. SharkAuth elevates agent identity to a first-class citizen, providing the same level of security, rotation, and auditing to autonomous systems as it does to human users.

---

## Key Pillars

### 1. Human-Centric Authentication
Secure your applications with modern industry standards out of the box:
- **Passkey Support (WebAuthn):** Full support for biometric and hardware security keys.
- **Enterprise SSO:** Built-in SAML and OIDC connection management for B2B multi-tenancy.
- **Magic Links & MFA:** Passwordless email login and TOTP-based multi-factor authentication.
- **Organization Management:** Native support for multi-tenant structures with RBAC and member invitations.

### 2. Autonomous Agent Identity
Specifically tuned for M2M and AI-driven workloads:
- **Long-lived Token Management:** Secure issuance and automatic rotation for agent secrets.
- **OAuth 2.1 Authorization Server:** A complete authorization server implementation supporting RFC 6749 and RFC 7636 (PKCE).
- **Dynamic Client Registration (DCR):** Allow systems to programmatically register and manage their own credentials.

### 3. Integrated Security Proxy
SharkAuth includes a programmable reverse proxy that gates access to internal services:
- **Rule-based Authorization:** Gate paths and methods based on user roles, agent scopes, or tenant membership.
- **Low-Latency Gateway:** Evaluates rules in-process before forwarding requests to upstreams.
- **Circuit Breaking & Session Caching:** High-performance session validation at the edge.

### 4. The OAuth Vault
Securely store and proxy third-party OAuth tokens. Agents can retrieve or use user-delegated tokens (e.g., GitHub, Google) through SharkAuth without ever touching the raw credentials.

---

## Technical Architecture

SharkAuth is designed for operational simplicity and "Impeccable" performance:
- **Single Binary:** Everything—backend, migration scripts, and the React admin dashboard—is embedded into a single executable.
- **CGO-Free SQLite:** Uses a pure-Go SQLite implementation for easy cross-compilation and zero-config storage.
- **Developer First:** Managed via a powerful CLI, REST API, or the embedded Admin UI.
- **SDKs:** Official support for TypeScript and Python.

## Quick Start

```bash
# Build the single binary (requires Go 1.25+ and pnpm)
make build

# Initialize and start the server
./sharkauth init
./sharkauth serve --dev
```

## License

SharkAuth is Open Source Software released under the MIT License.
