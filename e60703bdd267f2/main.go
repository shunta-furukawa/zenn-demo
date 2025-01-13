package main

import (
	"fmt"
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes,
		Dir:  "./example", // 解析対象のディレクトリを指定
	}

	// パッケージをロード
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Println("Error loading packages:", err)
		return
	}

	for _, pkg := range pkgs {
		fmt.Println("Package:", pkg.PkgPath)
		if len(pkg.Syntax) == 0 {
			fmt.Println("No syntax information found.")
			continue
		}

		// ファイルごとに解析
		for _, file := range pkg.Syntax {
			analyzeFile(file, pkg.Fset)
		}
	}
}

func analyzeFile(file *ast.File, fset *token.FileSet) {
	ast.Inspect(file, func(n ast.Node) bool {
		// 関数定義を解析
		if fn, ok := n.(*ast.FuncDecl); ok {
			fmt.Printf("- Function: %s\n", fn.Name.Name)
		}
		return true
	})
}
