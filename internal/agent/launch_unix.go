package agent

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os/exec"
	"syscall"
)

func asExitError(err error, target **exec.ExitError) bool {
	return errors.As(err, target)
}

func signalName(exitErr *exec.ExitError) string {
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return ""
	}
	return status.Signal().String()
}

// offlineUUID reproduces the identity Mojang derives for offline players: an
// MD5 of "OfflinePlayer:<name>", stamped as a version 3 UUID.
func offlineUUID(username string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + username))
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80

	hexed := hex.EncodeToString(sum[:])
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32]
}
