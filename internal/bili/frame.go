package bili

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
)

const (
	headerSize = 16

	// Body encoding versions
	VerNormal    uint16 = 0
	VerHeartbeat uint16 = 1
	VerDeflate   uint16 = 2
	VerBrotli    uint16 = 3

	// Operations
	OpHeartbeat      uint32 = 2
	OpHeartbeatReply uint32 = 3
	OpSendMsgReply   uint32 = 5
	OpAuth           uint32 = 7
	OpAuthReply      uint32 = 8
)

// Header is the 16-byte big-endian frame header.
type Header struct {
	PackLen       uint32
	RawHeaderSize uint16
	Ver           uint16
	Operation     uint32
	SeqID         uint32
}

// Frame is a decoded frame with its body.
type Frame struct {
	Header Header
	Body   []byte
}

// EncodeFrame builds a single wire frame.
func EncodeFrame(op uint32, ver uint16, body []byte) []byte {
	totalLen := uint32(headerSize + len(body))
	buf := make([]byte, totalLen)
	binary.BigEndian.PutUint32(buf[0:4], totalLen)
	binary.BigEndian.PutUint16(buf[4:6], headerSize)
	binary.BigEndian.PutUint16(buf[6:8], ver)
	binary.BigEndian.PutUint32(buf[8:12], op)
	binary.BigEndian.PutUint32(buf[12:16], 1)
	copy(buf[16:], body)
	return buf
}

// DecodeFrames parses a raw byte slice into one or more Frames,
// recursively decompressing DEFLATE/BROTLI payloads.
func DecodeFrames(data []byte) ([]Frame, error) {
	var frames []Frame
	offset := 0
	for offset < len(data) {
		if len(data)-offset < headerSize {
			return frames, fmt.Errorf("incomplete header at offset %d", offset)
		}
		h := Header{
			PackLen:       binary.BigEndian.Uint32(data[offset : offset+4]),
			RawHeaderSize: binary.BigEndian.Uint16(data[offset+4 : offset+6]),
			Ver:           binary.BigEndian.Uint16(data[offset+6 : offset+8]),
			Operation:     binary.BigEndian.Uint32(data[offset+8 : offset+12]),
			SeqID:         binary.BigEndian.Uint32(data[offset+12 : offset+16]),
		}
		if h.PackLen < headerSize || int(h.PackLen) > len(data)-offset {
			return frames, fmt.Errorf("invalid pack_len %d at offset %d", h.PackLen, offset)
		}
		bodyStart := offset + int(h.RawHeaderSize)
		bodyEnd := offset + int(h.PackLen)
		body := data[bodyStart:bodyEnd]

		switch h.Ver {
		case VerDeflate:
			decompressed, err := zlibDecompress(body)
			if err != nil {
				return frames, fmt.Errorf("zlib decompress: %w", err)
			}
			inner, err := DecodeFrames(decompressed)
			if err != nil {
				return frames, err
			}
			frames = append(frames, inner...)
		case VerBrotli:
			decompressed, err := brotliDecompress(body)
			if err != nil {
				return frames, fmt.Errorf("brotli decompress: %w", err)
			}
			inner, err := DecodeFrames(decompressed)
			if err != nil {
				return frames, err
			}
			frames = append(frames, inner...)
		default:
			frames = append(frames, Frame{Header: h, Body: body})
		}
		offset = bodyEnd
	}
	return frames, nil
}

func zlibDecompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func brotliDecompress(data []byte) ([]byte, error) {
	r := brotli.NewReader(bytes.NewReader(data))
	return io.ReadAll(r)
}
