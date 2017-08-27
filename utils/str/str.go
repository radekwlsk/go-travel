package str

import "strings"

func EmptyDefault(s string, d string) string {
	if len(strings.TrimSpace(string(s))) == 0 {
		return d
	} else {
		return s
	}
}
