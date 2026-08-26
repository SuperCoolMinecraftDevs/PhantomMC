#!/usr/bin/env bash
# Boots the built image under QEMU and asserts that the root filesystem ends up
# in memory with the boot medium released. Runs headless with no KVM so it works
# in CI, which makes it slow but honest.
#
# QEMU is killed as soon as the last marker appears rather than being left to
# run out the clock, so a passing run costs only as long as the boot takes.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly REPO_ROOT

ISO="${1:-}"
MEMORY="${PHANTOM_SMOKE_MEMORY:-3072}"
TIMEOUT="${PHANTOM_SMOKE_TIMEOUT:-600}"

readonly MARKERS=(
	"found boot medium|initramfs located the boot medium"
	"copying [0-9]+ MiB into memory|root image copied into RAM"
	"boot medium released|boot medium unmounted"
	"root is in memory|overlay mounted"
)

if [ -z "$ISO" ]; then
	ISO=$(find "${REPO_ROOT}/out" -maxdepth 1 -name '*.iso' | sort | head -1)
fi
if [ -z "$ISO" ] || [ ! -f "$ISO" ]; then
	echo "no iso found, run os/build.sh first" >&2
	exit 1
fi

LOG=$(mktemp)
QEMU_PID=""

cleanup() {
	if [ -n "$QEMU_PID" ] && kill -0 "$QEMU_PID" 2>/dev/null; then
		kill "$QEMU_PID" 2>/dev/null || true
		wait "$QEMU_PID" 2>/dev/null || true
	fi
	rm -f "$LOG"
}
trap cleanup EXIT

echo "booting $(basename "$ISO") with ${MEMORY}M, giving it ${TIMEOUT}s"

qemu-system-x86_64 \
	-machine q35 \
	-m "$MEMORY" \
	-cdrom "$ISO" \
	-boot d \
	-display none \
	-serial "file:${LOG}" \
	-no-reboot &
QEMU_PID=$!

final_pattern="${MARKERS[-1]%%|*}"
waited=0
while [ "$waited" -lt "$TIMEOUT" ]; do
	if grep -qiE "$final_pattern" "$LOG" 2>/dev/null; then
		break
	fi
	if ! kill -0 "$QEMU_PID" 2>/dev/null; then
		echo "qemu exited before the guest finished booting" >&2
		break
	fi
	sleep 2
	waited=$((waited + 2))
done

echo "checking serial output after ${waited}s"

failed=0
for marker in "${MARKERS[@]}"; do
	pattern="${marker%%|*}"
	description="${marker#*|}"
	if grep -qiE "$pattern" "$LOG"; then
		echo "  ok: ${description}"
	else
		echo "  FAIL: ${description}" >&2
		failed=1
	fi
done

if [ "$failed" -ne 0 ]; then
	echo "--- serial log tail ---" >&2
	tail -80 "$LOG" >&2
	exit 1
fi

echo "smoke test passed"
