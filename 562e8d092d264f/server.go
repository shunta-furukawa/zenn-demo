package main

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/shunta-furukawa/zenn-demo/562e8d092d264f/example"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
