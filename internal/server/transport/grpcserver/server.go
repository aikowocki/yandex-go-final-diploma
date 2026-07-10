package grpcserver

import (
	"context"
	"net"

	"google.golang.org/grpc"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/mapper"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/blob"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
)

// Server оборачивает gRPC-сервер
type Server struct {
	pb.UnimplementedAuthServiceServer
	pb.UnimplementedVaultServiceServer
	pb.UnimplementedSecretServiceServer
	pb.UnimplementedBlobServiceServer

	grpcServer *grpc.Server
	auth       *auth.UseCase
	vault      *vault.UseCase
	secret     *secret.UseCase
	blob       *blob.UseCase
}

// New создаёт новый gRPC-сервер с зарегистрированными сервисами. blobUseCase может быть nil,
// если сервер запущен без MinIO — в этом случае UploadBlob/DownloadBlob вернут Unimplemented.
func New(
	authUseCase *auth.UseCase,
	vaultUseCase *vault.UseCase,
	secretUseCase *secret.UseCase,
	blobUseCase *blob.UseCase,
	tokenVerifier interceptor.TokenVerifier,
) *Server {
	s := &Server{
		auth:   authUseCase,
		vault:  vaultUseCase,
		secret: secretUseCase,
		blob:   blobUseCase,
		grpcServer: grpc.NewServer(
			grpc.UnaryInterceptor(interceptor.Auth(tokenVerifier)),
			grpc.StreamInterceptor(interceptor.StreamAuth(tokenVerifier)),
		),
	}
	pb.RegisterAuthServiceServer(s.grpcServer, s)
	pb.RegisterVaultServiceServer(s.grpcServer, s)
	pb.RegisterSecretServiceServer(s.grpcServer, s)
	pb.RegisterBlobServiceServer(s.grpcServer, s)
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

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	res, err := s.auth.Register(ctx, mapper.RegisterParams(req))
	if err != nil {
		return nil, mapAuthErr(err)
	}
	return mapper.RegisterResponse(res), nil
}

func (s *Server) SetupEncryption(ctx context.Context, req *pb.SetupEncryptionRequest) (*pb.SetupEncryptionResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, mapAuthErr(auth.ErrUserNotFound)
	}

	if err := s.auth.SetupEncryption(ctx, mapper.SetupEncryptionParams(userID, req)); err != nil {
		return nil, mapAuthErr(err)
	}
	return &pb.SetupEncryptionResponse{}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	res, err := s.auth.Login(ctx, mapper.LoginParams(req))
	if err != nil {
		return nil, mapAuthErr(err)
	}
	return mapper.LoginResponse(res), nil
}

func (s *Server) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	res, err := s.auth.RefreshToken(ctx, mapper.RefreshTokenParams(req))
	if err != nil {
		return nil, mapAuthErr(err)
	}
	return mapper.RefreshTokenResponse(res), nil
}
