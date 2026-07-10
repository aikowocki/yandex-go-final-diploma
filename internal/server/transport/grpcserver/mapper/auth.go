package mapper

import (
	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

func RegisterParams(req *pb.RegisterRequest) auth.RegisterParams {
	return auth.RegisterParams{
		Login:           req.GetLogin(),
		LoginCredential: req.GetLoginCredential(),
	}
}

func RegisterResponse(res auth.RegisterResult) *pb.RegisterResponse {
	return &pb.RegisterResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		UserId:       res.UserID,
	}
}

func SetupEncryptionParams(userID string, req *pb.SetupEncryptionRequest) auth.SetupEncryptionParams {
	return auth.SetupEncryptionParams{
		UserID:       userID,
		EncKDFSalt:   req.GetEncKdfSalt(),
		EncKDFParams: req.GetEncKdfParams(),
	}
}

func LoginParams(req *pb.LoginRequest) auth.LoginParams {
	return auth.LoginParams{
		Login:           req.GetLogin(),
		LoginCredential: req.GetLoginCredential(),
	}
}

func LoginResponse(res auth.AuthResult) *pb.LoginResponse {
	return &pb.LoginResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		EncKdfSalt:   res.EncKDFSalt,
		EncKdfParams: res.EncKDFParams,
		UserId:       res.UserID,
	}
}

func RefreshTokenParams(req *pb.RefreshTokenRequest) auth.RefreshParams {
	return auth.RefreshParams{
		RefreshToken: req.GetRefreshToken(),
	}
}

func RefreshTokenResponse(res auth.AuthResult) *pb.RefreshTokenResponse {
	return &pb.RefreshTokenResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		EncKdfSalt:   res.EncKDFSalt,
		EncKdfParams: res.EncKDFParams,
		UserId:       res.UserID,
	}
}
