package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/auth"
)

func TestRegisterParams(t *testing.T) {
	req := &pb.RegisterRequest{Login: "alice", LoginCredential: []byte("cred")}
	got := RegisterParams(req)
	assert.Equal(t, auth.RegisterParams{Login: "alice", LoginCredential: []byte("cred")}, got)
}

func TestRegisterResponse(t *testing.T) {
	res := auth.RegisterResult{UserID: "u1", AccessToken: "at", RefreshToken: "rt"}
	got := RegisterResponse(res)
	assert.Equal(t, "at", got.GetAccessToken())
	assert.Equal(t, "rt", got.GetRefreshToken())
	assert.Equal(t, "u1", got.GetUserId())
}

func TestSetupEncryptionParams(t *testing.T) {
	req := &pb.SetupEncryptionRequest{
		EncKdfSalt:   []byte("salt"),
		EncKdfParams: []byte("params"),
		EncMasterKey: []byte("mk"),
	}
	got := SetupEncryptionParams("u1", req)
	assert.Equal(t, auth.SetupEncryptionParams{
		UserID:       "u1",
		EncKDFSalt:   []byte("salt"),
		EncKDFParams: []byte("params"),
		EncMasterKey: []byte("mk"),
	}, got)
}

func TestLoginParams(t *testing.T) {
	req := &pb.LoginRequest{Login: "bob", LoginCredential: []byte("cred2")}
	got := LoginParams(req)
	assert.Equal(t, auth.LoginParams{Login: "bob", LoginCredential: []byte("cred2")}, got)
}

func TestLoginResponse(t *testing.T) {
	res := auth.Result{
		UserID:       "u2",
		AccessToken:  "at2",
		RefreshToken: "rt2",
		EncKDFSalt:   []byte("salt"),
		EncKDFParams: []byte("params"),
		EncMasterKey: []byte("mk"),
	}
	got := LoginResponse(res)
	assert.Equal(t, "at2", got.GetAccessToken())
	assert.Equal(t, "rt2", got.GetRefreshToken())
	assert.Equal(t, "u2", got.GetUserId())
	assert.Equal(t, []byte("salt"), got.GetEncKdfSalt())
	assert.Equal(t, []byte("params"), got.GetEncKdfParams())
	assert.Equal(t, []byte("mk"), got.GetEncMasterKey())
}

func TestRefreshTokenParams(t *testing.T) {
	req := &pb.RefreshTokenRequest{RefreshToken: "rt3"}
	got := RefreshTokenParams(req)
	assert.Equal(t, auth.RefreshParams{RefreshToken: "rt3"}, got)
}

func TestRefreshTokenResponse(t *testing.T) {
	res := auth.Result{
		UserID:       "u3",
		AccessToken:  "at3",
		RefreshToken: "rt4",
		EncKDFSalt:   []byte("salt3"),
		EncKDFParams: []byte("params3"),
		EncMasterKey: []byte("mk3"),
	}
	got := RefreshTokenResponse(res)
	assert.Equal(t, "at3", got.GetAccessToken())
	assert.Equal(t, "rt4", got.GetRefreshToken())
	assert.Equal(t, "u3", got.GetUserId())
	assert.Equal(t, []byte("salt3"), got.GetEncKdfSalt())
	assert.Equal(t, []byte("params3"), got.GetEncKdfParams())
	assert.Equal(t, []byte("mk3"), got.GetEncMasterKey())
}
