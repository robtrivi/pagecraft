package boxes

import (
	"strconv"
	"strings"
	"unicode"
)

func countTracks(s string) int {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, "repeat("); ok && strings.HasSuffix(rest, ")") {
		contents := strings.TrimSuffix(rest, ")")
		countText, pattern, found := strings.Cut(contents, ",")
		n, err := strconv.Atoi(strings.TrimSpace(countText))
		patternCount := len(splitGridTrackTokens(pattern))
		if found && err == nil && n > 0 && patternCount > 0 && n <= maxGridTracks/patternCount {
			return n * patternCount
		}
		return 1
	}
	if n := len(splitGridTrackTokens(s)); n > 0 && n <= maxGridTracks {
		return n
	}
	return 1
}

func parseGridTracks(s string) ([]GridTrack, bool) {
	s = strings.TrimSpace(s)
	var tokens []string
	if rest, ok := strings.CutPrefix(s, "repeat("); ok && strings.HasSuffix(rest, ")") {
		contents := strings.TrimSuffix(rest, ")")
		countText, pattern, found := strings.Cut(contents, ",")
		count, err := strconv.Atoi(strings.TrimSpace(countText))
		patternTokens := splitGridTrackTokens(pattern)
		if !found || err != nil || count < 1 || len(patternTokens) == 0 || count > maxGridTracks/len(patternTokens) {
			return nil, false
		}
		for i := 0; i < count; i++ {
			tokens = append(tokens, patternTokens...)
		}
	} else {
		tokens = splitGridTrackTokens(s)
	}
	if len(tokens) == 0 || len(tokens) > maxGridTracks {
		return nil, false
	}
	tracks := make([]GridTrack, 0, len(tokens))
	for _, token := range tokens {
		if strings.HasPrefix(token, "minmax(") && strings.HasSuffix(token, ")") {
			contents := strings.TrimSuffix(strings.TrimPrefix(token, "minmax("), ")")
			minimum, maximum, found := strings.Cut(contents, ",")
			minimum = strings.TrimSpace(minimum)
			min, minOK := parseGapLength(minimum)
			minAuto := minimum == "auto"
			if minAuto {
				minOK = true
			}
			minPercent := 0.0
			if strings.HasSuffix(minimum, "%") {
				minPercent, minOK = parseNonnegativePercent(minimum)
				min = 0
			}
			maximum = strings.TrimSpace(maximum)
			fraction, err := strconv.ParseFloat(strings.TrimSuffix(maximum, "fr"), 64)
			if !found || !minOK || !strings.HasSuffix(maximum, "fr") || err != nil || fraction <= 0 {
				return nil, false
			}
			tracks = append(tracks, GridTrack{Fraction: fraction, Min: min, MinPercent: minPercent, MinAuto: minAuto})
			continue
		}
		if strings.HasSuffix(token, "%") {
			percent, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 64)
			if err != nil || percent < 0 {
				return nil, false
			}
			tracks = append(tracks, GridTrack{Percent: percent})
			continue
		}
		if strings.HasSuffix(token, "fr") {
			fraction, err := strconv.ParseFloat(strings.TrimSuffix(token, "fr"), 64)
			if err != nil || fraction < 0 {
				return nil, false
			}
			tracks = append(tracks, GridTrack{Fraction: fraction, MinAuto: true})
			continue
		}
		fixed, ok := parseGapLength(token)
		if !ok {
			return nil, false
		}
		tracks = append(tracks, GridTrack{Fixed: fixed})
	}
	return tracks, true
}

func parseGridAutoColumns(s string) ([]float64, []float64, []bool, []bool, []float64, []float64, []float64, []bool, bool) {
	tokens := splitGridTrackTokens(strings.TrimSpace(s))
	if len(tokens) == 0 || len(tokens) > maxGridTracks {
		return nil, nil, nil, nil, nil, nil, nil, nil, false
	}
	widths := make([]float64, 0, len(tokens))
	percentages := make([]float64, 0, len(tokens))
	percentageTracks := make([]bool, 0, len(tokens))
	fixedZeroTracks := make([]bool, 0, len(tokens))
	fractions := make([]float64, 0, len(tokens))
	minimums := make([]float64, 0, len(tokens))
	minimumPercentages := make([]float64, 0, len(tokens))
	automaticMinimums := make([]bool, 0, len(tokens))
	for _, token := range tokens {
		if token == "auto" {
			widths = append(widths, 0)
			percentages = append(percentages, 0)
			percentageTracks = append(percentageTracks, false)
			fixedZeroTracks = append(fixedZeroTracks, false)
			fractions = append(fractions, 0)
			minimums = append(minimums, 0)
			minimumPercentages = append(minimumPercentages, 0)
			automaticMinimums = append(automaticMinimums, false)
			continue
		}
		if percent, ok := parseGridAutoColumnPercentage(token); ok {
			widths = append(widths, 0)
			percentages = append(percentages, percent)
			percentageTracks = append(percentageTracks, true)
			fixedZeroTracks = append(fixedZeroTracks, false)
			fractions = append(fractions, 0)
			minimums = append(minimums, 0)
			minimumPercentages = append(minimumPercentages, 0)
			automaticMinimums = append(automaticMinimums, false)
			continue
		}
		tracks, valid := parseGridTracks(token)
		if !valid || len(widths)+len(tracks) > maxGridTracks {
			return nil, nil, nil, nil, nil, nil, nil, nil, false
		}
		for _, track := range tracks {
			if track.Percent != 0 || (track.Fraction == 0 && (track.Min != 0 || track.MinPercent != 0)) {
				return nil, nil, nil, nil, nil, nil, nil, nil, false
			}
			widths = append(widths, track.Fixed)
			percentages = append(percentages, 0)
			percentageTracks = append(percentageTracks, false)
			fixedZeroTracks = append(fixedZeroTracks, track.Fixed == 0 && track.Fraction == 0)
			fractions = append(fractions, track.Fraction)
			minimums = append(minimums, track.Min)
			minimumPercentages = append(minimumPercentages, track.MinPercent)
			automaticMinimums = append(automaticMinimums, track.MinAuto)
		}
	}
	return widths, percentages, percentageTracks, fixedZeroTracks, fractions, minimums, minimumPercentages, automaticMinimums, true
}

func isZeroGridLength(token string) bool {
	length, ok := parseGapLength(strings.TrimSpace(token))
	return ok && length == 0
}

func isZeroPercent(token string) bool {
	token = strings.TrimSpace(token)
	if !strings.HasSuffix(token, "%") {
		return false
	}
	value, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 64)
	return err == nil && value == 0
}

func parseNonnegativePercent(token string) (float64, bool) {
	token = strings.TrimSpace(token)
	if !strings.HasSuffix(token, "%") {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 64)
	return value, err == nil && value >= 0
}

func parseGridAutoColumnPercentage(token string) (float64, bool) {
	if percent, ok := parsePercent(token); ok {
		return percent, true
	}
	token = strings.TrimSpace(token)
	if !strings.HasSuffix(token, "%") {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSuffix(token, "%"), 64)
	return 0, err == nil && value == 0
}

func parseGridTemplateRows(s string) ([]float64, []bool, bool) {
	tokens := splitGridTrackTokens(strings.TrimSpace(s))
	if len(tokens) == 0 || len(tokens) > maxGridTracks {
		return nil, nil, false
	}
	heights := make([]float64, 0, len(tokens))
	fixedZeroTracks := make([]bool, 0, len(tokens))
	for _, token := range tokens {
		if token == "auto" {
			heights = append(heights, 0)
			fixedZeroTracks = append(fixedZeroTracks, false)
			continue
		}
		tracks, valid := parseGridTracks(token)
		if !valid || len(heights)+len(tracks) > maxGridTracks {
			return nil, nil, false
		}
		for _, track := range tracks {
			if track.Fraction != 0 || track.Percent != 0 || track.Min != 0 || track.MinPercent != 0 {
				return nil, nil, false
			}
			heights = append(heights, track.Fixed)
			fixedZeroTracks = append(fixedZeroTracks, track.Fixed == 0)
		}
	}
	return heights, fixedZeroTracks, true
}

func parseGridAutoRows(s string) ([]float64, []bool, bool) {
	tokens := splitGridTrackTokens(strings.TrimSpace(s))
	if len(tokens) == 0 || len(tokens) > maxGridTracks {
		return nil, nil, false
	}
	heights := make([]float64, 0, len(tokens))
	fixedZeroTracks := make([]bool, 0, len(tokens))
	for _, token := range tokens {
		if isZeroPercent(token) || isZeroGridLength(token) {
			heights = append(heights, 0)
			fixedZeroTracks = append(fixedZeroTracks, true)
			continue
		}
		parsed, valid := parseGridRowHeights(token)
		if !valid || len(heights)+len(parsed) > maxGridTracks {
			return nil, nil, false
		}
		heights = append(heights, parsed...)
		for range parsed {
			fixedZeroTracks = append(fixedZeroTracks, false)
		}
	}
	return heights, fixedZeroTracks, true
}

func parseGridRowHeights(s string) ([]float64, bool) {
	tokens := splitGridTrackTokens(strings.TrimSpace(s))
	if len(tokens) == 0 || len(tokens) > maxGridTracks {
		return nil, false
	}
	heights := make([]float64, 0, len(tokens))
	for _, token := range tokens {
		if token == "auto" {
			heights = append(heights, 0)
			continue
		}
		tracks, valid := parseGridTracks(token)
		if !valid || len(heights)+len(tracks) > maxGridTracks {
			return nil, false
		}
		for _, track := range tracks {
			if track.Fraction != 0 || track.Percent != 0 || track.Min != 0 || track.MinPercent != 0 {
				return nil, false
			}
			heights = append(heights, track.Fixed)
		}
	}
	return heights, true
}

func splitGridTrackTokens(s string) []string {
	var tokens []string
	depth, start := 0, -1
	for i, r := range s {
		if unicode.IsSpace(r) && depth == 0 {
			if start >= 0 {
				tokens = append(tokens, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		}
	}
	if start >= 0 {
		tokens = append(tokens, s[start:])
	}
	return tokens
}

// setBorderSide assigns a border side by name.
