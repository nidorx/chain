package pubsub

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// ============================================================================
// Internal Helper Functions
// ============================================================================

// bytesBuffer is a simple buffer for building messages.
type bytesBuffer struct {
	buf bytes.Buffer
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

func (b *bytesBuffer) WriteByte(c byte) error {
	return b.buf.WriteByte(c)
}

func (b *bytesBuffer) WriteString(s string) (n int, err error) {
	return b.buf.WriteString(s)
}

func (b *bytesBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

// binaryBigEndianPutUint32 writes a uint32 in big-endian order.
func binaryBigEndianPutUint32(buf []byte, v uint32) {
	binary.BigEndian.PutUint32(buf, v)
}

// binaryBigEndianUint32 reads a uint32 in big-endian order.
func binaryBigEndianUint32(buf []byte) uint32 {
	return binary.BigEndian.Uint32(buf)
}

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// errNew creates a new error with the given message.
func errNew(msg string) error {
	return errors.New(msg)
}

// errJoin joins multiple errors, returning nil if all are nil.
func errJoin(errs ...error) error {
	var nonNil []error
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}
	if len(nonNil) == 0 {
		return nil
	}
	if len(nonNil) == 1 {
		return nonNil[0]
	}
	return errors.Join(nonNil...)
}

// errFmt creates a formatted error.
func errFmt(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// minCompressionSize is the threshold below which compression is skipped.
// LZW compression has overhead that exceeds benefits for small messages.
// This constant can be adjusted based on performance testing.
const minCompressionSize = 128 // bytes
