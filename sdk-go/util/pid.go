package util

import (
	"fmt"
	"os"
	"strconv"
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

func ReadPID(path string) (int, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(string(b))
	if err != nil {
		return 0, false
	}

	if !IsProcessAlive(pid) {
		return 0, false
	}

	return pid, true
}

func WritePID(path string) (PID, error) {
	pf, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			// pidFile exists; read it
			b, err := os.ReadFile(path)
			if err == nil {
				if oldPid, err := strconv.Atoi(string(b)); err == nil {
					if IsProcessAlive(oldPid) {
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
