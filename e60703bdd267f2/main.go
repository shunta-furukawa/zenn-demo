package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes,
		Dir:  "./example", // 解析対象のディレクトリを指定
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Println("Error loading packages:", err)
		return
	}

	// `main` 関数を探して解析
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			analyzeMainFunction(file, pkg.Fset)
		}
	}
}

// `main` 関数の呼び出し順を解析
func analyzeMainFunction(file *ast.File, fset *token.FileSet) {
	// ファイル内のノードを探索
	ast.Inspect(file, func(n ast.Node) bool {
		// `main` 関数を探す
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			fmt.Printf("Analyzing calls in function: %s\n", fn.Name.Name)

			// 関数内の呼び出し順を解析
			extractCallSequence(fn.Body, 0)
		}
		return true
	})
}

// 関数呼び出しを順番に出力する
func extractCallSequence(node ast.Node, depth int) {
	// 関数呼び出しをリストアップする
	ast.Inspect(node, func(n ast.Node) bool {
		// 関数呼び出しを検出
		if call, ok := n.(*ast.CallExpr); ok {
			printIndentedCall(call, depth)

			// 再帰的に深い呼び出しを解析しない
			return false // 子ノードを探索しないことで重複を防止
		}
		return true
	})
}

// 呼び出しをインデント付きで出力
func printIndentedCall(call *ast.CallExpr, depth int) {
	indent := strings.Repeat("  ", depth)
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr: // パッケージ名を含む呼び出し
		fmt.Printf("%s%s.%s\n", indent, getIdentName(fun.X), fun.Sel.Name)
	case *ast.Ident: // ローカル関数呼び出し
		fmt.Printf("%s%s\n", indent, fun.Name)
	default:
		fmt.Printf("%sUnknown function call\n", indent)
	}
}

// セレクタのパッケージ名や変数名を取得
func getIdentName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "unknown"
}
