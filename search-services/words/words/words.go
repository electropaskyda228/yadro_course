package words

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
)

var forbittenWords []string = []string{"of", "the", "a", "and", "or",
	"will", "would", "i", "me", "you", "your",
	"he", "his", "him", "who", "it", "that",
	"she", "her", "we",
	"our", "they", "their", "them"}
var isPermissioned map[string]bool

func init() {
	isPermissioned = make(map[string]bool)
	for _, word := range forbittenWords {
		isPermissioned[word] = true
	}
}

func splitByNonAlphanumericUnicode(str string) []string {
	var result []string
	var current strings.Builder

	for _, r := range str {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func Norm(phrase string) []string {
	words := splitByNonAlphanumericUnicode(phrase)
	result := make(map[string]bool, 0)
	for _, word := range words {
		tmp, err := snowball.Stem(word, "english", true)
		if err == nil {
			if !isPermissioned[tmp] {
				result[tmp] = true
			}
		} else {
			if !isPermissioned[word] {
				result[word] = true
			}
		}
	}
	answer := make([]string, 0)
	for key := range result {
		answer = append(answer, key)
	}

	return answer
}
