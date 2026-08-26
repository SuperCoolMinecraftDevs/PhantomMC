# PhantomMC

A diskless Minecraft appliance. Boot it from a USB stick, pull the stick out, and
the machine runs Minecraft entirely from RAM. Nothing is written to any internal
drive, and the machine you booted is unchanged when you power it off.

## Status

Early development. Nothing here is usable yet. See `docs/adr/` for the decisions
made so far and `CHANGELOG.md` for what has landed.

## How it works

1. A small bootable image loads a kernel and a compressed root filesystem into RAM.
2. The root filesystem stays compressed and is loop mounted, with a tmpfs overlay
   on top for anything written at runtime.
3. The boot medium is released once the copy completes, so the USB stick can be
   removed.
4. An agent brings up networking, resolves a build manifest, fetches the game and
   mods into the overlay, and launches the game under a kiosk compositor.
5. On power off, all of it is gone.

## What is not shipped

PhantomMC does not distribute Minecraft. Game files come from Mojang's own
content delivery network and mods come from Modrinth, both fetched on the machine
that runs them. The image contains an operating system and a launcher, nothing
belonging to Mojang.

## Repository layout

    os/          bootable image build
    cmd/         Go binaries
    internal/    Go packages
    docs/        architecture notes and decision records

## License

GPL-3.0-or-later. See `LICENSE`.
