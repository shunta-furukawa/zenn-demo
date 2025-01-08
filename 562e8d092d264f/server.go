package main

import (
	"context"
	"log"
	"net"

	pb "github.com/shunta-furukawa/zenn-demo/562e8d092d264f/example"
	"google.golang.org/grpc"
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

	grpcServer := grpc.NewServer()
	pb.RegisterYourServiceServer(grpcServer, &server{})

	log.Println("Server is running on port 50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
