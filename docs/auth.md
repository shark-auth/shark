# SharkAuth API Documentation

Welcome to the documentation for SharkAuth. This system provides a robust, single-binary authentication service with support for traditional password flows, magic links, stateless sessions (cookies), multi-factor authentication (MFA), and Passkeys.

## Base URL
All API requests must be prefixed with:
`http://localhost:<PORT>/api/v1` (by default `http://localhost:8080/api/v1`).

Authentication state is maintained securely using browser cookies (`shark_session`). For cross-origin requests from frontends (like Next.js or Vite), ensure you include `credentials: 'include'` in your `fetch()` calls.

---

## 1. Traditional Auth Flow (Username/Password)

### Sign Up
Create a new user using an email and password.

- **Endpoint:** `POST /auth/signup`
- **Body Requirement:** JSON
```json
{
  "email": "user@example.com",
  "password": "mySecurePassword123",
  "name": "Jane Doe" // optional
}
```
- **Responses:**
  - `201 Created`: User created and session cookie set. Returns user object.
  - `400 Bad Request`: Invalid email or weak password.
  - `409 Conflict`: Email already exists.

### Log In
Authenticate an existing user. Requires a previously created account. 

- **Endpoint:** `POST /auth/login`
- **Body Requirement:** JSON
```json
{
  "email": "user@example.com",
  "password": "mySecurePassword123"
}
```
- **Responses:**
  - `200 OK`: Valid credentials, `shark_session` cookie set.
  - `401 Unauthorized`: Invalid credentials.

### Log Out
Revoke the current session and clear browser cookies.

- **Endpoint:** `POST /auth/logout`
- **Responses:**
  - `200 OK`: Successful logout.

### Get Current User
Fetch details about the currently authenticated user. Requires a valid session cookie.

- **Endpoint:** `GET /auth/me`
- **Responses:**
  - `200 OK`: Returns current user JSON (ID, Email, Verification state, MFA state).
  - `401 Unauthorized`: Session expired or invalid.

---

## 2. Magic Link Flow (Passwordless)

Users can sign in or create accounts without passwords by using an email verification logic. 

### Send Magic Link
Generates a secure verification link and emails it to the user.

- **Endpoint:** `POST /auth/magic-link/send`
- **Body Requirement:** JSON
```json
{
  "email": "user@example.com"
}
```
- **Behavior:** This endpoint is rate-limited (1 per 60 seconds per email). To prevent account enumeration, it always returns a `200 OK` regardless of whether the email exists in the database.

### Verify Magic Link
Triggered when the user clicks the URL in their email. 

- **Endpoint:** `GET /auth/magic-link/verify?token=<RAW_TOKEN>`
- **Behavior:**
  - Validates the token against the database.
  - If the user does not exist, registers them automatically and marks their email as verified.
  - Generates a secure session and issues a browser cookie.
  - Redirects the user back to the web application defined in `sharkauth.yaml` under `magic_link.redirect_url`.

---

## 3. Configuration & Startup

SharkAuth configurations are defined in `sharkauth.yaml` at the root directory. To run the service, simply execute:
```bash
go run cmd/shark/main.go --config sharkauth.yaml
```

**Key Configurations:**
```yaml
server:
  port: 8080
  secret: "replace-with-super-long-secure-random-string"

magic_link:
  token_lifetime: "10m"
  redirect_url: "http://localhost:3000/dashboard" # Where to send the user after clicking the email link

smtp:
  host: "smtp.resend.com" # Ex: Resend, Postmark
  port: 465
  username: "resend"
  password: "re_..." 
  from: "auth@yourdomain.com"
  from_name: "SharkAuth"
```

## Security Overview

- **Storage Details:** Data is safely stored in a local SQLite file (configured via `storage.path`).
- **Hashing Rules:** Rather than raw passwords, SharkAuth stores Argon2id cryptographic hashes. It's built to resist brute force and dictionary attacks natively.
- **Cookies:** Cookie management is handled entirely by gorilla/securecookie, which digitally signs and encrypts session IDs preventing man-in-the-middle reads or tampering.
