package utils

// Levenshtein takes two strings and a maximum edit distance and returns the number of edits
// to transform one string to another, or -1 if the distance is greater than the
// maximum distance.
func Levenshtein(str1, str2 string, maxDist int) int {
	return LevenshteinRunes([]rune(str1), []rune(str2), maxDist)
}

// LevenshteinRunes is the same as Levenshtein but accepts runes instead of
// strings
func LevenshteinRunes(r1, r2 []rune, maxDist int) int {
	return LevenshteinRunesBuffer(r1, r2, maxDist, nil)
}

// LevenshteinRunesBuffer is the same as LevenshteinRunes but accepts a memory
// buffer x which should be of length max(r1, r2)
func LevenshteinRunesBuffer(r1, r2 []rune, maxDist int, x []int) int {
	if CompareSlices(r1, r2) {
		return 0
	}

	r1, r2, r1Len, r2Len, toReturn := SwapRunes(r1, r2, maxDist)
	if toReturn != nil {
		return *toReturn
	}

	r1Len, r2Len = IgnoreSuffix(r1, r2, r1Len, r2Len)

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

	if x == nil {
		x = make([]int, r2Len)
	}

	x = GetCharCosts(r2Len, maxDist, x)

	jStartOffset := maxDist - lenDiff
	haveMax := maxDist < r2Len
	jStart := 0
	jEnd := maxDist

	current := 0
	for i := 0; i < r1Len; i++ {
		c := r1[start+i]

		left := i
		current = i

		if i > jStartOffset {
			jStart++
		}

		if jEnd < r2Len {
			jEnd++
		}

		for j := jStart; j < jEnd; j++ {
			above := current
			current = left
			left = x[j]

			if c != r2[j] {
				current++

				del := above + 1
				if del < current {
					current = del
				}

				ins := left + 1
				if ins < current {
					current = ins
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