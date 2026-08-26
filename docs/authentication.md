# Authentication

Two modes. Both have to work on a machine with no persistent storage, which
rules out most of the usual approaches.

## Microsoft device code flow

There is no browser in the image and no reason to add one. The OAuth device
authorization grant exists exactly for this situation, and it is the same flow a
smart TV uses.

```
   PhantomMC                    Microsoft                      your phone
       │                            │                              │
       │  request a device code     │                              │
       ├───────────────────────────▶│                              │
       │                            │                              │
       │  user_code, verification   │                              │
       │◀───────────────────────────┤                              │
       │                            │                              │
   ┌───┴──────────────────────┐     │                              │
   │  microsoft.com/link      │     │      user opens the link     │
   │  code: F8B2-X9AQ         │─────┼─────────────────────────────▶│
   └───┬──────────────────────┘     │                              │
       │                            │        signs in, approves    │
       │   poll for the token       │◀─────────────────────────────┤
       ├───────────────────────────▶│                              │
       │                            │                              │
       │   access token             │                              │
       │◀───────────────────────────┤                              │
       │                                                           │
       ├── Xbox Live user token ── XSTS token ── Minecraft token ──▶ launch
```

The chain after the Microsoft token is four hops: Xbox Live authentication, XSTS
authorization, exchange for a Minecraft token, then a profile lookup to obtain
the username and UUID. Each has its own failure mode and its own unhelpful error
message, which is why this is a milestone of its own rather than an afternoon.

Because the machine has no persistence, the token dies with the power. Signing
in happens on every boot. It takes about ten seconds on a phone and it is the
price of the guarantee that nothing is left behind.

### The client ID

Talking to `api.minecraftservices.com` requires an Azure application that has
been granted access to the Minecraft API. Newly registered applications are
refused until that grant is issued, and the grant has real lead time.

The client ID is supplied at build time and is never a constant in this
repository. Builds without one are valid and produce an image with Microsoft
login disabled. The reasoning, including why borrowing another launcher's ID is
a bad idea even though it is common, is in
[ADR 0005](adr/0005-microsoft-client-id-is-a-build-input.md).

## Offline mode

Offline mode is a real feature with real uses: authentication server outages,
local network play, and venues with no internet. It is also the single easiest
way to turn a launcher into a piracy tool, which is why every launcher that
ships it responsibly gates it.

PhantomMC gates it at manifest build time.

```
   builder (web)                                        image (RAM)
   ─────────────                                        ───────────
   user signs in once
          │
          ▼
   ownership confirmed
   against Minecraft services
          │
          ▼
   entitlement assertion         ──── manifest ────▶    agent checks:
   { issuedAt, expiresAt,                                 signature valid?
     signature }                                          not expired?
                                                              │
   nothing else is stored.                                    ▼
   no token. no account id.                          offline launch allowed
```

The assertion carries no token and no account identifier. It records that
ownership was confirmed and when that confirmation expires. The runtime honours
`auth.mode: offline` only when a valid assertion is present, and validation
fails otherwise, so this is enforced in code rather than by policy.

Expiry matters: without it a single manifest could be shared indefinitely as a
way around the check.

The full reasoning is in
[ADR 0006](adr/0006-offline-mode-requires-entitlement.md).

## What is deliberately not supported

Alternative authentication services that issue Minecraft sessions without a
Microsoft account are out of scope. So is any mechanism whose purpose is to
produce a playable session for someone who does not own the game. This is not a
technical limitation and it will not be accepted as a contribution.
