# flow_builder.tsx

**Path:** `admin/src/components/flow_builder.tsx`
**Type:** React component (page)
**LOC:** ~600

## Purpose
Visual authentication flow builder—design custom auth journeys (conditional logic, policy rules, step sequences).

## Exports
- `FlowBuilder()` (default) — function component

## Features
- **Canvas** — drag-drop auth step nodes (login, MFA, profile completion, etc.)
- **Step types** — password, passkey, TOTP, SMS, email verification, risk assessment, custom form
- **Conditions** — rules for branching (if MFA required → show TOTP)
- **Policies** — risk-based routing, geo-blocking, device enrollment requirements
- **Export** — save flow as JSON
- **Test** — simulate flow with test user

## Composed by
- App.tsx

## Notes
- Allows enterprises to design complex auth journeys
- Policy engine evaluates conditions at runtime
- Flow published to hosted auth pages
- Advanced feature (Enterprise phase)
