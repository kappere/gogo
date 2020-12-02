package util

func Ternary(expr bool, whenTrue, whenFalse interface{}) interface{} {
	if expr == true {
		return whenTrue
	}
	return whenFalse
}

func Min(a, b int) int {
	i := Ternary(a <= b, a, b)
	r, _ := i.(int)
	return r
}

func Max(a, b int) int {
	i := Ternary(a >= b, a, b)
	r, _ := i.(int)
	return r
}

func ValueOrDefault(v interface{}, dft interface{}) interface{} {
	if v == nil {
		return dft
	}
	return v
}
