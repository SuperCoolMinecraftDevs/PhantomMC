# FAQ

### Does this actually use zero disk space?

Yes, in the sense that matters: nothing is written to any drive inside the
machine. You can physically remove every drive and it still works. That is not a
figure of speech, it is a supported configuration and one of the reasons the
project exists.

The USB stick holds the image, so that is not zero. If you boot the same image
over the network there is no stick either.

### Can I really remove the USB stick?

Yes, once the console says so. The image is copied into memory and the medium is
unmounted before userspace starts. If the unmount fails the boot aborts rather
than continuing in a state where the stick is silently still required.

### Why is there no single player?

Because there is nowhere to put the world. Everything is in volatile memory and
a power cut takes it.

The alternative, synchronizing saves to a network share on a timer, was
considered and rejected as scope: it means configuring a remote target, handling
conflicts, surviving power loss mid write, and taking responsibility for
people's worlds. The appliance framing works without it. This is a machine for
joining servers.

### Will it run faster than Windows?

On AMD or Intel, probably, though the honest answer is that this has not been
benchmarked yet.

The reasons to expect it: Mesa's OpenGL path is strong, there is no desktop
environment or background service competing for the GPU, and asset loading comes
out of memory rather than a disk. The reason to be careful: none of that is
measured, and the compressed root costs CPU on read.

Numbers will go here when there are real ones.

### Do all mods work?

That is the intent, and it is why the base is Debian rather than Alpine.
Mojang's native libraries and mods carrying their own native code are built
against glibc, so a musl base would have broken exactly the mods people care
about. See [ADR 0002](adr/0002-glibc-base-instead-of-alpine.md).

Untested so far, because the game does not launch yet.

### How much RAM do I need?

4 GiB runs vanilla. 8 GiB is comfortable. 16 GiB or more handles large modpacks
without thinking about it. See [memory.md](memory.md).

### Why do I have to sign in every boot?

Because there is nowhere to keep the token. Storing it would mean a writable
partition, which breaks the guarantee the whole project rests on.

Device code flow makes it about ten seconds on a phone: a short code appears on
screen, you enter it, done.

### Can I play without owning Minecraft?

No, and that will not change. Offline mode exists for authentication outages and
local network play, and it is gated behind a one time ownership check exactly as
other launchers gate it. See
[ADR 0006](adr/0006-offline-mode-requires-entitlement.md).

### Why not Alpine? It is smaller.

It is, and it was the original plan. Mojang's prebuilt natives are linked
against glibc and Alpine uses musl, which breaks them. Working around it is
fragile for vanilla and worse for native mods. The extra 100 MiB buys
compatibility that is the entire point.

### Why not iPXE?

It runs before Linux, so it has its own drivers and its own network stack: no
modern wifi, and TFTP transport that consumer routers handle unpredictably. A
small Linux kernel on a stick does the same job with real drivers and ordinary
HTTPS. See [alternatives.md](alternatives.md).

### Does it work with Secure Boot?

No. It has to be disabled. Signing would need a key in the shim chain, which is
a project in itself.

### Is my Minecraft account safe?

Authentication uses Microsoft's device code flow, the same mechanism smart TVs
use. Your password is entered on Microsoft's own site on your own phone, never on
the PhantomMC machine. The token exists only in memory and only for that boot.

### Can I use this to cheat?

The image is a stock Linux with a stock Minecraft. It does not bundle, endorse or
make room for anything of that sort, and mods that do are yours to bring and
yours to answer for.

### When can I try it?

The boot chain works. Minecraft does not launch yet. See
[roadmap.md](roadmap.md), and specifically M2.
