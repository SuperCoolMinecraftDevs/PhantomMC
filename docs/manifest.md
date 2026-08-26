# Manifest format

A manifest is the complete description of one bootable configuration. It is the
only artifact PhantomMC servers store, and everything variable about a boot
comes from it.

Schema version 1. Defined in `internal/manifest/manifest.go`, validated in
`internal/manifest/validate.go`, example at
[`examples/manifest.json`](examples/manifest.json).

## Example

```json
{
  "schemaVersion": 1,
  "id": "phantom-starter",
  "name": "Phantom Starter",
  "createdAt": "2026-08-26T00:00:00Z",
  "minecraft": { "version": "1.21.8" },
  "loader": { "kind": "fabric", "version": "0.16.14" },
  "java": { "major": 21, "distribution": "temurin" },
  "auth": { "mode": "microsoft" },
  "graphics": { "vendor": "auto" },
  "jvm": { "heapMB": 4096, "args": ["-XX:+UseG1GC"] },
  "mods": [
    {
      "source": "modrinth",
      "projectId": "AANobbMI",
      "versionId": "vN2gRIJ8",
      "filename": "sodium-fabric-0.6.13.jar",
      "artifact": {
        "url": "https://cdn.modrinth.com/data/AANobbMI/versions/vN2gRIJ8/sodium.jar",
        "size": 1048576,
        "sha512": "..."
      }
    }
  ],
  "servers": [{ "name": "Home", "address": "mc.example.com:25565" }]
}
```

## Fields

### Top level

| Field | Type | Rules |
| --- | --- | --- |
| `schemaVersion` | int | Must equal 1. Unknown versions are refused, not guessed at |
| `id` | string | 3 to 64 characters, lowercase alphanumeric and hyphens |
| `name` | string | Human readable, must not be blank |
| `createdAt` | RFC 3339 | When the builder emitted this |

### `minecraft`

| Field | Rules |
| --- | --- |
| `version` | Required. A version id from Mojang's piston metadata, such as `1.21.8` |

### `loader`

| Field | Rules |
| --- | --- |
| `kind` | One of `vanilla`, `fabric`, `quilt`, `forge`, `neoforge` |
| `version` | Required for every kind except `vanilla`, which must leave it empty |

A `vanilla` loader with a non-empty `mods` array is rejected. This catches a
configuration that would otherwise fail silently at launch with the mods simply
ignored.

### `java`

| Field | Rules |
| --- | --- |
| `major` | 8 or greater. Must match what the chosen Minecraft version requires |
| `distribution` | Runtime to fetch when the image does not already carry the right major |

The image ships Java 21 and Java 25, which between them cover the 1.21 and 26
release lines. The agent selects between them by reading `javaVersion` from
Mojang's version document rather than trusting this field, because the version
document is authoritative and a manifest can be wrong.

The match is exact in both directions. Too old and the game will not start. Too
new and mod loaders and reflection heavy mods fail in ways that are difficult to
read out of a crash log.

### `auth`

| Field | Rules |
| --- | --- |
| `mode` | `microsoft` or `offline` |
| `offlineUsername` | Required in offline mode. 3 to 16 characters of `A-Za-z0-9_`. Must be empty in microsoft mode |
| `entitlement` | Required in offline mode. See [authentication.md](authentication.md) |

### `graphics`

| Field | Rules |
| --- | --- |
| `vendor` | `auto`, `amd`, `intel` or `nvidia`. `auto` covers everything Mesa handles |
| `renderWidth`, `renderHeight` | Optional. Render below native and let the compositor upscale |
| `upscaleTo` | Optional. Upscaler to apply |

### `jvm`

| Field | Rules |
| --- | --- |
| `heapMB` | At least 512. Checked against available memory before launch |
| `args` | Extra JVM arguments, appended after the generated ones |

### `mods[]`

| Field | Rules |
| --- | --- |
| `source` | `modrinth`, `curseforge` or `direct` |
| `projectId`, `versionId` | Upstream identifiers, kept for diagnostics and rebuilds |
| `filename` | A bare filename ending in `.jar`. No path separators. Must be unique within the manifest |
| `manual` | True when the upstream forbids third party download |
| `artifact` | See below |

### `artifact`

| Field | Rules |
| --- | --- |
| `url` | Absolute `https` URL. Plaintext `http` is refused. Must be empty when `manual` is true |
| `size` | Positive byte count |
| `sha512` | 128 lowercase hex characters. Recorded at build time and checked after download |

### `servers[]`

| Field | Rules |
| --- | --- |
| `name` | Must not be blank |
| `address` | Host or `host:port`. No whitespace |

Written into `servers.dat` so the multiplayer list is already populated on first
launch.

## Validation as a security boundary

Validation is not a convenience. It is the mechanism that stops a manifest from
being an arbitrary code execution vector, since a manifest is a document fetched
over the network that ends up controlling what gets downloaded and where it is
written.

The rules that matter:

- `filename` must be a bare name ending in `.jar`, so a manifest cannot write
  outside the mods directory by way of `../`
- `url` must be `https`, so a manifest cannot downgrade a fetch to plaintext
- `sha512` is mandatory and checked, so a compromised CDN cannot substitute a
  different file
- Unknown JSON fields are rejected during decoding, so a manifest written for a
  future schema fails loudly rather than being silently half understood

All of these are covered by tests in `internal/manifest/manifest_test.go`.

## Manual mods

CurseForge lets authors deny third party distribution, and PhantomMC respects
that flag. When a selected mod carries it, the builder emits the entry with
`manual: true` and no URL. The agent reports it and prompts the user to supply
the file rather than fetching it. See
[ADR 0004](adr/0004-no-game-file-redistribution.md).
