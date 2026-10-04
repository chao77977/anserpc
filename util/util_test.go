package util

import (
	"testing"
)

func TestStringSet_AddContainsRemove(t *testing.T) {
	s := NewStringSet()
	if s.Contains("a") {
		t.Fatal("empty set should not contain 'a'")
	}

	s.Add("a")
	s.Add("b")
	if !s.Contains("a") || !s.Contains("b") {
		t.Fatal("set should contain added elements")
	}
	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2", s.Len())
	}

	s.Remove("a")
	if s.Contains("a") {
		t.Fatal("'a' should have been removed")
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, want 1", s.Len())
	}

	// Removing a non-existent key is a no-op.
	s.Remove("zzz")
	if s.Len() != 1 {
		t.Fatalf("Len = %d after no-op remove, want 1", s.Len())
	}
}

func TestWithStringSet_SkipsEmpty(t *testing.T) {
	s := WithStringSet([]string{"x", "", "y"})
	if s.Len() != 2 {
		t.Fatalf("Len = %d, want 2 (empty string skipped)", s.Len())
	}
	if !s.Contains("x") || !s.Contains("y") {
		t.Fatal("expected x and y present")
	}
	if s.Contains("") {
		t.Fatal("empty string should be skipped")
	}
}

func TestWithLowerStringSet(t *testing.T) {
	s := WithLowerStringSet([]string{"ABC", "DeF"})
	if !s.Contains("abc") || !s.Contains("def") {
		t.Fatalf("expected lower-cased entries, got %v", s.List())
	}
	if s.Contains("ABC") {
		t.Fatal("original cased key should not be present")
	}
}

func TestStringSet_List(t *testing.T) {
	s := WithStringSet([]string{"a", "b", "c"})
	list := s.List()
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}
	seen := map[string]bool{}
	for _, v := range list {
		seen[v] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !seen[want] {
			t.Fatalf("List missing %q: %v", want, list)
		}
	}
}

func TestStringSet_Merge(t *testing.T) {
	a := WithStringSet([]string{"a", "b"})
	b := WithStringSet([]string{"b", "c"})
	a.Merge(b)
	if a.Len() != 3 {
		t.Fatalf("merged Len = %d, want 3", a.Len())
	}
	for _, want := range []string{"a", "b", "c"} {
		if !a.Contains(want) {
			t.Fatalf("merged set missing %q", want)
		}
	}

	// Merging an empty set changes nothing.
	a.Merge(NewStringSet())
	if a.Len() != 3 {
		t.Fatalf("Len = %d after merging empty, want 3", a.Len())
	}
}

func TestFormatName(t *testing.T) {
	cases := map[string]string{
		"Network":      "network",
		"ABC":          "abc",
		"abc":          "abc",
		"":             "",
		"MixedCase123": "mixedcase123",
	}
	for in, want := range cases {
		if got := FormatName(in); got != want {
			t.Errorf("FormatName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFmt(t *testing.T) {
	// No args: returned verbatim.
	if got := Fmt("plain string, no verbs"); got != "plain string, no verbs" {
		t.Errorf("Fmt without args = %q, want verbatim", got)
	}
	// With args: formats.
	if got := Fmt("%s:%d", "host", 80); got != "host:80" {
		t.Errorf("Fmt = %q, want host:80", got)
	}
}

func TestIsTemporaryError(t *testing.T) {
	if IsTemporaryError(nil) {
		t.Error("nil should not be temporary")
	}
	if IsTemporaryError(errPlain("boom")) {
		t.Error("plain error should not be temporary")
	}
	if !IsTemporaryError(tempErr{}) {
		t.Error("error with Temporary()==true should be temporary")
	}
	if IsTemporaryError(tempErr{notTemp: true}) {
		t.Error("error with Temporary()==false should not be temporary")
	}
}

type errPlain string

func (e errPlain) Error() string { return string(e) }

type tempErr struct{ notTemp bool }

func (e tempErr) Error() string   { return "temp" }
func (e tempErr) Temporary() bool { return !e.notTemp }
