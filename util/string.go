package util

import (
	"fmt"
	"unicode"
)

func FormatName(name string) string {
	rs := []rune(name)
	for i := range rs {
		rs[i] = unicode.ToLower(rs[i])
	}

	return string(rs)
}

// skip formatting if args are empty
func Fmt(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}

	return fmt.Sprintf(format, args...)
}
