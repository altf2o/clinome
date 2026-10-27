// Command clinome is a terminal metronome.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ebitengine/oto/v3"
)

//go:embed Metronomes/*.wav
var embedded embed.FS

const defaultSound = "Perc_MetronomeQuartz"

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	dir := flag.String("sounds", "", "directory of NAME_hi.wav / NAME_lo.wav files (default: built-in set)")
	bpm := flag.Int("bpm", 120, "initial tempo in quarter notes per minute")
	beats := flag.Int("beats", 4, "beats per bar (time signature numerator)")
	unit := flag.Int("unit", 4, "beat unit (time signature denominator): 1, 2, 4, 8 or 16")
	sound := flag.String("sound", defaultSound, "initial sound, e.g. Perc_Snap or Synth_Sine_A")
	latency := flag.Duration("latency", 50*time.Millisecond, "delay the beat display by this much so it lines up with the audio")
	list := flag.Bool("list", false, "list available sounds and exit")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("clinome", version)
		return
	}
	switch {
	case *bpm < minBPM || *bpm > maxBPM:
		usageError("-bpm must be between %d and %d", minBPM, maxBPM)
	case *beats < minBeats || *beats > maxBeats:
		usageError("-beats must be between %d and %d", minBeats, maxBeats)
	case !validUnit(*unit):
		usageError("-unit must be one of 1, 2, 4, 8, 16")
	case *latency < 0 || *latency > time.Second:
		usageError("-latency must be between 0s and 1s")
	}
	if err := run(*dir, *bpm, *beats, *unit, *sound, *latency, *list); err != nil {
		fmt.Fprintln(os.Stderr, "clinome:", err)
		os.Exit(1)
	}
}

func usageError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "clinome: "+format+"\n", args...)
	os.Exit(2)
}

func run(dir string, bpm, beats, unit int, soundName string, latency time.Duration, list bool) error {
	var fsys fs.FS
	if dir != "" {
		fsys = os.DirFS(dir)
	} else {
		fsys, _ = fs.Sub(embedded, "Metronomes")
	}
	sounds, err := loadSounds(fsys, sampleRate)
	if err != nil {
		return err
	}

	if list {
		for _, s := range sounds {
			fmt.Println(s.Name)
		}
		return nil
	}

	soundIdx := -1
	for i, s := range sounds {
		if s.Name == soundName {
			soundIdx = i
		}
	}
	if soundIdx < 0 && soundName == defaultSound {
		soundIdx = 0 // a custom -sounds directory needn't contain the default
	}
	if soundIdx < 0 {
		return fmt.Errorf("unknown sound %q (run with -list to see the available names)", soundName)
	}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channels,
		Format:       oto.FormatFloat32LE,
		BufferSize:   20 * time.Millisecond,
	})
	if err != nil {
		return fmt.Errorf("audio init: %w", err)
	}
	<-ready

	engine := NewEngine(float64(bpm), beats, unit)
	m := newModel(engine, sounds, soundIdx, bpm, beats, unit)
	prog := tea.NewProgram(m, tea.WithAltScreen())

	// Delay UI beat updates so the display lines up with what is heard.
	engine.latency = latency
	engine.onBeat = func(gen, beat int, delay time.Duration) {
		time.AfterFunc(delay, func() { prog.Send(beatMsg{gen, beat}) })
	}

	player := ctx.NewPlayer(engine)
	player.SetBufferSize(sampleRate * frameBytes * 30 / 1000)
	player.Play()
	// oto closes a Player once it's garbage collected, so keep it reachable
	// for as long as the UI runs.
	defer runtime.KeepAlive(player)

	_, err = prog.Run()
	return err
}
