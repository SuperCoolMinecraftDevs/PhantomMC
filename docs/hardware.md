# Hardware support

## Requirements

| | Minimum | Comfortable |
| --- | --- | --- |
| RAM | 4 GiB | 8 GiB or more |
| CPU | x86-64 with SSE4.2 | Any modern desktop or laptop part |
| GPU | Anything Mesa drives | Any AMD or Intel part from the last decade |
| Boot | UEFI or legacy BIOS | UEFI |
| Storage | None | None |

4 GiB is genuinely tight: the operating system takes around 400 MiB, and what
remains has to hold the game, its assets and the JVM heap. It will run vanilla.
It will not run a large modpack.

There is deliberately no storage requirement. A machine with no drive at all, or
with a dead one, is a supported configuration and one of the reasons the project
exists.

## GPUs

| Vendor | Driver | Expectation |
| --- | --- | --- |
| AMD | Mesa radeonsi, open source, in tree | Works with no configuration. The best case |
| Intel | Mesa iris, open source, in tree | Works with no configuration |
| Nvidia, Turing and newer | Open kernel modules | Separate image variant, see below |
| Nvidia, pre Turing | Proprietary driver only | Separate image variant, expected to be awkward |
| Nvidia, any, on Nouveau | In tree | Boots, performance is not competitive |

AMD and Intel are the easy path and the reason `auto` is the default. The
drivers ship with the kernel and with Mesa, they need no configuration, and
Minecraft's OpenGL path on Mesa is well trodden.

Nvidia is harder for a structural reason: the kernel module has to match the
running kernel exactly. In an image with a pinned kernel and no compiler at
runtime, that means building the module at image build time and shipping a
separate variant. `--gpu nvidia` selects it. Expect this milestone to take
longer than it looks.

## Firmware

Wifi and modern GPUs need firmware blobs loaded at initialization. Debian's
complete firmware set is over a gigabyte uncompressed, which is more than the
entire rest of the image, so `os/config/packages.firmware` selects only the
families likely to be present.

Currently selected: AMD graphics, Intel wifi, Realtek, and the general
miscellaneous set. Broadcom wifi is a known gap and affects a lot of older
laptops.

If your hardware needs firmware that is not in the image, it will fail at boot
with a kernel message naming the missing file. That is a bug worth reporting,
since the list is meant to grow.

## Testing

Two levels, and they check different things.

QEMU under software emulation, run by `make smoke` and by CI, exercises the boot
chain: finding the medium, copying to RAM, releasing the medium, stacking the
overlay, reaching userspace. It runs without KVM so it works on any CI runner.
It says nothing at all about graphics, because there is no real GPU involved.

Real hardware is the only way to test the parts that matter for actually
playing. There is no substitute and no plan to pretend otherwise.

## Java runtimes

| Minecraft | Java required | Shipped in the image |
| --- | --- | --- |
| 1.21.x | 21 | yes |
| 26.x | 25 | yes |
| 1.17 to 1.20.x | 17 | no, fetched at boot |
| 1.16 and older | 8 | no, fetched at boot |

Both shipped runtimes together cost roughly 35 MB compressed, which is a good
trade against downloading a runtime on every boot for the versions most people
play.

## Known gaps

- Nvidia variant is not built
- Broadcom wifi firmware is not included
- WPA Enterprise is not supported
- Secure Boot is not supported, so it has to be disabled. Signing would require
  a key in the shim chain, which is a project in itself
- 32 bit x86 and ARM are not targets
