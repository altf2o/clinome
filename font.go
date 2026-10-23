package main

import "strings"

// 3x5 pixel glyphs for the big beat display.
var glyphs = map[rune][5]string{
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	'1': {".#.", "##.", ".#.", ".#.", "###"},
	'2': {"###", "..#", "###", "#..", "###"},
	'3': {"###", "..#", "###", "..#", "###"},
	'4': {"#.#", "#.#", "###", "..#", "..#"},
	'5': {"###", "#..", "###", "..#", "###"},
	'6': {"###", "#..", "###", "#.#", "###"},
	'7': {"###", "..#", "..#", "..#", "..#"},
	'8': {"###", "#.#", "###", "#.#", "###"},
	'9': {"###", "#.#", "###", "..#", "###"},
	'-': {"...", "...", "###", "...", "..."},
}

const glyphW, glyphH = 3, 5

// bigTextSize returns the width and height in cells of s rendered at scale.
func bigTextSize(s string, scale int) (int, int) {
	n := len([]rune(s))
	return n*glyphW*2*scale + (n-1)*2*scale, glyphH * scale
}

// renderBig renders s using block glyphs. Each pixel is 2*scale cells wide
// and scale cells tall so it looks roughly square in a terminal.
func renderBig(s string, scale int) string {
	var rows []string
	px := strings.Repeat("█", 2*scale)
	blank := strings.Repeat(" ", 2*scale)
	gap := strings.Repeat(" ", 2*scale)
	for y := 0; y < glyphH; y++ {
		var line strings.Builder
		for i, r := range s {
			if i > 0 {
				line.WriteString(gap)
			}
			g := glyphs[r]
			for x := 0; x < glyphW; x++ {
				if g[y][x] == '#' {
					line.WriteString(px)
				} else {
					line.WriteString(blank)
				}
			}
		}
		for k := 0; k < scale; k++ {
			rows = append(rows, line.String())
		}
	}
	return strings.Join(rows, "\n")
}
