# TA (Text Analysis)


An implmentation in Go of  a blazing fast spellchecker inspired in [SymSpell](https://github.com/wolfgarbe/SymSpell).

## How to use it

```go
package main

import (
	"fmt"

	"github.com/agusnavce/ta"
	"github.com/agusnavce/ta/utils"
)

func main() {
	// Create a new instance of model
	t := ta.NewSpellModel()

	
	// Add words to the dictionary. Words require a frequency, but can have
	// other arbitrary metadata associated with them
	t.AddEntry(utils.Entry{
		Frequency: 100,
		Word:      "word",
		WordData: utils.WordData{
			"type": "noun",
		},
	})

	t.AddEntry(utils.Entry{
		Frequency: 1,
		Word:      "world",
		WordData: utils.WordData{
			"type": "noun",
		},
	})

	// Lookup a mismodeling, by default the "best" suggestion will be returned
	suggestions, _ := t.Lookup("wortd")
	fmt.Println(suggestions)
	// -> [word]


	suggestion := suggestions[0]

	// Get the frequency from the suggestion
	fmt.Println(suggestion.Frequency)
	// -> 100

	// Get metadata from the suggestion
	fmt.Println(suggestion.WordData["type"])
	// -> noun

	// Get multiple suggestions during lookup
	suggestions, _ = t.Lookup("wortd", ta.SuggestionLevel(ta.ALL))
	fmt.Println(suggestions)
	// -> [word, world]

    // Add multiple entries in one command. You can override the previos config or 
    // have a cumulative behaviour
	t.AddEntries(utils.Entries{
		Words: []string{"word", "word"}, 
		WordsData: utils.WordData{
			"type": "other",
		},
    }, ta.OverrideFrequency(true))
    
	// Save the dictionary
	t.Save("dict.model")

	// Load the dictionary
	t2, _ := ta.Load("dict.model")

	suggestions, _ = t2.Lookup("wortd", ta.SuggestionLevel(ta.ALL))
	fmt.Println(suggestions)
	// -> [word, world]

	// Create a Dictionary from a file, merges with any dictionary data already loaded.
	t2.CreateDictionary("test.txt")

	entry, err := t2.GetEntry("four")

	if err == nil {
		fmt.Println(entry.Word)
		// -> four
	}

	// Spell supports word segmentation
	t3 := ta.NewSpellModel()

	t3.AddEntry(utils.Entry{Frequency: 1, Word: "near"})
	t3.AddEntry(utils.Entry{Frequency: 1, Word: "the"})
	t3.AddEntry(utils.Entry{Frequency: 1, Word: "fireplace"})

	segmentResult, _ := t3.Segment("nearthefireplace")
	fmt.Println(segmentResult)
	// -> near the fireplace

	// Spell supports multiple dictionaries
	t4 := ta.NewSpellModel()

	t4.AddEntry(utils.Entry{Word: "quindici"}, ta.DictionaryName("italian"))
	suggestions, _ = t4.Lookup("quindici", ta.DictionaryOpts(
		ta.DictionaryName("italian"),
	))
	fmt.Println(suggestions)
	// -> [quindici]
}
```