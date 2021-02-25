package util

import (
	"unicode"
)

func FormatName(name string) string {
	rs := []rune(name)
	for i := range rs {
		rs[i] = unicode.ToLower(rs[i])
	}

	return string(rs)
}
