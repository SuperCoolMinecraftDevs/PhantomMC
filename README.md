<div align="center">

```
██████╗ ██╗  ██╗ █████╗ ███╗   ██╗████████╗ ██████╗ ███╗   ███╗
██╔══██╗██║  ██║██╔══██╗████╗  ██║╚══██╔══╝██╔═══██╗████╗ ████║
██████╔╝███████║███████║██╔██╗ ██║   ██║   ██║   ██║██╔████╔██║
██╔═══╝ ██╔══██║██╔══██║██║╚██╗██║   ██║   ██║   ██║██║╚██╔╝██║
██║     ██║  ██║██║  ██║██║ ╚████║   ██║   ╚██████╔╝██║ ╚═╝ ██║
╚═╝     ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝    ╚═════╝ ╚═╝     ╚═╝
                                                    M C
```

**A Minecraft appliance that lives entirely in memory.**

Boot it from a USB stick, pull the stick out, play. Nothing is written to any
drive in the machine, and when you power off there is nothing left to find.

[![ci](https://img.shields.io/github/actions/workflow/status/SuperCoolMinecraftDevs/PhantomMC/ci.yml?branch=main&style=flat-square&label=ci&logo=githubactions&logoColor=white)](https://github.com/SuperCoolMinecraftDevs/PhantomMC/actions/workflows/ci.yml)
[![license](https://img.shields.io/badge/license-GPL--3.0-blue?style=flat-square)](LICENSE)
[![go](https://img.shields.io/badge/go-1.27-00ADD8?style=flat-square&logo=go&logoColor=white)](go.mod)
[![base](https://img.shields.io/badge/base-Debian%20trixie-A81D33?style=flat-square&logo=debian&logoColor=white)](docs/adr/0002-glibc-base-instead-of-alpine.md)
[![disk usage](https://img.shields.io/badge/disk%20usage-0%20bytes-brightgreen?style=flat-square)](docs/architecture.md)
[![status](https://img.shields.io/badge/status-pre--alpha-orange?style=flat-square)](docs/roadmap.md)

</div>

---

> [!WARNING]
> **Pre-alpha.** The boot chain works and the image reaches userspace from RAM.
> Minecraft does not launch yet. See the [roadmap](docs/roadmap.md) for what is
> actually finished.

## What it does

```
   power on              ~3s                    ~10s                 done
      │                   │                      │                    │
      ▼                   ▼                      ▼                    ▼
 ┌─────────┐      ┌──────────────┐      ┌────────────────┐    ┌──────────────┐
 │  USB /  │─────▶│   root copied │─────▶│  medium ejected│───▶│   Minecraft  │
 │   ISO   │      │   into RAM    │      │  network up    │    │  full screen │
 └─────────┘      └──────────────┘      └────────────────┘    └──────────────┘
                                               │
                                    ┌──────────┴──────────┐
                                    │  pull the stick out │
                                    │  it is not needed   │
                                    └─────────────────────┘
```

The root filesystem is a compressed SquashFS image. It is copied into a tmpfs,
loop mounted read only, and given a writable tmpfs overlay. The boot medium is
unmounted before userspace starts. From that point the machine is a closed loop
running out of volatile memory, and the internal drives are never touched. Pull
the power and every trace is gone.

## Why you might want this

| Situation | What PhantomMC gives you |
| --- | --- |
| Friends bring laptops to a LAN party | Hand out sticks. No installs, no Java, no launcher setup, no cleanup afterwards |
| A machine whose drive has died | RAM and a GPU are enough. The dead disk is irrelevant |
| A shared or borrowed computer | Nothing is written, nothing persists, nothing to explain |
| A modpack that breaks constantly | Power cycle. The image is byte identical every boot |
| An old low end PC | No desktop environment, no telemetry, no background services competing for the GPU |

## Quick start

Nothing to try yet. When there is, it will look like this:

```sh
git clone https://github.com/SuperCoolMinecraftDevs/PhantomMC
cd PhantomMC
sudo make image      # builds out/phantommc-auto.iso
make smoke           # boots it under QEMU and checks it reaches RAM
```

Full instructions in [docs/building.md](docs/building.md).

## Documentation

| Document | Contents |
| --- | --- |
| [Architecture](docs/architecture.md) | How the whole system fits together and what runs where |
| [Boot sequence](docs/boot.md) | Every stage from firmware to game, in order |
| [Memory model](docs/memory.md) | Where every megabyte goes and why |
| [Manifest format](docs/manifest.md) | The schema that describes one bootable configuration |
| [Networking](docs/networking.md) | Bringing up ethernet, wifi and awkward home setups |
| [Authentication](docs/authentication.md) | Device code login and how offline mode is gated |
| [Failure handling](docs/failure-handling.md) | What happens when the game is your desktop and it crashes |
| [Hardware support](docs/hardware.md) | GPUs, firmware, what is expected to work |
| [Building](docs/building.md) | Building, testing and hacking on the image |
| [Alternatives considered](docs/alternatives.md) | Approaches that were rejected and why |
| [Roadmap](docs/roadmap.md) | Milestones and current position |
| [Glossary](docs/glossary.md) | Terms used throughout |
| [Decision records](docs/adr/) | Every expensive decision, with reasoning |

## What PhantomMC is not

It **does not distribute Minecraft**. Game files come from Mojang's own content
delivery network and mods come from Modrinth, both fetched on the machine that
runs them. This repository contains an operating system and a launcher. It
contains nothing belonging to Mojang, and it never will. See
[ADR 0004](docs/adr/0004-no-game-file-redistribution.md).

It **does not let you play without owning the game**. Offline mode exists,
because launcher outages and local network play are real, but it is gated behind
a one time ownership check exactly as other launchers gate it. See
[ADR 0006](docs/adr/0006-offline-mode-requires-entitlement.md).

It **is not a general purpose distribution**. There is no package manager at
runtime, no desktop, and no way to install anything. It runs one program.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Short version: small commits, decisions
get an ADR, and a broken build never becomes a release.

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).
