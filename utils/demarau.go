package utils

// DamerauLevenshtein takes two strings and a maximum edit distance and returns the number of edits
// to transform one string to another, or -1 if the distance is greater than the
// maximum distance.
func DamerauLevenshtein(str1, str2 string, maxDist int) int {
	return DamerauLevenshteinRunes([]rune(str1), []rune(str2), maxDist)
}

// DamerauLevenshteinRunes is the same as DamerauLevenshtein but accepts runes
// instead of strings
func DamerauLevenshteinRunes(r1, r2 []rune, maxDist int) int {
	return DamerauLevenshteinRunesBuffer(r1, r2, maxDist, nil, nil)
}

// DamerauLevenshteinRunesBuffer is the same as DamerauLevenshteinRunes but
// also accepts memory buffers x and y which should each be of size max(r1, r2).
func DamerauLevenshteinRunesBuffer(r1, r2 []rune, maxDist int, x, y []int) int {
	if CompareSlices(r1, r2) {
		return 0
	}

	r1, r2, r1Len, r2Len, toReturn := SwapRunes(r1, r2, maxDist)
	if toReturn != nil {
		return *toReturn
	}

	r1Len, r2Len = IgnoreSuffix(r1, r2, r1Len, r2Len)

	// Ignore prefix
	start := 0
	if r1[start] == r2[start] || r1Len == 0 {

		for start < r1Len && r1[start] == r2[start] {
			start++
		}
		r1Len -= start
		r2Len -= start

		if r1Len == 0 {
			if r2Len <= maxDist {
				return r2Len
			}
			return -1
		}
	}

	r2 = r2[start : start+r2Len]
	lenDiff, maxDist, toReturn := GetLenDiff(r1Len, r2Len, maxDist)
	if toReturn != nil {
		return *toReturn
	}

	x = GetCharCosts(r2Len, maxDist, x)
	if y == nil {
		y = make([]int, r2Len)
	}

	jStartOffset := maxDist - lenDiff
	haveMax := maxDist < r2Len
	jStart := 0
	jEnd := maxDist
	s1Char := r1[0]
	current := 0
	for i := 0; i < r1Len; i++ {
		prevS1Char := s1Char
		s1Char = r1[start+i]
		s2Char := r2[0]
		left := i
		current = left + 1
		nextTransCost := 0

		if i > jStartOffset {
			jStart++
		}

		if jEnd < r2Len {
			jEnd++
		}

		for j := jStart; j < jEnd; j++ {
			above := current
			thisTransCost := nextTransCost
			nextTransCost = y[j]
			current = left
			y[j] = current
			left = x[j]
			prevS2Char := s2Char
			s2Char = r2[j]
			if s1Char != s2Char {
				if left < current {
					current = left
				}
				if above < current {
					current = above
				}
				current++
				if i != 0 && j != 0 && s1Char == prevS2Char && prevS1Char ==
					s2Char {
					thisTransCost++
					if thisTransCost < current {
						current = thisTransCost
					}
				}
			}
			x[j] = current
		}

		if haveMax && x[i+lenDiff] > maxDist {
			return -1
		}
	}

	return current
}