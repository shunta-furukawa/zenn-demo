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

## toxiproxy を使った gRPC Keepalive 検証

---

## 1. toxiproxy のインストール

### macOSの場合

以下のコマンドで `toxiproxy` をインストールします。

```go
brew install toxiproxy
```

---

## 2. toxiproxy サーバーの起動

以下のコマンドで `toxiproxy` のサーバーを起動します。

```
toxiproxy-server
```

デフォルトでは、`localhost:8474` で REST API サーバーが起動します。

---

## 3. toxiproxy CLI を使ってプロキシを作成

toxiproxy を使って、gRPC サーバー（例: `localhost:50051`）へのプロキシを作成します。

```go
toxiproxy-cli create grpc_proxy --listen localhost:50052 --upstream localhost:50051
```

これにより、gRPC クライアントは `localhost:50052` を経由してサーバーに接続するようになります。

---

## 4. gRPC サーバーコード

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
            Time:    10 * time.Second, // サーバーからPINGを送信する間隔
            Timeout: 5 * time.Second,  // PING応答の待機時間
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

## 5. gRPC クライアントコード

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
            Time:                10 * time.Second, // クライアントからPINGを送信する間隔
            Timeout:             5 * time.Second,  // PING応答の待機時間
            PermitWithoutStream: true,             // ストリームがなくてもPINGを送信
        }),
    )
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewYourServiceClient(conn)

    for {
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()

        response, err := client.YourRPCMethod(ctx, &pb.YourRequest{Name: "World"})
        if err != nil {
            log.Printf("RPC failed: %v", err)
        } else {
            log.Printf("Response from server: %s", response.Message)
        }

        time.Sleep(5 * time.Second)
    }
}
```

---

## 6. toxiproxy を使った障害シミュレーション

### 接続を一時的に遮断する

以下のコマンドで、gRPC サーバーとの通信を一時的に遮断します。

```
toxiproxy-cli toggle grpc_proxy
```

- `toggle` コマンドを実行するたびに、接続の有効/無効が切り替わります。
- 接続が無効化されている間、クライアントは Keepalive の再試行を行います。

---

### 遅延を追加する

以下のコマンドで、通信に遅延を追加します（例: 1000ms）。

```
toxiproxy-cli toxic add grpc_proxy -t latency -a latency=1000
```

これにより、クライアントとサーバー間の通信に 1 秒の遅延が追加されます。

---

### パケットロスをシミュレーションする

以下のコマンドで、50% のパケットロスをシミュレーションします。

```
toxiproxy-cli toxic add grpc_proxy -t limit_data -a bytes=1024
```

---

## 7. 検証結果

- **接続断（`toggle`）**  
  クライアントが接続断を検知し、Keepalive の再接続動作を確認できます。

- **遅延（`latency`）**  
  クライアントが遅延に対してどのように応答するか、またタイムアウトが正しく発生するかを確認できます。

- **パケットロス（`limit_data`）**  
  パケットロス時に Keepalive の再送が適切に行われるかを確認できます。

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

