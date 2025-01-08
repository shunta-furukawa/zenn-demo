package main

import (
	"context"
	"log"
	"time"

	pb "github.com/shunta-furukawa/zenn-demo/562e8d092d264f/example"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewYourServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.YourRPCMethod(ctx, &pb.YourRequest{Name: "World"})
	if err != nil {
		log.Fatalf("Failed to call YourRPCMethod: %v", err)
	}

	log.Printf("Response from server: %s", response.Message)
}
