# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Build manifest schema and validation, covering Minecraft version, mod loader,
  Java runtime, mods, servers and authentication mode.
- Decision records for the base distribution, memory layout, distribution policy,
  Microsoft client credentials and offline mode gating.
- Makefile and continuous integration for the Go sources and shell scripts.
- Bootable image build producing a hybrid BIOS and UEFI ISO from a Debian
  base, with the root filesystem compressed and loaded into memory.
- Initramfs boot script that locates the boot medium by probing for the
  payload, copies it into RAM, releases the medium, and stacks a tmpfs
  overlay on the read only root.
- QEMU smoke test asserting the image reaches userspace with the boot medium
  released.
- Mojang version metadata model with platform rule evaluation, classpath
  assembly and argument substitution.
- Java runtime discovery and exact major version selection. The image ships
  Java 21 and Java 25.
- Kiosk session units, the phantom runtime user and a graphical default
  target.
- Concurrent asset fetching with digest verification and progress reporting.
- End to end install: resolve a version, download libraries and assets,
  select a matching Java runtime, assemble the command line and launch.
- Process supervision classifying a session as quit, crash or signalled, and
  flagging sessions too short to have reached the menu.
- Offline account identity derived the same way Minecraft has always derived
  it, so a player keeps a stable uuid without anything being stored.
- Microsoft device code sign in, including the Xbox Live, XSTS and Minecraft
  token chain, readable messages for the common XSTS refusals, and ownership
  detection. Sign in runs concurrently with the download.
- `auth.clientId` in the manifest, so the Azure application can be rotated
  without re-flashing a boot medium.
