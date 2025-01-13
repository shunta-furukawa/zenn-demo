package math

func Multiply(a, b int) int {
	for i := 0; i < b; i++ {
		a = Add(a, a)
	}
	return a
}
