package models

// TruncateText cuts text down to limit UTF-16 code units (Telegram counts
// message length this way), never splitting a rune in half. A non-positive
// limit yields an empty string.
func TruncateText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}

	units := 0

	for i, r := range text {
		next := units + runeUnits(r)
		if next > limit {
			return text[:i]
		}

		units = next
	}

	return text
}
