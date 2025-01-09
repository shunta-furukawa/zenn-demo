# gRPCのKeepalive設定について調べてみた

gRPCを使い始めると、クライアントとサーバー間の接続を安定して維持する **Keepalive** 機能を目にすることがあります。「そもそもHTTP/1.1ではKeepaliveなんて意識しなかったのに、なぜgRPCではわざわざ設定する必要があるの？」と思ったことはありませんか？

本記事では、Keepalive設定がなぜ必要なのか、その役割、そして設定方法や検証手順について、できるだけ噛み砕いて解説します。

---

## 1. なぜgRPCにはKeepalive設定が必要なのか？

### gRPCとHTTP/1.1の違い

gRPCは通信プロトコルとして**HTTP/2**を採用していますが、HTTP/2ではクライアントとサーバーが**長時間の接続を維持しながら多重化通信を行う**のが特徴です。一方、HTTP/1.1は基本的にリクエストごとに接続を作成し、リクエストが完了すると接続を切断する動作（あるいは短時間のみ接続を保持するKeep-Alive）が主流でした。

このように、gRPCでは**長期間の接続が前提**であるため、以下のような新しい課題が生じます。

- **接続の健全性確認が必要**: 長時間アイドル状態になると、ネットワーク障害や中間機器（ファイアウォールやロードバランサー）による切断が起きやすくなります。

- **リアルタイム性の確保**: クライアントが接続の不具合を即座に検知し、再接続やエラー処理を行う必要があります。

- **ストリームベース通信の特性**: gRPCでは単一の接続で複数のRPCを処理するため、接続の維持そのものがサービス全体の信頼性に直結します。

これらの理由から、gRPCでは**Keepalive設定**が導入されています。この設定により、定期的な「PINGフレーム」の送受信を通じて接続を維持し、問題が発生した場合には速やかに対処できるようになります。

---

## 2. Keepalive設定の主要項目

gRPCでは、クライアントとサーバーの両方でKeepalive設定を行うことができます。それぞれの役割と代表的な設定項目を見ていきましょう。

### クライアント側 (`keepalive.ClientParameters`)

- **Time**:  
  クライアントがサーバーに対してPINGフレームを送信する間隔を指定します。これを設定することで、定期的に接続が健全か確認できます（デフォルトでは無効）。

- **Timeout**:  
  クライアントがPINGフレームを送信後、サーバーからの応答を待つ時間を指定します。この時間内に応答がない場合、接続は切断されます。

- **PermitWithoutStream**:  
  アクティブなRPCストリームがない場合でもPINGを送信するかどうかを指定します。ストリームがなくても接続を維持したい場合に使用します。

### サーバー側 (`keepalive.ServerParameters`)

- **Time**:  
  サーバーがクライアントに対してPINGフレームを送信する間隔を指定します。定期的に接続を確認する際に使用します。

- **Timeout**:  
  サーバーがPINGフレームを送信後、クライアントからの応答を待つ時間を指定します。この時間内に応答がない場合、接続は切断されます。

### サーバー側 (`keepalive.EnforcementPolicy`)

- **MinTime**:  
  クライアントから受け入れるPINGフレームの最小間隔を指定します。この値より短い間隔でPINGを送信するクライアントは切断される可能性があります（負荷防止のため）。

- **PermitWithoutStream**:  
  アクティブなRPCストリームがない場合でもPINGを受け入れるかどうかを指定します。

---

## 3. gRPC の Keepalive の挙動を確かめる 

### 不安定なネットワーク環境を再現するために

gRPCのKeepalive設定を検証する際、ネットワーク障害や接続遅延などの「不安定な環境」をシミュレーションすることが重要です。しかし、通常のネットワーク環境では、こうした状況を簡単に再現することが難しい場合があります。

そこで役立つのが **`toxiproxy`** です。`toxiproxy` は、ネットワークの障害や遅延をシミュレーションできる軽量プロキシツールで、接続断やパケットロス、遅延など、さまざまな障害状況を簡単に再現できます。

---

### なぜ toxiproxy が有効なのか？

1. **柔軟なシミュレーション**  
   - 一時的な接続断、特定の遅延、パケットロスなど、ネットワークの異常を細かくコントロールできます。
   - これにより、Keepaliveの再接続やエラーハンドリングが正しく機能するか確認できます。

2. **簡単な導入**  
   - インストールと設定が簡単で、REST API や CLI を使って障害をリアルタイムで操作可能です。

3. **gRPCとの相性が良い**  
   - gRPCは通常、長期間接続を維持するため、toxiproxyのようなプロキシを挟むことで、障害発生時の動作を詳細にテストできます。

---

### toxiproxy の公式情報

`toxiproxy` の公式リポジトリは以下のURLから確認できます。  
[Shopify/toxiproxy - GitHub](https://github.com/Shopify/toxiproxy)

ここからダウンロードや使い方の詳細なドキュメントを確認できます。

---

### toxiproxy を用いた検証環境を作成 

#### 1. toxiproxy のインストール

##### macOSの場合

以下のコマンドで `toxiproxy` をインストールします。

```zsh
$ brew install toxiproxy
```

---

#### 2. toxiproxy サーバーの起動

以下のコマンドで `toxiproxy` のサーバーを起動します。

```zsh 
$ toxiproxy-server
```

デフォルトでは、`localhost:8474` で REST API サーバーが起動します。

---

#### 3. toxiproxy CLI を使ってプロキシを作成

toxiproxy を使って、gRPC サーバー（例: `localhost:50051`）へのプロキシを作成します。

```zsh 
$ toxiproxy-cli create --listen localhost:50052 --upstream localhost:50051 grpc_proxy
Created new proxy grpc_proxy
```

これにより、gRPC クライアントは `localhost:50052` を経由してサーバーに接続するようになります。

---

#### 4. gRPC サーバーコード

以下は、Keepaliveの設定を含んだサーバーコードの例です。

```go
package main

import (
    "context"
    "log"
    "net"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/keepalive"
    pb "example"
)

type server struct {
	pb.UnimplementedYourServiceServer
}

func (s *server) YourRPCMethod(ctx context.Context, in *pb.YourRequest) (*pb.YourResponse, error) {
	log.Printf("Received: %v", in.Name)
	return &pb.YourResponse{Message: "Hello " + in.Name}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Keepalive設定
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
      // 検証のため短めに
			Time:    1 * time.Second, // サーバーからPINGを送信する間隔
			Timeout: 1 * time.Second, // PING応答の待機時間
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // クライアントPINGの最小間隔
			PermitWithoutStream: true,            // ストリームがなくてもPINGを許可
		}),
	)

	pb.RegisterYourServiceServer(grpcServer, &server{})

	log.Println("Server is running on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

```

---

#### 5. gRPC クライアントコード

以下は、Keepaliveの設定を含んだクライアントコードの例です。

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/keepalive"
    pb "example"
)

func main() {
	conn, err := grpc.Dial(
		"localhost:50052", // toxiproxy 経由で接続
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                1 * time.Second, // 1秒ごとにPINGフレームを送信
			Timeout:             1 * time.Second, // 1秒間応答がない場合に接続を切断
			PermitWithoutStream: true,            // ストリームがなくてもPINGを送信
		}),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewYourServiceClient(conn)

	for {
		// アプリケーションのタイムアウトは十分に長く 30秒 に設定
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := client.YourRPCMethod(ctx, &pb.YourRequest{Name: "World"})
		if err != nil {
			log.Printf("RPC failed: %v", err)
		} else {
			log.Printf("Response from server: %s", response.Message)
		}

		time.Sleep(1 * time.Second)
	}
}

```

---

#### 6. toxiproxy の 基本的な機能 

toxiproxy を使うと以下のようなことができます。 

##### 接続を一時的に遮断する

以下のコマンドで、gRPC サーバーとの通信を一時的に遮断します。

```zsh
$ toxiproxy-cli toggle grpc_proxy
```

- `toggle` コマンドを実行するたびに、接続の有効/無効が切り替わります。
- 接続が無効化されている間、クライアントは Keepalive の再試行を行います。

---

##### 今の設定を確認する 

以下のコマンドで、toxiproxy の設定を確認できます。 

```zsh
$ toxiproxy-cli inspect grpc_proxy
Name: grpc_proxy        Listen: 127.0.0.1:50052 Upstream: localhost:50051
======================================================================
Proxy has no toxics enabled.

Hint: add a toxic with `toxiproxy-cli toxic add`
```

##### 遅延を追加する

以下のコマンドで、通信に遅延を追加します（例: 1000ms）。

```zsh
$ toxiproxy-cli toxic add -t latency -a latency=1000 grpc_proxy
```

これにより、クライアントとサーバー間の通信に 1 秒の遅延が追加されます。

```zsh 
$ toxiproxy-cli inspect grpc_proxy
Name: grpc_proxy        Listen: 127.0.0.1:50052 Upstream: localhost:50051
======================================================================
Upstream toxics:
Proxy has no Upstream toxics enabled.

Downstream toxics:
latency_downstream:     type=latency    stream=downstream       toxicity=1.00   attributes=[    jitter=0        latency=1000    ]

Hint: add a toxic with `toxiproxy-cli toxic add`
``` 

削除したい場合は 以下を実行すると削除できます 

```zsh 
$ toxiproxy-cli toxic remove -n latency_downstream grpc_proxy
```

---

#### 7. toxic を使った検証 

##### 7-1. **接続断のシミュレーション（`toxiproxy-cli toggle grpc_proxy`）**

**Keepalive設定値（クライアント側）**:

```go
keepalive.ClientParameters{
    Time:                5 * time.Second, // 10秒ごとにPINGフレームを送信
    Timeout:             2 * time.Second,  // 5秒間応答がない場合に接続を切断
    PermitWithoutStream: true,             // ストリームがなくてもPINGを送信
}
```

**手順**:
1. `toxiproxy-cli toggle grpc_proxy` を実行して接続を無効化。
2. クライアントがKeepaliveのPINGを送信し、応答がないことを検知。

**クライアントの標準出力**:
```zsh
Response from server: Hello World
RPC failed: rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing: dial tcp [::1]:50052: connect: connection refused"
```

- **解説**:  
  - クライアントは10秒ごとにPINGを送信しますが、`toxiproxy`によって接続が遮断されるため応答が得られません。
  - 5秒以内に応答がないため、gRPCのトランスポート層で接続が閉じられ、エラー `rpc error: code = Unavailable` が発生します。

---

##### 7-2. **遅延のシミュレーション（`toxiproxy-cli toxic add grpc_proxy -t latency -a latency=3000`）**

**手順**:
1. `toxiproxy-cli toxic add -t latency -a latency=3000 grpc_proxy` を実行して3秒の遅延を追加。
2. クライアントがPINGを送信するも、遅延によりタイムアウトが発生。

**クライアントの標準出力**:

```zsh
Response from server: Hello World
RPC failed: rpc error: code = Unavailable desc = error reading from server: EOF
```

- **解説**:

1. **最初のレスポンス成功 (`Response from server: Hello World`)**  
   クライアントは最初のリクエストを正常にサーバーに送り、サーバーからのレスポンスを受け取っています。この段階では、`toxiproxy` による遅延の影響を受けていないため、通信が成功しています。

2. **その後の切断 (`RPC failed: rpc error: code = Unavailable desc = error reading from server: EOF`)**  
   クライアントは `Keepalive.ClientParameters` の設定に基づき、1秒ごとにPINGフレームを送信します。しかし、`toxiproxy` による3秒の遅延が発生しているため、サーバーからのPING応答がタイムアウト (`Timeout: 1 * time.Second`) に間に合いません。

   加えて、サーバー側の `MinTime: 5 * time.Second` 設定により、クライアントが1秒ごとにPINGを送信することが「違反」と見なされ、サーバーがクライアントを切断します。その結果、クライアントがサーバーから接続終了の通知 (`EOF`) を受け取り、エラーとしてログに出力されます。

これらの要因が組み合わさり、最初は正常なレスポンスが出力され、その後にエラーが発生するログが記録されています。

---

##### 7-3. **Keepaliveを緩和した場合の結果**

**Keepalive設定値（サーバ側）**:
```go
keepalive.ServerParameters{
			Time:    10 * time.Second, // サーバーからPINGを送信する間隔
			Timeout: 10 * time.Second, // PING応答の待機時間
}
```

**Keepalive設定値（クライアント側）**:
```go
keepalive.ClientParameters{
    Time:                10 * time.Second, // 20秒ごとにPINGフレームを送信
    Timeout:             10 * time.Second, // 15秒間応答がない場合に接続を切断
    PermitWithoutStream: true,             // ストリームがなくてもPINGを送信
}
```

**手順**:
1. サーバのKeepalive の設定を上記の通りに伸ばして再起動します。
2. 緩やかなKeepalive設定のため、切断が発生するまでの時間が長くなります。

**クライアントの標準出力**:
```go
RPC succeeded: Response from server: Hello World
```

- **解説**:  
  - Timeoutが15秒に設定されているため、遅延やパケットロスが一時的であれば、PING応答が間に合う場合があります。
  - 接続が維持され、エラーが発生しないケースもあります。

---

##### 7-検証から得られる知見

- Keepaliveの設定値（`Time` や `Timeout`）は、ネットワークの特性に合わせて調整する必要があります。
- 遅延やパケットロスが頻発する環境では、**Timeoutを長めに設定**することで接続の安定性を確保できます。
- 一方で、迅速な障害検知が求められる場合には、**Timeoutを短めに設定**し、早期に再接続を試みる動作が有効です。

以上のように、`toxiproxy` を活用することで、gRPCのKeepalive設定値が接続の安定性やエラーハンドリングにどのような影響を与えるかを詳細に検証できます。適切な設定値を選ぶことで、システムの信頼性を高めることができます。

---

## 5. まとめ

gRPCのKeepalive設定は、HTTP/2ベースの長期間接続の特性を最大限活かしつつ、信頼性を高めるために欠かせない機能です。適切に設定することで、ネットワーク障害や中間機器の影響を最小限に抑え、サービス全体の安定性を向上させることができます。

本記事のコード例や検証手順を参考に、あなたのプロジェクトに最適な設定を探してみてください。

## 参考文献

以下の資料を参考になります。 

- [gRPC Keepalive に関するオプション - Qiita](https://qiita.com/k-akiyama/items/c0d5112be0e4858a3b22)  
  gRPCのKeepalive機能に関する詳細なオプション設定や、その役割について解説されています。

- [gRPCのkeepaliveで気をつけること - Carpe Diem](https://christina04.hatenablog.com/entry/grpc-keepalive)  
  gRPCにおけるKeepaliveの役割や設定時の注意点について、具体的な例を交えて説明されています。

- [gRPCのkeep alive動作検証 - tsuchinaga - Scrapbox](https://scrapbox.io/tsuchinaga/gRPC%E3%81%AEkeep_alive%E5%8B%95%E4%BD%9C%E6%A4%9C%E8%A8%BC)  
  gRPCのKeepalive機能の動作検証を行い、その結果や考察がまとめられています。

- [gRPCのkeepaliveで気をつけること その2 - Carpe Diem](https://christina04.hatenablog.com/entry/grpc-keepalive-2)  
  gRPCのKeepalive設定に関する追加の注意点やベストプラクティスが紹介されています。

