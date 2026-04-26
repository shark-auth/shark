# DEFERRED — Mock Customer Platform (Demo Surface)

**Status:** deferred to W+1 (Wed-Fri post-launch)
**Logged:** 2026-04-26
**Why:** Monday launch ships the binary + dashboard + 5-layer model. The mock customer platform is the *narrative artifact* — a fake SaaS that makes the customer-integration story visible without buyers needing to integrate first.

---

## What it is

A small standalone Node/Express (or Bun + Hono) app at e.g. `examples/mock-saas/` that pretends to be a customer of SharkAuth. It demonstrates the full integration shape:

1. End-user signs up at `mock-saas.localhost:3000`
2. Mock SaaS calls SharkAuth admin API to **create a `created_by` user**
3. Mock SaaS spawns 1-3 agents on behalf of that user via SharkAuth (`POST /api/v1/agents` w/ `created_by`)
4. Mock SaaS issues DPoP-bound tokens for those agents via SharkAuth's OAuth endpoints
5. Agents perform actions visible to the end-user in the mock SaaS UI
6. End-user can see/revoke their own agents from a "My Agents" page in the mock SaaS (NOT in SharkAuth admin — that's the auth-manager surface)
7. SharkAuth admin dashboard shows the customer's agents from the *operator's* POV

## Why it matters

- **Closes the loop visually.** Buyers see "this is what shipping SharkAuth into a product looks like" without writing code.
- **Makes the moat tangible.** Layer 3 cascade-revoke demos best when there's a mock customer fleet to nuke.
- **YC video material.** 30-60s screencast of a customer's user signing up → agents auto-provisioned → revoked from mock SaaS UI → visible cascade in SharkAuth dashboard = pure differentiation footage.

## Scope when shipped

- ~300-500 LOC (single file express or hono app)
- Auth: simple email+password OR magic link via shark
- DB: same shark sqlite (so the mock is read/writing through shark APIs only — no parallel state)
- 3 buttons: "Create my agents", "List my agents", "Revoke all my agents"
- README + screencast hook

## Owner

Founder + 1 sonnet subagent dispatched Wed.

## Defer triggers

- If launch lands flat, ship by Wednesday — it's the conversion artifact.
- If launch lands hot (HN top 10), ship by Friday — supplements organic momentum.

## NOT this

- Not a real customer or competitor product. Pure demo scaffolding.
- Not something that ever ships to production.
- Not a SharkAuth feature — it's an *example consumer* of SharkAuth.
