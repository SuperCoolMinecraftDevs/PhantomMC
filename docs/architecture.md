# Architecture

PhantomMC is three separate things that agree on one file format.

```
  ┌────────────────────────────────────────────────────────────────────┐
  │  BUILDER (web, Go)                                                 │
  │  ─────────────────                                                 │
  │  Pick a Minecraft version, a loader, mods, servers, an auth mode.   │
  │  Resolves everything against Modrinth and Mojang metadata.          │
  │  Verifies game ownership once, if offline mode was requested.       │
  │  Emits a manifest. Stores nothing else.                            │
  └───────────────────────────────┬────────────────────────────────────┘
                                  │  manifest.json  (a few KB)
                                  ▼
  ┌────────────────────────────────────────────────────────────────────┐
  │  IMAGE (os/, shell + Debian)                                       │
  │  ──────────────────────────                                        │
  │  A bootable ISO. Kernel, initramfs, compressed root, bootloader.    │
  │  Identical for every user. Contains no game files and no manifest.  │
  │  Its only job is to get a working Linux into RAM with a network.    │
  └───────────────────────────────┬────────────────────────────────────┘
                                  │  boots, then runs
                                  ▼
  ┌────────────────────────────────────────────────────────────────────┐
  │  AGENT (cmd/phantomd, Go)                                          │
  │  ────────────────────────                                          │
  │  Fetches the manifest, plans the work, downloads from Mojang and    │
  │  Modrinth into the overlay, authenticates, launches the game, and   │
  │  supervises it until it exits.                                     │
  └────────────────────────────────────────────────────────────────────┘
```

The separation matters. The image is a static artifact that almost never
changes, so it can be cached, mirrored and signed. Everything that varies per
user lives in a small JSON document. That is why a single generic ISO can serve
every configuration, and why updating the mod resolution logic does not require
anyone to re-flash a stick.

## The three delivery modes

| Mode | What is on the stick | Network needed at boot | Suits |
| --- | --- | --- | --- |
| Stream | Generic image, manifest fetched by URL | Yes, for everything | Fast connections, frequently changing packs |
| Pinned | Generic image plus an embedded manifest | Yes, for game and mods | Fixed configuration, still small |
| Prefetch | Image plus a fully populated game directory | No | Slow connections, offline venues |

Prefetch is produced by a separate cross platform tool that runs on the user's
own machine. It performs exactly the same downloads the agent would perform at
boot, from exactly the same upstreams, and writes the result to the stick. The
project never becomes a distributor of anyone else's files. See
[ADR 0004](adr/0004-no-game-file-redistribution.md).

## Filesystem layout at runtime

```
  /                     overlayfs
  ├── (lower)           /run/phantom/lower   read only squashfs, loop mounted
  │                                          from a copy held in tmpfs
  └── (upper)           /run/phantom/upper   tmpfs, everything written this boot

  /var/lib/phantom      the agent's working directory
  ├── minecraft/        game directory
  │   ├── mods/
  │   ├── assets/
  │   ├── libraries/
  │   └── versions/
  ├── manifest.json     the resolved manifest for this boot
  └── logs/
```

Nothing under `/` survives a power cycle, including the game directory. That is
the point, and it is why single player worlds are not supported. See
[the FAQ section on persistence](faq.md#why-is-there-no-single-player).

## Trust boundaries

The agent runs as an unprivileged user. It fetches from three kinds of upstream
and treats each differently:

| Upstream | Trust | Verification |
| --- | --- | --- |
| Mojang piston metadata and CDN | Authoritative for game files | TLS, plus the SHA1 digests Mojang publishes in its own metadata |
| Modrinth API and CDN | Authoritative for mods | TLS, plus the SHA512 digest recorded in the manifest at build time |
| PhantomMC manifest server | Configuration only | TLS. Manifests are validated before use and cannot reference arbitrary paths |

A manifest can only ask for files by https URL with a matching digest recorded
at build time. It cannot name a local path, cannot request plaintext http, and
cannot place a file outside the mods directory. Those rules are enforced in
`internal/manifest/validate.go` and covered by tests, not by convention.

## Why the agent is a separate binary

The obvious alternative is a shell script in the image. It was rejected because
the work involved is genuinely fiddly: resolving Mojang's version metadata into
a classpath, matching library artifacts against the current platform, verifying
digests, retrying partial downloads, and supervising a JVM while watching its
exit code and runtime. That is not shell work. It is also the part most likely
to change after release, and shipping it as a binary the image fetches means it
can change without anyone re-flashing anything.
