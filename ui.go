package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	minBPM, maxBPM     = 20, 300
	minBeats, maxBeats = 1, 16
	flashDuration      = 110 * time.Millisecond
	tapResetAfter      = 2 * time.Second
)

// beatUnits are the selectable time signature denominators.
var beatUnits = []int{1, 2, 4, 8, 16}

func validUnit(n int) bool {
	for _, u := range beatUnits {
		if u == n {
			return true
		}
	}
	return false
}

var (
	colAccent = lipgloss.Color("#FF5F87")
	colBeat   = lipgloss.Color("#5FD7FF")
	colDim    = lipgloss.Color("#4E4E4E")
	colMuted  = lipgloss.Color("#8A8A8A")
	colText   = lipgloss.Color("#E4E4E4")
	colGood   = lipgloss.Color("#87D787")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1C1C1C")).Background(colBeat).Padding(0, 1)
	labelStyle = lipgloss.NewStyle().Foreground(colMuted)
	valueStyle = lipgloss.NewStyle().Foreground(colText).Bold(true)
	helpKey    = lipgloss.NewStyle().Foreground(colBeat)
	helpDesc   = lipgloss.NewStyle().Foreground(colMuted)
	cardStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colDim).Padding(0, 1)
)

type beatMsg struct{ gen, beat int }
type flashOffMsg struct{ id int }

type model struct {
	engine *Engine
	sounds []Sound

	sound   int
	bpm     int
	beats   int
	unit    int // time signature denominator
	accent  bool
	volume  int // percent
	playing bool
	gen     int
	beat    int // current beat in bar, -1 when idle

	flashID  int
	flashing bool

	picking bool
	cursor  int
	offset  int // first visible row in the picker

	taps []time.Time

	width, height int
}

func newModel(e *Engine, sounds []Sound, sound, bpm, beats, unit int) model {
	m := model{
		engine: e, sounds: sounds, sound: sound,
		bpm: bpm, beats: beats, unit: unit, accent: true, volume: 80, beat: -1,
	}
	e.SetSound(sounds[sound])
	e.SetBPM(float64(bpm))
	e.SetBeatsPerBar(beats)
	e.SetBeatUnit(unit)
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampPicker()

	case beatMsg:
		if !m.playing || msg.gen != m.gen {
			return m, nil
		}
		m.beat = msg.beat
		m.flashing = true
		m.flashID++
		id := m.flashID
		return m, tea.Tick(flashDuration, func(time.Time) tea.Msg { return flashOffMsg{id} })

	case flashOffMsg:
		if msg.id == m.flashID {
			m.flashing = false
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.picking {
			return m.updatePicker(msg)
		}
		return m.updateMain(msg)
	}
	return m, nil
}

func (m model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit
	case " ":
		m.toggle()
	case "up", "k", "+":
		m.setBPM(m.bpm + 1)
	case "down", "j", "-":
		m.setBPM(m.bpm - 1)
	case "right", "l":
		m.setBPM(m.bpm + 5)
	case "left", "h":
		m.setBPM(m.bpm - 5)
	case "pgup":
		m.setBPM(m.bpm + 10)
	case "pgdown":
		m.setBPM(m.bpm - 10)
	case "]":
		m.setBeats(m.beats + 1)
	case "[":
		m.setBeats(m.beats - 1)
	case "}":
		m.stepUnit(1)
	case "{":
		m.stepUnit(-1)
	case "a":
		m.accent = !m.accent
		m.engine.SetAccent(m.accent)
	case "t":
		m.tap()
	case "p":
		m.engine.Preview(m.sounds[m.sound])
	case ".", ">":
		m.setVolume(m.volume + 10)
	case ",", "<":
		m.setVolume(m.volume - 10)
	case "s", "enter":
		m.picking = true
		m.cursor = m.sound
		m.clampPicker()
	}
	return m, nil
}

func (m model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prev := m.cursor
	switch msg.String() {
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "pgup":
		m.cursor -= m.pickerRows()
	case "pgdown":
		m.cursor += m.pickerRows()
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.sounds) - 1
	case "p", "tab":
		m.engine.Preview(m.sounds[m.cursor])
	case "enter":
		m.sound = m.cursor
		m.engine.SetSound(m.sounds[m.sound])
		m.picking = false
	case "esc", "s", "q":
		m.picking = false
	case " ":
		m.toggle()
	}
	m.cursor = max(0, min(len(m.sounds)-1, m.cursor))
	m.clampPicker()
	if m.cursor != prev {
		m.engine.Preview(m.sounds[m.cursor])
	}
	return m, nil
}

func (m *model) toggle() {
	if m.playing {
		m.engine.Stop()
		m.playing = false
		m.beat = -1
		m.flashing = false
	} else {
		m.gen = m.engine.Start()
		m.playing = true
	}
}

func (m *model) setBPM(bpm int) {
	m.bpm = max(minBPM, min(maxBPM, bpm))
	m.engine.SetBPM(float64(m.bpm))
}

func (m *model) setBeats(n int) {
	m.beats = max(minBeats, min(maxBeats, n))
	m.engine.SetBeatsPerBar(m.beats)
}

// stepUnit moves the time signature denominator up or down through beatUnits.
func (m *model) stepUnit(dir int) {
	for i, u := range beatUnits {
		if u == m.unit {
			m.unit = beatUnits[max(0, min(len(beatUnits)-1, i+dir))]
			break
		}
	}
	m.engine.SetBeatUnit(m.unit)
}

func (m *model) setVolume(v int) {
	m.volume = max(0, min(100, v))
	m.engine.SetVolume(float32(m.volume) / 100)
}

// tap sets the tempo from the average interval of recent taps, which are
// taken to be beats of the current beat unit.
func (m *model) tap() {
	now := time.Now()
	if n := len(m.taps); n > 0 && now.Sub(m.taps[n-1]) > tapResetAfter {
		m.taps = nil
	}
	m.taps = append(m.taps, now)
	if len(m.taps) > 6 {
		m.taps = m.taps[len(m.taps)-6:]
	}
	if len(m.taps) >= 2 {
		span := m.taps[len(m.taps)-1].Sub(m.taps[0])
		avg := span / time.Duration(len(m.taps)-1)
		// Taps mark clicked beats; convert that rate to quarter notes.
		m.setBPM(int(math.Round(float64(time.Minute) / float64(avg) * 4 / float64(m.unit))))
	}
}

// --- layout -------------------------------------------------------------

const (
	headerH = 1
	cardsH  = 4 // card content (2) + border (2)
	helpH   = 1
)

func (m model) panelHeight() int { return max(7, m.height-headerH-cardsH-helpH-1) }

func (m model) pickerRows() int { return max(1, m.panelHeight()-4) }

func (m *model) clampPicker() {
	rows := m.pickerRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = max(0, min(m.offset, len(m.sounds)-rows))
	m.offset = max(0, m.offset)
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	var panel string
	if m.picking {
		panel = m.viewPicker()
	} else {
		panel = m.viewBeat()
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewHeader(),
		panel,
		m.viewCards(),
		m.viewHelp(),
	)
}

func (m model) viewHeader() string {
	title := titleStyle.Render("♩ clinome")
	var status string
	if m.playing {
		status = lipgloss.NewStyle().Foreground(colGood).Bold(true).Render("▶ PLAYING")
	} else {
		status = labelStyle.Render("■ STOPPED")
	}
	gap := max(1, m.width-lipgloss.Width(title)-lipgloss.Width(status)-1)
	return title + strings.Repeat(" ", gap) + status
}

func (m model) viewBeat() string {
	w, h := m.width, m.panelHeight()
	innerW, innerH := w-2, h-2

	color := colDim
	digitColor := colMuted
	if m.beat >= 0 {
		digitColor = colBeat
		if m.beat == 0 && m.accent {
			digitColor = colAccent
		}
		if m.flashing {
			color = digitColor
		}
	}

	label := "-"
	if m.beat >= 0 {
		label = strconv.Itoa(m.beat + 1)
	}

	// Pick the largest scale that fits alongside the dot row.
	scale := 1
	for s := 6; s >= 1; s-- {
		bw, bh := bigTextSize(label, s)
		if bw <= innerW-2 && bh+2 <= innerH {
			scale = s
			break
		}
	}
	digitStyle := lipgloss.NewStyle().Foreground(digitColor)
	if !m.flashing && m.beat >= 0 {
		digitStyle = digitStyle.Faint(true)
	}
	big := digitStyle.Render(renderBig(label, scale))

	content := lipgloss.JoinVertical(lipgloss.Center, big, "", m.viewDots())
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Width(innerW).Height(innerH).
		Render(lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, content))
}

func (m model) viewDots() string {
	dots := make([]string, m.beats)
	for i := range dots {
		c := colBeat
		if i == 0 && m.accent {
			c = colAccent
		}
		st := lipgloss.NewStyle().Foreground(c)
		if i == m.beat {
			dots[i] = st.Bold(true).Render("●")
		} else {
			dots[i] = st.Faint(true).Render("○")
		}
	}
	return strings.Join(dots, " ")
}

func (m model) viewPicker() string {
	w, h := m.width, m.panelHeight()
	innerW, innerH := w-2, h-2

	header := valueStyle.Render("Select click sound") + "  " +
		labelStyle.Render(fmt.Sprintf("%d/%d · moving previews hi+lo", m.cursor+1, len(m.sounds)))

	rows := []string{header, ""}
	end := min(len(m.sounds), m.offset+m.pickerRows())
	for i := m.offset; i < end; i++ {
		s := m.sounds[i]
		mark := "  "
		if i == m.sound {
			mark = lipgloss.NewStyle().Foreground(colGood).Render("✓ ")
		}
		group := labelStyle.Render(fmt.Sprintf("%-6s", s.Group))
		name := s.Label
		line := mark + group + " " + name
		if i == m.cursor {
			line = lipgloss.NewStyle().Foreground(colBeat).Bold(true).Render("▸ ") + mark + group + " " +
				lipgloss.NewStyle().Foreground(colBeat).Bold(true).Render(name)
		} else {
			line = "  " + line
		}
		rows = append(rows, line)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBeat).
		Padding(0, 1).
		Width(innerW).Height(innerH).
		Render(strings.Join(rows, "\n"))
}

func (m model) viewCards() string {
	accent := "off"
	if m.accent {
		accent = "on"
	}
	cards := []struct{ label, value, sub string }{
		{"TEMPO", fmt.Sprintf("♩ = %d", m.bpm), m.tempoSub()},
		{"METER", fmt.Sprintf("%d / %d", m.beats, m.unit), "accent " + accent},
		{"SOUND", m.sounds[m.sound].Display(), "hi + lo"},
		{"VOLUME", fmt.Sprintf("%d%%", m.volume), volumeBar(m.volume)},
	}

	n := len(cards)
	avail := m.width
	var out []string
	for i, c := range cards {
		cw := avail / n
		if i == n-1 {
			cw = m.width - (avail/n)*(n-1)
		}
		body := labelStyle.Render(c.label) + " " + valueStyle.Render(c.value) + "\n" + labelStyle.Render(c.sub)
		out = append(out, cardStyle.Width(max(1, cw-2)).MaxHeight(cardsH).Render(body))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, out...)
}

func (m model) viewHelp() string {
	var items [][2]string
	if m.picking {
		items = [][2]string{{"↑↓", "browse"}, {"enter", "select"}, {"p", "replay"}, {"space", "start/stop"}, {"esc", "cancel"}}
	} else {
		items = [][2]string{
			{"space", "play"}, {"↑↓←→", "bpm"}, {"t", "tap"},
			{"[ ] { }", "meter"}, {"a", "accent"}, {"s", "sound"}, {"p", "preview"}, {", .", "vol"}, {"q", "quit"},
		}
	}
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = helpKey.Render(it[0]) + " " + helpDesc.Render(it[1])
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(" " + strings.Join(parts, helpDesc.Render(" · ")))
}

// tempoSub shows the tempo marking, plus the click rate when the beat unit
// isn't a quarter note.
func (m model) tempoSub() string {
	s := tempoMarking(m.bpm)
	if m.unit != 4 {
		s += fmt.Sprintf(" · %g/min", float64(m.bpm)*float64(m.unit)/4)
	}
	return s
}

func tempoMarking(bpm int) string {
	switch {
	case bpm < 40:
		return "Grave"
	case bpm < 60:
		return "Largo"
	case bpm < 66:
		return "Larghetto"
	case bpm < 76:
		return "Adagio"
	case bpm < 108:
		return "Andante"
	case bpm < 120:
		return "Moderato"
	case bpm < 156:
		return "Allegro"
	case bpm < 176:
		return "Vivace"
	case bpm < 200:
		return "Presto"
	default:
		return "Prestissimo"
	}
}

func volumeBar(v int) string {
	n := v / 10
	return strings.Repeat("▮", n) + strings.Repeat("▯", 10-n)
}
