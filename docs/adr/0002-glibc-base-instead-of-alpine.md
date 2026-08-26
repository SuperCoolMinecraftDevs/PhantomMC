# 0002. Use a glibc base rather than Alpine

Status: accepted

## Context

Alpine is the obvious pick for a minimal image and was the original plan. It does
not work here. Mojang ships LWJGL, GLFW and OpenAL as prebuilt native libraries
linked against glibc, and Alpine uses musl. `gcompat` alone does not bridge this.

The known working recipe on musl systems is to discard Mojang's bundled natives,
build replacements from source, redirect LWJGL at them through
`-Dorg.lwjgl.*.libname` properties, and override the allocator. That is fragile
for vanilla Minecraft and considerably worse for mods carrying their own native
code, such as voice chat and physics mods. Full mod compatibility is a core
promise of this project, so a base that fights it is the wrong base.

## Decision

Build the runtime image from Debian stable using `mmdebstrap`, with glibc.

Alpine may still be used later for a separate minimal network bootstrap stage if
one is needed. That would be a new record.

## Consequences

The image grows by roughly 100 MB before compression. Mojang's natives and every
Forge, Fabric, Quilt and NeoForge mod load without special handling. Debian's
package set covers Mesa, Wayland and firmware without custom builds.

`apt` is used at build time only. It is not present at runtime.
