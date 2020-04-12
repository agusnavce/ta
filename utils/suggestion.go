package utils

import "strings"

// Suggestion is used to represent a suggested word from a lookup.
type Suggestion struct {
	// The distance between this suggestion and the input word
	Distance int
	Entry
}

// SuggestionList is a slice of Suggestion
type SuggestionList []Suggestion

// GetWords returns a string slice of words for the suggestions
func (s SuggestionList) GetWords() []string {
	words := make([]string, 0, len(s))
	for _, v := range s {
		words = append(words, v.Entry.Word)
	}
	return words
}

// String returns a string representation of the SuggestionList.
func (s SuggestionList) String() string {
	return "[" + strings.Join(s.GetWords(), ", ") + "]"
}
