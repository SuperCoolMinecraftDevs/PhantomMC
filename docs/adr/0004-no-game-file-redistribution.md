# 0004. Never redistribute game or mod files

Status: accepted

## Context

The original design had the website assemble an image containing Minecraft, the
Java runtime and the user's mods, then serve it as a download. That makes the
project a distributor of Mojang's copyrighted files, which their terms do not
permit. It also ignores the CurseForge distribution flag, which many mod authors
set to deny third party redistribution.

Serving those files is also the single fastest way to get the repository and the
domain taken down, which ends the project regardless of the technical merits.

## Decision

PhantomMC servers store manifests. They never store or proxy game files, Java
runtimes, or mod archives.

Two delivery modes, both fetching from upstream:

- Stream. The image boots, resolves its manifest, and downloads from Mojang's
  content delivery network and Modrinth directly into the overlay.
- Prefetch. A small cross platform tool runs on the user's own machine, performs
  the same downloads from the same upstreams, and writes a populated USB stick.
  For users whose connection is too slow to do this at every boot.

The distinction is whose machine does the fetching. It is never ours.

## Consequences

Boot requires network access in stream mode. Prefetch mode covers the slow
connection case at the cost of a separate tool and a re-flash when the pack
changes.

Hosting cost stays near zero, because we serve small JSON documents.

A mod whose distribution flag denies third party downloads cannot be fetched
automatically. The manifest records it and the user is prompted to supply it.
