# Glossary

**Agent** — `phantomd`, the Go binary inside the image that resolves the
manifest, fetches the game, and supervises it.

**cage** — A kiosk compositor built on wlroots. Runs one application full screen
with no desktop, no taskbar and no way to minimize.

**Device code flow** — An OAuth grant for devices that cannot show a browser.
The device displays a short code, the user enters it on a phone, and the device
polls until approval. How smart TVs sign in.

**Entitlement assertion** — A signed statement from the builder recording that
game ownership was confirmed and when that confirmation expires. Carries no
token and no account identifier. Required for offline mode.

**GLFW** — The windowing and input library Minecraft uses through LWJGL.
Distributed by Mojang as a prebuilt binary linked against glibc, which is why
the base distribution has to use glibc.

**glibc and musl** — Two implementations of the C standard library. Debian uses
glibc, Alpine uses musl. Binaries built against one do not generally run on the
other. See [ADR 0002](adr/0002-glibc-base-instead-of-alpine.md).

**initramfs** — A small filesystem the kernel unpacks into memory at boot,
containing just enough to find and mount the real root. Where the medium probing
and the RAM copy happen.

**LWJGL** — The Java bindings that give Minecraft access to OpenGL, GLFW and
OpenAL. It loads native libraries, which is the source of most platform trouble.

**Manifest** — The JSON document describing one bootable configuration. The only
artifact PhantomMC servers store. See [manifest.md](manifest.md).

**Manual mod** — A mod whose upstream denies third party redistribution. Recorded
in the manifest without a URL and supplied by the user.

**Mesa** — The open source graphics driver stack. Covers AMD and Intel with no
configuration, and is a large part of why those are the easy path.

**mmdebstrap** — Builds a Debian root filesystem from packages. Faster than
`debootstrap` and does not require root for every mode.

**overlayfs** — Stacks a writable layer over a read only one. The read only layer
is the SquashFS image, the writable layer is a tmpfs, and together they are the
root filesystem.

**Payload** — `/phantom/root.squashfs` on the boot medium. Its presence is what
identifies a device as a PhantomMC medium.

**PXE and iPXE** — Network boot standards implemented in firmware, before Linux
starts. Considered and rejected, see [alternatives.md](alternatives.md).

**seatd** — Arbitrates access to input devices and the display without a login
manager. Needed because there is no desktop session to own the seat.

**SquashFS** — A compressed read only filesystem. Held compressed in memory for
the whole session and decompressed on read.

**switch_root** — The point where the initramfs replaces itself with the real
root and hands control to the actual init system.

**tmpfs** — A filesystem that lives in memory. Holds both the compressed image
and the writable overlay.

**toram** — The general technique of copying a boot medium into memory so it can
be removed. Named after the boot parameter other live systems use for it.

**XWayland** — Runs X11 clients under a Wayland compositor. Minecraft's bundled
GLFW targets X11, so this sits in the path until native Wayland is viable.

**zstd** — The compression algorithm used for the root image. Slow to compress at
level 19, fast to decompress, which is the correct trade for something built once
and booted often.
