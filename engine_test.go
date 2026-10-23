package main

import (
	"testing"
	"time"
)

func TestBeatSpacingIsSampleAccurate(t *testing.T) {
	e := NewEngine(120, 4, 4)
	e.SetSound(Sound{Hi: []float32{1, 1}, Lo: []float32{0.5, 0.5}})
	e.SetVolume(1)

	var beats []int
	e.onBeat = func(_, beat int, _ time.Duration) { beats = append(beats, beat) }
	e.Start()

	// Render 2 seconds in odd-sized chunks; at 120 BPM that is exactly 4 beats.
	buf := make([]byte, 0, 2*sampleRate*frameBytes)
	chunk := make([]byte, 333*frameBytes)
	for len(buf) < cap(buf) {
		n := min(len(chunk), cap(buf)-len(buf))
		e.Read(chunk[:n])
		buf = append(buf, chunk[:n]...)
	}

	var onsets []int
	for i := 0; i < len(buf)/frameBytes; i++ {
		if buf[i*frameBytes+3] != 0 { // non-zero float32 high byte
			onsets = append(onsets, i)
		}
	}
	want := []int{0, 24000, 48000, 72000}
	if len(onsets) != len(want) {
		t.Fatalf("onsets = %v, want %v", onsets, want)
	}
	for i := range want {
		if onsets[i] != want[i] {
			t.Fatalf("onsets = %v, want %v", onsets, want)
		}
	}
	if got := []int{0, 1, 2, 3}; len(beats) != 4 || beats[0] != got[0] || beats[3] != got[3] {
		t.Fatalf("beats = %v", beats)
	}
}

func TestBeatUnitScalesClickRate(t *testing.T) {
	for _, tc := range []struct{ unit, want int }{{2, 48000}, {4, 24000}, {8, 12000}, {16, 6000}} {
		e := NewEngine(120, 6, tc.unit)
		var n int
		e.onBeat = func(int, int, time.Duration) { n++ }
		e.Start()
		e.Read(make([]byte, (2*tc.want+1)*frameBytes))
		if n != 3 {
			t.Errorf("unit %d: got %d clicks in %d frames, want 3", tc.unit, n, 2*tc.want+1)
		}
	}
}
