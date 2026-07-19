package mapper

import (
	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

// RegisterParams  параметры usecase Register из proto-запроса.
func RegisterParams(req *pb.RegisterRequest) auth.RegisterParams {
	return auth.RegisterParams{
		Login:           req.GetLogin(),
		LoginCredential: req.GetLoginCredential(),
	}
}

// RegisterResponse proto-ответ Register из результата usecase.
func RegisterResponse(res auth.RegisterResult) *pb.RegisterResponse {
	return &pb.RegisterResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		UserId:       res.UserID,
	}
}

// SetupEncryptionParams  параметры usecase SetupEncryption из proto-запроса.
func SetupEncryptionParams(userID string, req *pb.SetupEncryptionRequest) auth.SetupEncryptionParams {
	return auth.SetupEncryptionParams{
		UserID:       userID,
		EncKDFSalt:   req.GetEncKdfSalt(),
		EncKDFParams: req.GetEncKdfParams(),
		EncMasterKey: req.GetEncMasterKey(),
	}
}

// LoginParams  параметры usecase Login из proto-запроса.
func LoginParams(req *pb.LoginRequest) auth.LoginParams {
	return auth.LoginParams{
		Login:           req.GetLogin(),
		LoginCredential: req.GetLoginCredential(),
	}
}

// LoginResponse  proto-ответ Login из результата usecase.
func LoginResponse(res auth.Result) *pb.LoginResponse {
	return &pb.LoginResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		EncKdfSalt:   res.EncKDFSalt,
		EncKdfParams: res.EncKDFParams,
		UserId:       res.UserID,
		EncMasterKey: res.EncMasterKey,
	}
}

// RefreshTokenParams  параметры usecase RefreshToken из proto-запроса.
func RefreshTokenParams(req *pb.RefreshTokenRequest) auth.RefreshParams {
	return auth.RefreshParams{
		RefreshToken: req.GetRefreshToken(),
	}
}

// RefreshTokenResponse  proto-ответ RefreshToken из результата usecase.
func RefreshTokenResponse(res auth.Result) *pb.RefreshTokenResponse {
	return &pb.RefreshTokenResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		EncKdfSalt:   res.EncKDFSalt,
		EncKdfParams: res.EncKDFParams,
		UserId:       res.UserID,
		EncMasterKey: res.EncMasterKey,
	}
}
