# Memory model

The only resource PhantomMC actually spends is RAM, so it is worth being precise
about where it goes.

## The trick

The naive way to run from memory is to extract the root filesystem into a tmpfs.
That costs the full uncompressed size for the whole session, and most of those
bytes are read once during boot and never touched again.

Instead the compressed SquashFS is copied into tmpfs and loop mounted in place.
It stays compressed for the entire session. A tmpfs overlay on top absorbs every
write.

```
        naive                          PhantomMC
   ┌──────────────┐                 ┌──────────────┐
   │              │                 │  overlay     │  what you actually wrote
   │  extracted   │  800 MiB        │  (tmpfs)     │  50 MiB idle
   │  rootfs      │                 ├──────────────┤
   │  (tmpfs)     │                 │  squashfs    │  280 MiB, stays compressed
   │              │                 │  (tmpfs)     │
   └──────────────┘                 └──────────────┘
      800 MiB                          330 MiB
```

The cost is CPU on read. Decompressing a zstd block is far cheaper than the disk
read it replaces, and the pages that matter end up in the page cache anyway, so
in practice the hot path is uncompressed memory either way.

## Budget

Measured where marked, estimated otherwise.

| Component | Size | Status |
| --- | --- | --- |
| Kernel and initramfs | 80 MiB | measured |
| Compressed root, minimal profile | 174 MiB | measured |
| Compressed root, full profile | 393 MiB | measured |
| Full profile ISO | 454 MiB | measured |
| Overlay at idle | 50 MiB | estimated |
| Vanilla game, assets and libraries | 400 to 600 MiB | estimated |
| A large modpack on top | up to 2 GiB | estimated |

The full profile came in at 393 MiB against an estimate of 250 to 350 MiB in
[ADR 0003](adr/0003-compressed-root-in-ram.md). The estimate was low and the ADR
has been left as written, because it records what was believed when the decision
was made.

So the operating system overhead lands around 470 MiB rather than 400 MiB. On a
32 GiB machine that is still noise. On an 8 GiB laptop it leaves roughly 7 GiB
for the JVM, which is more headroom than a Windows install with a browser open
would give you. On a 4 GiB machine it is now genuinely tight and trimming
matters.

## Where the weight is

Installed size of the directly listed packages, before dependencies:

| List | Size | Notes |
| --- | --- | --- |
| firmware | 233 MiB | Dominates everything else |
| graphics | 88 MiB | Mesa with every driver |
| base | 21 MiB | |
| network | 3 MiB | |
| java | 3 MiB | Metapackages. The real cost is in the headless runtimes they pull |
| boot | 2 MiB | |

Firmware is more than half the image and the obvious place to cut. Most of it is
for hardware any given machine does not have. Splitting firmware by GPU vendor at
build time, the way `--gpu` already splits drivers, would cut a substantial
amount for anyone who knows what card they are running.

## Trimming

Three levers, in order of how much they buy:

Firmware is the biggest single win. Debian's full firmware set is over a
gigabyte uncompressed. The package lists in `os/config/packages.firmware` select
only the families likely to be present on a desktop or laptop. A future variant
could strip further based on a GPU choice made at build time.

Documentation, manual pages and locales are excluded at bootstrap time through
`dpkg` path exclusions in `os/build.sh`. This is cheap and already done.

Compression level is set to zstd 19, which is slow to write and fast to read.
That is the right trade for something built once and booted many times.

## zram

Planned, not implemented. A compressed block device backing the overlay would
let low memory machines hold a larger working set before running out. It is
listed under [M10](roadmap.md) because it only matters once there is a real
workload to measure, and guessing at compression ratios before then is not
useful.

## What happens when memory runs out

Currently the kernel OOM killer takes the JVM and the user gets dropped to a
console. This is bad and is tracked as part of [failure
handling](failure-handling.md). The intended behaviour is that the agent checks
available memory against the manifest's declared heap before launching and
refuses with a clear message rather than letting the machine thrash.
