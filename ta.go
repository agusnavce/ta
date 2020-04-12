package ta


import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/agusnavce/ta/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/tidwall/gjson"
)

type suggestionLevel int
type deletes map[uint32]struct{}

// Verbosity constants
const (
	// BEST will yield 'best' suggestion
	BEST suggestionLevel = iota
	// CLOSEST will yield 'closest' suggestion
	CLOSEST 
	//ALL will yield 'all' suggestion
	ALL 
)

// SpellModel provides access to functions for spelling correction
type SpellModel struct {
	MaxEditDistance uint32
	PrefixLength uint32

	cumulativeFreq uint64
	dictionaryDeletes *utils.DictionaryDeletes
	longestWord uint32
	library *utils.Library
}

// Main constants
const (
	defaultDict         = "default"
	defaultEditDistance = 2
	defaultPrefixLength = 7
)

// Load a dictionary from disk from filename. Returns a new Spell instance on
// success, or will return an error if there's a problem reading the file.
func Load(filename string) (*SpellModel, error) {
	s := NewSpellModel()

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(gz)
	if err != nil {
		return nil, err
	}

	err = f.Close()
	if err != nil {
		return nil, err
	}

	err = gz.Close()
	if err != nil {
		return nil, err
	}

	// Load the words
	gj := gjson.ParseBytes(data)
	gj.Get("words").ForEach(func(dictionary, entries gjson.Result) bool {
		entries.ForEach(func(word, definition gjson.Result) bool {
			e := utils.Entry{}
			if err := mapstructure.Decode(definition.Value(), &e); err != nil {
				log.Fatal(err)
			}

			if _, err := s.AddEntry(e); err != nil {
				log.Fatal(err)
			}
			return true
		})
		return true
	})

	if gj.Get("options.editDistance").Exists() {
		s.MaxEditDistance = uint32(gj.Get("options.editDistance").Int())
	}

	if gj.Get("options.prefixLength").Exists() {
		s.PrefixLength = uint32(gj.Get("options.prefixLength").Int())
	}

	return s, nil
}

// NewSpellModel to instanciate
func NewSpellModel() *SpellModel {
	model := new(SpellModel)
	return model.Init()
}

// Init function
func (model *SpellModel) Init() *SpellModel {
	s := new(SpellModel)
	s.cumulativeFreq = 0
	s.dictionaryDeletes = utils.NewDictionaryDeletes()
	s.longestWord = 0
	s.MaxEditDistance = defaultEditDistance
	s.PrefixLength = defaultPrefixLength
	s.library = utils.NewLibrary()
	return s
}

// AddEntry adds an entry to the dictionary. If the word already exists its data
// will be overwritten if override is present if not it will update. Returns true if a new word was added, false otherwise.
// Will return an error if there was a problem adding a word
func (model *SpellModel) AddEntry(de utils.Entry, opts ...utils.DictionaryOption) (bool, error) {
	dictOptions := model.defaultDictOptions()

	for _, opt := range opts {
		if err := opt(dictOptions); err != nil {
			return false, err
		}
	}

	word := de.Word

	atomic.AddUint64(&model.cumulativeFreq, de.Frequency)

	// If the word already exists, just update its result - we don't need to
	// recalculate the deletes as these should never change
	if entry, exists := model.library.Load(dictOptions.Name, word); exists {
		atomic.AddUint64(&model.cumulativeFreq, ^(de.Frequency - 1))
		if !dictOptions.OverrideFrequency{
			de.Frequency = de.Frequency + entry.Frequency	
		}
		if !dictOptions.OverrideWordData {
			de.WordData = entry.WordData
		}
		model.library.Store(dictOptions.Name, word, de)
		return false, nil
	}

	model.library.Store(dictOptions.Name, word, de)

	// Keep track of the longest word in the dictionary
	wordLength := uint32(len([]rune(word)))
	if wordLength > atomic.LoadUint32(&model.longestWord) {
		atomic.StoreUint32(&model.longestWord, wordLength)
	}

	// Get the deletes for the word. For each delete, hash it and associate the
	// word with it
	deletes := model.getDeletes(word)
	if len(deletes) > 0 {
		wordRunes := []rune(word)

		de := utils.DeleteEntry{
			Len:   len(wordRunes),
			Runes: wordRunes,
			Str:   word,
		}
		for deleteHash := range deletes {
			model.dictionaryDeletes.Add(dictOptions.Name, deleteHash, &de)
		}
	}

	return true, nil
}



// AddEntries adds multiple string entries to the dictionary with
// same info.
func (model *SpellModel) AddEntries(entries utils.Entries,  opts ...utils.DictionaryOption) (bool, error){
	dictOptions := model.defaultDictOptions()

	for _, opt := range opts {
		if err := opt(dictOptions); err != nil {
			return false, err
		}
	}

	for index, word := range entries.Words {
		if index == 0 {
			model.AddEntry(utils.Entry{
					Frequency: 1,
					Word: word,
					WordData: entries.WordsData,
				}, 
				OverrideFrequency(dictOptions.OverrideFrequency),
				OverrideWordData(dictOptions.OverrideWordData),
			)
		} else {
			model.AddEntry(utils.Entry{
					Frequency: 1,
					Word: word,
					WordData: entries.WordsData,
				}, 
				OverrideWordData(dictOptions.OverrideWordData),
			)
		}
		
	}

	return true, nil
}


// CreateDictionary loads multiple dictionary entries from a file of
// words. Merges with any dictionary data already loaded.
func (model *SpellModel) CreateDictionary(filePath string, opts ...utils.DictionaryOption) (bool, error) {
	dictOpts := model.defaultDictOptions()

	for _, opt := range opts {
		if err := opt(dictOpts); err != nil {
			return false, err
		}
	}

	f, err := os.Open(filePath)

	if err != nil {
		return false, err
	}

	s := bufio.NewScanner(f)

	for s.Scan() {
		model.AddEntry(utils.Entry{
			Frequency: 1, 
			Word: s.Text(),
		})
	}
		
	err = s.Err()
	err = f.Close()

	if  err != nil {
		return false, err
	}

	return true, nil
}

func (model *SpellModel) defaultDictOptions() *utils.DictOptions {
	return &utils.DictOptions{
		Name: defaultDict,
	}
}

// DictionaryName defines the name of the dictionary that should be used when
// storing, deleting, looking up words, etc. If not set, the default dictionary
// will be used
func DictionaryName(name string) utils.DictionaryOption {
	return func(opts *utils.DictOptions) error {
		opts.Name = name
		return nil
	}
}

// OverrideWordData defines if when adding a new entry this overrides previous
// information.
func OverrideWordData(override bool) utils.DictionaryOption {
	return func(opts *utils.DictOptions) error {
		opts.OverrideWordData = override
		return nil
	}
}

// OverrideFrequency defines if when adding a new entry this overrides previous
// frequencies
func OverrideFrequency(override bool) utils.DictionaryOption {
	return func(opts *utils.DictOptions) error {
		opts.OverrideFrequency = override
		return nil
	}
}

// GetEntry returns the Entry for word. If a word does not exist, nil will
// be returned
func (model *SpellModel) GetEntry(word string, opts ...utils.DictionaryOption) (*utils.Entry, error) {
	dictOpts := model.defaultDictOptions()

	for _, opt := range opts {
		if err := opt(dictOpts); err != nil {
			return nil, err
		}
	}

	if entry, exists := model.library.Load(dictOpts.Name, word); exists {
		return &entry, nil
	}
	return nil, nil
}

// RemoveEntry removes a entry from the dictionary. Returns true if the entry
// was removed, false otherwise
func (model *SpellModel) RemoveEntry(word string, opts ...utils.DictionaryOption) (bool, error) {
	dictOpts := model.defaultDictOptions()

	for _, opt := range opts {
		if err := opt(dictOpts); err != nil {
			return false, err
		}
	}

	return model.library.Remove(dictOpts.Name, word), nil
}

// RemoveEntries bathc remove of entries
func (model *SpellModel) RemoveEntries(words []string, opts ...utils.DictionaryOption) (bool, error) {
	dictOpts := model.defaultDictOptions()

	for _, opt := range opts {
		if err := opt(dictOpts); err != nil {
			return false, err
		}
	}

	for _, word := range words {
		model.RemoveEntry(word, DictionaryName(dictOpts.Name))
	}

	return true, nil
}


// Save a representation of spell to disk at filename
func (model *SpellModel) Save(filename string) error {
	jsonStr, _ := json.Marshal(map[string]interface{}{
		"options": map[string]interface{}{
			"editDistance": model.MaxEditDistance,
			"prefixLength": model.PrefixLength,
		},
		"words": model.library.Dictionaries,
	})

	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	w := gzip.NewWriter(f)
	_, err = w.Write(jsonStr)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return nil
}


type lookupParams struct {
	dictOpts         *utils.DictOptions
	distanceFunction func([]rune, []rune, int) int
	editDistance     uint32
	prefixLength     uint32
	sortFunc         func(utils.SuggestionList)
	suggestionLevel  suggestionLevel
}

func (model *SpellModel) defaultLookupParams() *lookupParams {
	return &lookupParams{
		dictOpts:         model.defaultDictOptions(),
		distanceFunction: utils.DamerauLevenshteinRunes,
		editDistance:     model.MaxEditDistance,
		prefixLength:     model.PrefixLength,
		sortFunc: func(results utils.SuggestionList) {
			sort.Slice(results, func(i, j int) bool {
				s1 := results[i]
				s2 := results[j]

				if s1.Distance < s2.Distance {
					return true
				} else if s1.Distance == s2.Distance {
					return s1.Frequency > s2.Frequency
				}

				return false
			})
		},
		suggestionLevel: BEST,
	}
}

// LookupOption is a function that controls how a Lookup is performed. An error
// will be returned if the LookupOption is invalid.
type LookupOption func(*lookupParams) error

// DictionaryOpts accepts multiple DictionaryOption and controls what
// dictionary should be used during lookup
func DictionaryOpts(opts ...utils.DictionaryOption) LookupOption {
	return func(params *lookupParams) error {
		for _, opt := range opts {
			if err := opt(params.dictOpts); err != nil {
				return err
			}
		}
		return nil
	}
}

// DistanceFunc accepts a function, f(str1, str2, maxDist), which calculates the
// distance between two strings. It should return -1 if the distance between the
// strings is greater than maxDist.
func DistanceFunc(df func([]rune, []rune, int) int) LookupOption {
	return func(lp *lookupParams) error {
		lp.distanceFunction = df
		return nil
	}
}

// EditDistance allows the max edit distance to be set for the Lookup. Reducing
// the edit distance will improve lookup performance.
func EditDistance(dist uint32) LookupOption {
	return func(lp *lookupParams) error {
		lp.editDistance = dist
		return nil
	}
}

// SortFunc allows the sorting of the SuggestionList to be configured. By
// default, suggestions will be sorted by their edit distance, then their
// frequency.
func SortFunc(sf func(utils.SuggestionList)) LookupOption {
	return func(lp *lookupParams) error {
		lp.sortFunc = sf
		return nil
	}
}

// SuggestionLevel defines how many results are returned for the lookup. See the
// package constants for the levels available.
func SuggestionLevel(level suggestionLevel) LookupOption {
	return func(lp *lookupParams) error {
		lp.suggestionLevel = level
		return nil
	}
}

// PrefixLength defines how much of the input word should be used for the
// lookup.
func PrefixLength(prefixLength uint32) LookupOption {
	return func(lp *lookupParams) error {
		if prefixLength < 1 {
			return errors.New("prefix length must be greater than 0")
		}
		lp.prefixLength = prefixLength
		return nil
	}
}

func (model *SpellModel) newDictSuggestion(input string, dist int, dp *utils.DictOptions) utils.Suggestion {
	entry, _ := model.library.Load(dp.Name, input)

	return utils.Suggestion{
		Distance: dist,
		Entry:    entry,
	}
}

// Lookup takes an input and returns suggestions from the dictionary for that
// word. By default it will return the best suggestion for the word if it
// exists.
//
// Accepts zero or more LookupOption that can be used to configure how lookup
// occurs.
func (model *SpellModel) Lookup(input string, opts ...LookupOption) (utils.SuggestionList, error) {
	lookupParams := model.defaultLookupParams()

	for _, opt := range opts {
		if err := opt(lookupParams); err != nil {
			return nil, err
		}
	}

	results := utils.SuggestionList{}
	dict := lookupParams.dictOpts.Name

	// Check for an exact match
	if _, exists := model.library.Load(dict, input); exists {
		results = append(results, model.newDictSuggestion(input, 0, lookupParams.dictOpts))

		if lookupParams.suggestionLevel != ALL {
			return results, nil
		}
	}

	editDistance := int(lookupParams.editDistance)

	// If edit distance is 0, just check if input is in the dictionary
	if editDistance == 0 {
		return results, nil
	}

	inputRunes := []rune(input)
	inputLen := len(inputRunes)
	prefixLength := int(lookupParams.prefixLength)

	// Keep track of the deletes we've already considered
	consideredDeletes := make(map[string]struct{})

	// Keep track of the suggestions we've already considered
	consideredSuggestions := make(map[string]struct{})
	consideredSuggestions[input] = struct{}{}

	// Keep a list of words we want to try
	var candidates []string

	// Restrict the length of the input we'll examine
	inputPrefixLen := utils.Min(inputLen, prefixLength)
	candidates = append(candidates, utils.Substring(input, 0, inputPrefixLen))

	for i := 0; i < len(candidates); i++ {
		candidate := candidates[i]
		candidateLen := len([]rune(candidate))
		lengthDiff := inputPrefixLen - candidateLen

		// If the difference between the prefixed input and candidate is larger
		// than the max edit distance then skip the candidate
		if lengthDiff > editDistance {
			if lookupParams.suggestionLevel == ALL {
				continue
			}
			break
		}

		candidateHash := utils.GetStringHash(candidate)
		if suggestions, exists := model.dictionaryDeletes.Load(dict, candidateHash); exists {
			for _, suggestion := range suggestions {
				suggestionLen := suggestion.Len

				// Ignore the suggestion if it equals the input
				if suggestion.Str == input {
					continue
				}

				// Skip the suggestion if:
				// * Its length difference to the input is greater than the max
				//   edit distance
				// * Its length is less than the current candidate (occurs in
				//   the tae of hash collision)
				// * Its length is the same as the candidate and is *not* the
				//   candidate (in the tae of a hash collision)
				if utils.Abs(suggestionLen-inputLen) > editDistance ||
					suggestionLen < candidateLen ||
					(suggestionLen == candidateLen && suggestion.Str != candidate) {
					continue
				}

				// Skip suggestion if its edit distance is too far from input
				suggPrefixLen := utils.Min(suggestionLen, prefixLength)
				if suggPrefixLen > inputPrefixLen &&
					(suggPrefixLen-candidateLen) > editDistance {
					continue
				}

				var dist int

				// If the candidate is an empty string and maps to a bin with
				// suggestions (i.e. hash collision), ignore the suggestion if
				// its edit distance with the input is greater than max edit
				// distance
				if candidateLen == 0 {
					dist = utils.Max(inputLen, suggestionLen)
					if dist > editDistance ||
						!utils.AddKey(consideredSuggestions, suggestion.Str) {
						continue
					}
				} else if suggestionLen == 1 {

					// If the length of the suggestion is 1, determine if the
					// input contains the suggestion. If it does than the edit
					// distance is input - 1, otherwise it's the length of the
					// input
					if strings.Contains(input, suggestion.Str) {
						dist = inputLen - 1
					} else {
						dist = inputLen
					}

					if dist > editDistance ||
						!utils.AddKey(consideredSuggestions, suggestion.Str) {
						continue
					}
				} else {
					if !utils.AddKey(consideredSuggestions, suggestion.Str) {
						continue
					}
					if dist = lookupParams.distanceFunction(inputRunes, suggestion.Runes, editDistance); dist < 1 {
						continue
					}
				}

				// Determine whether or not this suggestion should be added to
				// the results and if so, how.
				if dist <= editDistance {
					if len(results) > 0 {
						switch lookupParams.suggestionLevel {
						case CLOSEST:
							if dist < editDistance {
								results = utils.SuggestionList{}
							}
						case BEST:
							entry, _ := model.library.Load(lookupParams.dictOpts.Name, suggestion.Str)

							curFreq := entry.Frequency
							closestFreq := results[0].Frequency

							if dist < editDistance || curFreq > closestFreq {
								editDistance = dist
								results[0] = model.newDictSuggestion(suggestion.Str, dist, lookupParams.dictOpts)
							}
							continue
						}
					}

					if lookupParams.suggestionLevel != ALL {
						editDistance = dist
					}

					results = append(results,
						model.newDictSuggestion(suggestion.Str, dist, lookupParams.dictOpts))
				}

			}
		}

		// Add additional candidates
		if lengthDiff < editDistance && candidateLen <= prefixLength {

			if lookupParams.suggestionLevel != ALL && lengthDiff > editDistance {
				continue
			}

			for i := 0; i < candidateLen; i++ {
				deleteWord := utils.RemoveChar(candidate, i)

				if utils.AddKey(consideredDeletes, deleteWord) {
					candidates = append(candidates, deleteWord)
				}
			}
		}
	}

	// Order the results
	lookupParams.sortFunc(results)

	return results, nil
}

type segmentParams struct {
	lookupOptions []LookupOption
}

func (model *SpellModel) defaultSegmentParams() *segmentParams {
	return &segmentParams{
		lookupOptions: []LookupOption{
			SuggestionLevel(BEST),
		},
	}
}

// SegmentOption is a function that controls how a Segment is performed. An
// error will be returned if the SegmentOption is invalid.
type SegmentOption func(*segmentParams) error

// SegmentLookupOpts allows the Lookup() options for the current segmentation to
// be configured
func SegmentLookupOpts(opt ...LookupOption) SegmentOption {
	return func(sp *segmentParams) error {
		sp.lookupOptions = opt
		return nil
	}
}

// Segment contains details about an individual segment
type Segment struct {
	Input string
	Entry *utils.Entry
	Word  string
}

// SegmentResult holds the result of a call to Segment()
type SegmentResult struct {
	Distance int
	Segments []Segment
}

// GetWords returns a string slice of words for the segments
func (s SegmentResult) GetWords() []string {
	words := make([]string, 0, len(s.Segments))
	for _, s := range s.Segments {
		words = append(words, s.Word)
	}
	return words
}

// String returns a string representation of the SegmentList.
func (s SegmentResult) String() string {
	return strings.Join(s.GetWords(), " ")
}

// Segment takes an input string which may have word concatenations, and
// attempts to divide it into the most likely set of words by adding spaces at
// the most appropriate positions.
//
// Accepts zero or more SegmentOption that can be used to configure how
// segmentation occurs
func (model *SpellModel) Segment(input string, opts ...SegmentOption) (*SegmentResult, error) {
	segmentParams := model.defaultSegmentParams()

	for _, opt := range opts {
		if err := opt(segmentParams); err != nil {
			return nil, err
		}
	}

	longestWord := int(atomic.LoadUint32(&model.longestWord))
	if longestWord == 0 {
		return nil, errors.New("longest word in dictionary has zero length")
	}

	cumulativeFreq := float64(atomic.LoadUint64(&model.cumulativeFreq))
	if cumulativeFreq == 0 {
		return nil, errors.New("cumulative frequency is zero")
	}

	inputLen := len([]rune(input))

	arraySize := utils.Min(inputLen, longestWord)
	circularIdx := -1

	type composition struct {
		segmentedString string
		correctedString string
		distanceSum     int
		probability     float64
	}
	compositions := make([]composition, arraySize)

	for i := 0; i < inputLen; i++ {

		jMax := utils.Min(inputLen-i, longestWord)

		for j := 1; j <= jMax; j++ {
			part := utils.Substring(input, i, i+j)

			separatorLength := 0
			topEd := 0
			topProbabilityLog := 0.0
			topResult := ""

			if unicode.Is(unicode.White_Space, rune(part[0])) {
				part = utils.Substring(input, i+1, i+j)
			} else {
				separatorLength = 1
			}

			topEd += len([]rune(part))
			part = strings.Replace(part, " ", "", -1)
			topEd -= len([]rune(part))

			suggestions, err := model.Lookup(part, segmentParams.lookupOptions...)
			if err != nil {
				return nil, err
			}

			if len(suggestions) > 0 {
				topResult = suggestions[0].Entry.Word
				topEd += suggestions[0].Distance

				freq := suggestions[0].Frequency
				topProbabilityLog = math.Log10(float64(freq) / cumulativeFreq)
			} else {
				// Unknown word
				topResult = part
				topEd += len([]rune(part))
				topProbabilityLog = math.Log10(10.0 / (cumulativeFreq *
					math.Pow(10.0, float64(len([]rune(part))))))
			}

			destinationIdx := (j + circularIdx) % arraySize

			if i == 0 {
				compositions[destinationIdx] = composition{
					segmentedString: part,
					correctedString: topResult,
					distanceSum:     topEd,
					probability:     topProbabilityLog,
				}
			} else if j == longestWord ||
				((compositions[circularIdx].distanceSum+topEd ==
					compositions[destinationIdx].distanceSum ||
					compositions[circularIdx].distanceSum+separatorLength+topEd ==
						compositions[destinationIdx].distanceSum) &&
					compositions[destinationIdx].probability < compositions[circularIdx].probability+topProbabilityLog) ||
				compositions[circularIdx].distanceSum+separatorLength+topEd <
					compositions[destinationIdx].distanceSum {
				compositions[destinationIdx] = composition{
					segmentedString: compositions[circularIdx].segmentedString + " " + part,
					correctedString: compositions[circularIdx].correctedString + " " + topResult,
					distanceSum:     compositions[circularIdx].distanceSum + separatorLength + topEd,
					probability:     compositions[circularIdx].probability + topProbabilityLog,
				}
			}
		}

		circularIdx++
		if circularIdx == arraySize {
			circularIdx = 0
		}
	}

	segmentedString := compositions[circularIdx].segmentedString
	correctedString := compositions[circularIdx].correctedString
	segmentedWords := strings.Split(segmentedString, " ")
	correctedWords := strings.Split(correctedString, " ")
	segments := make([]Segment, len(correctedWords))

	for i, word := range correctedWords {
		e, err := model.GetEntry(word)
		if err != nil {
			return nil, err
		}

		segments[i] = Segment{
			Input: segmentedWords[i],
			Word:  word,
			Entry: e,
		}
	}

	result := SegmentResult{
		Distance: compositions[circularIdx].distanceSum,
		Segments: segments,
	}

	return &result, nil
}

func (model *SpellModel) generateDeletes(word string, editDistance uint32, deletes deletes) deletes {
	editDistance++

	if wordLen := len([]rune(word)); wordLen > 1 {
		for i := 0; i < wordLen; i++ {
			deleteWord := utils.RemoveChar(word, i)
			deleteHash := utils.GetStringHash(deleteWord)

			if _, exists := deletes[deleteHash]; !exists {
				deletes[deleteHash] = struct{}{}

				if editDistance < model.MaxEditDistance {
					model.generateDeletes(deleteWord, editDistance, deletes)
				}
			}

		}
	}

	return deletes
}


func (model *SpellModel) getDeletes(word string) deletes {
	deletes := deletes{}
	wordLen := len([]rune(word))

	// Restrict the size of the word to the max length of the prefix we'll
	// examine
	if wordLen > int(model.PrefixLength) {
		word = utils.Substring(word, 0, int(model.PrefixLength))
	}

	wordHash := utils.GetStringHash(word)
	deletes[wordHash] = struct{}{}

	return model.generateDeletes(word, 0, deletes)
}




