package secret

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/aikowocki/yandex-go-final-diploma/internal/client/contracts"
	"github.com/aikowocki/yandex-go-final-diploma/internal/client/grpcclient"
)

type GenericConflict struct {
	SecretID string
	VaultID  string
	IsDelete bool

	MineRow, MineIndex, MinePayload       map[string]any
	ServerRow, ServerIndex, ServerPayload map[string]any
	ServerVersion                         int64
	ServerType                            int32

	serverEncRow, serverEncIndex, serverEncPayload []byte

	retryMine func(ctx context.Context, baseVersion int64) (*GenericConflict, error)
}

func (u *UseCase) GenericResolveConflict(ctx context.Context, c *GenericConflict, choice ConflictChoice) (*GenericConflict, error) {
	if c == nil {
		return nil, ErrNilConflict
	}
	switch choice {
	case ChoiceMine:
		return c.retryMine(ctx, c.ServerVersion)
	case ChoiceServer:
		return nil, u.cacheFullSecret(ctx, c.SecretID, c.VaultID, c.ServerType,
			c.serverEncRow, c.serverEncIndex, c.serverEncPayload, c.ServerVersion, false)
	default:
		return nil, ErrUnknownChoice
	}
}

func createTyped[R, I, P any](ctx context.Context, u *UseCase, vaultID string, secretType int32, row R, index I, payload P) (string, error) {
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return "", err
	}

	secretID := uuid.NewString()
	encRow, encIndex, encPayload, err := encryptTiers(u, vaultKey, vaultID, secretID, createVersion, row, index, payload)
	if err != nil {
		return "", err
	}

	if err := u.server.CreateSecret(ctx, token, secretID, vaultID, secretType, encRow, encIndex, encPayload); err != nil {
		if errors.Is(err, grpcclient.ErrUnavailable) {
			return u.createOffline(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload)
		}
		return "", err
	}

	if err := u.cacheCreated(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload, false); err != nil {
		return "", err
	}
	return secretID, nil
}

func updateTyped[R, I, P any](ctx context.Context, u *UseCase, vaultID, secretID string, baseVersion int64, secretType int32, row R, index I, payload P) (*GenericConflict, error) {
	vaultKey, token, err := u.vaultContext(vaultID)
	if err != nil {
		return nil, err
	}

	newVersion := baseVersion + 1
	encRow, encIndex, encPayload, err := encryptTiers(u, vaultKey, vaultID, secretID, newVersion, row, index, payload)
	if err != nil {
		return nil, err
	}

	_, err = u.server.UpdateSecret(ctx, token, secretID, baseVersion, encRow, encIndex, encPayload)
	if err != nil {
		var conflict *grpcclient.ConflictError
		switch {
		case errors.As(err, &conflict):
			return buildGenericConflict(u, vaultKey, vaultID, secretID, secretType, row, index, payload, conflict.Server, false)
		case errors.Is(err, grpcclient.ErrUnavailable):
			if oerr := u.updateOffline(ctx, secretID, vaultID, secretType, baseVersion, encRow, encIndex, encPayload); oerr != nil {
				return nil, oerr
			}
			return nil, nil
		default:
			return nil, err
		}
	}

	if err := u.cacheFullSecret(ctx, secretID, vaultID, secretType, encRow, encIndex, encPayload, newVersion, false); err != nil {
		return nil, err
	}
	return nil, nil
}

func buildGenericConflict[R, I, P any](u *UseCase, vaultKey []byte, vaultID, secretID string, secretType int32, row R, index I, payload P, server contracts.ServerSecret, isDelete bool) (*GenericConflict, error) {
	mineRow, err := toMap(row)
	if err != nil {
		return nil, err
	}
	mineIndex, err := toMap(index)
	if err != nil {
		return nil, err
	}
	minePayload, err := toMap(payload)
	if err != nil {
		return nil, err
	}

	var srvRow, srvIndex, srvPayload map[string]any
	if err := decryptTiers(u, vaultKey, vaultID, secretID, server.Version, server.EncRow, server.EncIndex, server.EncPayload, &srvRow, &srvIndex, &srvPayload); err != nil {
		return nil, fmt.Errorf("decrypt server version: %w", err)
	}

	return &GenericConflict{
		SecretID:         secretID,
		VaultID:          vaultID,
		IsDelete:         isDelete,
		MineRow:          mineRow,
		MineIndex:        mineIndex,
		MinePayload:      minePayload,
		ServerRow:        srvRow,
		ServerIndex:      srvIndex,
		ServerPayload:    srvPayload,
		ServerVersion:    server.Version,
		ServerType:       server.Type,
		serverEncRow:     server.EncRow,
		serverEncIndex:   server.EncIndex,
		serverEncPayload: server.EncPayload,
		retryMine: func(ctx context.Context, baseVersion int64) (*GenericConflict, error) {
			return updateTyped(ctx, u, vaultID, secretID, baseVersion, secretType, row, index, payload)
		},
	}, nil
}

func toMap(value any) (map[string]any, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal for display: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("unmarshal for display: %w", err)
	}
	return m, nil
}

func remarshal(src, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("remarshal: marshal: %w", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("remarshal: unmarshal: %w", err)
	}
	return nil
}
