package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	// サーバを初期化
	templ := "The result of the calculation is: %d"

	calcService := server.NewCulcService()
	printService := server.NewPrintService(templ)
	exampleServer := server.NewExampleServer(calcService, printService)

	// gRPC サーバを起動
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterExampleServiceServer(grpcServer, exampleServer)

	log.Println("Server is running on port :50051")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
