# Building

## Requirements

Debian or Ubuntu, root access, roughly 8 GiB of free disk, and:

```sh
sudo apt-get install --no-install-recommends \
  mmdebstrap squashfs-tools xorriso mtools dosfstools \
  grub-efi-amd64-bin grub-pc-bin qemu-system-x86 shellcheck \
  debian-archive-keyring
```

`debian-archive-keyring` is easy to overlook on an Ubuntu host, where it is
not installed by default. Without it the bootstrap fails with apt reporting
the Debian repository as unsigned rather than saying a key is missing.

Go 1.27 or newer for the agent. Root is required because `mmdebstrap` creates
device nodes and `mksquashfs` preserves ownership.

## Building an image

```sh
sudo make image
```

Around two minutes for the minimal profile on a single core, most of it spent in
`mksquashfs` at compression level 19. Output lands in `out/`.

### Options

```sh
sudo os/build.sh --help
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `--suite` | `trixie` | Debian suite to bootstrap |
| `--mirror` | `deb.debian.org` | Debian mirror. Point at a local one to build faster |
| `--gpu` | `auto` | `auto` for Mesa, `nvidia` for the proprietary variant |
| `--lists` | all five | Comma separated package lists to include |
| `--out` | `out/` | Where the ISO lands |
| `--work` | `build/` | Scratch space |
| `--keep-rootfs` | off | Leave the bootstrapped tree for inspection |

### Fast iteration

When working on the boot chain, the graphics and firmware packages are dead
weight. Skip them:

```sh
sudo os/build.sh --lists base,boot
```

This cuts the build to about two minutes and produces an image that boots to a
console. It is what the smoke test exercises.

### Package lists

| File | Contents |
| --- | --- |
| `packages.base` | Minimum to boot and reach a shell |
| `packages.boot` | Kernel and initramfs tooling |
| `packages.network` | Ethernet, wifi, DNS |
| `packages.graphics` | Mesa, the compositor, XWayland, fonts |
| `packages.firmware` | Firmware blobs, deliberately narrow |

Lines starting with `#` are comments. One package per line.

## Testing

```sh
make smoke
```

Boots the built ISO under QEMU with no KVM, watches the serial console, and
asserts four markers in order:

```
found boot medium on /dev/sr0
copying 173 MiB into memory
boot medium released, the drive can be removed now
root is in memory
```

QEMU is killed as soon as the last marker appears, so a passing run costs only
as long as the boot takes. Roughly 90 seconds under software emulation.

| Variable | Default | Purpose |
| --- | --- | --- |
| `PHANTOM_SMOKE_MEMORY` | 3072 | Guest memory in MiB. Must exceed the image size |
| `PHANTOM_SMOKE_TIMEOUT` | 600 | Seconds before giving up |

To watch a boot interactively:

```sh
qemu-system-x86_64 -machine q35 -m 4096 -cdrom out/phantommc-auto.iso -boot d
```

Pick the verbose entry in the GRUB menu to see the full kernel log, or the third
entry to keep the boot medium mounted for inspection.

## Writing to a USB stick

Any of these work:

```sh
sudo dd if=out/phantommc-auto.iso of=/dev/sdX bs=4M status=progress oflag=sync
```

The ISO is a hybrid image, so `dd`, Rufus, BalenaEtcher and Raspberry Pi Imager
all handle it.

Because the medium is located by probing for the payload rather than by label,
copying the ISO contents onto an existing FAT32 partition also works, and needs
no flashing tool at all. See [boot.md](boot.md#why-the-medium-is-found-by-probing-not-by-label).

## Go targets

```sh
make build        # binaries into out/, version stamped from git describe
make test         # with the race detector
make lint         # go vet, gofmt check, shellcheck
```

## Verifying microsoft sign in

```sh
./out/phantomd -signin-test -client-id <application-id>
```

Runs only the sign in chain, prints a device code to approve on a phone, and
reports the resulting username and uuid. No image, no boot, no downloads. See
[authentication.md](authentication.md).

## Running the agent by hand

The agent runs outside an image, which makes iterating on manifest handling
quick:

```sh
./out/phantomd -manifest docs/examples/manifest.json -dry-run
```

It reports the plan without touching anything.

## Troubleshooting

**`no medium labelled` or a busybox prompt at boot.** The initramfs could not
find a device carrying the payload. The message lists every block device it saw.
Usually a missing storage driver.

**`Disk full` during `build_efi_image`.** The GRUB payload outgrew the EFI
partition. It is sized from the artifact now, so this should not recur, but
adding GRUB modules is the thing that would cause it.

**QEMU boots to GRUB then hangs.** Almost always insufficient guest memory. The
image has to fit in RAM with room to spare. Raise `PHANTOM_SMOKE_MEMORY`.

**The build fails fetching packages.** Usually a suite name that no longer
exists or a mirror that is out of date. Try `--mirror`.
