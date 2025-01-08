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

## 3. 実際のコード例

以下は、クライアントとサーバーでKeepaliveを設定するgRPCコードの例です。

### サーバー側

```go
package main

import (
    "log"
    "net"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/keepalive"
    pb "example/protobuf"
)

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    server := grpc.NewServer(
        grpc.KeepaliveParams(keepalive.ServerParameters{
            Time:    10 * time.Second,
            Timeout: 5 * time.Second,
        }),
        grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
            MinTime:             5 * time.Second,
            PermitWithoutStream: true,
        }),
    )

    pb.RegisterYourServiceServer(server, &YourService{})
    log.Println("Server is running on port 50051...")
    server.Serve(lis)
}
```

### クライアント側

```go
package main

import (
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/keepalive"
    pb "example/protobuf"
)

func main() {
    conn, err := grpc.Dial(
        "localhost:50051",
        grpc.WithInsecure(),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             5 * time.Second,
            PermitWithoutStream: true,
        }),
    )
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewYourServiceClient(conn)
    log.Println("Client connected")
}
```

---

## 4. 設定を変更して挙動を検証する


### サーバーの起動
上記の`server.go`を実行し、サーバーを起動します。

```
go run server.go
```

### クライアントの実行
別のターミナルで`client.go`を実行し、サーバーに接続してRPCを呼び出します。

```
go run client.go
```

### ネットワークの切断シミュレーション
サーバーまたはクライアントのネットワーク接続をシミュレートするため、`iptables`コマンドを使用して特定のポートへの通信をブロックします。

#### サーバー側でのブロック例
```
sudo iptables -A INPUT -p tcp --dport 50051 -j DROP
```

#### クライアント側でのブロック例
```
sudo iptables -A OUTPUT -p tcp --dport 50051 -j DROP
```


3. Keepaliveの設定値を変更して接続の維持や切断の挙動を確認します。

### 検証ポイント

- **TimeやTimeoutを短く設定**: ネットワーク障害が検知されるまでの時間を観察。
- **PermitWithoutStreamの有効/無効化**: ストリームがない状態での接続維持をテスト。

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

