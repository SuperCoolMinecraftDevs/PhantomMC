#!/usr/bin/env bash
# Boots the built image under QEMU and asserts that the root filesystem ends up
# in memory with the boot medium released. Runs headless with no KVM so it works
# in CI, which makes it slow but honest.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly REPO_ROOT

ISO="${1:-}"
MEMORY="${PHANTOM_SMOKE_MEMORY:-3072}"
TIMEOUT="${PHANTOM_SMOKE_TIMEOUT:-600}"

if [ -z "$ISO" ]; then
	ISO=$(find "${REPO_ROOT}/out" -maxdepth 1 -name '*.iso' | sort | head -1)
fi
if [ -z "$ISO" ] || [ ! -f "$ISO" ]; then
	echo "no iso found, run os/build.sh first" >&2
	exit 1
fi

LOG=$(mktemp)
trap 'rm -f "$LOG"' EXIT

echo "booting $(basename "$ISO") with ${MEMORY}M, timeout ${TIMEOUT}s"

set +e
timeout "$TIMEOUT" qemu-system-x86_64 \
	-machine q35 \
	-m "$MEMORY" \
	-cdrom "$ISO" \
	-boot d \
	-display none \
	-serial "file:${LOG}" \
	-no-reboot
set -e

assert() {
	if grep -qiE "$1" "$LOG"; then
		echo "  ok: $2"
	else
		echo "  FAIL: $2" >&2
		echo "--- serial log tail ---" >&2
		tail -60 "$LOG" >&2
		exit 1
	fi
}

echo "checking serial output"
assert "found boot medium" "initramfs located the boot medium"
assert "copying [0-9]+ MiB into memory" "root image copied into RAM"
assert "boot medium released" "boot medium unmounted"
assert "root is in memory" "overlay mounted"

echo "smoke test passed"
