package utils

import (
	"sync"
)


// Library is a collection of Dictionaries
type Library struct {
	sync.RWMutex
	Dictionaries map[string]Dictionary
}

// NewLibrary creates a new library
func NewLibrary() *Library {
	return &Library{
		Dictionaries: make(map[string]Dictionary),
	}
}

// Load checks if a word exists in a given dictionary
func (l *Library) Load(dict, word string) (Entry, bool) {
	l.RLock()
	definition, exists := l.Dictionaries[dict][word]
	l.RUnlock()
	return definition, exists
}

// Store adds a word to a given dictionary
func (l *Library) Store(dict, word string, definition Entry) {
	l.Lock()
	if _, exists := l.Dictionaries[dict]; !exists {
		l.Dictionaries[dict] = make(Dictionary)
	}

	l.Dictionaries[dict][word] = definition

	l.Unlock()
}

// Remove deletes a word from a given dictionary
func (l *Library) Remove(dict, word string) bool {
	l.Lock()
	defer l.Unlock()

	if _, exists := l.Dictionaries[dict][word]; exists {
		delete(l.Dictionaries[dict], word)
		return true
	}

	return false
}

