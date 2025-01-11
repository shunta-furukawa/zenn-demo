# protoファイルからのコード生成: protoc と buf の比較

この記事では、[gRPCのKeepalive設定について調べてみた](https://zenn.dev/shunta_furukawa/articles/562e8d092d264f)で使用したプロジェクトフォルダ構成を基に、`.proto`ファイルからGoおよびgRPCのコードを生成する方法を、従来の`protoc`と新しいツールである`buf`を使って比較し、`buf`の利便性を紹介します。

## プロジェクトフォルダ構成

まず、プロジェクトのディレクトリ構成は以下のとおりです。
（関係ないファイルは省略しています） 

```protobuf
.
├── example/
├── example.proto
└── go.mod
```

`example.proto`の内容は以下のとおりです。

```protobuf
syntax = "proto3";

package example;

option go_package = "github.com/shunta-furukawa/zenn-demo/986d1e236326cd/example";

service YourService {
  rpc YourRPCMethod (YourRequest) returns (YourResponse);
}

message YourRequest {
  string name = 1;
}

message YourResponse {
  string message = 1;
}
```

この構成で `example` 配下に protoファイルから生成した goファイルを出力することを目指します。

## `protoc`を使用したコード生成

従来、`protoc`を使用してコード生成を行う手順は以下のとおりです。

1. **必要なツールのインストール:**

   Go用のプラグインをインストールします。

   ```zsh
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

2. **環境変数の設定:**

   Goのバイナリディレクトリを`PATH`に追加します。

   ```zsh
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

3. **コード生成の実行:**

   `protoc`コマンドを使用して、GoおよびgRPCのコードを生成します。

   ```zsh
   protoc --go_out=./example --go_opt=paths=source_relative \
   --go-grpc_out=./example --go-grpc_opt=paths=source_relative \
   ./example.proto 
   ```

この手順では、複数のツールのインストールや環境設定が必要であり、コマンドも複雑です。

## `buf`を使用したコード生成

`buf`を使用すると、これらの手順が簡素化されます。

1. **`buf`のインストール:**

   `buf`をインストールします。

   ```zsh
   brew install bufbuild/buf/buf
   ```

2. **プロジェクトの初期化:**

   プロジェクトディレクトリで以下のコマンドを実行して、`buf.yaml`を作成します。

   ```zsh
   buf config init
   ```

   これにより、次のような`buf.yaml`ファイルが生成されます。

   ```zsh
   version: v2
   ```

3. **コード生成設定の作成:**

   `buf.gen.yaml`ファイルを手動で作成し、次の内容を記述します。

   ```yaml
   version: v2
   plugins:
     - remote: buf.build/protocolbuffers/go
       out: example
       opt: paths=source_relative
     - remote: buf.build/grpc/go
       out: example
       opt: paths=source_relative
   ```

4. **コード生成の実行:**

   以下のコマンドを実行して、コード生成を行います。

   ```zsh
   buf generate
   ```

`buf`を使用することで、設定ファイルに必要な情報を記述し、シンプルなコマンドでコード生成が可能となります。

## `buf`のメリット

- **一元管理:** 設定ファイルでプロジェクトの構成やコード生成の設定を一元管理できます。
- **依存関係の自動管理:** リモートプラグインを指定することで、ローカル環境にプラグインをインストールする必要がなく、依存関係の管理が容易です。
- **簡潔なコマンド:** `buf generate`コマンドだけで複数のコード生成を実行でき、コマンドの複雑さが軽減されます。
- **高機能:** `buf`はLintやブレイキングチェンジの検出など、プロトコルバッファの管理に役立つ多くの機能を提供しています。

これらの理由から、`buf`を使用することで、プロトコルバッファのコード生成や管理がより効率的かつ簡潔になります。
