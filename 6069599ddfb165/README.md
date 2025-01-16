# ✨ gRPCサーバを入り口にプログラマティックに把握する

[前回の記事](https://zenn.dev/shunta_furukawa/articles/e60703bdd267f2)では、`go/packages`を活用し、「`main`関数から呼び出される関数を再帰的に解析する」手法を紹介しました。今回はそれを発展させ、**gRPC サーバを入り口にした静的解析**に挑戦します。

---

## 前回のおさらい

前回の記事では、`go/packages`を活用して、「`main`関数がどの関数を呼び出しているか」を再帰的に追うことが出来ました。

1. AST (`go/ast`)を解析
2. 呼び出し先の型情報を用いて関数を特定
3. 追いた関数内の呼び出し先も再帰的に追う

このフローを完成したことで、Go プロジェクトの関数間の依存関係を自動化して推定する道筋が見えてきました。

これを基にし、今回は「**gRPC サーバに登録された RPC メソッド**から、実際に呼び出される関数を見ていきます。

---

## 今回のテーマ

今回の記事では「**gRPC サーバを入り口にした関数呼び出し解析**」を行います。
具体的には次のようなことを目指します。

1. `grpcServer := grpc.NewServer()` に対して `RegisterExampleServiceServer(grpcServer, &ExampleServer{})` している箇所を探し、そこに渡されている **`ExampleServer` 構造体** を取り出す。
2. その構造体が実装している **gRPC の RPC メソッド** (例: `Culc`) をエントリポイントに、AST を再帰的に解析して「どの関数をどんな順番で呼んでいるか」を可視化する。

ポイントは、**gRPC のインターフェイス定義（`example_grpc.pb.go`）** があるパッケージと、**実装クラス（`ExampleServer`）があるパッケージ** が違う点です。「`go/packages` を使えば自動的に見つかる」と思いきや、しっかりパッケージをたどらないと `Culc` メソッドが見つからずに終わってしまう場合もあります。

---

## 今回のサンプルアプリ構成

今回は、次の構成のサンプルアプリを用意しました。

```
├── example/
│   ├── example/
│   │   ├── example.proto
│   │   └── example_grpc.pb.go   // gRPC インターフェイス定義
│   ├── server/
│   │   ├── server.go            // ExampleServer の実装
│   │   ├── calc_service.go
│   │   └── print_service.go
│   └── main.go                  // gRPC サーバ起動処理
└── main.go (解析ツール)
```

- **`example_grpc.pb.go`**
  - `.proto`ファイルから自動生成された **`ExampleServiceServer` インターフェイス**が定義されています。  
- **`server.go`**
  - 実際のサーバ実装。`type ExampleServer struct { ... }`や `func (s *ExampleServer) Culc(...)` が寫かれています。
- **`main.go` (gRPC サーバ起動)**
  - `RegisterExampleServiceServer(grpcServer, &ExampleServer{})`のように **gRPC サーバに RPC 実装を登録しています**。

この構成で、実際に実装された RPC メソッドが呼び出している関数フローを分析します。

---

## 分析用の `main.go`の解説

今回の分析は「**RegisterExampleServiceServer の第2引数で指定されたサーバ実装からスタート**する」方式を取っています。主に次のステップを課しています。

1. **`RegisterExampleServiceServer(...)`の呼び出しを AST から検出**
2. **第2引数として渡されたサーバ実装を特定**
3. **サーバ実装の実装構造体を分析し、具体の RPC メソッドを追察**
4. **RPC メソッドの本文から呼び出される関数を再帰的に解析**

---

## ポイント: 「実装構造体のあるパッケージ」を追いかける

多くの方がつまずくのが、**インターフェイス定義のパッケージ** と **実装構造体のパッケージ** が異なる点です。`obj.Pkg().Path()` を見ると、インターフェイス側（`example` パッケージ）を指してしまい、そこで探しても `ExampleServer` が見つからない…という状況になりがちです。

そこで、以下のように「ポインタを剥がしたあと、**`named.Obj().Pkg().Path()`** で“構造体が所属する実装パッケージ”を取得」するアプローチを取ります。

```go
// ポインタ型等を剥がして最終的に *types.Named を取り出す
underlying := serverType
for {
    ptr, ok := underlying.(*types.Pointer)
    if !ok {
        break
    }
    underlying = ptr.Elem()
}

// これが Named かチェック
named, _ := underlying.(*types.Named)
if named == nil {
    return
}

// 実際の実装パッケージ
serverPkgPath := named.Obj().Pkg().Path()
serverPkg := pkgMap[serverPkgPath]
```

こうすることで、**`example/server`** パッケージの AST (`server.go` など) を巡回できるようになり、実際の `func (s *ExampleServer) Culc(...)` を発見→解析できます。

---

## 解析コード例

以下に解析用のコードを例示します。

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

// ... 中略 ...

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedSyntax | packages.NeedFiles |
		      packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps,
		Dir: "./example", // 解析対象ディレクトリ
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		fmt.Println("Error loading packages:", err)
		return
	}

	// パッケージをマップにまとめる
	pkgMap := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		pkgMap[pkg.PkgPath] = pkg
	}

	// RegisterExampleServiceServer を探し、解析開始
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			analyzeGRPCRegistration(file, pkg.Fset, pkg.TypesInfo, pkgMap)
		}
	}
}

// RegisterExampleServiceServer(...) を探す
func analyzeGRPCRegistration(file *ast.File, fset *token.FileSet, typesInfo *types.Info, pkgMap map[string]*packages.Package) {
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "RegisterExampleServiceServer" {
			if len(call.Args) == 2 {
				// 第2引数 (サーバ実装)
				serverArg := call.Args[1]
				analyzeServerArg(serverArg, typesInfo, pkgMap)
			}
		}
		return true
	})
}

// 実際のサーバ実装を解析 (構造体のメソッド呼び出しフローを辿る)
func analyzeServerArg(serverArg ast.Expr, typesInfo *types.Info, pkgMap map[string]*packages.Package) {
	obj := typesInfo.ObjectOf(getIdent(serverArg))
	if obj == nil {
		return
	}
	serverType := obj.Type()
	if serverType == nil {
		return
	}

	// ポインタを剥がすなど
	underlying := serverType
	for {
		ptr, ok := underlying.(*types.Pointer)
		if !ok {
			break
		}
		underlying = ptr.Elem()
	}
	named, _ := underlying.(*types.Named)
	if named == nil {
		return
	}

	// 実装パッケージを正しく取得
	serverPkgPath := named.Obj().Pkg().Path()
	serverPkg := pkgMap[serverPkgPath]
	if serverPkg == nil {
		return
	}

	// サーバ構造体のメソッドを列挙し、AST を元に呼び出し解析
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if !method.Exported() {
			continue
		}
		fnDef := findFunctionDeclInPackage(serverPkg, method)
		if fnDef == nil {
			continue
		}
		fmt.Printf("Analyzing RPC method: %s.%s\n", fnDef.Pkg, fnDef.Name)
		visited := make(map[string]bool)
		extractCallSequence(fnDef.Node.Body, 0, serverPkg.TypesInfo, pkgMap, visited)
	}
}

// ... 従来の extractCallSequence, findFunctionDeclInPackage などは前回同様 ...

```

---

## 工夫した点

1. **RegisterExampleServiceServer を探索**  
   - “どのサーバが登録されているか” を動的に拾うことで、分析対象を “本当に使っているサービス” に限定できます。

2. **ポインタ剥がし**  
   - `&ExampleServer{}` のようにポインタ型で渡されると、`obj.Type()` は最初 `*types.Pointer` になりがち。そこでループを使って **`*types.Pointer` → 要素型** を剥がし、“素”の型 (`*types.Named`) を取得しています。

3. **構造体のあるパッケージを追う**  
   - **`named.Obj().Pkg().Path()`** で、実装構造体が定義されているパッケージを引き当て、そこから `server.go` の AST を探索。これにより、RPC 実装メソッド (`func (s *ExampleServer) Culc(...)`) を発見し、再帰的に呼び出し解析が可能になります。

---

## 実行例

実際にこのコードを走らせると、「**どの RPC メソッドを実装しており、その中で何が呼ばれているか**」が階層表示されます。  
例えば、以下のような出力が得られます。

```
=== Analyzing gRPC service registrations ===
[ServerArg] Type: *github.com/shunta-furukawa/zenn-demo/6069599ddfb165/example/server.ExampleServer
Analyzing server implementation package: github.com/shunta-furukawa/zenn-demo/6069599ddfb165/example/server
Analyzing RPC method: server.Culc
unknown.Multiply
  int32
  s.Add
unknown.Print
  fmt.Sprintf
```

この出力により、`Culc` メソッドが `Multiply` (さらに `Add`) を呼んでいる様子がインデント付きで表示され、サービス実装の全体像が把握しやすくなります。

---

## 今後の展望

1. **複数の RPC 呼び出しに対応**  
   - 現在の解析ツールは単一の RPC メソッドにフォーカスしていますが、
     **複数の RPC メソッドが連鎖的に呼び出される場合** でも対応できるよう、
     呼び出しフローをさらに拡張したいです。

2. **"RegisterExampleServiceServer" との比較を動的に行う**  
   - 今回のコードでは "RegisterExampleServiceServer" をハードコードしていますが、
     他のサービス登録メソッドにも対応するように、
     **比較対象を動的に設定できる仕組み** を追加したいです。

3. **パッケージ名が unknown となる問題の解決**  
   - 一部の呼び出しで "unknown" として出力されているパッケージ名を特定し、
     **AST の解析手法や型情報の取得方法を見直して**、正確なパッケージ名を取得できるように改善できたらいいなと思っています。

---

## まとめ

前回は「`main` 関数から呼び出される関数」を中心に解析する方法を紹介しましたが、今回はさらに一歩進めて「**gRPC サーバ実装を入り口に** 呼び出しフローを可視化する」方法を紹介しました。複数のパッケージに散らばる RPC の実装を AST + 型情報でつなぎ合わせるのがポイントです。

- **前回との違い**: gRPC サービスは **インターフェイス定義** と **実装** が別パッケージになりがちなので、単純に `obj.Pkg().Path()` で探索しても引っかからないことが多い。  
- **今回の工夫**: 「実装構造体 (`ExampleServer`) のパッケージ」を取りに行き、そこでメソッドを検索することで、正しくメソッド本体を特定できるようにしました。

大規模プロジェクトであっても、**このように `go/packages` を使って静的解析**を行うと、意外と複雑な依存関係や呼び出しフローを自動抽出できます。今後はさらに発展させて、HTTP ハンドラや CLI コマンドなど、他の “エントリーポイント” に対しても同様の解析ができるよう取り組んでいきたいと思います。

---

