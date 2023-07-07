package gotils

import "testing"

func TestOrString(t *testing.T) {
	s1 := "hello"
	s2 := "world"
	s3 := OrString(s1, s2)
	if s3 != s1 {
		t.Error("OrString error 1")
	}
	s4 := OrString("", s2)
	if s4 != s2 {
		t.Error("OrString error 2")
	}

}

func TestSubstring(t *testing.T) {
	s := "Mozilla"
	t.Log(Substring(s, 1, 3)) // "M"
	s2 := Substring(s, 1, 3)
	if s2 != "oz" {
		t.Error("Substring error 1")
	}
	s3 := Substring(s, 2, 0)
	t.Log(s3)
	if s3 != "zilla" {
		t.Error("Substring error 2", s3)
	}
	s4 := Substring(s, 0, 7)
	if s4 != "Mozilla" {
		t.Error("Substring error 3")
	}
	s5 := Substring(s, 0, 10)
	if s5 != "Mozilla" {
		t.Error("Substring error 4")
	}
}
