package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// FunctionDefinition は、呼び出し先関数の情報をまとめた構造体
type FunctionDefinition struct {
	Pkg  string        // パッケージ名 (構造体やメソッドが属する実装パッケージ)
	Name string        // 関数(メソッド)名
	Node *ast.FuncDecl // 関数ノード
}

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedSyntax |
			packages.NeedFiles |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedDeps,
		Dir: "./example", // 解析対象ディレクトリ。必要に応じて調整
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Println("Error loading packages:", err)
		return
	}

	// パッケージ情報をマップに格納 (あとで依存関係解析に使用)
	pkgMap := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		pkgMap[pkg.PkgPath] = pkg
	}

	// --- 例: main 関数の呼び出し解析をやりたい場合 ---
	fmt.Println("=== Analyzing main function calls ===")
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			analyzeMainFunction(file, pkg.Fset, pkg.TypesInfo, pkgMap)
		}
	}

	// --- gRPC サービス実装の解析 ---
	fmt.Println("\n=== Analyzing gRPC service registrations ===")
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			analyzeGRPCRegistration(file, pkg.Fset, pkg.TypesInfo, pkgMap)
		}
	}
}

// analyzeMainFunction は、main 関数を探して呼び出しを解析
func analyzeMainFunction(file *ast.File, fset *token.FileSet, typesInfo *types.Info, pkgMap map[string]*packages.Package) {
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		if fn.Name.Name == "main" {
			fmt.Printf("Analyzing calls in function: %s\n", fn.Name.Name)
			visited := make(map[string]bool)
			extractCallSequence(fn.Body, 0, typesInfo, pkgMap, visited)
		}
		return true
	})
}

// analyzeGRPCRegistration は、RegisterExampleServiceServer(...) の呼び出しを探し、
// 第2引数 (サーバ実装) の型を調べてその実装メソッドを解析する。
func analyzeGRPCRegistration(file *ast.File, fset *token.FileSet, typesInfo *types.Info, pkgMap map[string]*packages.Package) {
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// 関数呼び出しが RegisterExampleServiceServer(...) かどうか
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel != nil && sel.Sel.Name == "RegisterExampleServiceServer" {
				// 引数が2つ: RegisterExampleServiceServer(registrar, server)
				if len(call.Args) == 2 {
					serverArg := call.Args[1]
					analyzeServerArg(serverArg, typesInfo, pkgMap)
				}
			}
		}
		return true
	})
}

// analyzeServerArg は RegisterExampleServiceServer の第2引数 (サーバ実装) の型を取得し、
// その「実装パッケージ」へ移動してメソッドを AST 解析する。
func analyzeServerArg(serverArg ast.Expr, typesInfo *types.Info, pkgMap map[string]*packages.Package) {
	obj := typesInfo.ObjectOf(getIdent(serverArg))
	if obj == nil {
		return
	}
	serverType := obj.Type()
	if serverType == nil {
		return
	}
	fmt.Printf("[ServerArg] Type: %s\n", serverType.String())

	// ポインタ型等を剥がして最終的に *types.Named を取り出す
	underlying := serverType
	for {
		if ptr, ok := underlying.(*types.Pointer); ok {
			underlying = ptr.Elem()
		} else {
			break
		}
	}
	named, _ := underlying.(*types.Named)
	if named == nil {
		fmt.Printf("Not a named type: %s\n", underlying.String())
		return
	}

	// ---- ここがポイント: 実際の「構造体を定義しているパッケージ」を取得 ----
	serverPkgPath := named.Obj().Pkg().Path()
	serverPkg := pkgMap[serverPkgPath]
	if serverPkg == nil {
		fmt.Printf("Server package not found in pkgMap: %s\n", serverPkgPath)
		return
	}
	fmt.Printf("Analyzing server implementation package: %s\n", serverPkgPath)

	// サーバ型(構造体)のメソッド一覧を列挙
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if !method.Exported() {
			continue
		}
		// AST から該当のメソッド定義 (FuncDecl) を探す
		fnDef := findFunctionDeclInPackage(serverPkg, method)
		if fnDef == nil {
			continue
		}
		// RPC 実装メソッドを解析
		fmt.Printf("Analyzing RPC method: %s.%s\n", fnDef.Pkg, fnDef.Name)
		visited := make(map[string]bool)
		extractCallSequence(fnDef.Node.Body, 0, serverPkg.TypesInfo, pkgMap, visited)
	}
}

// findFunctionDeclInPackage は指定パッケージ内で、指定されたメソッド (types.Object) に合致する
// `FuncDecl` を探し、見つかったら返す。
func findFunctionDeclInPackage(pkg *packages.Package, method types.Object) *FunctionDefinition {
	for _, f := range pkg.Syntax {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			// 関数名 (メソッド名) が合致するか
			// レシーバー型の一致チェックなどを厳密にやる場合は、fn.Recv をさらに確認
			// （本サンプルでは名前だけを比較）
			if fn.Name.Name == method.Name() {
				return &FunctionDefinition{
					Pkg:  method.Pkg().Name(), // or pkg.Name
					Name: method.Name(),
					Node: fn,
				}
			}
		}
	}
	return nil
}

// extractCallSequence は与えられたノード配下の関数呼び出しを順番に出力し、
// さらに呼び出し先関数が同パッケージ内にあれば再帰的に解析する。
func extractCallSequence(node ast.Node, depth int, typesInfo *types.Info, pkgMap map[string]*packages.Package, visited map[string]bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok {
			printIndentedCall(call, depth)

			// 呼び出し先の関数定義を取得
			if fn := getFunctionDefinition(call, typesInfo, pkgMap); fn != nil {
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

// getFunctionDefinition は、呼び出し先関数が同パッケージ内に定義されている場合に探し出す
func getFunctionDefinition(call *ast.CallExpr, typesInfo *types.Info, pkgMap map[string]*packages.Package) *FunctionDefinition {
	obj := typesInfo.ObjectOf(getCallIdent(call))
	if obj == nil || obj.Pkg() == nil {
		return nil
	}
	pkg := pkgMap[obj.Pkg().Path()]
	if pkg == nil {
		return nil
	}
	// パッケージ内の関数を探索
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == obj.Name() {
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

// printIndentedCall は呼び出しをインデント付きで表示するユーティリティ
func printIndentedCall(call *ast.CallExpr, depth int) {
	indent := strings.Repeat("  ", depth)
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		fmt.Printf("%s%s.%s\n", indent, getIdentName(fun.X), fun.Sel.Name)
	case *ast.Ident:
		fmt.Printf("%s%s\n", indent, fun.Name)
	default:
		fmt.Printf("%s(Unknown call)\n", indent)
	}
}

// getCallIdent は、CallExpr の呼び出し先識別子 (関数名) を取得
func getCallIdent(call *ast.CallExpr) *ast.Ident {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun
	case *ast.SelectorExpr:
		return fun.Sel
	}
	return nil
}

// getIdentName は SelectorExpr のパッケージ名や変数名を取得
func getIdentName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "unknown"
}

// getIdent は、CallExpr の引数などが Ident / SelectorExpr の場合に対応して識別子を抜き出す
func getIdent(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return e.Sel
	}
	return nil
}
