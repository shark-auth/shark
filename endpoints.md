# Shark Auth API Endpoints Documentation

## Overview

Shark Auth provides a robust Go API with endpoints mounted under the `/api/v1` prefix.
The authentication model primarily uses an **HTTP-only cookie** (`shark_session`) for frontend applications (Next.js) and can accept API keys (`Authorization: Bearer <token>`) for backend-to-backend communication.

**Base URL Template:** `https://your-railway-app-url.railway.app/api/v1`

---

## Core Authentication Endpoints

### 1. User Signup

- **Endpoint:** `POST /api/v1/auth/signup`
- **Description:** Registers a new user and sets the `shark_session` cookie.
- **Request Body (JSON):**
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword123"
  }
  ```

### 2. User Login

- **Endpoint:** `POST /api/v1/auth/login`
- **Description:** Authenticates a user and sets the `shark_session` cookie.
- **Request Body (JSON):**
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword123"
  }
  ```

### 3. Get Current User (`me`)

- **Endpoint:** `GET /api/v1/auth/me`
- **Description:** Fetches the authenticated user profile based on the current session cookie.
- **Headers Needed:** Cookie (`shark_session`) must be present in the request.

### 4. User Logout

- **Endpoint:** `POST /api/v1/auth/logout`
- **Description:** Clears the user's `shark_session` cookie.

---

## Integrations

### 1. Next.js Frontend Integration

Since SharkAuth relies on the `shark_session` cookie, your Next.js frontend calls must include `credentials: "include"` (or handle cookies server-side).

**Example: Client-Side Login Component**

```javascript
// Next.js (React Client Component)
export default function handleLogin(email, password) {
  fetch("https://<your-railway-url>/api/v1/auth/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    // CRITICAL: This ensures the browser saves the set-cookie returned by the server
    credentials: "include",
    body: JSON.stringify({ email, password }),
  })
    .then((res) => res.json())
    .then((data) => {
      console.log("Logged in:", data);
    })
    .catch((err) => console.error("Login failed:", err));
}
```

**Example: Server Component Data Fetching (Next.js `app/` router)**
If you are doing checks from Next.js server components, you need to manually forward the cookie to the SharkAuth backend.

```javascript
import { cookies } from "next/headers";

export async function getUserProfile() {
  const cookieStore = await cookies();
  const sessionCookie = cookieStore.get("shark_session");

  const response = await fetch("https://<your-railway-url>/api/v1/auth/me", {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      // Forward the exact cookie back to shark auth
      Cookie: `shark_session=${sessionCookie?.value}`,
    },
  });

  if (!response.ok) {
    return null;
  }
  return response.json();
}
```

---

### 2. Python Backend Integration

For a Python backend acting on behalf of a user, you should preserve cookies across requests using a `requests.Session()`.
Alternatively, if your Python backend connects to Shark Auth using an **Admin Key** or an **API Key**, you simply provide the Auth header.

**Scenario A: Replicating User Browser Steps (Using Sessions)**

```python
import requests

API_BASE = "https://<your-railway-url>/api/v1"

# Initialize a session that automatically manages cookies
session = requests.Session()

# 1. Login to get the cookie
login_res = session.post(f"{API_BASE}/auth/login", json={
    "email": "user@example.com",
    "password": "securepassword123"
})
print("Login status:", login_res.status_code)

# 2. Get User Profile (Cookie is sent automatically)
profile_res = session.get(f"{API_BASE}/auth/me")
print("Profile data:", profile_res.json())
```

**Scenario B: Accessing with an Admin / API Key**
For administrative endpoints (e.g. Roles, SSO Connections), use the admin API key generated on first `shark serve`.

```python
import requests

API_BASE = "https://<your-railway-url>/api/v1"
ADMIN_KEY = "sk_live_..."  # printed to stdout on first run

# Example: Get all roles
headers = {
    "Authorization": f"Bearer {ADMIN_KEY}",
    "Content-Type": "application/json"
}

roles_res = requests.get(f"{API_BASE}/roles", headers=headers)
print("Roles:", roles_res.json())
```

---

### Additional Core Features Included in the API

Shark Auth incorporates many other features out of the box requiring their respective configurations:

- **Passkeys (Fingerprint / FaceID):** endpoints at `/api/v1/auth/passkey/...`
- **MFA:** endpoints at `/api/v1/auth/mfa/...`
- **SSO & SAML:** endpoints at `/api/v1/sso/ connections` and callbacks
- **Magic Links:** endpoints at `/api/v1/auth/magic-link/...`
