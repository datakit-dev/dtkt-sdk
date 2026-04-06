//go:build !windows

package util

import "syscall"

func IsProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
