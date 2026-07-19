package thingino

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildSEINAL builds a length-prefixed AVCC NAL carrying a Thingino SEI
// user_data_unregistered payload, mirroring the reference Python encoder.
func buildSEINAL(payload []byte) []byte {
	var rbsp bytes.Buffer

	ptype := 5
	for ptype >= 255 {
		rbsp.WriteByte(0xFF)
		ptype -= 255
	}
	rbsp.WriteByte(byte(ptype))

	size := len(payload)
	for size >= 255 {
		rbsp.WriteByte(0xFF)
		size -= 255
	}
	rbsp.WriteByte(byte(size))

	rbsp.Write(payload)
	rbsp.WriteByte(0x80) // rbsp_trailing_bits

	nal := append([]byte{0x06}, rbsp.Bytes()...) // NAL header: type 6 (SEI)

	out := make([]byte, 4+len(nal))
	binary.BigEndian.PutUint32(out, uint32(len(nal)))
	copy(out[4:], nal)
	return out
}

func TestExtractSEI(t *testing.T) {
	body := []byte(`{"rotation":90,"elements":[{"t":"timestamp","text":"2026-01-01 00:00:00","x":10,"y":10}]}`)
	payload := append(append([]byte{}, seiUUID[:]...), body...)

	seiNAL := buildSEINAL(payload)

	// A slice NAL (type 1) with no special meaning, just filler after the SEI,
	// like a real access unit: [SEI NAL][slice NAL].
	sliceNAL := make([]byte, 4+5)
	binary.BigEndian.PutUint32(sliceNAL, 5)
	sliceNAL[4] = 0x01 // NAL header: type 1
	copy(sliceNAL[5:], []byte{1, 2, 3, 4})

	avcc := append(append([]byte{}, seiNAL...), sliceNAL...)

	got, ok := ExtractSEI(avcc)
	if !ok {
		t.Fatal("ExtractSEI: expected to find a Thingino SEI payload")
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("ExtractSEI: got %q, want %q", got, body)
	}
}

func TestExtractSEI_NoSEI(t *testing.T) {
	sliceNAL := make([]byte, 4+5)
	binary.BigEndian.PutUint32(sliceNAL, 5)
	sliceNAL[4] = 0x01
	copy(sliceNAL[5:], []byte{1, 2, 3, 4})

	_, ok := ExtractSEI(sliceNAL)
	if ok {
		t.Fatal("ExtractSEI: expected no SEI payload in a stream with no SEI NAL")
	}
}

func TestExtractSEI_WrongUUID(t *testing.T) {
	payload := append([]byte{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, // not the Thingino UUID
	}, []byte(`{"foo":"bar"}`)...)

	avcc := buildSEINAL(payload)

	_, ok := ExtractSEI(avcc)
	if ok {
		t.Fatal("ExtractSEI: expected no match for a non-Thingino UUID")
	}
}

func TestRemoveEmulationPrevention(t *testing.T) {
	in := []byte{0x00, 0x00, 0x03, 0x01, 0x00, 0x00, 0x03, 0x02, 0x00, 0x00, 0x03}
	want := []byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x02, 0x00, 0x00}
	got := removeEmulationPrevention(in)
	if !bytes.Equal(got, want) {
		t.Fatalf("removeEmulationPrevention: got %v, want %v", got, want)
	}
}
