# 0001. Record architecture decisions

Status: accepted

## Context

PhantomMC makes a lot of unusual choices that look arbitrary without the
reasoning behind them. Someone reading `os/build.sh` in six months should be able
to find out why the base is Debian rather than Alpine without asking.

## Decision

Every decision that is expensive to reverse gets a record in `docs/adr/`.
Anything that can be changed in an afternoon does not.

## Consequences

Slightly more overhead per decision. In exchange the reasoning survives past the
conversation it happened in.
