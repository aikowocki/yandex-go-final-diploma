package mapper

import (
	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

// CreateSecretParams параметры usecase CreateSecret из proto-запроса.
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

// UpdateSecretParams параметры usecase UpdateSecret из proto-запроса.
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

// DeleteSecretParams параметры usecase DeleteSecret из proto-запроса.
func DeleteSecretParams(userID string, req *pb.DeleteSecretRequest) secret.DeleteParams {
	return secret.DeleteParams{
		UserID:      userID,
		SecretID:    req.GetSecretId(),
		BaseVersion: req.GetBaseVersion(),
	}
}

// AttachBlobParams параметры usecase AttachBlob из proto-запроса.
func AttachBlobParams(userID string, req *pb.AttachBlobRequest) secret.AttachBlobParams {
	return secret.AttachBlobParams{
		UserID:      userID,
		SecretID:    req.GetSecretId(),
		BaseVersion: req.GetBaseVersion(),
		BlobRef:     req.GetBlobRef(),
		BlobSize:    req.GetBlobSize(),
	}
}

// SecretConflictDetail proto-деталь конфликта из полной серверной версии секрета.
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

// ListRowResponse proto-ответ ListRow из списка строк секретов.
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

// ListIndexResponse proto-ответ ListIndex из списка индексных записей.
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

// GetPayloadResponse proto-ответ GetPayload из результата usecase.
func GetPayloadResponse(p secret.Payload) *pb.GetPayloadResponse {
	return &pb.GetPayloadResponse{
		SecretId:   p.ID,
		Type:       pb.SecretType(p.Type),
		Version:    p.Version,
		EncPayload: p.EncPayload,
	}
}
