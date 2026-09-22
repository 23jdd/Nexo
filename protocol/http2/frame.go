// Package http2 implements the binary framing layer defined by RFC 9113.
package http2

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	ClientPreface       = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	FrameHeaderSize     = 9
	DefaultMaxFrameSize = 16 * 1024
	MaxFrameSize        = 1<<24 - 1
)

type FrameType uint8

const (
	FrameData         FrameType = 0x0
	FrameHeaders      FrameType = 0x1
	FramePriority     FrameType = 0x2
	FrameRSTStream    FrameType = 0x3
	FrameSettings     FrameType = 0x4
	FramePushPromise  FrameType = 0x5
	FramePing         FrameType = 0x6
	FrameGoAway       FrameType = 0x7
	FrameWindowUpdate FrameType = 0x8
	FrameContinuation FrameType = 0x9
)

type Flags uint8

const (
	FlagEndStream  Flags = 0x1
	FlagAck        Flags = 0x1
	FlagEndHeaders Flags = 0x4
	FlagPadded     Flags = 0x8
	FlagPriority   Flags = 0x20
)

type FrameHeader struct {
	Length   uint32
	Type     FrameType
	Flags    Flags
	StreamID uint32
}

type Frame struct {
	Header  FrameHeader
	Payload []byte
}

func (h FrameHeader) MarshalBinary() ([]byte, error) {
	if h.Length > MaxFrameSize {
		return nil, fmt.Errorf("HTTP/2 frame length %d exceeds %d", h.Length, MaxFrameSize)
	}
	if h.StreamID > 0x7fffffff {
		return nil, fmt.Errorf("invalid HTTP/2 stream ID %d", h.StreamID)
	}
	data := make([]byte, FrameHeaderSize)
	data[0] = byte(h.Length >> 16)
	data[1] = byte(h.Length >> 8)
	data[2] = byte(h.Length)
	data[3] = byte(h.Type)
	data[4] = byte(h.Flags)
	binary.BigEndian.PutUint32(data[5:], h.StreamID)
	return data, nil
}

func (h *FrameHeader) UnmarshalBinary(data []byte) error {
	if len(data) != FrameHeaderSize {
		return fmt.Errorf("HTTP/2 frame header length is %d, want %d", len(data), FrameHeaderSize)
	}
	h.Length = uint32(data[0])<<16 | uint32(data[1])<<8 | uint32(data[2])
	h.Type = FrameType(data[3])
	h.Flags = Flags(data[4])
	h.StreamID = binary.BigEndian.Uint32(data[5:]) & 0x7fffffff
	return nil
}

func ReadFrame(r io.Reader) (*Frame, error) {
	return ReadFrameLimit(r, DefaultMaxFrameSize)
}

func ReadFrameLimit(r io.Reader, maxSize uint32) (*Frame, error) {
	if maxSize == 0 || maxSize > MaxFrameSize {
		return nil, fmt.Errorf("invalid maximum frame size %d", maxSize)
	}
	headerBytes := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return nil, err
	}
	var header FrameHeader
	if err := header.UnmarshalBinary(headerBytes); err != nil {
		return nil, err
	}
	if header.Length > maxSize {
		return nil, fmt.Errorf("HTTP/2 frame length %d exceeds peer limit %d", header.Length, maxSize)
	}
	frame := &Frame{Header: header, Payload: make([]byte, header.Length)}
	if _, err := io.ReadFull(r, frame.Payload); err != nil {
		return nil, err
	}
	if err := frame.Validate(); err != nil {
		return nil, err
	}
	return frame, nil
}

func WriteFrame(w io.Writer, frame *Frame) error {
	if frame == nil {
		return fmt.Errorf("nil HTTP/2 frame")
	}
	if len(frame.Payload) > MaxFrameSize {
		return fmt.Errorf("HTTP/2 frame payload is too large")
	}
	frame.Header.Length = uint32(len(frame.Payload))
	if err := frame.Validate(); err != nil {
		return err
	}
	header, err := frame.Header.MarshalBinary()
	if err != nil {
		return err
	}
	if err := writeAll(w, header); err != nil {
		return err
	}
	return writeAll(w, frame.Payload)
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func (frame *Frame) Validate() error {
	if frame.Header.Length != uint32(len(frame.Payload)) {
		return fmt.Errorf("HTTP/2 frame length is %d, payload length is %d", frame.Header.Length, len(frame.Payload))
	}
	switch frame.Header.Type {
	case FrameSettings:
		if frame.Header.StreamID != 0 {
			return fmt.Errorf("SETTINGS frame has non-zero stream ID")
		}
		if frame.Header.Flags&FlagAck != 0 && len(frame.Payload) != 0 {
			return fmt.Errorf("SETTINGS ACK has a payload")
		}
		if len(frame.Payload)%6 != 0 {
			return fmt.Errorf("SETTINGS payload length is not a multiple of 6")
		}
	case FramePing:
		if frame.Header.StreamID != 0 || len(frame.Payload) != 8 {
			return fmt.Errorf("invalid PING frame")
		}
	case FrameGoAway:
		if frame.Header.StreamID != 0 || len(frame.Payload) < 8 {
			return fmt.Errorf("invalid GOAWAY frame")
		}
	case FrameRSTStream:
		if frame.Header.StreamID == 0 || len(frame.Payload) != 4 {
			return fmt.Errorf("invalid RST_STREAM frame")
		}
	case FrameWindowUpdate:
		if len(frame.Payload) != 4 || binary.BigEndian.Uint32(frame.Payload)&0x7fffffff == 0 {
			return fmt.Errorf("invalid WINDOW_UPDATE frame")
		}
	case FrameData, FrameHeaders:
		if frame.Header.StreamID == 0 {
			return fmt.Errorf("frame type %d requires a stream", frame.Header.Type)
		}
	}
	return nil
}
