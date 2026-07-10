package secret

import (
	"encoding/binary"
	"fmt"
	"io"
)

const maxFrameSize = 16 * 1024 * 1024

func writeFrame(w io.Writer, data []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write frame length: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write frame data: %w", err)
	}
	return nil
}

func readFrame(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err // io.EOF/io.ErrUnexpectedEOF пробрасываются как есть — сигнал конца потока
	}
	size := binary.BigEndian.Uint32(lenBuf[:])
	if size > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d bytes", size)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read frame data: %w", err)
	}
	return data, nil
}
