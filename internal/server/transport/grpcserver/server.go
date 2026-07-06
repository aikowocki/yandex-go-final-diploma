package grpcserver

import (
	"context"
	"net"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"google.golang.org/grpc"
)

// Server оборачивает gRPC-сервер.
type Server struct {
	pb.UnimplementedAuthServiceServer
	grpcServer *grpc.Server
}

// New создаёт новый gRPC-сервер с зарегистрированными сервисами.
func New() *Server {
	s := &Server{
		grpcServer: grpc.NewServer(),
	}
	pb.RegisterAuthServiceServer(s.grpcServer, s)
	return s
}

// Run запускает gRPC-сервер на указанном адресе.
func (s *Server) Run(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.grpcServer.Serve(lis)
}

// Stop gracefully останавливает сервер.
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// Ping — проверка связности.
func (s *Server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Message: "pong"}, nil
}
