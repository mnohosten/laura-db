package text

import (
	"regexp"
	"strings"
	"unicode"
)

// Analyzer handles text tokenization, normalization, and stemming
type Analyzer struct {
	stopWords map[string]bool
	stemmer   *PorterStemmer
}

// NewAnalyzer creates a new text analyzer with English stop words
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		stopWords: defaultStopWords(),
		stemmer:   NewPorterStemmer(),
	}
}

// Analyze processes text and returns normalized tokens
func (a *Analyzer) Analyze(text string) []string {
	// Tokenize
	tokens := a.tokenize(text)

	// Normalize and filter
	var result []string
	for _, token := range tokens {
		// Convert to lowercase
		token = strings.ToLower(token)

		// Skip if too short
		if len(token) < 2 {
			continue
		}

		// Skip stop words
		if a.stopWords[token] {
			continue
		}

		// Apply stemming
		token = a.stemmer.Stem(token)

		result = append(result, token)
	}

	return result
}

// tokenize breaks text into words
func (a *Analyzer) tokenize(text string) []string {
	// Split on whitespace and punctuation
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	parts := re.Split(text, -1)

	var tokens []string
	for _, part := range parts {
		if len(part) > 0 {
			tokens = append(tokens, part)
		}
	}

	return tokens
}

// AnalyzeWithPositions returns tokens with their positions in the original text
func (a *Analyzer) AnalyzeWithPositions(text string) []TokenPosition {
	tokens := a.tokenize(text)
	var result []TokenPosition
	position := 0

	for _, token := range tokens {
		// Convert to lowercase
		normalized := strings.ToLower(token)

		// Skip if too short
		if len(normalized) < 2 {
			position++
			continue
		}

		// Skip stop words
		if a.stopWords[normalized] {
			position++
			continue
		}

		// Apply stemming
		stemmed := a.stemmer.Stem(normalized)

		result = append(result, TokenPosition{
			Token:    stemmed,
			Position: position,
		})

		position++
	}

	return result
}

// TokenPosition represents a token with its position in the text
type TokenPosition struct {
	Token    string
	Position int
}

// IsWord checks if a rune is a letter or number
func IsWord(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// defaultStopWords returns common English stop words
func defaultStopWords() map[string]bool {
	words := []string{
		"a", "an", "and", "are", "as", "at", "be", "but", "by",
		"for", "if", "in", "into", "is", "it", "no", "not", "of",
		"on", "or", "such", "that", "the", "their", "then", "there",
		"these", "they", "this", "to", "was", "will", "with",
		// Additional common words
		"i", "you", "he", "she", "we", "they", "me", "him", "her",
		"us", "them", "what", "which", "who", "when", "where", "why",
		"how", "all", "each", "every", "both", "few", "more", "most",
		"other", "some", "can", "could", "may", "might", "must",
		"shall", "should", "would", "am", "been", "being", "have",
		"has", "had", "do", "does", "did", "doing",
	}

	stopWords := make(map[string]bool)
	for _, word := range words {
		stopWords[word] = true
	}

	return stopWords
}
