# Alternatives considered

Approaches that were seriously considered and rejected. Recorded so they are not
proposed again without new information.

## Alpine Linux as the base

The obvious pick for a small image, and the original plan.

Rejected because Mojang ships LWJGL, GLFW and OpenAL as prebuilt native
libraries linked against glibc, and Alpine uses musl. `gcompat` alone does not
bridge this. The working recipe on musl systems is to discard Mojang's natives,
build replacements, redirect LWJGL at them through system properties, and
override the allocator. That is fragile for vanilla and worse for mods carrying
their own native code, which is exactly the compatibility the project promises.

Cost of the decision: roughly 100 MiB. Worth it.

Full reasoning in [ADR 0002](adr/0002-glibc-base-instead-of-alpine.md).

## iPXE network boot

The original design had a tiny iPXE bootloader fetch the payload over the
network, so the stick never needed re-flashing.

Rejected because iPXE runs before Linux and therefore has its own drivers, its
own network stack and its own TLS. Modern wifi is largely absent, WPA
Enterprise is not happening, and TFTP depends on broadcast behaviour that
consumer routers and hand rolled NAT setups handle unpredictably.

Letting a small Linux kernel do the networking instead gives real drivers, `iwd`,
and ordinary HTTPS. The same goal, reached with the network stack that already
solves these problems. See [networking.md](networking.md).

## Debian live-boot

`live-boot` already implements `toram` and would have saved writing a boot
script.

Rejected because it brings a large amount of Debian Live machinery for one
feature, and its medium discovery is built around labels and its own conventions.
Our discovery has to handle a stick populated by copying files, which live-boot
does not target. The script we wrote is around 140 lines and we control all of
it, which matters for the thing most likely to break on unfamiliar hardware.

## Extracting the root filesystem into tmpfs

Simpler than loop mounting a compressed image.

Rejected because it costs the full uncompressed size for the entire session,
roughly two and a half times more memory, and every one of those bytes is taken
from the JVM. See [memory.md](memory.md).

## Building the ISO on the server per user

The original plan: the website compiles a custom image containing the user's
chosen mods and the game.

Rejected twice over. It makes the project a distributor of Mojang's files and of
mods whose authors denied redistribution, which is not permitted and is the
fastest route to a takedown. It also means paying CPU to pack a SquashFS image
for every download, when the same result is achieved by serving a few kilobytes
of JSON and letting the client fetch from upstream.

See [ADR 0004](adr/0004-no-game-file-redistribution.md).

## A shell script instead of the agent

Keeping everything in the image as shell.

Rejected because the work is not shell shaped: resolving Mojang version metadata
into a classpath, matching library artifacts to the platform, verifying digests,
resuming partial downloads, and supervising a JVM while tracking exit codes and
runtime. It is also the component most likely to change after release, and a
binary the image fetches can change without anyone re-flashing a stick.

## Persistent storage for the authentication token

Keeping a small writable partition on the stick so the Microsoft login survives
reboots, as Tails does for its persistent volume.

Rejected because it breaks the guarantee the project is built on, requires
custom partitioning that rules out drag and drop installation, and buys ten
seconds. Device code flow on a phone is fast enough that the trade is not close.

## Supporting single player worlds

Synchronizing saves to a network share on a timer.

Rejected as scope. It requires configuring a remote target, handling conflicts,
surviving power loss mid write, and taking responsibility for people's worlds.
The appliance framing is coherent without it: this is a machine for joining
servers. Someone who wants persistent local worlds wants an ordinary computer.
