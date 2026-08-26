# 0007. One Go module for the whole repository

Status: accepted

## Context

The runtime agent, the web backend and the prefetch tool all operate on the same
manifest format and the same upstream clients. Splitting them into separate
modules means versioning an internal schema against ourselves.

## Decision

One module, `github.com/SuperCoolMinecraftDevs/PhantomMC`, with binaries under
`cmd/` and shared code under `internal/`. No `go.work`.

## Consequences

A change to the manifest format updates every consumer in the same commit, and
CI catches drift immediately.

Nothing under `internal/` can be imported by outside projects. That is intended
for now. If any of it becomes worth publishing it moves to `pkg/` in a later
record.
