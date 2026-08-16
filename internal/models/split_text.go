package models

import (
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// SplitText splits text into chunks whose length does not exceed limit,
// counted in UTF-16 code units (Telegram counts message length this way).
// Chunks are cut at line boundaries when possible, then at word boundaries,
// and only mid-word when a single word exceeds the limit. Whitespace around
// cut points is trimmed. Empty and whitespace-only text yields nil: Telegram
// rejects blank messages, and the result must not depend on the limit.
func SplitText(text string, limit int) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	if limit <= 0 || utf16Len(text) <= limit {
		return []string{text}
	}

	var chunks []string

	remaining := text
	for remaining != "" {
		if utf16Len(remaining) <= limit {
			if tail := strings.TrimRightFunc(remaining, unicode.IsSpace); tail != "" {
				chunks = append(chunks, tail)
			}

			break
		}

		cut := cutIndex(remaining, limit)
		if cut == 0 {
			// The first rune alone exceeds the limit (e.g. an astral symbol
			// with limit 1): force progress by emitting that single rune.
			_, size := utf8.DecodeRuneInString(remaining)
			cut = size
		}

		chunk := strings.TrimRightFunc(remaining[:cut], unicode.IsSpace)
		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		remaining = strings.TrimLeftFunc(remaining[cut:], unicode.IsSpace)
	}

	return chunks
}

// cutIndex returns the byte index at which to cut s so that the head fits
// into limit UTF-16 code units, preferring the last line boundary, then the
// last word boundary, then a hard cut.
func cutIndex(s string, limit int) int {
	units := 0
	lastNewline := -1
	lastSpace := -1

	for i, r := range s {
		runeLen := runeUnits(r)

		if units+runeLen > limit {
			if lastNewline >= 0 {
				return lastNewline
			}

			if lastSpace >= 0 {
				return lastSpace
			}

			return i
		}

		units += runeLen

		switch {
		case r == '\n':
			lastNewline = i + 1
		case unicode.IsSpace(r):
			lastSpace = i + utf8.RuneLen(r)
		}
	}

	return len(s)
}

// utf16Len returns the length of s in UTF-16 code units.
func utf16Len(s string) int {
	units := 0

	for _, r := range s {
		units += runeUnits(r)
	}

	return units
}

// runeUnits returns how many UTF-16 code units r takes. Ranging over a string
// never yields a surrogate or an out-of-range rune (invalid bytes come out as
// utf8.RuneError), but a negative length would make the counters run backwards,
// so it is clamped here once for both callers.
func runeUnits(r rune) int {
	if units := utf16.RuneLen(r); units > 0 {
		return units
	}

	return 1
}
