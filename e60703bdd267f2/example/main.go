package main

import (
	"github.com/shunta-furukawa/zenn-demo/e60703bdd267f2/example/math"
	"github.com/shunta-furukawa/zenn-demo/e60703bdd267f2/example/utils"
)

func main() {
	result := math.Add(3, 5)
	utils.PrintResult("Addition", result)

	result = math.Multiply(4, 7)
	utils.PrintResult("Multiplication", result)
}
