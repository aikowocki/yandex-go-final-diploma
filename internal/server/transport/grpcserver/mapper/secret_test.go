package mapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/aikowocki/yandex-go-final-diploma/api/proto/gen/gophkeeper/v1"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/domain"
	"github.com/aikowocki/yandex-go-final-diploma/internal/server/usecase/secret"
)

func TestCreateSecretParams(t *testing.T) {
	req := &pb.CreateSecretRequest{
		VaultId:    "v1",
		SecretId:   "s1",
		Type:       pb.SecretType_SECRET_TYPE_TEXT,
		EncRow:     []byte("row"),
		EncIndex:   []byte("idx"),
		EncPayload: []byte("payload"),
	}
	got := CreateSecretParams("u1", req)
	assert.Equal(t, secret.CreateParams{
		UserID:     "u1",
		VaultID:    "v1",
		SecretID:   "s1",
		Type:       domain.SecretType(pb.SecretType_SECRET_TYPE_TEXT),
		EncRow:     []byte("row"),
		EncIndex:   []byte("idx"),
		EncPayload: []byte("payload"),
	}, got)
}

func TestUpdateSecretParams(t *testing.T) {
	req := &pb.UpdateSecretRequest{
		SecretId:    "s1",
		BaseVersion: 3,
		EncRow:      []byte("row"),
		EncIndex:    []byte("idx"),
		EncPayload:  []byte("payload"),
	}
	got := UpdateSecretParams("u1", req)
	assert.Equal(t, secret.UpdateParams{
		UserID:      "u1",
		SecretID:    "s1",
		BaseVersion: 3,
		EncRow:      []byte("row"),
		EncIndex:    []byte("idx"),
		EncPayload:  []byte("payload"),
	}, got)
}

func TestDeleteSecretParams(t *testing.T) {
	req := &pb.DeleteSecretRequest{SecretId: "s1", BaseVersion: 4}
	got := DeleteSecretParams("u1", req)
	assert.Equal(t, secret.DeleteParams{UserID: "u1", SecretID: "s1", BaseVersion: 4}, got)
}

func TestAttachBlobParams(t *testing.T) {
	req := &pb.AttachBlobRequest{
		SecretId:    "s1",
		BaseVersion: 2,
		BlobRef:     "ref",
		BlobSize:    1024,
	}
	got := AttachBlobParams("u1", req)
	assert.Equal(t, secret.AttachBlobParams{
		UserID:      "u1",
		SecretID:    "s1",
		BaseVersion: 2,
		BlobRef:     "ref",
		BlobSize:    1024,
	}, got)
}

func TestSecretConflictDetail(t *testing.T) {
	s := domain.Secret{
		ID:         "s1",
		Type:       domain.SecretTypeBankCard,
		Version:    9,
		EncRow:     []byte("row"),
		EncIndex:   []byte("idx"),
		EncPayload: []byte("payload"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	got := SecretConflictDetail(s)
	assert.Equal(t, "s1", got.GetSecretId())
	assert.Equal(t, pb.SecretType(domain.SecretTypeBankCard), got.GetType())
	assert.Equal(t, int64(9), got.GetVersion())
	assert.Equal(t, []byte("row"), got.GetEncRow())
	assert.Equal(t, []byte("idx"), got.GetEncIndex())
	assert.Equal(t, []byte("payload"), got.GetEncPayload())
}

func TestListRowResponse(t *testing.T) {
	rows := []secret.Row{
		{ID: "s1", Type: domain.SecretTypeText, Version: 1, EncRow: []byte("r1")},
		{ID: "s2", Type: domain.SecretTypeTOTP, Version: 2, EncRow: []byte("r2")},
	}
	got := ListRowResponse(rows)
	require.Len(t, got.GetSecrets(), 2)
	assert.Equal(t, "s1", got.GetSecrets()[0].GetSecretId())
	assert.Equal(t, pb.SecretType(domain.SecretTypeText), got.GetSecrets()[0].GetType())
	assert.Equal(t, int64(1), got.GetSecrets()[0].GetVersion())
	assert.Equal(t, []byte("r1"), got.GetSecrets()[0].GetEncRow())
}

func TestListRowResponse_Empty(t *testing.T) {
	got := ListRowResponse(nil)
	assert.Empty(t, got.GetSecrets())
}

func TestListIndexResponse(t *testing.T) {
	rows := []secret.IndexItem{
		{ID: "s1", Version: 1, EncIndex: []byte("i1")},
	}
	got := ListIndexResponse(rows)
	require.Len(t, got.GetSecrets(), 1)
	assert.Equal(t, "s1", got.GetSecrets()[0].GetSecretId())
	assert.Equal(t, int64(1), got.GetSecrets()[0].GetVersion())
	assert.Equal(t, []byte("i1"), got.GetSecrets()[0].GetEncIndex())
}

func TestListIndexResponse_Empty(t *testing.T) {
	got := ListIndexResponse(nil)
	assert.Empty(t, got.GetSecrets())
}

func TestGetPayloadResponse(t *testing.T) {
	p := secret.Payload{ID: "s1", Type: domain.SecretTypeBinary, Version: 5, EncPayload: []byte("p1")}
	got := GetPayloadResponse(p)
	assert.Equal(t, "s1", got.GetSecretId())
	assert.Equal(t, pb.SecretType(domain.SecretTypeBinary), got.GetType())
	assert.Equal(t, int64(5), got.GetVersion())
	assert.Equal(t, []byte("p1"), got.GetEncPayload())
}
