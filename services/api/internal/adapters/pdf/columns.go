package pdf

import "strings"

const (
	minGutterSpaces   = 2
	minColumnLines    = 3
	maxCrossingShare  = 0.25
	minGutterFraction = 0.3
	maxGutterFraction = 0.7
)

func mergePages(raw, layout string) string {
	rawPages, layoutPages := strings.Split(raw, "\f"), strings.Split(layout, "\f")
	if len(rawPages) != len(layoutPages) {
		return raw
	}
	out := make([]string, len(rawPages))
	for i, page := range rawPages {
		out[i] = page
		if untangled, ok := untangleColumns(layoutPages[i]); ok {
			out[i] = untangled
		}
	}
	return strings.Join(out, "\f")
}

func untangleColumns(page string) (string, bool) {
	lines := strings.Split(page, "\n")
	rows := make([][]rune, len(lines))
	for i, l := range lines {
		rows[i] = []rune(strings.TrimRight(l, " "))
	}
	gutter, ok := findGutter(rows)
	if !ok {
		return "", false
	}

	var out, left, right []string
	flush := func() {
		out = append(append(out, left...), right...)
		left, right = nil, nil
	}
	for _, row := range rows {
		if len(strings.TrimSpace(string(row))) == 0 {
			left, right = appendBlank(left), appendBlank(right)
			continue
		}
		l, r, split := splitAt(row, gutter)
		if !split {
			flush()
			out = append(out, collapse(string(row)))
			continue
		}
		if l != "" {
			left = append(left, l)
		}
		if r != "" {
			right = append(right, r)
		}
	}
	flush()
	return strings.Join(trimBlankEnds(out), "\n"), true
}

func findGutter(rows [][]rune) (int, bool) {
	width := 0
	for _, row := range rows {
		width = max(width, len(row))
	}
	lo, hi := max(int(float64(width)*minGutterFraction), minGutterSpaces), int(float64(width)*maxGutterFraction)
	counts := map[int]int{}
	for _, row := range rows {
		for p := lo; p <= hi && p < len(row); p++ {
			if row[p] != ' ' && isGap(row, p) {
				counts[p]++
			}
		}
	}
	best := 0
	for p, n := range counts {
		if n > counts[best] || (n == counts[best] && p < best) {
			best = p
		}
	}
	return best, counts[best] >= minColumnLines && crossingShare(rows, best) <= maxCrossingShare
}

func crossingShare(rows [][]rune, gutter int) float64 {
	text, crossing := 0, 0
	for _, row := range rows {
		if strings.TrimSpace(string(row)) == "" {
			continue
		}
		text++
		if !isGap(row, gutter) {
			crossing++
		}
	}
	if text == 0 {
		return 1
	}
	return float64(crossing) / float64(text)
}

func isGap(row []rune, p int) bool {
	for k := p - minGutterSpaces; k < p; k++ {
		if k < len(row) && row[k] != ' ' {
			return false
		}
	}
	return true
}

func splitAt(row []rune, gutter int) (string, string, bool) {
	if !isGap(row, gutter) {
		return "", "", false
	}
	if len(row) <= gutter {
		return collapse(string(row)), "", true
	}
	return collapse(string(row[:gutter])), collapse(string(row[gutter:])), true
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func appendBlank(lines []string) []string {
	if len(lines) == 0 || lines[len(lines)-1] == "" {
		return lines
	}
	return append(lines, "")
}

func trimBlankEnds(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	return lines
}
