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
