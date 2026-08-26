# Roadmap

Milestone based rather than date based. Estimates assume weekend work by one
person and are the honest numbers, not the optimistic ones.

## Position

```
  M0 ████████████████████  done   foundation
  M1 ████████████████████  done   bootable image, root in RAM
  M2 ██████████████░░░░░░         graphics and a running game   ◀── here
  M3 ██████████████████░░         launcher core
  M4 ████████████████░░░░         authentication
  M5 ░░░░░░░░░░░░░░░░░░░░         network bring up
  M6 ░░░░░░░░░░░░░░░░░░░░         mods and loaders
  M7 ░░░░░░░░░░░░░░░░░░░░         the builder
  M8 ░░░░░░░░░░░░░░░░░░░░         failure handling
  M9 ░░░░░░░░░░░░░░░░░░░░         prefetch tool
 M10 ░░░░░░░░░░░░░░░░░░░░         nvidia, tuning, size
```

## M0 Foundation — done

Repository, licence, CI, decision records, the manifest schema with validation
and tests, the agent skeleton with dry run planning.

## M1 Bootable image — done

Image build producing a hybrid BIOS and UEFI ISO from a Debian base. Custom
initramfs that probes for the payload, copies it into RAM, releases the medium
and stacks a tmpfs overlay. QEMU smoke test asserting all of it.

Verified: boots to a login prompt on the RAM backed overlay with the medium
already released.

## M2 Graphics and a running game — mostly done

The milestone that decides whether the project works.

- Mesa, `seatd`, `cage`, XWayland in the image: done
- Java runtimes in the image: done, 21 and 25
- Mojang's glibc natives load: **confirmed**. LWJGL 3.3.3 extracted and loaded
  `liblwjgl.so` on Debian and reached GLFW initialization, which is the whole
  premise of [ADR 0002](adr/0002-glibc-base-instead-of-alpine.md)
- Vanilla Minecraft launching full screen on real hardware: not yet verified

Remaining risk is narrow now: the game runs, so what is left is whether cage,
seatd and XWayland hand it a working surface outside a desktop session. That
cannot be tested without a real GPU.

## M3 Launcher core — mostly done

Version manifest and version document resolution, platform rule evaluation,
library selection, asset index handling, concurrent digest verified downloads,
classpath assembly, argument substitution, Java runtime selection by exact
major, and process supervision.

Verified against the live Mojang servers: 71 libraries and 4271 asset objects
downloaded, and the game launched as far as GLFW.

Left: resuming interrupted downloads by range request, and a disk space check
before starting a half gigabyte of downloads.

## M4 Authentication — implemented, unverified

Device code flow, the Xbox Live and XSTS chain, the Minecraft token exchange and
the profile lookup are all written and tested against a fake chain. The request
format is confirmed correct against Microsoft's live endpoint, which rejects an
invalid application with a well formed error rather than a malformed request
complaint.

End to end verification needs a real Azure application with the
`XboxLive.signin` grant. That is the outstanding blocker and it is an
administrative one, not a coding one.

Entitlement assertion checking for offline mode is still to do and belongs with
the builder in M7, since that is where assertions are issued.

## M5 Network bring up — 1 weekend

Ethernet with DHCP, wifi selection and passphrase entry through `iwd`,
reachability checking, captive portal detection.

## M6 Mods and loaders — 2 weekends

Modrinth resolution, Fabric installation, then NeoForge. Manual mod handling for
entries whose upstream denies redistribution. CurseForge behind a flag once
Modrinth is solid.

## M7 The builder — 2 weekends

Go backend and a web interface. Mod search, dependency resolution, manifest
emission, the entitlement check for offline mode.

## M8 Failure handling — 1 weekend

Supervision, the fallback interface, `mclo.gs` upload with a QR code, the
circuit breaker, safe mode. Design is in
[failure-handling.md](failure-handling.md).

## M9 Prefetch tool — 1 weekend

Cross platform binary that performs the agent's downloads on the user's own
machine and writes a populated stick, for connections too slow to stream at
every boot.

## M10 Nvidia, tuning, size — ongoing

Nvidia image variant, zram, compositor upscaling, firmware trimming.

## Total

Roughly three months of weekends to something publishable. M2 is the milestone
worth being nervous about.

## Explicitly out of scope

- Single player worlds and save synchronization
- Any mechanism for playing without owning the game
- Running as an installed operating system
- Architectures other than x86-64
- Secure Boot
