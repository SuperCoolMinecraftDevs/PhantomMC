# 0006. Offline mode requires proof of entitlement

Status: accepted

## Context

Offline mode is a legitimate and widely shipped launcher feature. It covers
authentication server outages, local network play, and machines that cannot reach
Microsoft at launch time.

Launchers that ship it responsibly gate it. Prism Launcher will not create an
offline account until a Microsoft account that owns the game has been added,
which is why a small ecosystem of patches exists purely to defeat that check.
An ungated offline toggle is not a convenience feature, it is a way to play
without owning the game.

PhantomMC keeps no state between boots, so the usual approach of remembering a
previous successful login is not available.

## Decision

Entitlement is verified once, at manifest build time, in the web builder. The
user signs in, ownership is confirmed against the Minecraft services API, and
nothing from that exchange is stored. The resulting manifest carries an
entitlement assertion alongside the chosen offline username.

The runtime honours `auth.mode: offline` only when that assertion is present and
valid. Images built without it fall back to interactive Microsoft login.

## Consequences

Boot stays instant and requires no login, which was the point of offline mode
here.

One sign in is required when the image is configured, on a device with a browser,
rather than on the machine being booted.

The assertion is signed by our backend and carries an expiry, so a manifest
cannot be shared indefinitely as a way around the check.
