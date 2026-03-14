package util

import (
	"bufio"
	"io"
	"sync"
)

// SyncScanner wraps an bufio.Scanner and allows pausing/resuming reads.
type SyncScanner struct {
	scanner *bufio.Scanner
	mu      sync.Mutex
	cond    *sync.Cond
	paused  bool
}

// NewSyncScanner creates a new PausableReader.
func NewSyncScanner(r io.Reader) *SyncScanner {
	s := &SyncScanner{
		scanner: bufio.NewScanner(r),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *SyncScanner) Buffer(buf []byte, max int) {
	s.mu.Lock()
	for s.paused {
		s.cond.Wait() // Wait while paused
	}
	s.mu.Unlock()

	s.scanner.Buffer(buf, max)
}

func (s *SyncScanner) Bytes() []byte {
	s.mu.Lock()
	for s.paused {
		s.cond.Wait() // Wait while paused
	}
	s.mu.Unlock()

	return s.scanner.Bytes()
}

func (s *SyncScanner) Err() error {
	s.mu.Lock()
	for s.paused {
		s.cond.Wait() // Wait while paused
	}
	s.mu.Unlock()

	return s.scanner.Err()
}

func (s *SyncScanner) Scan() bool {
	s.mu.Lock()
	for s.paused {
		s.cond.Wait() // Wait while paused
	}
	s.mu.Unlock()

	return s.scanner.Scan()
}

func (s *SyncScanner) Split(split bufio.SplitFunc) {
	s.mu.Lock()
	for s.paused {
		s.cond.Wait() // Wait while paused
	}
	s.mu.Unlock()

	s.scanner.Split(split)
}

func (s *SyncScanner) Text() string {
	s.mu.Lock()
	for s.paused {
		s.cond.Wait() // Wait while paused
	}
	s.mu.Unlock()

	return s.scanner.Text()
}

// Pause pauses the scanner.
func (s *SyncScanner) Pause() {
	s.mu.Lock()
	s.paused = true
	s.mu.Unlock()
}

// Resume resumes the scanner.
func (s *SyncScanner) Resume() {
	s.mu.Lock()
	s.paused = false
	s.cond.Broadcast()
	s.mu.Unlock()
}
