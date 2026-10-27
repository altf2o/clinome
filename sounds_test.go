package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestLoadSounds(t *testing.T) {
	hi := makeWAV(1, 1, sampleRate, 16, []byte{0, 0x40})
	lo := makeWAV(1, 1, sampleRate, 16, []byte{0, 0x20})
	fsys := fstest.MapFS{
		"Synth_Sine_A_hi.wav": {Data: hi},
		"Synth_Sine_A_lo.wav": {Data: lo},
		"Perc_Snap_lo.wav":    {Data: lo}, // missing hi half
		"Click_hi.wav":        {Data: hi}, // no group
		"notes.txt":           {Data: []byte("ignored")},
		"Stray.wav":           {Data: hi}, // no _hi/_lo suffix
	}
	sounds, err := loadSounds(fsys, sampleRate)
	if err != nil {
		t.Fatal(err)
	}

	want := []struct{ name, display string }{
		{"Click", "Other · Click"},
		{"Perc_Snap", "Perc · Snap"},
		{"Synth_Sine_A", "Synth · Sine A"},
	}
	if len(sounds) != len(want) {
		t.Fatalf("got %d sounds, want %d", len(sounds), len(want))
	}
	for i, w := range want {
		s := sounds[i]
		if s.Name != w.name || s.Display() != w.display {
			t.Errorf("sound %d = %q (%q), want %q (%q)", i, s.Name, s.Display(), w.name, w.display)
		}
		if len(s.Hi) == 0 || len(s.Lo) == 0 {
			t.Errorf("%s: hi or lo is empty", s.Name)
		}
	}
	if snap := sounds[1]; snap.Hi[0] != snap.Lo[0] {
		t.Errorf("Perc_Snap should reuse lo for hi")
	}
	if sine := sounds[2]; sine.Hi[0] == sine.Lo[0] {
		t.Errorf("Synth_Sine_A hi and lo should differ")
	}
}

func TestLoadSoundsErrors(t *testing.T) {
	if _, err := loadSounds(fstest.MapFS{}, sampleRate); err == nil {
		t.Error("empty dir: expected an error")
	}
	bad := fstest.MapFS{"X_hi.wav": {Data: []byte("junk")}}
	if _, err := loadSounds(bad, sampleRate); err == nil {
		t.Error("corrupt file: expected an error")
	}
}

func TestEmbeddedSoundsLoad(t *testing.T) {
	fsys, err := fs.Sub(embedded, "Metronomes")
	if err != nil {
		t.Fatal(err)
	}
	sounds, err := loadSounds(fsys, sampleRate)
	if err != nil {
		t.Fatal(err)
	}
	if len(sounds) != 60 {
		t.Errorf("got %d embedded sounds, want 60", len(sounds))
	}
	found := false
	for _, s := range sounds {
		found = found || s.Name == defaultSound
	}
	if !found {
		t.Errorf("default sound %q is not embedded", defaultSound)
	}
}
