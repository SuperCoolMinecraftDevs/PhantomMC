# Roadmap

Milestone based rather than date based. Estimates assume weekend work by one
person and are the honest numbers, not the optimistic ones.

## Position

```
  M0 ████████████████████  done   foundation
  M1 ████████████████████  done   bootable image, root in RAM
  M2 ░░░░░░░░░░░░░░░░░░░░         graphics and a running game   ◀── here
  M3 ░░░░░░░░░░░░░░░░░░░░         launcher core
  M4 ░░░░░░░░░░░░░░░░░░░░         authentication
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

## M2 Graphics and a running game — 2 weekends

The milestone that decides whether the project works.

- Mesa, `seatd`, `cage`, XWayland in the image
- A JRE fetched at build time
- Vanilla Minecraft launching full screen on real hardware
- Confirm Mojang's glibc natives load, which is the whole premise of ADR 0002

Risk: high. Wayland, seat management and GPU initialization outside a desktop
session is where the unknowns are. Everything after this is ordinary software.

## M3 Launcher core — 1 to 2 weekends

Resolving Mojang's piston metadata into something launchable: version manifest,
asset index, library artifacts matched to the platform, natives extraction,
classpath assembly, digest verification, resumable downloads.

## M4 Authentication — 1 weekend, blocked

Device code flow, the Xbox Live and XSTS chain, entitlement assertion checking.

Blocked on the Azure application grant, which has lead time and must be started
well before this milestone.

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
