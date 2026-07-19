package grpcserver

import (
	"context"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/transport/grpcserver/interceptor"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

// StoreRecoveryCodes сохраняет комплект recovery-кодов текущего пользователя.
func (s *Server) StoreRecoveryCodes(ctx context.Context, req *pb.StoreRecoveryCodesRequest) (*pb.StoreRecoveryCodesResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, mapAuthErr(auth.ErrUserNotFound)
	}

	codes := make([]auth.RecoveryCodeEntry, 0, len(req.GetCodes()))
	for _, c := range req.GetCodes() {
		codes = append(codes, auth.RecoveryCodeEntry{
			CodeID:       c.GetCodeId(),
			EncMasterKey: c.GetEncMasterKey(),
		})
	}

	if err := s.auth.StoreRecoveryCodes(ctx, userID, codes); err != nil {
		return nil, mapAuthErr(err)
	}
	return &pb.StoreRecoveryCodesResponse{}, nil
}

// GetRecoveryBlob возвращает зашифрованный master key по recovery-коду пользователя.
func (s *Server) GetRecoveryBlob(ctx context.Context, req *pb.GetRecoveryBlobRequest) (*pb.GetRecoveryBlobResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, mapAuthErr(auth.ErrUserNotFound)
	}

	blob, err := s.auth.GetRecoveryBlob(ctx, userID, req.GetCodeId())
	if err != nil {
		return nil, mapAuthErr(err)
	}
	return &pb.GetRecoveryBlobResponse{EncMasterKey: blob}, nil
}

// MarkRecoveryCodeUsed помечает recovery-код текущего пользователя как использованный.
func (s *Server) MarkRecoveryCodeUsed(ctx context.Context, req *pb.MarkRecoveryCodeUsedRequest) (*pb.MarkRecoveryCodeUsedResponse, error) {
	userID, ok := interceptor.UserIDFromContext(ctx)
	if !ok {
		return nil, mapAuthErr(auth.ErrUserNotFound)
	}

	if err := s.auth.MarkRecoveryCodeUsed(ctx, userID, req.GetCodeId()); err != nil {
		return nil, mapAuthErr(err)
	}
	return &pb.MarkRecoveryCodeUsedResponse{}, nil
}
