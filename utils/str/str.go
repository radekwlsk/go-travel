package str

import "strings"

type String = string

func (s String) EmptyDefault(d string) string {
	if len(strings.TrimSpace(s)) == 0 {
		return d
	} else {
		return s
	}
}
