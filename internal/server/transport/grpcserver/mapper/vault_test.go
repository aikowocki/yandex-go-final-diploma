package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
)

func TestCreateVaultParams(t *testing.T) {
	req := &pb.CreateVaultRequest{WrappedVaultKey: []byte("wvk"), EncName: []byte("name")}
	got := CreateVaultParams("u1", req)
	assert.Equal(t, vault.CreateParams{UserID: "u1", WrappedVaultKey: []byte("wvk"), EncName: []byte("name")}, got)
}

func TestListVaultsResponse(t *testing.T) {
	vaults := []vault.Tier1{
		{ID: "v1", WrappedVaultKey: []byte("k1"), EncName: []byte("n1"), Version: 1},
		{ID: "v2", WrappedVaultKey: []byte("k2"), EncName: []byte("n2"), Version: 2},
	}
	got := ListVaultsResponse(vaults)
	require.Len(t, got.GetVaults(), 2)
	assert.Equal(t, "v1", got.GetVaults()[0].GetVaultId())
	assert.Equal(t, []byte("k1"), got.GetVaults()[0].GetWrappedVaultKey())
	assert.Equal(t, []byte("n1"), got.GetVaults()[0].GetEncName())
	assert.Equal(t, int64(1), got.GetVaults()[0].GetVersion())
	assert.Equal(t, "v2", got.GetVaults()[1].GetVaultId())
}

func TestListVaultsResponse_Empty(t *testing.T) {
	got := ListVaultsResponse(nil)
	assert.Empty(t, got.GetVaults())
}

func TestCheckFreshnessResponse(t *testing.T) {
	versions := []vault.Version{
		{ID: "v1", Version: 5},
		{ID: "v2", Version: 7},
	}
	got := CheckFreshnessResponse(versions)
	require.Len(t, got.GetVaults(), 2)
	assert.Equal(t, "v1", got.GetVaults()[0].GetVaultId())
	assert.Equal(t, int64(5), got.GetVaults()[0].GetVersion())
	assert.Equal(t, "v2", got.GetVaults()[1].GetVaultId())
	assert.Equal(t, int64(7), got.GetVaults()[1].GetVersion())
}

func TestCheckFreshnessResponse_Empty(t *testing.T) {
	got := CheckFreshnessResponse(nil)
	assert.Empty(t, got.GetVaults())
}
