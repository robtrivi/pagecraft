package layout

import "strings"

// BreakLines splits text at spaces without exceeding the usable width.
// It measures incrementally with measure, which must be monotonic. A word
// wider than the available width is not split; it overflows, as with CSS
// overflow when hyphenation is disabled.
func BreakLines(text string, usable float64, measure func(string, float64) float64, fontSize float64) []string {
	words := strings.Fields(text)
	var lines []string
	cur := ""
	for _, w := range words {
		trial := w
		if cur != "" {
			trial = cur + " " + w
		}
		if cur == "" || measure(trial, fontSize) <= usable {
			cur = trial
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	return append(lines, cur)
}
