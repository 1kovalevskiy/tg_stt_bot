package models

import "testing"

func TestTruncateText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		limit int
		want  string
	}{
		{name: "empty text", text: "", limit: 10, want: ""},
		{name: "shorter than limit", text: "hello", limit: 10, want: "hello"},
		{name: "exactly limit", text: "hello", limit: 5, want: "hello"},
		{name: "cut to limit", text: "hello world", limit: 5, want: "hello"},
		{name: "non positive limit", text: "hello", limit: 0, want: ""},
		{name: "cyrillic counted per rune", text: "привет мир", limit: 6, want: "привет"},
		{
			name:  "astral rune counted as two units",
			text:  "\U0001F600\U0001F600",
			limit: 3,
			want:  "\U0001F600",
		},
		{
			name:  "astral rune is never cut in half",
			text:  "\U0001F600",
			limit: 1,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateText(tt.text, tt.limit)
			if got != tt.want {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tt.text, tt.limit, got, tt.want)
			}

			if utf16Len(got) > tt.limit && tt.limit > 0 {
				t.Errorf("TruncateText(%q, %d) = %q, which is %d UTF-16 units",
					tt.text, tt.limit, got, utf16Len(got))
			}
		})
	}
}
