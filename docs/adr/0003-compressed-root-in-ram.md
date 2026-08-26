# 0003. Keep the root filesystem compressed in RAM

Status: accepted

## Context

Every byte the operating system holds is a byte Minecraft cannot use. The naive
approach of extracting the image into a tmpfs costs the full uncompressed size
for the entire session, and most of those bytes are read once at boot and never
touched again.

## Decision

Copy the zstd compressed SquashFS into RAM, loop mount it read only, and stack a
tmpfs overlay on top with overlayfs. The boot medium is released as soon as the
copy finishes.

Enable zram for the overlay so that machines with little memory get compression
on runtime writes as well.

## Consequences

Resident cost is the compressed image plus whatever the overlay actually holds,
rather than the uncompressed image. Expected budget:

| Component                    | RAM          |
| ---------------------------- | ------------ |
| kernel, initramfs, firmware  | 80 MB        |
| compressed root              | 250 to 350 MB |
| overlay at idle              | 50 MB        |
| game assets, libraries, mods | 400 MB to 2 GB |

Reads from the root pay a decompression cost. This is cheap relative to the disk
reads it replaces, and the hot paths end up in the page cache anyway.

The USB stick can be removed once the copy completes, which is the behaviour the
project is named after.
