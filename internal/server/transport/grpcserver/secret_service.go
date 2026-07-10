package grpcserver

import (
	"context"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/mapper"
)

func (s *Server) CreateSecret(ctx context.Context, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	id, err := s.secret.CreateSecret(ctx, mapper.CreateSecretParams(userID, req))
	if err != nil {
		return nil, mapSecretErr(err)
	}
	return &pb.CreateSecretResponse{SecretId: id}, nil
}

func (s *Server) UpdateSecret(ctx context.Context, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	version, err := s.secret.UpdateSecret(ctx, mapper.UpdateSecretParams(userID, req))
	if err != nil {
		return nil, mapSecretErr(err)
	}
	return &pb.UpdateSecretResponse{Version: version}, nil
}

func (s *Server) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	if _, err := s.secret.DeleteSecret(ctx, mapper.DeleteSecretParams(userID, req)); err != nil {
		return nil, mapSecretErr(err)
	}
	return &pb.DeleteSecretResponse{}, nil
}

func (s *Server) ListRow(ctx context.Context, req *pb.ListRowRequest) (*pb.ListRowResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	rows, err := s.secret.ListRow(ctx, userID, req.GetVaultId())
	if err != nil {
		return nil, mapSecretErr(err)
	}
	return mapper.ListRowResponse(rows), nil
}

func (s *Server) ListIndex(ctx context.Context, req *pb.ListIndexRequest) (*pb.ListIndexResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	items, err := s.secret.ListIndex(ctx, userID, req.GetVaultId())
	if err != nil {
		return nil, mapSecretErr(err)
	}
	return mapper.ListIndexResponse(items), nil
}

func (s *Server) GetPayload(ctx context.Context, req *pb.GetPayloadRequest) (*pb.GetPayloadResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, errNoUser()
	}

	payload, err := s.secret.GetPayload(ctx, userID, req.GetSecretId())
	if err != nil {
		return nil, mapSecretErr(err)
	}
	return mapper.GetPayloadResponse(payload), nil
}
