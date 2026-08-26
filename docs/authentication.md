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
message.

Two of those are worth calling out because they are not really errors:

XSTS refuses with a bare numeric code and an empty message. `2148916233` means
the Microsoft account has never had an Xbox profile created, `2148916238` means
it is a child account that has not been added to a family. Both are common and
both are fixable by the user, so the agent translates them into a sentence
rather than surfacing the number.

A 404 from the profile endpoint is the ownership check. The account signed in
perfectly well, it simply does not own Minecraft Java Edition.

Sign in runs concurrently with the game download, since both take about a
minute and neither depends on the other. The user reads the code off the screen
and approves on their phone while assets stream in.

Because the machine has no persistence, the token dies with the power. Signing
in happens on every boot. It takes about ten seconds on a phone and it is the
price of the guarantee that nothing is left behind.

### The client ID

Talking to `api.minecraftservices.com` requires an Azure application that has
been granted access to the Minecraft API. Newly registered applications are
refused until that grant is issued, and the grant has real lead time.

The client ID travels in the manifest, under `auth.clientId`. It is not
compiled into the agent and it is not baked into the image.

That placement is deliberate. The image on a boot medium is expected to outlive
many configuration changes, so a credential embedded in it can only be rotated
by asking every user to re-flash. A manifest is fetched fresh on every boot, so
replacing the value takes effect the next time anyone powers on. It also means
this repository carries no credential of its own.

Obtaining one is the genuinely hard part. Registering an Azure application is
free, but the `XboxLive.signin` permission that the Minecraft API requires is
gated, and Microsoft has told hobbyist launcher developers asking about it that
they need to be enrolled in the Xbox Developer program.

Several open source launchers publish their client IDs, and the official
launcher's is widely circulated, so reusing one is common. It is also not
permitted: Prism Launcher's build documentation instructs forks to replace the
value, and using the official launcher's means presenting your application to
Microsoft as the Minecraft Launcher. PhantomMC ships with no default. Whoever
operates a manifest server chooses what to put there and owns that choice.

See [ADR 0005](adr/0005-microsoft-client-id-is-a-build-input.md).

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
