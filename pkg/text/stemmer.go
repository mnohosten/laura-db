package text

import (
	"strings"
	"unicode"
)

// PorterStemmer implements the Porter stemming algorithm
// This is a simplified version focusing on common suffixes
type PorterStemmer struct{}

// NewPorterStemmer creates a new Porter stemmer
func NewPorterStemmer() *PorterStemmer {
	return &PorterStemmer{}
}

// Stem reduces a word to its stem
func (ps *PorterStemmer) Stem(word string) string {
	word = strings.ToLower(word)

	// Don't stem very short words
	if len(word) < 3 {
		return word
	}

	// Step 1a: plurals and -ed/-ing
	word = ps.step1a(word)

	// Step 1b: -ed, -ing
	word = ps.step1b(word)

	// Step 1c: y -> i
	word = ps.step1c(word)

	// Step 2: double suffixes
	word = ps.step2(word)

	// Step 3: -ic, -full, -ness
	word = ps.step3(word)

	// Step 4: common suffixes
	word = ps.step4(word)

	// Step 5: -e removal
	word = ps.step5(word)

	return word
}

// step1a handles plurals and -ED or -ING
func (ps *PorterStemmer) step1a(word string) string {
	// SSES -> SS (caresses -> caress)
	if strings.HasSuffix(word, "sses") {
		return word[:len(word)-2]
	}

	// IES -> I (ponies -> poni)
	if strings.HasSuffix(word, "ies") {
		return word[:len(word)-2]
	}

	// SS -> SS (caress -> caress)
	if strings.HasSuffix(word, "ss") {
		return word
	}

	// S -> (nothing) (cats -> cat)
	if strings.HasSuffix(word, "s") && len(word) > 3 {
		return word[:len(word)-1]
	}

	return word
}

// step1b handles -ED and -ING
func (ps *PorterStemmer) step1b(word string) string {
	// EED -> EE (agreed -> agree)
	if strings.HasSuffix(word, "eed") {
		if ps.measure(word[:len(word)-3]) > 0 {
			return word[:len(word)-1]
		}
		return word
	}

	// ED -> (nothing) (played -> play)
	if strings.HasSuffix(word, "ed") {
		stem := word[:len(word)-2]
		if ps.containsVowel(stem) {
			return ps.step1bHelper(stem)
		}
		return word
	}

	// ING -> (nothing) (playing -> play)
	if strings.HasSuffix(word, "ing") {
		stem := word[:len(word)-3]
		if ps.containsVowel(stem) {
			return ps.step1bHelper(stem)
		}
		return word
	}

	return word
}

// step1bHelper handles post-processing after removing -ED/-ING
func (ps *PorterStemmer) step1bHelper(word string) string {
	// AT -> ATE, BL -> BLE, IZ -> IZE
	if strings.HasSuffix(word, "at") || strings.HasSuffix(word, "bl") || strings.HasSuffix(word, "iz") {
		return word + "e"
	}

	// Double consonant -> single (hopping -> hop)
	if len(word) >= 2 {
		last := word[len(word)-1]
		prev := word[len(word)-2]
		if last == prev && ps.isConsonant(rune(last)) && last != 'l' && last != 's' && last != 'z' {
			return word[:len(word)-1]
		}
	}

	// Short word -> add E (hop -> hope)
	if ps.measure(word) == 1 && ps.endsWithCVC(word) {
		return word + "e"
	}

	return word
}

// step1c handles Y -> I
func (ps *PorterStemmer) step1c(word string) string {
	if strings.HasSuffix(word, "y") {
		stem := word[:len(word)-1]
		if ps.containsVowel(stem) {
			return stem + "i"
		}
	}
	return word
}

// step2 handles double suffixes
func (ps *PorterStemmer) step2(word string) string {
	suffixes := map[string]string{
		"ational": "ate",
		"tional":  "tion",
		"enci":    "ence",
		"anci":    "ance",
		"izer":    "ize",
		"alli":    "al",
		"entli":   "ent",
		"eli":     "e",
		"ousli":   "ous",
		"ization": "ize",
		"ation":   "ate",
		"ator":    "ate",
		"alism":   "al",
		"iveness": "ive",
		"fulness": "ful",
		"ousness": "ous",
		"aliti":   "al",
		"iviti":   "ive",
		"biliti":  "ble",
	}

	for suffix, replacement := range suffixes {
		if strings.HasSuffix(word, suffix) {
			stem := word[:len(word)-len(suffix)]
			if ps.measure(stem) > 0 {
				return stem + replacement
			}
		}
	}

	return word
}

// step3 handles -ic-, -full, -ness etc.
func (ps *PorterStemmer) step3(word string) string {
	suffixes := map[string]string{
		"icate": "ic",
		"ative": "",
		"alize": "al",
		"iciti": "ic",
		"ical":  "ic",
		"ful":   "",
		"ness":  "",
	}

	for suffix, replacement := range suffixes {
		if strings.HasSuffix(word, suffix) {
			stem := word[:len(word)-len(suffix)]
			if ps.measure(stem) > 0 {
				return stem + replacement
			}
		}
	}

	return word
}

// step4 handles various suffixes
func (ps *PorterStemmer) step4(word string) string {
	suffixes := []string{
		"al", "ance", "ence", "er", "ic", "able", "ible", "ant",
		"ement", "ment", "ent", "ion", "ou", "ism", "ate", "iti",
		"ous", "ive", "ize",
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) {
			stem := word[:len(word)-len(suffix)]
			if ps.measure(stem) > 1 {
				// Special case for -ion
				if suffix == "ion" && len(stem) > 0 {
					last := stem[len(stem)-1]
					if last == 's' || last == 't' {
						return stem
					}
				} else {
					return stem
				}
			}
		}
	}

	return word
}

// step5 handles -e removal
func (ps *PorterStemmer) step5(word string) string {
	if strings.HasSuffix(word, "e") {
		stem := word[:len(word)-1]
		m := ps.measure(stem)
		if m > 1 || (m == 1 && !ps.endsWithCVC(stem)) {
			return stem
		}
	}

	// Remove double L if measure > 1
	if len(word) > 1 && strings.HasSuffix(word, "ll") {
		if ps.measure(word) > 1 {
			return word[:len(word)-1]
		}
	}

	return word
}

// measure counts the number of consonant-vowel sequences
func (ps *PorterStemmer) measure(word string) int {
	count := 0
	inVowelSeq := false

	for _, r := range word {
		if ps.isVowel(r) {
			inVowelSeq = true
		} else if inVowelSeq {
			count++
			inVowelSeq = false
		}
	}

	return count
}

// containsVowel checks if word contains a vowel
func (ps *PorterStemmer) containsVowel(word string) bool {
	for _, r := range word {
		if ps.isVowel(r) {
			return true
		}
	}
	return false
}

// isVowel checks if a rune is a vowel
func (ps *PorterStemmer) isVowel(r rune) bool {
	r = unicode.ToLower(r)
	return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u'
}

// isConsonant checks if a rune is a consonant
func (ps *PorterStemmer) isConsonant(r rune) bool {
	return !ps.isVowel(r) && unicode.IsLetter(r)
}

// endsWithCVC checks if word ends with consonant-vowel-consonant
func (ps *PorterStemmer) endsWithCVC(word string) bool {
	if len(word) < 3 {
		return false
	}

	runes := []rune(word)
	n := len(runes)

	last := runes[n-1]
	middle := runes[n-2]
	first := runes[n-3]

	// Check pattern: consonant-vowel-consonant (but not w, x, or y at end)
	return ps.isConsonant(first) &&
		ps.isVowel(middle) &&
		ps.isConsonant(last) &&
		last != 'w' && last != 'x' && last != 'y'
}
