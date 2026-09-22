package http2

import "fmt"

func NewDataFrame(streamID uint32, data []byte, endStream bool) (*Frame, error) {
	if streamID == 0 || streamID > 0x7fffffff {
		return nil, fmt.Errorf("invalid stream ID")
	}
	flags := Flags(0)
	if endStream {
		flags |= FlagEndStream
	}
	return newFrame(FrameData, flags, streamID, data), nil
}

func NewHeadersFrame(streamID uint32, headerBlock []byte, endHeaders, endStream bool) (*Frame, error) {
	if streamID == 0 || streamID > 0x7fffffff {
		return nil, fmt.Errorf("invalid stream ID")
	}
	flags := Flags(0)
	if endHeaders {
		flags |= FlagEndHeaders
	}
	if endStream {
		flags |= FlagEndStream
	}
	return newFrame(FrameHeaders, flags, streamID, headerBlock), nil
}
