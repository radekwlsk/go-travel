package utils

func IfThenElse(cond bool, a interface{}, b interface{}) interface{} {
	if cond {
		return a
	} else {
		return b
	}
}
