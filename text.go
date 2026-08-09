package main

import (
	"unicode"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func isCJK(r rune) bool {
	return unicode.In(r,
		unicode.Han,      // kanji
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

func WrapText(s string, face text.Face, maxWidth float64) []string {
	var lines []string
	paragraphs := strings.Split(s, "\n")

	for _, para := range paragraphs {
		tokens := tokenize(para)
		if len(tokens) == 0 {
			lines = append(lines, "")
			continue
		}

		current := tokens[0]
		for _, tok := range tokens[1:] {
			// Only insert a space if we're joining two Latin words.
			// CJK glue directly with no space.
			sep := " "
			if isCJK([]rune(tok)[0]) || isCJK([]rune(current)[len([]rune(current))-1]) {
				sep = ""
			}
			candidate := current + sep + tok

			w := text.Advance(candidate, face)
			if w <= maxWidth {
				current = candidate
			} else {
				lines = append(lines, current)
				current = tok
			}
		}
		lines = append(lines, current)
	}
	return lines
}
