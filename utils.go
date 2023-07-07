package gotils

// OrString returns s1 if it's not "", otherwise s2
// Similar to JavaScript's x || y syntax.
func OrString(s1, s2 string) string {
	if s1 != "" {
		return s1
	}
	return s2
}

// Substring follows same rules as: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/substring
func Substring(s string, indexStart, indexEnd int) string {
	if indexStart < 0 {
		indexStart = 0
	}
	if indexEnd <= 0 {
		indexEnd = len(s)
	}
	if indexEnd > len(s) {
		indexEnd = len(s)
	}
	if indexStart > indexEnd {
		indexStart = indexEnd
	}
	// fmt.Println("indexStart", indexStart, "indexEnd", indexEnd)
	return s[indexStart:indexEnd]
}
