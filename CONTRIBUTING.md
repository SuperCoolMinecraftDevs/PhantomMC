# Contributing

## Commits

Small and focused. One logical change per commit, and a message that says what
changed and, when it is not obvious, why.

```
Find the boot medium by probing for the payload

Looking up /dev/disk/by-label only works when the medium was written by a
flashing tool. A stick populated by copying files onto an existing FAT32
partition keeps its old label, and that is a flow we want to support.
```

Subject line in the imperative, under about 70 characters, no trailing period. A
body when the reasoning is not obvious from the diff. A commit that touches the
build system, the docs and three packages at once should have been three
commits.

## Decisions

Anything expensive to reverse gets a record in `docs/adr/`. Anything changeable
in an afternoon does not.

Copy the shape of an existing one: context, decision, consequences. Records are
append only. If a decision is reversed, add a new record that supersedes the old
one rather than editing history.

## Code

Modular and readable. Comments explain why, never what. A comment that restates
the function name is noise and will be removed.

```go
// bad
// GetUser gets the user.

// fine
// Probing every block device is cheap and, unlike a label lookup, works on a
// stick that was populated by copying files rather than being flashed.
```

Before adding a dependency, ask whether it can be written instead. Prefer
writing it. A few hundred lines under our control beats a transitive tree we do
not read, especially in something that has to work on unfamiliar hardware with
no way to ship a hotfix mid session.

Go code is `gofmt` formatted and passes `go vet`. Shell scripts pass
`shellcheck`. `make lint` runs all of it.

## Tests

Anything that can be tested without hardware should be. Validation logic, the
manifest schema, planning: all covered and expected to stay covered.

The boot chain is covered by the QEMU smoke test, which runs without KVM so it
works anywhere. It proves the image reaches userspace from RAM. It proves
nothing about graphics, because there is no real GPU involved. Do not claim
otherwise in a pull request.

## CI and releases

CI is allowed to fail on a branch. That is what it is for.

A tag is never cut while CI is red. If a release build fails, the fix comes
before anything else. There is no such thing as a known broken release here.

## Changelog

Update `CHANGELOG.md` under `Unreleased` in the same commit as a user visible
change. Format is Keep a Changelog.

## Scope

Two things will not be accepted regardless of implementation quality:

Anything that redistributes Minecraft, its assets, or mods whose authors denied
third party distribution. See [ADR 0004](docs/adr/0004-no-game-file-redistribution.md).

Anything whose purpose is producing a playable session for someone who does not
own the game, including alternative session services and any route around the
entitlement check. See [ADR 0006](docs/adr/0006-offline-mode-requires-entitlement.md).

Both exist for the same reason: this project is only interesting if it can be
distributed publicly, and neither of those survives contact with that goal.
