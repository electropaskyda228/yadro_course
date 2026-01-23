package words

import (
	"reflect"
	"sort"
	"testing"
)

func TestSplitByNonAlphanumericUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "hello, world!",
			expected: []string{"hello", "world"},
		},
		{
			name:     "multiple spaces",
			input:    "hello   world  test",
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "mixed alphanumeric",
			input:    "hello123 world456",
			expected: []string{"hello123", "world456"},
		},
		{
			name:     "unicode characters",
			input:    "привет мир",
			expected: []string{"привет", "мир"},
		},
		{
			name:     "with hyphens and underscores",
			input:    "hello-world test_phrase",
			expected: []string{"hello", "world", "test", "phrase"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only delimiters",
			input:    "!!! , ,, ...",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitByNonAlphanumericUnicode(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("splitByNonAlphanumericUnicode(%q) = %v (len=%d), want %v (len=%d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitByNonAlphanumericUnicode(%q)[%d] = %q, want %q",
						tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestNormFunction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple phrase",
			input:    "Hello beautiful world",
			expected: []string{"beauti", "hello", "world"},
		},
		{
			name:     "filter stop words",
			input:    "the cat and the dog",
			expected: []string{"cat", "dog"},
		},
		{
			name: "with punctuation and stop words",
			// Слова "is" и "this" не в списке forbittenWords, поэтому они остаются
			input:    "Hello, world! This is a test.",
			expected: []string{"hello", "is", "test", "this", "world"},
		},
		{
			name:     "stemming words",
			input:    "running jumping swimmer",
			expected: []string{"jump", "run", "swimmer"},
		},
		{
			name:     "mixed case and stemming",
			input:    "Running JUMPING swimmer",
			expected: []string{"jump", "run", "swimmer"},
		},
		{
			name:     "only stop words",
			input:    "the and or a",
			expected: []string{},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "duplicate words",
			input:    "hello hello world world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "words with digits",
			input:    "test123 hello456",
			expected: []string{"hello456", "test123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Norm(tt.input)

			sort.Strings(result)
			expected := make([]string, len(tt.expected))
			copy(expected, tt.expected)
			sort.Strings(expected)

			if !reflect.DeepEqual(result, expected) {
				t.Errorf("Norm(%q) = %v, want %v", tt.input, result, expected)
			}
		})
	}
}

func TestNormFiltersKnownForbiddenWords(t *testing.T) {
	forbiddenTests := []struct {
		word     string
		expected bool
	}{
		{"of", true},
		{"the", true},
		{"a", true},
		{"and", true},
		{"or", true},
		{"will", true},
		{"would", true},
		{"i", true},
		{"me", true},
		{"you", true},
		{"your", true},
		{"he", true},
		{"his", true},
		{"him", true},
		{"who", true},
		{"it", true},
		{"that", true},
		{"she", true},
		{"her", true},
		{"we", true},
		{"our", true},
		{"they", true},
		{"their", true},
		{"them", true},
		{"is", false},
		{"this", false},
		{"hello", false},
		{"world", false},
		{"test", false},
	}

	for _, tt := range forbiddenTests {
		t.Run(tt.word, func(t *testing.T) {
			result := Norm(tt.word)
			shouldBeFiltered := len(result) == 0

			if shouldBeFiltered != tt.expected {
				t.Errorf("Word %q: filtered = %v, expected filtered = %v", tt.word, shouldBeFiltered, tt.expected)
			}
		})
	}
}
