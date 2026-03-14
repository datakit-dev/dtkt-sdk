package util

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

type (
	pidFile struct {
		*os.File
	}
	PID interface{ Unlock() error }
)

func (p *pidFile) Unlock() error {
	path := p.Name()
	if err := p.Close(); err != nil {
		return err
	}

	return RemovePID(path)
}

// ReadPID reads the PID from the file and checks if the process is still running.
// Returns (pid, true) if the PID file exists and the process is alive.
// Returns (0, false) if the file doesn't exist, can't be read, or the process is not running.
// Note: Uses signal 0 to check process existence without affecting it.
func ReadPID(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, false
	}

	// Check if process is alive using signal 0 (null signal - doesn't affect the process)
	if err := syscall.Kill(pid, 0); err != nil {
		return 0, false
	}

	return pid, true
}

// IsProcessAlive checks if a process with the given PID is running.
// Uses signal 0 which doesn't affect the process, only checks its existence.
func IsProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func WritePID(path string) (PID, error) {
	pf, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			// pidFile exists; read it
			b, err := os.ReadFile(path)
			if err == nil {
				if oldPid, err := strconv.Atoi(string(b)); err == nil {
					// Check if process is alive using signal 0 (doesn't affect the process)
					if err := syscall.Kill(oldPid, 0); err == nil {
						return nil, fmt.Errorf("daemon already running with PID: %d", oldPid)
					}
					// Process not alive; remove stale pidFile and retry
					_ = RemovePID(path)
					return WritePID(path)
				}
			}
			return nil, fmt.Errorf("pidFile exists but cannot read PID: %w", err)
		}
		return nil, fmt.Errorf("failed to create pidFile: %w", err)
	}

	_, err = fmt.Fprintf(pf, "%d", os.Getpid())
	if err != nil {
		return nil, fmt.Errorf("failed to write pidFile: %w", err)
	}

	return &pidFile{File: pf}, nil
}

func RemovePID(path string) error {
	return os.Remove(path)
}
