package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps,
		Dir:  "./example", // 解析対象のディレクトリを指定
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Println("Error loading packages:", err)
		return
	}

	// パッケージ情報をマップに格納 (後で依存関係解析に使用)
	pkgMap := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		pkgMap[pkg.PkgPath] = pkg
	}

	// `main` 関数を探して解析
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			analyzeMainFunction(file, pkg.Fset, pkg.TypesInfo, pkgMap)
		}
	}
}

// `main` 関数の呼び出し順を解析
func analyzeMainFunction(file *ast.File, fset *token.FileSet, typesInfo *types.Info, pkgMap map[string]*packages.Package) {
	ast.Inspect(file, func(n ast.Node) bool {
		// `main` 関数を探す
		if fn, ok := n.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			fmt.Printf("Analyzing calls in function: %s\n", fn.Name.Name)

			// 関数内の呼び出し順を解析
			visited := make(map[string]bool)
			extractCallSequence(fn.Body, 0, typesInfo, pkgMap, visited)
		}
		return true
	})
}

// 関数呼び出しを順番に出力し、再帰的に解析
func extractCallSequence(node ast.Node, depth int, typesInfo *types.Info, pkgMap map[string]*packages.Package, visited map[string]bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			printIndentedCall(call, depth)

			// 呼び出し先の関数を再帰的に解析
			if fn := getFunctionDefinition(call, typesInfo, pkgMap); fn != nil {
				// 関数のフルパスを取得してループを防ぐ
				fullName := fmt.Sprintf("%s.%s", fn.Pkg, fn.Name)
				if !visited[fullName] {
					visited[fullName] = true
					extractCallSequence(fn.Node.Body, depth+1, typesInfo, pkgMap, visited)
				}
			}
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

// 呼び出し先関数の定義を取得
type FunctionDefinition struct {
	Pkg  string        // パッケージ名
	Name string        // 関数名
	Node *ast.FuncDecl // 関数ノード
}

func getFunctionDefinition(call *ast.CallExpr, typesInfo *types.Info, pkgMap map[string]*packages.Package) *FunctionDefinition {
	// 型情報から関数オブジェクトを取得
	obj := typesInfo.ObjectOf(getCallIdent(call))
	if obj == nil || obj.Pkg() == nil {
		// 無効な呼び出しまたは外部依存関係
		return nil
	}

	// 呼び出し元のパッケージを特定
	pkg := pkgMap[obj.Pkg().Path()]
	if pkg == nil {
		return nil
	}

	// パッケージ内の関数定義を探索
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == obj.Name() {
				return &FunctionDefinition{
					Pkg:  obj.Pkg().Name(),
					Name: obj.Name(),
					Node: fn,
				}
			}
		}
	}
	return nil
}

// 関数呼び出しの識別子を取得
func getCallIdent(call *ast.CallExpr) *ast.Ident {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun
	case *ast.SelectorExpr:
		return fun.Sel
	}
	return nil
}

// セレクタのパッケージ名や変数名を取得
func getIdentName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "unknown"
}
