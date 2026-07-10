package contracts

import "context"

// LocalVault — строка локального кеша папки (те же шифротексты, что на сервере, + sync-состояние).
type LocalVault struct {
	ID              string
	WrappedVaultKey []byte
	EncName         []byte
	Version         int64
	SyncedVersion   int64
	Deleted         bool
}

// LocalSecret — строка локального кеша секрета. enc_index/enc_payload могут быть nil (лениво).
type LocalSecret struct {
	ID            string
	VaultID       string
	Type          int32
	EncRow        []byte
	EncIndex      []byte
	EncPayload    []byte
	Version       int64
	IndexLoaded   bool
	PayloadLoaded bool
	Dirty         bool // Изменён локально и ещё не синхронизирован (оффлайн)
	Deleted       bool
}

// OutboxOp — тип отложенной оффлайн-операции.
type OutboxOp string

const (
	OutboxOpCreate OutboxOp = "create"
	OutboxOpUpdate OutboxOp = "update"
	OutboxOpDelete OutboxOp = "delete"
)

// OutboxEntry — запись очереди оффлайн-изменений.
type OutboxEntry struct {
	ID          int64
	Op          OutboxOp
	Entity      string // secret/vault
	EntityID    string
	BaseVersion int64
	Payload     []byte // Сериализованные зашифрованные поля к отправке
	CreatedAt   string
}

// OutboxSecretCreate — сериализуемое тело outbox-операции создания секрета (op=create, entity=secret).
// Хранит уже зашифрованные тиры и временный локальный id, по которому идёт reconcile после отправки.
type OutboxSecretCreate struct {
	VaultID    string `json:"vault_id"`
	TempID     string `json:"temp_id"`
	Type       int32  `json:"type"`
	EncRow     []byte `json:"enc_row"`
	EncIndex   []byte `json:"enc_index"`
	EncPayload []byte `json:"enc_payload"`
}

// LocalStorage — локальное SQLite-хранилище клиента (кеш + оффлайн-очередь).
type LocalStorage interface {
	UpsertVault(ctx context.Context, v LocalVault) error
	ListVaults(ctx context.Context) ([]LocalVault, error)
	GetVault(ctx context.Context, id string) (LocalVault, bool, error)
	SetVaultSyncedVersion(ctx context.Context, id string, syncedVersion int64) error

	UpsertSecretRow(ctx context.Context, s LocalSecret) error
	SetSecretPayload(ctx context.Context, id string, encPayload []byte, version int64) error
	ListSecretsByVault(ctx context.Context, vaultID string) ([]LocalSecret, error)
	GetSecret(ctx context.Context, id string) (LocalSecret, bool, error)
	DeleteSecret(ctx context.Context, id string) error

	EnqueueOutbox(ctx context.Context, e OutboxEntry) (int64, error)
	ListPendingOutbox(ctx context.Context) ([]OutboxEntry, error)
	RemoveOutbox(ctx context.Context, id int64) error

	KVGet(ctx context.Context, key string) ([]byte, bool, error)
	KVSet(ctx context.Context, key string, value []byte) error

	Close() error
}
