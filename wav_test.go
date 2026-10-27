package main

import (
	"encoding/binary"
	"math"
	"testing"
)

// makeWAV builds a minimal WAV file. samples are interleaved and already
// encoded at the given bit depth.
func makeWAV(format, channels uint16, rate uint32, bits uint16, samples []byte) []byte {
	le := binary.LittleEndian
	b := []byte("RIFF\x00\x00\x00\x00WAVEfmt ")
	b = le.AppendUint32(b, 16)
	b = le.AppendUint16(b, format)
	b = le.AppendUint16(b, channels)
	b = le.AppendUint32(b, rate)
	b = le.AppendUint32(b, rate*uint32(channels)*uint32(bits/8))
	b = le.AppendUint16(b, channels*bits/8)
	b = le.AppendUint16(b, bits)
	b = append(b, "data"...)
	b = le.AppendUint32(b, uint32(len(samples)))
	b = append(b, samples...)
	le.PutUint32(b[4:], uint32(len(b)-8))
	return b
}

func TestDecodeWAVFormats(t *testing.T) {
	le := binary.LittleEndian
	for _, tc := range []struct {
		name     string
		format   uint16
		bits     uint16
		channels uint16
		samples  []byte
		want     []float32 // interleaved stereo
	}{
		{"pcm16 mono", 1, 16, 1, le.AppendUint16(le.AppendUint16(nil, 0x4000), 0xC000), []float32{0.5, 0.5, -0.5, -0.5}},
		{"pcm16 stereo", 1, 16, 2, le.AppendUint16(le.AppendUint16(nil, 0x4000), 0xC000), []float32{0.5, -0.5}},
		{"pcm24 mono", 1, 24, 1, []byte{0x00, 0x00, 0x40}, []float32{0.5, 0.5}},
		{"pcm32 mono", 1, 32, 1, le.AppendUint32(nil, 0xC0000000), []float32{-0.5, -0.5}},
		{"float32 mono", 3, 32, 1, le.AppendUint32(nil, math.Float32bits(0.25)), []float32{0.25, 0.25}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeWAV(makeWAV(tc.format, tc.channels, sampleRate, tc.bits, tc.samples), sampleRate)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestDecodeWAVResamples(t *testing.T) {
	samples := make([]byte, 2*24000) // 1 s of 16-bit mono at 24 kHz
	got, err := decodeWAV(makeWAV(1, 1, 24000, 16, samples), sampleRate)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2*sampleRate {
		t.Fatalf("got %d frames, want %d", len(got)/2, sampleRate)
	}
}

func TestDecodeWAVRejectsBadInput(t *testing.T) {
	one := []byte{0, 0}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"not riff", []byte("RIFX\x00\x00\x00\x00WAVE")},
		{"no chunks", []byte("RIFF\x04\x00\x00\x00WAVE")},
		{"8-bit", makeWAV(1, 1, sampleRate, 8, []byte{0})},
		{"zero channels", makeWAV(1, 0, sampleRate, 16, one)},
		{"zero rate", makeWAV(1, 1, 0, 16, one)},
		{"huge rate", makeWAV(1, 1, 1<<30, 16, one)},
		{"tiny rate", makeWAV(1, 1, 1, 16, one)},
		{"empty data", makeWAV(1, 1, sampleRate, 16, nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := decodeWAV(tc.data, sampleRate); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// FuzzDecodeWAV checks that no input makes the decoder panic, since -sounds
// loads arbitrary user files.
func FuzzDecodeWAV(f *testing.F) {
	seed, err := embedded.ReadFile("Metronomes/Perc_Snap_hi.wav")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add(makeWAV(3, 2, 44100, 32, make([]byte, 64)))
	f.Fuzz(func(t *testing.T, data []byte) {
		decodeWAV(data, sampleRate)
	})
}
