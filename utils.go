package gotils

// OrString returns s1 if it's not "", otherwise s2
// Similar to JavaScript's x || y syntax.
func OrString(s1, s2 string) string {
	if s1 != "" {
		return s1
	}
	return s2
}
