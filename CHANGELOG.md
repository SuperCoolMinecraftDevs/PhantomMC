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
