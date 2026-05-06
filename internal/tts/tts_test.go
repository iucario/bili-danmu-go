package tts

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Punctuation-only → spoken form
		{name: "ascii question mark alone", input: "?", want: "问号"},
		{name: "full-width question mark alone", input: "？", want: "问号"},
		{name: "multiple question marks", input: "???", want: "问号问号问号"},
		{name: "ascii exclamation alone", input: "!", want: "感叹号"},
		{name: "full-width exclamation alone", input: "！", want: "感叹号"},
		{name: "mixed punctuation only", input: "?!", want: "问号感叹号"},

		// Real words present → punctuation dropped silently
		{name: "chinese with question mark", input: "哈哈？", want: "哈哈"},
		{name: "ascii word with question mark", input: "ok?", want: "ok"},
		{name: "chinese with exclamation", input: "牛！", want: "牛"},
		{name: "sentence with trailing punct", input: "一个人吃这么多？", want: "一个人吃这么多"},

		// Emoji removal
		{name: "custom emoji only", input: "[爱心]", want: ""},
		{name: "unicode emoji only", input: "😀", want: ""},
		{name: "text with custom emoji", input: "hello[爱心]world", want: "helloworld"},
		{name: "text with unicode emoji", input: "哈哈😂", want: "哈哈"},

		// Combination
		{name: "emoji then question mark", input: "😂？", want: "问号"},
		{name: "text emoji and punct", input: "好[爱心]？", want: "好"},

		// Edge cases
		{name: "empty string", input: "", want: ""},
		{name: "whitespace only", input: "   ", want: ""},
		{name: "plain chinese", input: "你好", want: "你好"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitize(tc.input)
			if got != tc.want {
				t.Errorf("sanitize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
