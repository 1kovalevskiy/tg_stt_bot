package models

import (
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// TelegramMessageLimit is the maximum Telegram message length in UTF-16 code units.
const TelegramMessageLimit = 4096

// SplitText splits text into chunks whose length does not exceed limit,
// counted in UTF-16 code units (Telegram counts message length this way).
// Chunks are cut at line boundaries when possible, then at word boundaries,
// and only mid-word when a single word exceeds the limit. Whitespace around
// cut points is trimmed. Empty text yields nil.
func SplitText(text string, limit int) []string {
	if text == "" {
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
		runeLen := utf16.RuneLen(r)
		if runeLen < 0 {
			runeLen = 1
		}

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
		runeLen := utf16.RuneLen(r)
		if runeLen < 0 {
			runeLen = 1
		}

		units += runeLen
	}

	return units
}
