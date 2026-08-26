# Failure handling

On an ordinary desktop a crashed game returns you to a desktop. Here the game is
the entire user interface, so a crash drops the user onto a bare console with no
mouse, no window manager and no idea what happened. Handling that well is not
polish, it is a core requirement.

Nothing in this document is implemented yet. It is the design being built
towards in [M8](roadmap.md).

## Supervision

The agent does not exec the JVM and disappear. It runs it as a child and watches
the exit status.

| Exit | Meaning | Action |
| --- | --- | --- |
| 0 | User quit from the in-game menu | Power off cleanly |
| non-zero | Crash, failed mod load, or the compositor died | Show the fallback interface |
| killed by OOM | Heap larger than the machine could back | Fallback interface, with the memory cause named explicitly |

The distinction matters because quitting the game is the normal way to end a
session here. There is no desktop to return to, so a clean exit means power off.

## The fallback interface

A terminal interface drawn on the framebuffer, navigable with the keyboard
alone, since there is no cursor.

```
   ╔══════════════════════════════════════════════════════════╗
   ║                    FATAL EXCEPTION                       ║
   ╠══════════════════════════════════════════════════════════╣
   ║                                                          ║
   ║   The Java virtual machine terminated unexpectedly.      ║
   ║   Exit code 1 after 4 seconds.                           ║
   ║                                                          ║
   ║   Likely cause: a mod failed during initialization.      ║
   ║                                                          ║
   ║     [ Restart Minecraft ]                                ║
   ║     [ Restart without mods ]                             ║
   ║     [ Upload the crash log ]                             ║
   ║     [ Power off ]                                        ║
   ║                                                          ║
   ╚══════════════════════════════════════════════════════════╝
```

Instant to draw, because everything is already in memory.

## Uploading the log

This is the part that actually matters, and it exists because of a constraint
nothing else has. With no persistent storage, the moment the machine loses power
the crash log is gone permanently. A user who cannot get the log off the machine
cannot report the bug, and cannot ask a mod author for help.

So the log goes to `mclo.gs`, which returns a short URL. That URL is rendered as
a QR code directly in the terminal using block characters, and the user
photographs the screen. The log is on the internet before the machine is powered
off.

## The circuit breaker

Restarting after a crash is often correct, because transient failures exist. But
a fatally incompatible mod will crash the same way every time, and blindly
restarting produces an infinite loop of black screens with no way out.

The rule is based on how long the game survived:

| Runtime before crash | Restart offered |
| --- | --- |
| More than 60 seconds | Yes, without limit |
| Under 60 seconds, fewer than 3 times in a row | Yes |
| Under 60 seconds, 3 times in a row | No. The restart option is removed and the cause is stated |

A game that ran for two hours and crashed is a different event from one that
never reached the main menu, and treating them the same is what produces the
loop.

## Safe mode

Restarting without mods launches the same Minecraft version with an empty mods
directory. It answers the first diagnostic question immediately: is this the
game or is this the pack? It also gives a working session to someone who just
wants to play, which is worth more than a correct error message.

## The audio cue

A short tone through ALSA when the fallback interface appears.

This looks like a joke and is partly one, but it is doing real work. If the GPU
pipeline is what failed, the screen may show nothing at all, and a machine that
appears to be doing nothing is indistinguishable from a machine that has hung.
An audible signal tells the user something definite happened. Arcade hardware
did this for the same reason.

## Failures before userspace

Everything above assumes userspace is running. Failures inside the initramfs are
handled differently, because there is no framebuffer interface, no network and
nowhere to send anything. Those drop to a busybox shell with the reason printed
on the console. See [boot.md](boot.md#failure-behaviour).
