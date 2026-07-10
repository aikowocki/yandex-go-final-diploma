package mapper

import (
	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

func CreateSecretParams(userID string, req *pb.CreateSecretRequest) secret.CreateParams {
	return secret.CreateParams{
		UserID:     userID,
		VaultID:    req.GetVaultId(),
		SecretID:   req.GetSecretId(),
		Type:       domain.SecretType(req.GetType()),
		EncRow:     req.GetEncRow(),
		EncIndex:   req.GetEncIndex(),
		EncPayload: req.GetEncPayload(),
	}
}

func UpdateSecretParams(userID string, req *pb.UpdateSecretRequest) secret.UpdateParams {
	return secret.UpdateParams{
		UserID:      userID,
		SecretID:    req.GetSecretId(),
		BaseVersion: req.GetBaseVersion(),
		EncRow:      req.GetEncRow(),
		EncIndex:    req.GetEncIndex(),
		EncPayload:  req.GetEncPayload(),
	}
}

func DeleteSecretParams(userID string, req *pb.DeleteSecretRequest) secret.DeleteParams {
	return secret.DeleteParams{
		UserID:      userID,
		SecretID:    req.GetSecretId(),
		BaseVersion: req.GetBaseVersion(),
	}
}

// SecretConflictDetail строит proto-деталь конфликта из полной серверной версии секрета.
func SecretConflictDetail(s domain.Secret) *pb.SecretConflict {
	return &pb.SecretConflict{
		SecretId:   s.ID,
		Type:       pb.SecretType(s.Type),
		Version:    s.Version,
		EncRow:     s.EncRow,
		EncIndex:   s.EncIndex,
		EncPayload: s.EncPayload,
	}
}

func ListRowResponse(rows []secret.Row) *pb.ListRowResponse {
	items := make([]*pb.SecretRow, 0, len(rows))
	for _, r := range rows {
		items = append(items, &pb.SecretRow{
			SecretId: r.ID,
			Type:     pb.SecretType(r.Type),
			Version:  r.Version,
			EncRow:   r.EncRow,
		})
	}
	return &pb.ListRowResponse{Secrets: items}
}

func ListIndexResponse(rows []secret.IndexItem) *pb.ListIndexResponse {
	items := make([]*pb.SecretIndex, 0, len(rows))
	for _, r := range rows {
		items = append(items, &pb.SecretIndex{
			SecretId: r.ID,
			Version:  r.Version,
			EncIndex: r.EncIndex,
		})
	}
	return &pb.ListIndexResponse{Secrets: items}
}

func GetPayloadResponse(p secret.Payload) *pb.GetPayloadResponse {
	return &pb.GetPayloadResponse{
		SecretId:   p.ID,
		Type:       pb.SecretType(p.Type),
		Version:    p.Version,
		EncPayload: p.EncPayload,
	}
}
