package utils

import (
	"sync"
)

// Dictionary is a mapping of a word to its dictionary entry
type Dictionary map[string]Entry

// Entry represents a word in the dictionary
type Entry struct {
	Frequency uint64 `json:",omitempty"`
	Word      string
	WordData  WordData `json:",omitempty"`
}

// WordData stores metadata about a word.
type WordData map[string]interface{}

// DictionaryDeletes stores the deletes entries
type DictionaryDeletes struct {
	sync.RWMutex
	dictionaries map[string]deletesMap
}

type deletesMap map[uint32][]*DeleteEntry

// DeleteEntry is a delete word
type DeleteEntry struct {
	Len   int
	Runes []rune
	Str   string
}

// DictOptions are the dictionary options
type DictOptions struct {
	Name string
	OverrideFrequency bool
	OverrideWordData bool
}

// DictionaryOption is a function that controls the dictionary being used.
// An error will be returned if a dictionary option is invalid
type DictionaryOption func(*DictOptions) error


// NewDictionaryDeletes creates a new dictionary with deletes
func NewDictionaryDeletes() *DictionaryDeletes {
	return &DictionaryDeletes{
		dictionaries: make(map[string]deletesMap),
	}
}

// Load checks if a word exists in a given dictionary
func (dd *DictionaryDeletes) Load(dict string, key uint32) ([]*DeleteEntry, bool) {
	dd.RLock()
	entry, exists := dd.dictionaries[dict][key]
	dd.RUnlock()
	return entry, exists
}

// Add a word to a given dictionary
func (dd *DictionaryDeletes) Add(dict string, key uint32, entry *DeleteEntry) {
	dd.Lock()
	if _, exists := dd.dictionaries[dict]; !exists {
		dd.dictionaries[dict] = make(deletesMap)
	}

	dd.dictionaries[dict][key] = append(dd.dictionaries[dict][key], entry)
	dd.Unlock()
}
