# 0005. The Microsoft client ID is a build input

Status: accepted

## Context

Authenticating against `api.minecraftservices.com` requires an Azure application
whose client ID has been granted access to the Minecraft API. Newly registered
applications are refused until that grant is issued.

Several open source launchers publish their client IDs, since a public OAuth
client cannot keep a secret. Reusing one is common practice. It is also something
those projects ask people not to do: Prism Launcher's build documentation
instructs forks to remove or replace the client ID, and the equivalent request
from MultiMC was contentious enough to contribute to a fork of the project.

The operational risk is concrete. If a borrowed ID is rotated or revoked, every
PhantomMC user loses the ability to log in at the same moment.

## Decision

The client ID is supplied at build time through configuration. It is never a
constant in this repository.

Builds without one are valid and produce an image where Microsoft login is
disabled.

Registering a PhantomMC application and requesting the Minecraft API grant is
tracked separately and should be started early, because approval has lead time.

## Consequences

Rotating the ID is a configuration change and a rebuild rather than a code
change, so recovery from a revocation is fast.

Anyone building from source supplies their own value, which is the same
expectation every other launcher sets.
