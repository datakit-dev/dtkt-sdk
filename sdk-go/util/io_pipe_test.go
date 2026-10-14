package util

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// TestIOPipe_ImplementsReadWriteCloser verifies the interface at compile time.
func TestIOPipe_ImplementsReadWriteCloser(t *testing.T) {
	var _ io.ReadWriteCloser = (*IOPipe)(nil)
}

func TestIOPipe_WriteAndRead(t *testing.T) {
	p := NewIOPipe()

	want := []byte("hello pipe")
	go func() {
		if _, err := p.Write(want); err != nil {
			t.Errorf("Write: %v", err)
		}
	}()

	got := make([]byte, len(want))
	n, err := p.Read(got)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != len(want) || string(got[:n]) != string(want) {
		t.Errorf("got %q, want %q", got[:n], want)
	}
	p.Close()
}

func TestIOPipe_ReadBlocksUntilWrite(t *testing.T) {
	p := NewIOPipe()
	defer p.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4)
		if _, err := p.Read(buf); err != nil {
			t.Errorf("Read: %v", err)
		}
	}()

	// Give the reader goroutine time to block before we write.
	time.Sleep(20 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("Read returned before Write — should have blocked")
	default:
	}

	p.Write([]byte("data")) //nolint:errcheck

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Read did not unblock after Write")
	}
}

func TestIOPipe_CloseDeliversEOFToReader(t *testing.T) {
	p := NewIOPipe()

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 4)
		_, err := p.Read(buf)
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) {
			t.Errorf("Read after Close got %v, want io.EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Read did not unblock after Close")
	}
}

func TestIOPipe_WriteAfterCloseReturnsError(t *testing.T) {
	p := NewIOPipe()
	p.Close()

	_, err := p.Write([]byte("after close"))
	if err == nil {
		t.Fatal("expected error writing to closed pipe, got nil")
	}
}

func TestIOPipe_CloseWithError(t *testing.T) {
	p := NewIOPipe()
	sentinel := errors.New("custom error")

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 4)
		_, err := p.Read(buf)
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	p.CloseWithError(sentinel) //nolint:errcheck

	select {
	case err := <-done:
		if !errors.Is(err, sentinel) {
			t.Errorf("Read got %v, want %v", err, sentinel)
		}
	case <-time.After(time.Second):
		t.Fatal("Read did not unblock after CloseWithError")
	}
}

func TestIOPipe_CloseWithErrorNilUsesEOF(t *testing.T) {
	p := NewIOPipe()

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 4)
		_, err := p.Read(buf)
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	p.CloseWithError(nil) //nolint:errcheck

	select {
	case err := <-done:
		if !errors.Is(err, io.EOF) {
			t.Errorf("Read got %v, want io.EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Read did not unblock after CloseWithError(nil)")
	}
}

func TestIOPipe_ConcurrentWriters(t *testing.T) {
	p := NewIOPipe()

	const writers = 8
	const chunkSize = 256
	var wg sync.WaitGroup
	wg.Add(writers)

	for range writers {
		go func() {
			defer wg.Done()
			data := make([]byte, chunkSize)
			if _, err := p.Write(data); err != nil {
				t.Errorf("concurrent Write: %v", err)
			}
		}()
	}

	// Read all expected bytes.
	total := 0
	go func() {
		wg.Wait()
		p.Close()
	}()

	buf := make([]byte, 512)
	for {
		n, err := p.Read(buf)
		total += n
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
	}

	if total != writers*chunkSize {
		t.Errorf("read %d bytes, want %d", total, writers*chunkSize)
	}
}

func TestIOPipe_MultipleCloseIdempotent(t *testing.T) {
	p := NewIOPipe()
	if err := p.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close: io.Pipe returns io.ErrClosedPipe on double-close; we just
	// verify it doesn't panic.
	_ = p.Close()
}
