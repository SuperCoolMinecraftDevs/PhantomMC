#!/usr/bin/env bash
# Builds the PhantomMC boot image. Must run as root because mmdebstrap and
# mksquashfs need to create device nodes and preserve ownership.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly REPO_ROOT
readonly CONFIG_DIR="${REPO_ROOT}/os/config"
readonly OVERLAY_DIR="${REPO_ROOT}/os/overlay"
readonly INITRAMFS_DIR="${REPO_ROOT}/os/initramfs"

SUITE="trixie"
MIRROR="http://deb.debian.org/debian"
GPU="auto"
LISTS="base,boot,network,graphics,java,firmware"
LABEL="PHANTOMMC"
KEYRING="/usr/share/keyrings/debian-archive-keyring.gpg"
WORK="${REPO_ROOT}/build"
OUT="${REPO_ROOT}/out"
ROOTFS=""
KEEP_ROOTFS=0

usage() {
	cat <<-EOF
		Usage: $(basename "$0") [options]

		  --suite NAME    Debian suite to bootstrap (default: ${SUITE})
		  --mirror URL    Debian mirror (default: ${MIRROR})
		  --gpu VENDOR    auto or nvidia (default: ${GPU})
		  --keyring PATH  Debian archive keyring (default: ${KEYRING})
		  --lists NAMES   Comma separated package lists (default: ${LISTS})
		  --out DIR       Output directory (default: ${OUT})
		  --work DIR      Scratch directory (default: ${WORK})
		  --keep-rootfs   Do not delete the bootstrapped tree on exit
		  -h, --help      This message
	EOF
}

log() { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
die() {
	printf '\033[1;31merror:\033[0m %s\n' "$*" >&2
	exit 1
}

parse_args() {
	while [ $# -gt 0 ]; do
		case "$1" in
		--suite) SUITE="$2"; shift 2 ;;
		--mirror) MIRROR="$2"; shift 2 ;;
		--gpu) GPU="$2"; shift 2 ;;
		--keyring) KEYRING="$2"; shift 2 ;;
		--lists) LISTS="$2"; shift 2 ;;
		--out) OUT="$2"; shift 2 ;;
		--work) WORK="$2"; shift 2 ;;
		--keep-rootfs) KEEP_ROOTFS=1; shift ;;
		-h | --help) usage; exit 0 ;;
		*) die "unknown option: $1" ;;
		esac
	done

	case "$GPU" in
	auto | nvidia) ;;
	*) die "--gpu must be auto or nvidia" ;;
	esac
}

require_root() {
	[ "$(id -u)" -eq 0 ] || die "must run as root"
}

require_tools() {
	local missing=()
	for tool in mmdebstrap mksquashfs xorriso grub-mkstandalone grub-mkimage mkfs.vfat mcopy; do
		command -v "$tool" >/dev/null 2>&1 || missing+=("$tool")
	done
	[ ${#missing[@]} -eq 0 ] || die "missing tools: ${missing[*]}"

	# Bootstrapping Debian from a non-Debian host fails at the signature check
	# unless the archive keyring is present, and apt reports it as an unsigned
	# repository rather than a missing key, which is not an obvious diagnosis.
	if [ ! -f "$KEYRING" ]; then
		die "no Debian archive keyring at ${KEYRING}, install debian-archive-keyring"
	fi
}

package_list() {
	local lists
	IFS="," read -r -a lists <<<"$LISTS"
	[ "$GPU" = "nvidia" ] && lists+=(nvidia)

	for name in "${lists[@]}"; do
		local file="${CONFIG_DIR}/packages.${name}"
		[ -f "$file" ] || die "missing package list: ${file}"
		grep -vE '^\s*(#|$)' "$file"
	done | sort -u | paste -sd,
}

bootstrap() {
	log "bootstrapping ${SUITE} into ${ROOTFS}"
	# Excluding man pages removes the directories they live in, and packages
	# that register an alternative for a man page fail in postinst when the
	# target directory is missing. openjdk is one of them.
	mmdebstrap \
		--variant=minbase \
		--keyring="$KEYRING" \
		--components="main,contrib,non-free-firmware" \
		--include="$(package_list)" \
		--aptopt='Apt::Install-Recommends "false"' \
		--dpkgopt='path-exclude=/usr/share/man/*' \
		--dpkgopt='path-exclude=/usr/share/doc/*' \
		--dpkgopt='path-exclude=/usr/share/locale/*' \
		--essential-hook='mkdir -p "$1/usr/share/man/man1"' \
		--customize-hook="copy-in ${INITRAMFS_DIR}/hooks/phantom /etc/initramfs-tools/hooks" \
		--customize-hook="copy-in ${INITRAMFS_DIR}/scripts/phantom /etc/initramfs-tools/scripts" \
		--customize-hook="chroot \$1 chmod 0755 /etc/initramfs-tools/hooks/phantom /etc/initramfs-tools/scripts/phantom" \
		"$SUITE" "$ROOTFS" "$MIRROR"
}

configure() {
	log "applying overlay"
	if [ -d "$OVERLAY_DIR" ] && [ -n "$(ls -A "$OVERLAY_DIR")" ]; then
		cp -a "${OVERLAY_DIR}/." "${ROOTFS}/"
	fi

	if [ -x "${OUT}/phantomd" ]; then
		install -m 0755 "${OUT}/phantomd" "${ROOTFS}/usr/bin/phantomd"
	else
		log "no agent binary in ${OUT}, image will boot without phantomd"
	fi

	echo "phantom" >"${ROOTFS}/etc/hostname"
	printf 'PhantomMC \\n \\l\n\n' >"${ROOTFS}/etc/issue"

	chroot "$ROOTFS" useradd --system --create-home \
		--home-dir /var/lib/phantom \
		--shell /usr/sbin/nologin \
		--groups video,input,render \
		phantom

	chroot "$ROOTFS" systemctl enable seatd.service
	chroot "$ROOTFS" systemctl enable phantom-agent.service
	chroot "$ROOTFS" systemctl enable phantom-session.service
	chroot "$ROOTFS" systemctl set-default graphical.target

	# apt is a build time tool only. Nothing at runtime may use it.
	chroot "$ROOTFS" apt-get clean
	rm -rf "${ROOTFS:?}/var/lib/apt/lists/"* "${ROOTFS:?}/var/cache/apt/"*
}

build_initramfs() {
	log "regenerating initramfs"
	local version
	version=$(basename "$(find "${ROOTFS}/lib/modules" -maxdepth 1 -mindepth 1 -type d | head -1)")
	[ -n "$version" ] || die "no kernel found in rootfs"

	echo "MODULES=most" >"${ROOTFS}/etc/initramfs-tools/conf.d/phantom"
	echo "COMPRESS=zstd" >>"${ROOTFS}/etc/initramfs-tools/conf.d/phantom"

	mount --bind /dev "${ROOTFS}/dev"
	chroot "$ROOTFS" update-initramfs -c -k "$version"
	umount "${ROOTFS}/dev"

	mkdir -p "${WORK}/iso/boot"
	cp "${ROOTFS}/boot/vmlinuz-${version}" "${WORK}/iso/boot/vmlinuz"
	cp "${ROOTFS}/boot/initrd.img-${version}" "${WORK}/iso/boot/initrd.img"
}

build_squashfs() {
	log "compressing root filesystem"
	mkdir -p "${WORK}/iso/phantom"

	# The kernel and initramfs already live on the medium. Shipping a second copy
	# inside the squashfs costs memory for the whole session and buys nothing.
	mksquashfs "$ROOTFS" "${WORK}/iso/phantom/root.squashfs" \
		-comp zstd -Xcompression-level 19 \
		-b 1M -noappend -no-progress \
		-e boot/vmlinuz-\* boot/initrd.img-\*

	log "root filesystem is $(du -h "${WORK}/iso/phantom/root.squashfs" | cut -f1)"
}

write_grub_config() {
	mkdir -p "${WORK}/iso/boot/grub"
	cat >"${WORK}/iso/boot/grub/grub.cfg" <<-EOF
		set default=0
		set timeout=3

		menuentry "PhantomMC" {
			linux /boot/vmlinuz boot=phantom phantom.label=${LABEL} console=tty0 console=ttyS0,115200 quiet loglevel=3
			initrd /boot/initrd.img
		}

		menuentry "PhantomMC (verbose)" {
			linux /boot/vmlinuz boot=phantom phantom.label=${LABEL} console=tty0 console=ttyS0,115200
			initrd /boot/initrd.img
		}

		menuentry "PhantomMC (keep boot medium mounted)" {
			linux /boot/vmlinuz boot=phantom phantom.label=${LABEL} phantom.keepmedium
			initrd /boot/initrd.img
		}
	EOF
}

build_efi_image() {
	log "building efi bootloader"
	local staging="${WORK}/efi"
	rm -rf "$staging"
	mkdir -p "${staging}/EFI/BOOT"

	grub-mkstandalone \
		--format=x86_64-efi \
		--output="${staging}/EFI/BOOT/BOOTX64.EFI" \
		--modules="part_gpt part_msdos fat iso9660 normal linux search search_label configfile" \
		"boot/grub/grub.cfg=${WORK}/iso/boot/grub/grub.cfg"

	# grub-mkstandalone embeds a memdisk, so the payload is several megabytes and
	# its size moves with the module list. Size the partition from the artefact
	# rather than guessing.
	local esp payload_kb esp_kb
	esp="${WORK}/iso/boot/grub/efi.img"
	payload_kb=$(( ($(stat -c %s "${staging}/EFI/BOOT/BOOTX64.EFI") + 1023) / 1024 ))
	esp_kb=$(( payload_kb + 1024 ))
	esp_kb=$(( (esp_kb + 31) / 32 * 32 ))

	rm -f "$esp"
	mkfs.vfat -C -n PHANTOMEFI "$esp" "$esp_kb" >/dev/null
	mmd -i "$esp" ::/EFI ::/EFI/BOOT
	mcopy -i "$esp" "${staging}/EFI/BOOT/BOOTX64.EFI" ::/EFI/BOOT/
	log "efi partition is ${esp_kb} KiB for a ${payload_kb} KiB payload"
}

build_bios_image() {
	log "building bios bootloader"
	local core="${WORK}/core.img"
	grub-mkimage \
		--format=i386-pc \
		--output="$core" \
		--prefix="/boot/grub" \
		biosdisk iso9660 part_msdos normal linux search configfile

	cat /usr/lib/grub/i386-pc/cdboot.img "$core" >"${WORK}/iso/boot/grub/bios.img"
}

build_iso() {
	log "assembling iso"
	mkdir -p "$OUT"
	local iso="${OUT}/phantommc-${GPU}.iso"

	xorriso -as mkisofs \
		-iso-level 3 \
		-volid "$LABEL" \
		-full-iso9660-filenames \
		-eltorito-boot boot/grub/bios.img \
		-no-emul-boot -boot-load-size 4 -boot-info-table \
		--eltorito-catalog boot/grub/boot.cat \
		--grub2-boot-info \
		--grub2-mbr /usr/lib/grub/i386-pc/boot_hybrid.img \
		-eltorito-alt-boot \
		-e boot/grub/efi.img \
		-no-emul-boot \
		-append_partition 2 0xef "${WORK}/iso/boot/grub/efi.img" \
		-output "$iso" \
		"${WORK}/iso"

	log "wrote ${iso} ($(du -h "$iso" | cut -f1))"
}

cleanup() {
	if [ -n "$ROOTFS" ] && mountpoint -q "${ROOTFS}/dev"; then
		umount "${ROOTFS}/dev" || true
	fi
	if [ "$KEEP_ROOTFS" -eq 0 ] && [ -n "$ROOTFS" ] && [ -d "$ROOTFS" ]; then
		rm -rf "$ROOTFS"
	fi
}

main() {
	parse_args "$@"
	require_root
	require_tools

	ROOTFS="${WORK}/rootfs"
	trap cleanup EXIT

	rm -rf "${WORK:?}/iso" "$ROOTFS"
	mkdir -p "$WORK" "$OUT"

	bootstrap
	configure
	build_initramfs
	build_squashfs
	write_grub_config
	build_efi_image
	build_bios_image
	build_iso
}

main "$@"
