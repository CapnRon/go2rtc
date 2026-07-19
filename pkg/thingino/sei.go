// Package thingino parses Thingino's custom SEI on-screen-display metadata.
//
// Thingino's RTSP server (prudynt-t) can embed OSD text (timestamp, labels,
// etc.) directly in the H.264 bitstream as a user_data_unregistered SEI NAL
// (payload_type 5), identified by a fixed 16-byte UUID, followed by a UTF-8
// JSON document:
//
//	{
//	  "rotation": 0,
//	  "elements": [
//	    {"t": "timestamp", "text": "2026-01-01 00:00:00", "x": 12, "y": 12},
//	    {"t": "label", "text": "FRONT DOOR", "x": -12, "y": -12}
//	  ]
//	}
package thingino

import (
	"bytes"

	"github.com/AlexxIT/go2rtc/pkg/h264"
)

var seiUUID = [16]byte{
	0xa1, 0xb2, 0xc3, 0xd4, 0xe5, 0xf6, 0x47, 0x80,
	0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90,
}

// ExtractSEI scans an AVCC-format access unit (as produced by go2rtc's
// h264.RTPDepay/RepairAVCC - a concatenation of 4-byte-length-prefixed NALs)
// for a Thingino SEI payload and returns its JSON body, if present.
func ExtractSEI(avcc []byte) ([]byte, bool) {
	if len(avcc) < 4 {
		return nil, false
	}

	for _, nalu := range h264.SplitNALU(avcc) {
		if len(nalu) < 5 || h264.NALUType(nalu) != h264.NALUTypeSEI {
			continue
		}

		// nalu = [4-byte length][1-byte NAL header][RBSP...]
		rbsp := removeEmulationPrevention(nalu[5:])
		if payload, ok := parseSEIPayload(rbsp); ok {
			return payload, true
		}
	}

	return nil, false
}

func removeEmulationPrevention(data []byte) []byte {
	out := make([]byte, 0, len(data))
	zeroes := 0
	for _, b := range data {
		if zeroes == 2 && b == 0x03 {
			zeroes = 0
			continue
		}
		out = append(out, b)
		if b == 0 {
			zeroes++
		} else {
			zeroes = 0
		}
	}
	return out
}

// parseSEIPayload walks SEI payload_type/payload_size fields (each an
// 0xFF-continued sum, per H.264 Annex D) looking for a user_data_unregistered
// (type 5) payload tagged with the Thingino UUID.
func parseSEIPayload(rbsp []byte) ([]byte, bool) {
	pos := 0
	n := len(rbsp)

	for pos < n-1 {
		payloadType := 0
		for pos < n && rbsp[pos] == 0xFF {
			payloadType += 255
			pos++
		}
		if pos >= n {
			break
		}
		payloadType += int(rbsp[pos])
		pos++

		payloadSize := 0
		for pos < n && rbsp[pos] == 0xFF {
			payloadSize += 255
			pos++
		}
		if pos >= n {
			break
		}
		payloadSize += int(rbsp[pos])
		pos++

		if pos+payloadSize > n {
			break
		}
		payload := rbsp[pos : pos+payloadSize]
		pos += payloadSize

		if payloadType == 5 && len(payload) >= 16 && bytes.Equal(payload[:16], seiUUID[:]) {
			return payload[16:], true
		}
		if payloadType == 0x80 {
			break
		}
	}

	return nil, false
}
