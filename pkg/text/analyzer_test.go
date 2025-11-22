package text

import (
	"reflect"
	"testing"
)

func TestTokenization(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple text",
			input:    "The quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
		{
			name:     "With punctuation",
			input:    "Hello, world! How are you?",
			expected: []string{"hello", "world"}, // "how" and "are" are stop words, "you" is stop word
		},
		{
			name:     "Mixed case",
			input:    "MongoDB is a Database",
			expected: []string{"mongodb", "databas"}, // stemmed
		},
		{
			name:     "Stop words removed",
			input:    "the quick and the brown",
			expected: []string{"quick", "brown"},
		},
		{
			name:     "Numbers",
			input:    "Version 2024 release",
			expected: []string{"version", "2024", "releas"}, // stemmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestStemming(t *testing.T) {
	stemmer := NewPorterStemmer()

	tests := []struct {
		input    string
		expected string
	}{
		// Plurals
		{"cats", "cat"},
		{"caresses", "caress"},
		{"ponies", "poni"},

		// -ed, -ing
		{"agreed", "agre"},
		{"playing", "plai"}, // Porter stemmer output
		{"played", "plai"},  // Porter stemmer output

		// -ly
		{"quickly", "quickli"}, // Porter stemmer output

		// -tion
		{"relation", "relat"},
		{"conditional", "condit"},

		// -ness
		{"goodness", "good"},

		// Various
		{"running", "run"},
		{"databases", "databas"},
		{"indexing", "index"},
		{"computing", "comput"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stemmer.Stem(tt.input)
			if result != tt.expected {
				t.Errorf("Stem(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAnalyzerWithPositions(t *testing.T) {
	analyzer := NewAnalyzer()

	text := "The quick brown fox jumps"
	positions := analyzer.AnalyzeWithPositions(text)

	// Should get: quick(1), brown(2), fox(3), jump(4) [stemmed]
	// "The" is a stop word at position 0

	if len(positions) != 4 {
		t.Fatalf("Expected 4 tokens, got %d", len(positions))
	}

	expected := []struct {
		token string
		pos   int
	}{
		{"quick", 1},
		{"brown", 2},
		{"fox", 3},
		{"jump", 4}, // stemmed from "jumps"
	}

	for i, exp := range expected {
		if positions[i].Token != exp.token {
			t.Errorf("Position %d: expected token %q, got %q", i, exp.token, positions[i].Token)
		}
		if positions[i].Position != exp.pos {
			t.Errorf("Position %d: expected position %d, got %d", i, exp.pos, positions[i].Position)
		}
	}
}

func TestStopWords(t *testing.T) {
	analyzer := NewAnalyzer()

	// Text with only stop words
	text := "the a an and or but"
	tokens := analyzer.Analyze(text)

	if len(tokens) != 0 {
		t.Errorf("Expected all stop words to be filtered, got %v", tokens)
	}
}

func TestShortWords(t *testing.T) {
	analyzer := NewAnalyzer()

	// Text with short words (< 2 characters)
	text := "I am a go developer"
	tokens := analyzer.Analyze(text)

	// "I", "a" should be filtered (stop words and short)
	// "am" is a stop word
	// "go" stays (2 characters, not a stop word)
	// "developer" -> "develop" (stemmed)

	expected := []string{"go", "develop"}
	if !reflect.DeepEqual(tokens, expected) {
		t.Errorf("Expected %v, got %v", expected, tokens)
	}
}
