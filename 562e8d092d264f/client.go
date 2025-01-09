package main

import (
	"context"
	"log"
	"time"

	pb "github.com/shunta-furukawa/zenn-demo/562e8d092d264f/example"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
