# Security

## Reporting

Open a private security advisory through the repository's Security tab rather
than a public issue.

## Threat model

PhantomMC boots an unfamiliar machine from removable media and fetches
configuration and executable content over the network. That is the shape of the
problem.

### What it protects against

**Persistence.** Nothing is written to any drive in the machine. The root
filesystem is read only and the overlay is memory backed, so a compromise ends
at power off. There is no mechanism by which anything survives a reboot.

**Modification of the running system.** The lower layer is a read only SquashFS
image. Writes land in the overlay, which is discarded.

**Substituted downloads.** Every artifact is fetched over TLS and checked against
a SHA512 digest recorded in the manifest at build time. A compromised CDN cannot
substitute a different file without the digest failing.

**Path traversal from a manifest.** A manifest is a network document that
controls what gets written where. `filename` must be a bare name ending in
`.jar`, `url` must be `https`, and unknown JSON fields are rejected at decode
time. Enforced in `internal/manifest/validate.go` and covered by tests.

### What it does not protect against

**Mods.** A Minecraft mod is arbitrary Java running with the game's privileges.
PhantomMC verifies that the file it downloaded is the file the manifest named. It
makes no claim about what that file does. This is the same position every
launcher is in.

**A hostile manifest server.** Whoever serves the manifest chooses which mods
get downloaded. Point the image at a manifest you trust.

**Firmware and hardware attacks.** Anything below the kernel is out of scope.
Booting untrusted media on a machine you care about is a decision, not a default.

**Physical access.** Not addressed at all. There is no disk encryption because
there is no disk.

**Traffic analysis.** The machine reaches Mojang, Modrinth and the manifest
server. Anyone watching the network sees that.

## Secure Boot

Not supported. It has to be disabled to boot the image. Signing would require a
key in the shim chain, which is a substantial project of its own and is not
currently planned.

## Credentials

The Microsoft client ID is a build time input and is never committed. See
[ADR 0005](docs/adr/0005-microsoft-client-id-is-a-build-input.md).

Authentication tokens exist only in memory and only for the current boot. There
is nowhere to write them and no attempt is made to preserve them, which is why
signing in happens on every boot.

The entitlement assertion carries no token and no account identifier. It records
only that ownership was confirmed and when that confirmation expires.
