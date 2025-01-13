# Go のコードをプログラマティックに把握する

大規模な Go プロジェクトに携わっていると、関数やモジュール間の依存関係を正確に把握するのは非常に重要です。しかし、手動でそれを行うのは非効率であり、全体像を見失いがちです。本記事では、Go のコード解析ツールである `go/packages` を活用し、依存関係をプログラマティックに把握する方法を紹介します。

特に、`main` 関数がどのような関数を呼び出しているのかを解析し、その依存関係を明らかにする具体的な例を示します。

---

## どのツールを選ぶべき？

Go のコードを解析するための主要なツールには、`go/ast/inspector` と `go/packages` があります。それぞれに得意分野があり、用途に応じて選択する必要があります。

### go/ast/inspector
- **特徴**:
  - Go の抽象構文木 (AST) を効率的に探索するためのツール。
  - 特定のノード (関数定義や変数宣言など) を迅速に抽出可能。
- **適用場面**:
  - 軽量なコード解析ツールや静的解析ツールを構築する際に適している。
  - 単一ファイルや小規模な解析タスクに強い。

### go/packages
- **特徴**:
  - パッケージ全体を解析し、型情報、依存関係、構文情報 (AST) を取得可能。
  - 大規模なプロジェクトや複数パッケージをまたぐ解析に対応。
- **適用場面**:
  - プロジェクト全体の依存関係を追跡したり、型情報を伴う詳細な解析を行う場合に適している。
  - ドキュメント生成ツールやリファクタリングツールの基盤として利用可能。

**選択理由**:
本記事では、型情報や複数パッケージにまたがる解析を行いたいため、`go/packages` を選択します。

---

## go/packages を使って main 関数から呼ばれている関数を出力してみる

### サンプルプロジェクトの説明

今回の例では、以下のような構成の簡単なプロジェクト `example` を用意しました。

```
example/
├── go.mod
├── main.go
├── math/
│   ├── add.go
│   └── multiply.go
└── utils/
    └── printer.go
```

#### 各ファイルの内容

**`main.go`**:
```go
package main

import (
	"example/math"
	"example/utils"
)

func main() {
	result := math.Add(3, 5)
	utils.PrintResult("Addition", result)

	result = math.Multiply(4, 7)
	utils.PrintResult("Multiplication", result)
}
```

**`math/add.go`**:
```go
package math

func Add(a, b int) int {
	return a + b
}
```

**`math/multiply.go`**:
```go
package math

func Multiply(a, b int) int {
	var r int
	for i := 0; i < b; i++ {
		r = Add(r, a)
	}
	return r
}
```

**`utils/printer.go`**:
```go
package utils

import "fmt"

func PrintResult(operation string, result int) {
	fmt.Printf("%s result: %d\n", operation, result)
}
```

---

### 解析している main.go の説明

#### 全体の流れ
本プログラムでは、以下の手順で `main` 関数から呼び出される関数を出力します。

1. `go/packages` を使ってプロジェクト全体をロードします。
2. ロードされたパッケージの中から `main` 関数を探します。
3. `main` 関数内の関数呼び出し (`CallExpr`) を抽出します。
4. 型情報を使って呼び出し先関数を特定し、再帰的に解析します。

#### コード

以下が解析プログラムのコードです。

```go
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
```

## 解析コードの詳細な解説 

### 関数単位でのコード解説

---

#### **`main` 関数**
```go
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
```
**解説:**
- `cfg`:
  - `packages.Config` を設定して、解析するディレクトリや取得する情報を指定。
  - `Mode` フィールドで必要な情報 (名前、構文、型情報、依存関係など) を指定。
- `packages.Load`:
  - Go プロジェクト全体をロードし、解析対象のパッケージ情報を取得。
  - ロードされたパッケージを `pkgs` に格納。
- `pkgMap`:
  - パッケージ情報をマップ形式で保存。
- `analyzeMainFunction`:
  - 各パッケージの構文情報 (`pkg.Syntax`) を走査して `main` 関数を解析。

---

#### **`analyzeMainFunction` 関数**
```go
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
```
**解説:**
- `ast.Inspect`:
  - AST (抽象構文木) を再帰的に走査し、ノード情報を取得。
- `FuncDecl` チェック:
  - `main` 関数 (`FuncDecl`) を検出し、その中身を解析。
- `extractCallSequence`:
  - `main` 関数の中で呼び出される関数を解析し、依存関係を出力。

---

#### **`extractCallSequence` 関数**
```go
func extractCallSequence(node ast.Node, depth int, typesInfo *types.Info, pkgMap map[string]*packages.Package, visited map[string]bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			printIndentedCall(call, depth)

			// 呼び出し先の関数を再帰的に解析
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
```
**解説:**
- `CallExpr` チェック:
  - 関数呼び出し (`CallExpr`) を検出。
- `printIndentedCall`:
  - 現在の関数呼び出しを出力。
- `getFunctionDefinition`:
  - 呼び出された関数の定義を取得。
- 再帰処理:
  - 呼び出し先の関数を再帰的に解析。
  - `visited` マップで再解析を防止。

---

#### **`printIndentedCall` 関数**
```go
func printIndentedCall(call *ast.CallExpr, depth int) {
	indent := strings.Repeat("  ", depth)
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		fmt.Printf("%s%s.%s\n", indent, getIdentName(fun.X), fun.Sel.Name)
	case *ast.Ident:
		fmt.Printf("%s%s\n", indent, fun.Name)
	default:
		fmt.Printf("%sUnknown function call\n", indent)
	}
}
```
**解説:**
- 呼び出しの種類に応じて出力形式を調整。
  - `SelectorExpr`: パッケージ名付きの呼び出し (`pkg.Func` 形式)。
  - `Ident`: ローカル関数呼び出し。
  - その他の形式も出力。

---

#### **`getFunctionDefinition` 関数**
```go
func getFunctionDefinition(call *ast.CallExpr, typesInfo *types.Info, pkgMap map[string]*packages.Package) *FunctionDefinition {
	// 型情報から関数オブジェクトを取得
	obj := typesInfo.ObjectOf(getCallIdent(call))
	if obj == nil || obj.Pkg() == nil {
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
```
**解説:**
- `typesInfo.ObjectOf`:
  - 呼び出し元の型情報を取得し、関数オブジェクトを特定。
- パッケージ内の探索:
  - 対象の関数名 (`obj.Name()`) と一致する関数を AST 内で探す。
- 結果として関数定義 (`FunctionDefinition`) を返す。

---

#### **`getCallIdent` 関数**
```go
func getCallIdent(call *ast.CallExpr) *ast.Ident {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun
	case *ast.SelectorExpr:
		return fun.Sel
	}
	return nil
}
```
**解説:**
- 呼び出し元 (`CallExpr.Fun`) から識別子を取得。
  - `Ident`: ローカル関数。
  - `SelectorExpr`: パッケージ付き関数呼び出し。

---

#### **`getIdentName` 関数**
```go
func getIdentName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "unknown"
}
```
**解説:**
- 任意の式 (`ast.Expr`) から識別子名を取得。
- 未知の形式の場合は "unknown" を返す。

### Goコード解析ツールの仕組み

`main.go` の全体の仕組みは、Go のコードベースを解析して、`main` 関数内で呼び出される関数の依存関係を出力することを目的としています。以下に、その全体の流れを簡潔に説明します。

---

#### 1. プロジェクトのロードと設定
まず、`go/packages` を利用して、指定したディレクトリ (`./example`) に含まれる Go プロジェクト全体をロードします。この際、以下の情報を取得します:
- パッケージ名 (`NeedName`)
- 抽象構文木 (AST) (`NeedSyntax`)
- 型情報 (`NeedTypes`, `NeedTypesInfo`)
- 依存関係 (`NeedDeps`)

これにより、プロジェクト全体の解析が可能となります。

---

#### 2. `main` 関数の検出
ロードしたパッケージ内のすべてのファイルを走査し、AST を解析して `main` 関数 (`FuncDecl`) を特定します。これにより、解析対象を限定します。

---

#### 3. 関数呼び出しの解析
`main` 関数内のすべての関数呼び出し (`CallExpr`) を再帰的に解析します。この際、次の手順を踏みます:
1. 関数呼び出しを検出 (`CallExpr`)。
2. 型情報 (`types.Info`) を使用して、呼び出し元の関数定義を特定。
3. 再帰的に依存関係を解析し、結果を出力。

---

#### 4. インデント付きで関数の依存関係を出力
各関数呼び出しは、深さ (`depth`) に応じたインデント付きで出力されます。これにより、依存関係の階層構造を視覚的に把握できます。

---

#### 主な使用関数のまとめ
- **`main`**:
  全体の流れを管理し、`go/packages` を使ってプロジェクトをロード。
- **`analyzeMainFunction`**:
  AST を解析して `main` 関数を特定。
- **`extractCallSequence`**:
  関数呼び出しを再帰的に解析し、依存関係を追跡。
- **`getFunctionDefinition`**:
  型情報を基に呼び出し先の関数定義を取得。
- **`printIndentedCall`**:
  関数呼び出しをインデント付きで出力。

---

## 出力例

このコードを実行すると、以下のような出力が得られます。

```
Analyzing calls in function: main
math.Add
utils.PrintResult
  fmt.Printf
math.Multiply
  Add
utils.PrintResult
```

---

## まとめ

本記事では、`go/packages` を使って `main` 関数から呼び出される関数を解析し、依存関係を出力する仕組みを構築しました。この手法を発展させることで、将来的には gRPC や HTTP サーバのハンドラーから呼び出される関数の依存関係を検出し、それをシーケンス図として可視化する仕組みを構築したいと考えています。

また、`go/packages` は依存関係や型情報の解析にも非常に強力であり、プロジェクト全体の構造を理解するのに役立つツールです。

このプロジェクトはその第一歩となる取り組みです。これからさらに深掘りしていく予定ですので、ぜひ参考にしてみてください！

--- 

## 本日のサンプルコード

↓ サンプルコード

https://github.com/shunta-furukawa/zenn-demo/tree/main/e60703bdd267f2

---

## 参考文献

- [go/ast/inspector - Go Documentation](https://pkg.go.dev/golang.org/x/tools/go/ast/inspector)
- [go/packages - Go Documentation](https://pkg.go.dev/golang.org/x/tools/go/packages)



