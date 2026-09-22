package http2

import (
	"encoding/binary"
	"fmt"
)

type SettingID uint16

const (
	SettingHeaderTableSize      SettingID = 0x1
	SettingEnablePush           SettingID = 0x2
	SettingMaxConcurrentStreams SettingID = 0x3
	SettingInitialWindowSize    SettingID = 0x4
	SettingMaxFrameSize         SettingID = 0x5
	SettingMaxHeaderListSize    SettingID = 0x6
)

type Setting struct {
	ID    SettingID
	Value uint32
}

type ErrorCode uint32

const (
	ErrorNo                 ErrorCode = 0x0
	ErrorProtocol           ErrorCode = 0x1
	ErrorInternal           ErrorCode = 0x2
	ErrorFlowControl        ErrorCode = 0x3
	ErrorSettingsTimeout    ErrorCode = 0x4
	ErrorStreamClosed       ErrorCode = 0x5
	ErrorFrameSize          ErrorCode = 0x6
	ErrorRefusedStream      ErrorCode = 0x7
	ErrorCancel             ErrorCode = 0x8
	ErrorCompression        ErrorCode = 0x9
	ErrorConnect            ErrorCode = 0xa
	ErrorEnhanceYourCalm    ErrorCode = 0xb
	ErrorInadequateSecurity ErrorCode = 0xc
	ErrorHTTP11Required     ErrorCode = 0xd
)

func NewSettingsFrame(settings []Setting, ack bool) (*Frame, error) {
	if ack && len(settings) != 0 {
		return nil, fmt.Errorf("SETTINGS ACK cannot contain settings")
	}
	payload := make([]byte, len(settings)*6)
	for index, setting := range settings {
		if err := validateSetting(setting); err != nil {
			return nil, err
		}
		offset := index * 6
		binary.BigEndian.PutUint16(payload[offset:], uint16(setting.ID))
		binary.BigEndian.PutUint32(payload[offset+2:], setting.Value)
	}
	flags := Flags(0)
	if ack {
		flags = FlagAck
	}
	return newFrame(FrameSettings, flags, 0, payload), nil
}

func ParseSettings(frame *Frame) ([]Setting, error) {
	if frame == nil || frame.Header.Type != FrameSettings {
		return nil, fmt.Errorf("not a SETTINGS frame")
	}
	if err := frame.Validate(); err != nil {
		return nil, err
	}
	settings := make([]Setting, 0, len(frame.Payload)/6)
	for offset := 0; offset < len(frame.Payload); offset += 6 {
		setting := Setting{ID: SettingID(binary.BigEndian.Uint16(frame.Payload[offset:])), Value: binary.BigEndian.Uint32(frame.Payload[offset+2:])}
		if err := validateSetting(setting); err != nil {
			return nil, err
		}
		settings = append(settings, setting)
	}
	return settings, nil
}

func validateSetting(setting Setting) error {
	switch setting.ID {
	case SettingEnablePush:
		if setting.Value > 1 {
			return fmt.Errorf("SETTINGS_ENABLE_PUSH must be 0 or 1")
		}
	case SettingInitialWindowSize:
		if setting.Value > 0x7fffffff {
			return fmt.Errorf("SETTINGS_INITIAL_WINDOW_SIZE is too large")
		}
	case SettingMaxFrameSize:
		if setting.Value < DefaultMaxFrameSize || setting.Value > MaxFrameSize {
			return fmt.Errorf("SETTINGS_MAX_FRAME_SIZE is out of range")
		}
	}
	return nil
}

func NewPingFrame(data [8]byte, ack bool) *Frame {
	flags := Flags(0)
	if ack {
		flags = FlagAck
	}
	return newFrame(FramePing, flags, 0, data[:])
}

func ParsePing(frame *Frame) ([8]byte, error) {
	var data [8]byte
	if frame == nil || frame.Header.Type != FramePing {
		return data, fmt.Errorf("not a PING frame")
	}
	if err := frame.Validate(); err != nil {
		return data, err
	}
	copy(data[:], frame.Payload)
	return data, nil
}

func NewGoAwayFrame(lastStreamID uint32, code ErrorCode, debug []byte) (*Frame, error) {
	if lastStreamID > 0x7fffffff {
		return nil, fmt.Errorf("invalid last stream ID")
	}
	payload := make([]byte, 8+len(debug))
	binary.BigEndian.PutUint32(payload, lastStreamID)
	binary.BigEndian.PutUint32(payload[4:], uint32(code))
	copy(payload[8:], debug)
	return newFrame(FrameGoAway, 0, 0, payload), nil
}

func ParseGoAway(frame *Frame) (uint32, ErrorCode, []byte, error) {
	if frame == nil || frame.Header.Type != FrameGoAway {
		return 0, 0, nil, fmt.Errorf("not a GOAWAY frame")
	}
	if err := frame.Validate(); err != nil {
		return 0, 0, nil, err
	}
	lastStreamID := binary.BigEndian.Uint32(frame.Payload) & 0x7fffffff
	code := ErrorCode(binary.BigEndian.Uint32(frame.Payload[4:]))
	return lastStreamID, code, append([]byte(nil), frame.Payload[8:]...), nil
}

func NewRSTStreamFrame(streamID uint32, code ErrorCode) (*Frame, error) {
	if streamID == 0 || streamID > 0x7fffffff {
		return nil, fmt.Errorf("invalid stream ID")
	}
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(code))
	return newFrame(FrameRSTStream, 0, streamID, payload), nil
}

func ParseRSTStream(frame *Frame) (ErrorCode, error) {
	if frame == nil || frame.Header.Type != FrameRSTStream {
		return 0, fmt.Errorf("not an RST_STREAM frame")
	}
	if err := frame.Validate(); err != nil {
		return 0, err
	}
	return ErrorCode(binary.BigEndian.Uint32(frame.Payload)), nil
}

func NewWindowUpdateFrame(streamID, increment uint32) (*Frame, error) {
	if streamID > 0x7fffffff || increment == 0 || increment > 0x7fffffff {
		return nil, fmt.Errorf("invalid WINDOW_UPDATE values")
	}
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, increment)
	return newFrame(FrameWindowUpdate, 0, streamID, payload), nil
}

func ParseWindowUpdate(frame *Frame) (uint32, error) {
	if frame == nil || frame.Header.Type != FrameWindowUpdate {
		return 0, fmt.Errorf("not a WINDOW_UPDATE frame")
	}
	if err := frame.Validate(); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(frame.Payload) & 0x7fffffff, nil
}

func newFrame(frameType FrameType, flags Flags, streamID uint32, payload []byte) *Frame {
	copyPayload := append([]byte(nil), payload...)
	return &Frame{Header: FrameHeader{Length: uint32(len(copyPayload)), Type: frameType, Flags: flags, StreamID: streamID}, Payload: copyPayload}
}
