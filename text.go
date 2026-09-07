package main

import (
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func isCJK(r rune) bool {
	return unicode.In(r,
		unicode.Han, // kanji
		unicode.Hiragana,
		unicode.Katakana,
	)
}

func tokenize(s string) []string {
	var tokens []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}
	for _, r := range s {
		switch {
		case unicode.IsSpace(r):
			flush()
		case isCJK(r):
			flush()
			tokens = append(tokens, string(r))
		default:
			word.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// splitLongWord breaks a single token into pieces that each fit within
// maxWidth, splitting rune-by-rune. Used when a word alone is too wide
// to fit on a line.
func splitLongWord(word string, face text.Face, maxWidth float64) []string {
	var pieces []string
	var current strings.Builder

	for _, r := range word {
		candidate := current.String() + string(r)
		w := text.Advance(candidate, face)
		if w <= maxWidth || current.Len() == 0 {
			// Always allow at least one rune per piece, even if that
			// single rune alone exceeds maxWidth (can't split further).
			current.WriteRune(r)
		} else {
			pieces = append(pieces, current.String())
			current.Reset()
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		pieces = append(pieces, current.String())
	}
	return pieces
}

// addToken appends tok to lines/current, splitting tok itself across
// multiple lines if it's too wide to ever fit on one line by itself.
func addToken(lines []string, current string, tok string, sep string, face text.Face, maxWidth float64) ([]string, string) {
	candidate := current + sep + tok
	if current == "" {
		candidate = tok
	}

	if text.Advance(candidate, face) <= maxWidth {
		return lines, candidate
	}

	// Doesn't fit as-is. First, push whatever we were building onto lines.
	if current != "" {
		lines = append(lines, current)
		current = ""
	}

	// Does tok fit on its own line untouched?
	if text.Advance(tok, face) <= maxWidth {
		return lines, tok
	}

	// tok itself is too wide — split it into pieces.
	pieces := splitLongWord(tok, face, maxWidth)
	for i, p := range pieces {
		if i == len(pieces)-1 {
			current = p // last piece becomes the new "current" line-in-progress
		} else {
			lines = append(lines, p)
		}
	}
	return lines, current
}

func WrapText(s string, face text.Face, maxWidth float64) []string {
	var lines []string
	paragraphs := strings.Split(s, "\n")
	for _, para := range paragraphs {
		tokens := tokenize(para)
		if len(tokens) == 0 {
			lines = append(lines, "")
			continue
		}

		var current string
		for i, tok := range tokens {
			sep := " "
			if current == "" {
				sep = ""
			} else if isCJK([]rune(tok)[0]) || isCJK([]rune(current)[len([]rune(current))-1]) {
				sep = ""
			}
			_ = i
			lines, current = addToken(lines, current, tok, sep, face, maxWidth)
		}
		if current != "" {
			lines = append(lines, current)
		}
	}
	return lines
}
