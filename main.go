// Command clinome is a terminal metronome.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ebitengine/oto/v3"
)

//go:embed Metronomes/*.wav
var embedded embed.FS

func main() {
	dir := flag.String("sounds", "", "directory of NAME_hi.wav / NAME_lo.wav files (default: built-in set)")
	bpm := flag.Int("bpm", 120, "initial tempo in quarter notes per minute")
	beats := flag.Int("beats", 4, "beats per bar (time signature numerator)")
	unit := flag.Int("unit", 4, "beat unit (time signature denominator): 1, 2, 4, 8 or 16")
	sound := flag.String("sound", "Perc_MetronomeQuartz", "initial sound, e.g. Perc_Snap or Synth_Sine_A")
	list := flag.Bool("list", false, "list available sounds and exit")
	flag.Parse()

	if !validUnit(*unit) {
		fmt.Fprintln(os.Stderr, "clinome: -unit must be one of 1, 2, 4, 8, 16")
		os.Exit(2)
	}
	if err := run(*dir, *bpm, *beats, *unit, *sound, *list); err != nil {
		fmt.Fprintln(os.Stderr, "clinome:", err)
		os.Exit(1)
	}
}

func run(dir string, bpm, beats, unit int, soundName string, list bool) error {
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

	soundIdx := 0
	for i, s := range sounds {
		if s.Name == soundName {
			soundIdx = i
		}
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
	m := newModel(engine, sounds, soundIdx, max(minBPM, min(maxBPM, bpm)), max(minBeats, min(maxBeats, beats)), unit)
	prog := tea.NewProgram(m, tea.WithAltScreen())

	// Delay UI beat updates so the display lines up with what is heard.
	engine.latency = 50 * time.Millisecond
	engine.onBeat = func(gen, beat int, delay time.Duration) {
		time.AfterFunc(delay, func() { prog.Send(beatMsg{gen, beat}) })
	}

	player := ctx.NewPlayer(engine)
	player.SetBufferSize(sampleRate * frameBytes * 30 / 1000)
	player.Play()
	defer player.Close()

	_, err = prog.Run()
	return err
}
