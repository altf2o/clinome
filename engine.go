package main

import (
	"encoding/binary"
	"math"
	"sync"
	"time"
)

const (
	sampleRate = 48000
	channels   = 2
	frameBytes = channels * 4 // float32 per channel
	maxVoices  = 32
)

type voice struct {
	data    []float32 // interleaved stereo
	pos     int       // frame position
	delay   int       // frames to wait before starting
	preview bool
}

// Engine is a sample-accurate metronome and mixer. It implements io.Reader
// producing interleaved float32 little-endian stereo PCM for the audio player.
// Beats are scheduled in the sample domain, so timing does not depend on
// goroutine scheduling or timers.
type Engine struct {
	mu sync.Mutex

	running     bool
	bpm         float64 // quarter notes per minute
	beatsPerBar int
	beatUnit    int // time signature denominator; one click per beatUnit note
	accent      bool
	hi, lo      []float32
	volume      float32

	voices   []voice
	pos      int64   // frames rendered so far
	nextBeat float64 // frame position of the next beat
	lastBeat float64 // frame position of the previous beat
	beat     int     // beats played since start
	gen      int     // incremented on every Start so stale beat events can be ignored

	// latency approximates the time between a frame being rendered and heard.
	latency time.Duration
	// onBeat is called (under the lock, must not block) when a beat is
	// scheduled; delay is how long until it is actually audible.
	onBeat func(gen, beat int, delay time.Duration)
}

func NewEngine(bpm float64, beatsPerBar, beatUnit int) *Engine {
	return &Engine{bpm: bpm, beatsPerBar: beatsPerBar, beatUnit: beatUnit, accent: true, volume: 0.8}
}

func (e *Engine) framesPerBeat() float64 {
	return sampleRate * 60 / e.bpm * 4 / float64(e.beatUnit)
}

// retime moves the upcoming beat relative to the last one so tempo changes
// take effect immediately without a stutter.
func (e *Engine) retime() {
	if e.running && e.beat > 0 {
		e.nextBeat = e.lastBeat + e.framesPerBeat()
	}
}

func (e *Engine) Read(p []byte) (int, error) {
	frames := len(p) / frameBytes

	e.mu.Lock()
	defer e.mu.Unlock()

	for i := 0; i < frames; i++ {
		if e.running && float64(e.pos) >= e.nextBeat {
			e.trigger(i)
		}

		var l, r float32
		for v := range e.voices {
			vc := &e.voices[v]
			if vc.delay > 0 {
				vc.delay--
				continue
			}
			if 2*vc.pos+1 < len(vc.data) {
				l += vc.data[2*vc.pos]
				r += vc.data[2*vc.pos+1]
				vc.pos++
			}
		}
		l, r = clamp(l*e.volume), clamp(r*e.volume)

		binary.LittleEndian.PutUint32(p[i*frameBytes:], math.Float32bits(l))
		binary.LittleEndian.PutUint32(p[i*frameBytes+4:], math.Float32bits(r))
		e.pos++
	}

	// Drop finished voices.
	live := e.voices[:0]
	for _, v := range e.voices {
		if v.delay > 0 || 2*v.pos+1 < len(v.data) {
			live = append(live, v)
		}
	}
	e.voices = live

	return frames * frameBytes, nil
}

// trigger plays the current beat; offset is its frame index within the
// buffer currently being rendered.
func (e *Engine) trigger(offset int) {
	b := e.beat % e.beatsPerBar
	click := e.lo
	if b == 0 && e.accent {
		click = e.hi
	}
	e.addVoice(click, 0, false)

	if e.onBeat != nil {
		delay := e.latency + time.Duration(offset)*time.Second/sampleRate
		e.onBeat(e.gen, b, delay)
	}

	e.lastBeat = e.nextBeat
	e.nextBeat += e.framesPerBeat()
	e.beat++
}

func (e *Engine) addVoice(data []float32, delay int, preview bool) {
	if len(data) == 0 {
		return
	}
	if len(e.voices) >= maxVoices {
		e.voices = e.voices[1:]
	}
	e.voices = append(e.voices, voice{data: data, delay: delay, preview: preview})
}

func clamp(x float32) float32 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x
}

// Start begins playback on the next rendered frame and returns the new
// generation number used to tag beat events.
func (e *Engine) Start() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = true
	e.beat = 0
	e.gen++
	e.nextBeat = float64(e.pos)
	return e.gen
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.running = false
}

func (e *Engine) SetBPM(bpm float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.bpm = bpm
	e.retime()
}

func (e *Engine) SetBeatUnit(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.beatUnit = n
	e.retime()
}

func (e *Engine) SetBeatsPerBar(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.beatsPerBar = n
}

func (e *Engine) SetAccent(on bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.accent = on
}

func (e *Engine) SetVolume(v float32) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.volume = v
}

func (e *Engine) SetSound(s Sound) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.hi, e.lo = s.Hi, s.Lo
}

// Preview plays a sound's hi click followed by its lo click, cutting off any
// preview still playing so scrolling through sounds doesn't pile up.
func (e *Engine) Preview(s Sound) {
	e.mu.Lock()
	defer e.mu.Unlock()
	live := e.voices[:0]
	for _, v := range e.voices {
		if !v.preview {
			live = append(live, v)
		}
	}
	e.voices = live
	e.addVoice(s.Hi, 0, true)
	e.addVoice(s.Lo, sampleRate*35/100, true)
}
