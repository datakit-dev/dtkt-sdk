package util

import (
	"io"
)

var _ io.ReadWriteCloser = (*IOPipe)(nil)

// IOPipe wraps an [io.Pipe] and implements [io.ReadWriteCloser].
// Reads consume data written to the same pipe, making it suitable as a
// drop-in for stdin/stdout/stderr buffers that require proper blocking
// semantics and goroutine safety.
//
// Close closes both ends of the pipe. The write end is closed first so that
// any blocked or future Read calls receive [io.EOF] rather than
// [io.ErrClosedPipe].
type IOPipe struct {
	r *io.PipeReader
	w *io.PipeWriter
}

// NewIOPipe returns an [IOPipe] backed by a new [io.Pipe].
func NewIOPipe() *IOPipe {
	r, w := io.Pipe()
	return &IOPipe{r: r, w: w}
}

// Read reads from the pipe. It blocks until data is available, the pipe is
// closed, or the context of the associated writer is cancelled.
func (p *IOPipe) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

// Write writes to the pipe. It blocks until all bytes have been consumed by a
// reader or the read end is closed.
func (p *IOPipe) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

// Close closes the pipe. Reads after Close return [io.EOF]. Writes after
// Close return [io.ErrClosedPipe].
//
// Only the write end is explicitly closed. Closing the write end is sufficient
// to unblock any in-progress reads (they receive [io.EOF]) and to close the
// pipe's internal done channel, which also unblocks any blocked writes.
// Closing the read end as well would cause reads to receive [io.ErrClosedPipe]
// instead of [io.EOF] due to how [io.Pipe] resolves close errors.
func (p *IOPipe) Close() error {
	return p.w.Close()
}

// CloseWithError closes the pipe with a specific error. Readers will receive
// err instead of [io.EOF]. If err is nil, [io.EOF] is used. Writes after
// CloseWithError return [io.ErrClosedPipe].
func (p *IOPipe) CloseWithError(err error) error {
	return p.w.CloseWithError(err)
}
