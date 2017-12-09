package utils

func StringIn(value string, strings []string) bool {
	for _, s := range strings {
		if s == value {
			return true
		}
	}
	return false
}
