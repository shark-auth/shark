# Acme Travel — the AI Concierge demo

A working illustration of what shark gives you: a single coherent travel-booking
flow that exercises every primitive a real agent product needs in production.
Not a contrived RFC walkthrough. Not a microbenchmark. A story you can hand to
a CTO and have them recognise their own roadmap inside it.

## The story

Maria Chen runs an AI travel concierge product. Her customers type things like
"book me a trip to Lisbon next month, hotel near the river, dinner reservation
on Friday, and please file the receipts." Her product replies "on it" — and
behind the curtain, a Concierge agent fans out to five specialists:

- a **Flight Booker** that calls Amadeus
- a **Hotel Booker** that calls Booking.com
- a **Calendar Sync** that talks to Google Calendar
- an **Expense Filer** that posts to Concur
- a **Payment Processor** that charges Stripe

Five customer connections. Five upstream tokens. Sub-agents calling sub-agents.
Step-up authentication when the dollar amount gets serious. Cascade revoke when
a customer churns. **This is what every AI agent product looks like under the
hood.** The question is not whether your product needs these primitives. The
question is whether you've built them yet, and whether they're correct.

The demo runs end-to-end in about three minutes against a local shark instance.
It is not a mock. It signs up a real user, mints real OAuth clients via DCR,
issues real DPoP-bound access tokens, performs real RFC 8693 token exchange to
build a depth-3 act chain, retrieves real vault credentials with the proof key
on every request, enrols real TOTP, and produces a real audit trail you can
inspect in the dashboard.

## What you just saw

**Identity.** Maria signs up with email + password. The dev-email tab catches
the magic link, the demo extracts the token, verification flips her account.
Login mints a session cookie. The whole human-onboarding round trip — email
verification, session establishment, password validation — is one HTTP round
each, no glue code on Maria's side.

**Agents as first-class identities.** Five specialists are created via the
admin agents endpoint, each with its own client credentials, each with its own
DPoP keypair generated client-side. The Concierge is registered through Dynamic
Client Registration (RFC 7591) so that the customer-facing "install this
integration" path looks exactly like a real OAuth app onboarding. None of these
agents share secrets with each other. None of them share secrets with Maria.
Compromise of one specialist does not compromise the others, and proves it by
construction, not by audit.

**Delegation that's actually a chain.** The Concierge calls the Flight Booker
via RFC 8693 token exchange — the resulting JWT has `act.sub = Concierge`,
visibly recording who delegated to whom. The Flight Booker then exchanges
*its* token to the Payment Processor, producing `act.act.sub` — a depth-3
chain where every hop is cryptographically attestable. When the Payment
Processor talks to Stripe at the bottom of the chain, the audit log can
truthfully say "Maria asked the Concierge to book a trip; the Concierge asked
the Flight Booker to handle the flight; the Flight Booker handed off the card
charge to the Payment Processor." That sentence becomes a structured query.

**Vault credentials with proof of possession.** Maria connected five upstream
SaaS accounts. Each connection is encrypted at rest by the field encryptor.
When the Flight Booker fetches the Amadeus token, it presents both its own
DPoP-bound access token and a fresh DPoP proof signed with its private key.
The retrieval is gated on `cnf.jkt` matching the prover's thumbprint. A stolen
access token is useless without the private key. A stolen private key is
useless without the access token. Both at once is the only way in.

**Parallelism for free.** Three of the specialists fetch their vault
credentials concurrently. The wall-clock total is dominated by the slowest
single retrieval, not the sum. Production AI agents are bottlenecked by these
fan-outs; shark adds zero serialisation overhead on top of what you'd write by
hand, and the audit log records the fan-out as one event tree, not three
unrelated events.

**Step-up authentication that actually steps up.** The Payment Processor
charges $850 — fine, under the policy threshold. It then tries to charge
$1500 and is rejected with `step_up_required`. Maria enrols a TOTP
authenticator (real RFC 6238: HMAC-SHA1, 30-second window, 6 digits — the
demo computes the codes inline so it has no `pyotp` dependency), confirms
the first code, then completes a real `/mfa/challenge` round trip. The
session is now elevated. The same $1500 charge succeeds, identical code
path, different session state. **The amount is not the gate. The proof of
human-in-the-loop is.**

**Surgical revocation.** The demo rotates the Flight Booker's DPoP key:
the old `jkt` is dead, in-flight proofs are rejected, the new keypair takes
over. It bulk-revokes every token whose `client_id` matches the Payment
Processor pattern — Flight Booker keeps working, Payment Processor stops
mid-step. It deletes the Stripe vault connection: Payment Processor loses
data access without affecting Amadeus, Booking, Google Calendar, or Concur.
Each of these is a single API call. Each writes one audit row. None of them
require a deploy.

**Cascade revoke for the churn case.** Finally, a single call cascade-revokes
Maria. Every agent she created is deactivated. Every OAuth consent she
granted is revoked. Every token they hold is invalidated. One row in the
audit log links the cascade to the admin who triggered it and the reason
they gave. This is the call your support team makes at 2am when a customer
emails to say their employee left.

## The moat

> **Your agents are already doing this. They're just not doing it safely.**

Sub-agents calling sub-agents. Connections to upstream SaaS. Delegated tokens
that need to record who-asked-whom. Step-up auth on high-value operations.
Cascade revoke when a customer churns. These are not optional features for
production agents. Every team building real agent products has them, or is
about to need them, or has shipped a broken version of them and doesn't yet
know.

**Without shark.** You spend three months on OAuth + DPoP + RFC 8693 + audit
infrastructure + cascade-revoke schema design. Token theft becomes a P0
incident before you've built the controls to prevent it. You discover that
"MFA on agent tokens" isn't a thing the day a customer asks why a chatbot
just charged their card $40,000. The retrofit is not three months. It's a
rewrite, because the audit log doesn't have the columns you need.

**With shark.** Your team writes the same product code they would have
written. They get DPoP binding, act-chain attestation, vault retrieval with
proof-of-possession, RFC-correct token exchange, real MFA, surgical and
cascade revocation, and a queryable audit log out of the box. They do not
get "detection". They get **cryptographic prevention** — a stolen token is
not a security incident if the attacker also needs the matching private key
they never had.

The right way. The secure way. **The secure AND fast way.**

## What shark gave you for free

- **DPoP token binding** → stolen tokens are useless on attacker hardware
- **RFC 8693 token exchange** → delegation chain is in the JWT, not the wiki
- **`act.act.sub` chain attestation** → "who asked whom" is a query, not a meeting
- **Field-encrypted vault** → upstream credentials never touch logs in cleartext
- **Per-agent DPoP keypairs** → compromise of one specialist contains the blast radius
- **Real TOTP enrolment + challenge** → step-up auth that actually steps up
- **Bulk revoke by `client_id` pattern** → kill one specialist, leave the rest alive
- **Vault disconnect** → revoke data access without revoking the agent
- **Cascade revoke per user** → one call for the customer-churn case
- **Append-only audit log** → every action above is queryable by actor, target, and time

## Try it

```bash
git clone https://github.com/sharkauth/shark
cd shark
make build
./shark.exe serve

# in another terminal
pip install -e sdk/python/
python tools/agent_demo_concierge.py --fast
```

Then open `http://localhost:8080/admin` and walk the Audit, Agents, Vault,
Sessions, and Users tabs. Every step the demo printed has a corresponding
row in the dashboard, with the actor, the target, the act-chain, the
delegated scopes, and the timing.

The demo also runs interactively without `--fast` — it pauses for ENTER
between each of the 23 steps, so you can talk through the narrative, point
at the dashboard, and let the room ask "wait, how does *that* work?"

If you Ctrl-C mid-demo, state is left intact for inspection — Maria's
account, her agents, her vault connections, her audit trail. The terminal
prints `[LEAVING STATE FOR INSPECTION]` and her `user_id` so you can pick
up where you stopped. On natural finish, step 23 cascade-revokes everything
the demo created, so the next run is clean.
