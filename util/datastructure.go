package util

import (
	"strings"
)

var (
	_exists = struct{}{}
)

type StringSet map[string]struct{}

func NewStringSet() StringSet {
	return make(StringSet)
}

func WithStringSet(ws []string) StringSet {
	ss := NewStringSet()
	for _, s := range ws {
		if s != "" {
			ss.Add(s)
		}
	}

	return ss
}

func WithLowerStringSet(ws []string) StringSet {
	ss := NewStringSet()
	for _, s := range ws {
		if s != "" {
			ss.Add(strings.ToLower(s))
		}
	}

	return ss
}

func (s StringSet) Add(x string) {
	s[x] = _exists
}

func (s StringSet) Remove(x string) {
	if s.Contains(x) {
		delete(s, x)
	}
}

func (s StringSet) Contains(x string) bool {
	_, ok := s[x]
	return ok
}

func (s StringSet) Len() int {
	return len(s)
}

func (s StringSet) List() []string {
	list := make([]string, s.Len())

	i := 0
	for k, _ := range s {
		list[i] = k
		i++
	}

	return list
}

func (s StringSet) Merge(x StringSet) {
	if x.Len() != 0 {
		for _, v := range x.List() {
			s.Add(v)
		}
	}
}
