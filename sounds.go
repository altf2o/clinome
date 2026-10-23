package main

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Sound is a click with an accented (hi) and regular (lo) variant.
type Sound struct {
	Name  string // e.g. "Perc_Snap"
	Group string // e.g. "Perc"
	Label string // e.g. "Snap"
	Hi    []float32
	Lo    []float32
}

func (s Sound) Display() string { return s.Group + " · " + s.Label }

// loadSounds finds every *_hi.wav / *_lo.wav pair in fsys.
func loadSounds(fsys fs.FS, rate int) ([]Sound, error) {
	files, err := fs.Glob(fsys, "*.wav")
	if err != nil {
		return nil, err
	}

	byName := map[string]*Sound{}
	for _, f := range files {
		base := strings.TrimSuffix(path.Base(f), path.Ext(f))
		var name string
		var hi bool
		switch {
		case strings.HasSuffix(base, "_hi"):
			name, hi = strings.TrimSuffix(base, "_hi"), true
		case strings.HasSuffix(base, "_lo"):
			name = strings.TrimSuffix(base, "_lo")
		default:
			continue
		}

		raw, err := fs.ReadFile(fsys, f)
		if err != nil {
			return nil, err
		}
		pcm, err := decodeWAV(raw, rate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}

		s := byName[name]
		if s == nil {
			group, label, ok := strings.Cut(name, "_")
			if !ok {
				group, label = "Other", name
			}
			s = &Sound{Name: name, Group: group, Label: strings.ReplaceAll(label, "_", " ")}
			byName[name] = s
		}
		if hi {
			s.Hi = pcm
		} else {
			s.Lo = pcm
		}
	}

	var sounds []Sound
	for _, s := range byName {
		// Tolerate a missing half of a pair by reusing the other one.
		if s.Hi == nil {
			s.Hi = s.Lo
		}
		if s.Lo == nil {
			s.Lo = s.Hi
		}
		sounds = append(sounds, *s)
	}
	sort.Slice(sounds, func(i, j int) bool {
		if sounds[i].Group != sounds[j].Group {
			return sounds[i].Group < sounds[j].Group
		}
		return sounds[i].Label < sounds[j].Label
	})
	if len(sounds) == 0 {
		return nil, fmt.Errorf("no *_hi.wav / *_lo.wav sounds found")
	}
	return sounds, nil
}
