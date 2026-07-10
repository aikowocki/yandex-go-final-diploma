package mapper

import (
	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/vault"
)

func CreateVaultParams(userID string, req *pb.CreateVaultRequest) vault.CreateParams {
	return vault.CreateParams{
		UserID:          userID,
		WrappedVaultKey: req.GetWrappedVaultKey(),
		EncName:         req.GetEncName(),
	}
}

func ListVaultsResponse(vaults []vault.Tier1) *pb.ListVaultsResponse {
	items := make([]*pb.Vault, 0, len(vaults))
	for _, v := range vaults {
		items = append(items, &pb.Vault{
			VaultId:         v.ID,
			WrappedVaultKey: v.WrappedVaultKey,
			EncName:         v.EncName,
			Version:         v.Version,
		})
	}
	return &pb.ListVaultsResponse{Vaults: items}
}

func CheckFreshnessResponse(versions []vault.Version) *pb.CheckFreshnessResponse {
	items := make([]*pb.VaultVersion, 0, len(versions))
	for _, v := range versions {
		items = append(items, &pb.VaultVersion{
			VaultId: v.ID,
			Version: v.Version,
		})
	}
	return &pb.CheckFreshnessResponse{Vaults: items}
}
