package grpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/mapper"
)

func (s *Server) CreateVault(ctx context.Context, req *pb.CreateVaultRequest) (*pb.CreateVaultResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	id, err := s.vault.CreateVault(ctx, mapper.CreateVaultParams(userID, req))
	if err != nil {
		return nil, mapVaultErr(err)
	}
	return &pb.CreateVaultResponse{VaultId: id}, nil
}

func (s *Server) ListVaults(ctx context.Context, _ *pb.ListVaultsRequest) (*pb.ListVaultsResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	vaults, err := s.vault.ListVaults(ctx, userID)
	if err != nil {
		return nil, mapVaultErr(err)
	}
	return mapper.ListVaultsResponse(vaults), nil
}

func (s *Server) CheckFreshness(ctx context.Context, _ *pb.CheckFreshnessRequest) (*pb.CheckFreshnessResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	versions, err := s.vault.CheckFreshness(ctx, userID)
	if err != nil {
		return nil, mapVaultErr(err)
	}
	return mapper.CheckFreshnessResponse(versions), nil
}

// errNoUser — защитный ответ, если userID не оказался в контексте.
func errNoUser() error {
	return status.Error(codes.Unauthenticated, "missing authenticated user")
}
