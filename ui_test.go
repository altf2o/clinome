package main

import (
	"testing"
	"time"
)

func testModel() model {
	sounds := []Sound{{Name: "A_a", Hi: []float32{0, 0}, Lo: []float32{0, 0}}}
	return newModel(NewEngine(120, 4, 4), sounds, 0, 120, 4, 4)
}

func TestTempoMarking(t *testing.T) {
	for bpm, want := range map[int]string{20: "Grave", 60: "Larghetto", 100: "Andante", 120: "Allegro", 300: "Prestissimo"} {
		if got := tempoMarking(bpm); got != want {
			t.Errorf("tempoMarking(%d) = %q, want %q", bpm, got, want)
		}
	}
}

func TestStepUnitStaysInRange(t *testing.T) {
	m := testModel()
	for range 10 {
		m.stepUnit(1)
	}
	if m.unit != 16 {
		t.Errorf("unit = %d after stepping up, want 16", m.unit)
	}
	for range 10 {
		m.stepUnit(-1)
	}
	if m.unit != 1 {
		t.Errorf("unit = %d after stepping down, want 1", m.unit)
	}
}

func TestTapTempo(t *testing.T) {
	for _, tc := range []struct {
		unit     int
		interval time.Duration
		want     int
	}{
		{4, 500 * time.Millisecond, 120},
		{8, 250 * time.Millisecond, 120}, // eighth notes at 240/min
		{4, 10 * time.Millisecond, maxBPM},
		{4, 1900 * time.Millisecond, 32},
	} {
		m := testModel()
		m.unit = tc.unit
		// Fake three earlier taps; tap() adds the fourth at time.Now.
		now := time.Now()
		for i := 3; i > 0; i-- {
			m.taps = append(m.taps, now.Add(-time.Duration(i)*tc.interval))
		}
		m.tap()
		if diff := m.bpm - tc.want; diff < -1 || diff > 1 {
			t.Errorf("unit %d, interval %v: bpm = %d, want ≈%d", tc.unit, tc.interval, m.bpm, tc.want)
		}
	}
}

func TestPickerClamp(t *testing.T) {
	m := testModel()
	m.sounds = make([]Sound, 50)
	m.height = 20
	rows := m.pickerRows()

	m.cursor = 49
	m.clampPicker()
	if m.offset != 50-rows {
		t.Errorf("offset = %d at last item, want %d", m.offset, 50-rows)
	}
	m.cursor = 0
	m.clampPicker()
	if m.offset != 0 {
		t.Errorf("offset = %d at first item, want 0", m.offset)
	}
}
