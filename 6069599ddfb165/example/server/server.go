package server

import (
	"context"

	"github.com/shunta-furukawa/zenn-demo/6069599ddfb165/example/example"
)

type ExampleServer struct {
	example.UnimplementedExampleServiceServer
	CulcService  *CulcService
	PrintService *PrintService
}

func NewExampleServer(c *CulcService, p *PrintService) *ExampleServer {
	return &ExampleServer{
		CulcService:  c,
		PrintService: p,
	}
}

// Culc RPC
func (s *ExampleServer) Culc(ctx context.Context, req *example.CulcRequest) (*example.CulcResponse, error) {
	// CulcService の Multiply を呼び出して計算
	result := s.CulcService.Multiply(req.A, req.B)

	// PrintService の Print を使用して結果を整形
	message := s.PrintService.Print(result)

	return &example.CulcResponse{Message: message}, nil
}
