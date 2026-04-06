//go:build windows

package util

import "syscall"

const processQueryLimitedInformation = 0x1000

func IsProcessAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(h)
	return true
}
