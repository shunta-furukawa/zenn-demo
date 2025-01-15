package math

func Multiply(a, b int) int {
	var r int
	for i := 0; i < b; i++ {
		r = Add(r, a)
	}
	return r
}
