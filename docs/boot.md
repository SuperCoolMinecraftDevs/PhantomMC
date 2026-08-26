# Boot sequence

Every stage, in order, with the code that implements it.

```
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 1  FIRMWARE                                                          │
 │    UEFI reads the FAT partition and runs EFI/BOOT/BOOTX64.EFI.       │
 │    Legacy BIOS reads the El Torito record instead.                   │
 │    Implemented by: os/build.sh build_efi_image, build_bios_image     │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 2  GRUB                                                              │
 │    Three menu entries: normal, verbose, and keep the medium mounted. │
 │    Loads the kernel with boot=phantom on the command line.           │
 │    Implemented by: os/build.sh write_grub_config                     │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 3  KERNEL AND INITRAMFS                                              │
 │    Debian kernel with MODULES=most, so storage and USB controllers   │
 │    are available without knowing the target hardware in advance.     │
 │    boot=phantom selects our mountroot implementation.                │
 │    Implemented by: os/initramfs/hooks/phantom                        │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 4  FINDING THE MEDIUM                              ◀── the hard part │
 │    Load loop, squashfs, overlay, isofs, vfat, ext4. Wait for udev.   │
 │    Probe every block device: mount read only, look for the payload,  │
 │    unmount if absent. Retry for 45 seconds.                          │
 │    Implemented by: os/initramfs/scripts/phantom find_medium          │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 5  COPY INTO RAM                                                     │
 │    tmpfs sized to the image plus 8 MiB slack. Copy, then sync.       │
 │    Implemented by: os/initramfs/scripts/phantom copy_to_ram          │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 6  RELEASE THE MEDIUM                                                │
 │    Unmount. If this fails the boot aborts rather than continuing in  │
 │    a state where the stick is silently still required.               │
 │    The stick can be removed from here on.                            │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 7  STACK THE OVERLAY                                                 │
 │    Loop mount the squashfs read only as the lower layer.             │
 │    tmpfs upper layer. overlayfs onto ${rootmnt}. switch_root.        │
 └──────────────────────────────────────────────────────────────────────┘
                                   ▼
 ┌──────────────────────────────────────────────────────────────────────┐
 │ 8  USERSPACE                     ◀── everything below is not built   │
 │    systemd starts. Network comes up. seatd takes the seat.           │
 │    phantomd resolves the manifest and fetches the game.              │
 │    cage starts, Minecraft launches full screen.                      │
 └──────────────────────────────────────────────────────────────────────┘
```

## Why the medium is found by probing, not by label

The first implementation looked for `/dev/disk/by-label/PHANTOMMC` and failed
immediately in testing. Two reasons to abandon that approach entirely:

Labels are not reliable. A stick written with `dd` or a flashing tool carries
our label. A stick populated by copying files onto an existing FAT32 partition
keeps whatever label it already had, and that copy based flow is one we want to
support because it removes the need for any flashing tool at all.

Probing is not expensive. Mounting a handful of block devices read only and
checking for one file costs milliseconds. The label lookup is kept as a fast
path that short circuits the scan when it happens to work.

The consequence is that the payload path is the real identifier. Change
`PHANTOM_IMAGE` and you change what counts as a PhantomMC medium.

## Kernel command line

| Parameter | Effect |
| --- | --- |
| `boot=phantom` | Required. Selects our `mountroot` over the default |
| `phantom.label=NAME` | Fast path label to try before scanning. Default `PHANTOMMC` |
| `phantom.image=PATH` | Payload path relative to the medium root. Default `/phantom/root.squashfs` |
| `phantom.keepmedium` | Do not unmount the boot medium. For debugging only |
| `console=ttyS0,115200` | Mirror the console to serial. Used by the smoke test |

## Failure behaviour

Every failure in `mountroot` calls `panic`, which drops to a busybox shell with
the reason printed on the console. This is deliberate. At this stage there is no
graphical output available and no way to report anything to a server, so the
only useful thing to do is stop somewhere a human can look around. The friendly
error handling described in [failure-handling.md](failure-handling.md) applies
after userspace is running, not here.
