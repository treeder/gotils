package gotils

import "math/rand"

// ChooseRandomString chooses a random element from slice
func ChooseRandomString(sl []string) string {
	if sl == nil {
		return ""
	}
	return sl[rand.Intn(len(sl))]
}
