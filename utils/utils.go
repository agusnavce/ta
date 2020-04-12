package utils

// Entries defines multiple entries
type Entries struct {
	Words []string
	WordsData WordData
}

// Abs function
func Abs( arg int ) int {
	
	if arg < 0{
		arg = -arg
	}
	
	return arg 
}

//Min function
func Min(arg ...int  ) int {
	
	res := findMin(arg)	

	return res
}

func findMin(a [] int)  int{
	var min int = a[0] 
	for _,arg := range a{
		if (min>arg){
			min = arg
		}
	}
	return min
}

//Max function
func Max(arg ...int  ) int {
	
	res := findMax(arg)	

	return res
}


func findMax(a [] int)  int{
	var max int = 0
	for _,arg := range a{
		if (max<arg){
			max = arg
		}
	}
	return max
}

//CompareSlices function
func CompareSlices(a, b []rune) bool {
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// GetCharCosts function
func GetCharCosts(length, maxDist int, x []int) []int {
	if x == nil {
		x = make([]int, length)
	}

	i := 0
	for ; i < maxDist; i++ {
		x[i] = i + 1
	}
	for ; i < length; i++ {
		x[i] = maxDist + 2
	}

	return x
}

//GetLenDiff function
func GetLenDiff(s1Len, s2Len, maxDist int) (int, int, *int) {
	lenDiff := s2Len - s1Len
	toReturn := -1

	if maxDist > s2Len {
		maxDist = s2Len
	} else if lenDiff > maxDist {
		return lenDiff, maxDist, &toReturn
	}

	return lenDiff, maxDist, nil
}

// SwapRunes function
func SwapRunes(r1, r2 []rune, maxDist int) ([]rune, []rune, int, int, *int) {
	toReturn := -1
	r1Len := len(r1)
	r2Len := len(r2)

	if maxDist < 0 {
		return r1, r2, r1Len, r2Len, &toReturn
	}

	if r1Len > r2Len {
		r1, r2 = r2, r1
		r1Len, r2Len = r2Len, r1Len
	}

	if r1Len == 0 {
		if r2Len <= maxDist {
			return r1, r2, r1Len, r2Len, &r2Len
		}
		return r1, r2, r1Len, r2Len, &toReturn
	}

	return r1, r2, r1Len, r2Len, nil
}

// IgnoreSuffix function
func IgnoreSuffix(s1, s2 []rune, s1Len, s2Len int) (int, int) {
	for s1Len > 0 && s1[s1Len-1] == s2[s2Len-1] {
		s1Len--
		s2Len--
	}

	return s1Len, s2Len
}

// AddKey function
func AddKey(hash map[string]struct{}, key string) bool {
	if _, exists := hash[key]; exists {
		return false
	}

	hash[key] = struct{}{}

	return true
}

// GetStringHash FNV-1a hash implementation
func GetStringHash(str string) uint32 {
	var h uint32 = 2166136261
	for _, c := range []byte(str) {
		h ^= uint32(c)
		h *= 16777619
	}
	return h
}

// RemoveChar function
func RemoveChar(str string, index int) string {
	return Substring(str, 0, index) + Substring(str, index+1, len([]rune(str)))
}

// Substring function
func Substring(s string, start int, end int) string {
	if start >= len([]rune(s)) {
		return ""
	}

	startStrIdx := 0
	i := 0

	for j := range s {
		if i == start {
			startStrIdx = j
		}
		if i == end {
			return s[startStrIdx:j]
		}
		i++
	}
	return s[startStrIdx:]
}