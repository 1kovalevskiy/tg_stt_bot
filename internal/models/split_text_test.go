package models

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		limit int
		want  []string
	}{
		{
			name:  "empty text",
			text:  "",
			limit: 10,
			want:  nil,
		},
		{
			name:  "shorter than limit",
			text:  "hello",
			limit: 10,
			want:  []string{"hello"},
		},
		{
			// Telegram rejects a blank message, and the answer must not
			// depend on whether the text happens to fit the limit.
			name:  "whitespace only, fits the limit",
			text:  "   ",
			limit: 10,
			want:  nil,
		},
		{
			name:  "whitespace only, over the limit",
			text:  "   ",
			limit: 2,
			want:  nil,
		},
		{
			name:  "exactly limit",
			text:  "hello",
			limit: 5,
			want:  []string{"hello"},
		},
		{
			name:  "split at word boundary",
			text:  "hello world",
			limit: 7,
			want:  []string{"hello", "world"},
		},
		{
			name:  "split at line boundary preferred over word boundary",
			text:  "one two\nthree four",
			limit: 12,
			want:  []string{"one two", "three four"},
		},
		{
			name:  "long word without spaces hard cut",
			text:  "aaaaaaaaaa",
			limit: 4,
			want:  []string{"aaaa", "aaaa", "aa"},
		},
		{
			name:  "emoji counted as two utf16 units",
			text:  "\U0001F600\U0001F600\U0001F600",
			limit: 4,
			want:  []string{"\U0001F600\U0001F600", "\U0001F600"},
		},
		{
			name:  "emoji not split in half",
			text:  "\U0001F600\U0001F600",
			limit: 3,
			want:  []string{"\U0001F600", "\U0001F600"},
		},
		{
			name:  "limit smaller than single astral rune still progresses",
			text:  "\U0001F600\U0001F600",
			limit: 1,
			want:  []string{"\U0001F600", "\U0001F600"},
		},
		{
			name:  "mixed ascii and astral counted in utf16 units",
			text:  "ab\U0001F600cd",
			limit: 4,
			want:  []string{"ab\U0001F600", "cd"},
		},
		{
			name:  "cyrillic counted as one utf16 unit each",
			text:  "привет мир",
			limit: 6,
			want:  []string{"привет", "мир"},
		},
		{
			name:  "whitespace around cut trimmed",
			text:  "hello   world",
			limit: 8,
			want:  []string{"hello", "world"},
		},
		{
			name:  "non positive limit returns whole text",
			text:  "hello world",
			limit: 0,
			want:  []string{"hello world"},
		},
		{
			name:  "multiple chunks from long text",
			text:  strings.Repeat("word ", 5),
			limit: 9,
			want:  []string{"word", "word", "word", "word", "word"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SplitText(tt.text, tt.limit)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitText(%q, %d) = %q, want %q", tt.text, tt.limit, got, tt.want)
			}
		})
	}
}

func TestSplitText_LongTextChunksWithinLimit(t *testing.T) {
	t.Parallel()

	// Astral symbols make the UTF-16 length twice the rune count,
	// so a rune-counting implementation would produce oversized chunks.
	text := strings.Repeat("\U0001F600\U0001F600\U0001F600 слово word ", 400)

	chunks := SplitText(text, TelegramMessageLimit)
	if len(chunks) < 2 {
		t.Fatalf("SplitText() returned %d chunks, want at least 2", len(chunks))
	}

	for i, chunk := range chunks {
		if chunk == "" {
			t.Errorf("chunk %d is empty", i)
		}
		if got := utf16Len(chunk); got > TelegramMessageLimit {
			t.Errorf("chunk %d is %d UTF-16 units, want <= %d", i, got, TelegramMessageLimit)
		}
	}

	// No content is lost: joining chunks back with spaces normalizes to the original words.
	want := strings.Fields(text)
	got := strings.Fields(strings.Join(chunks, " "))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("joined chunks lost content: got %d words, want %d", len(got), len(want))
	}
}

func TestUTF16Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want int
	}{
		{name: "empty", text: "", want: 0},
		{name: "ascii", text: "abc", want: 3},
		{name: "cyrillic", text: "мир", want: 3},
		{name: "astral emoji", text: "\U0001F600", want: 2},
		{name: "mixed", text: "a\U0001F600б", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := utf16Len(tt.text); got != tt.want {
				t.Errorf("utf16Len(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}
