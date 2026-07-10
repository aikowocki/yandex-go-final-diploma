-- name: CreateSecret :one
-- id генерирует клиент (нужен для AAD-привязки шифротекста ещё до отправки).
INSERT INTO secrets (id, vault_id, type, enc_row, enc_index, enc_payload)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, version, created_at, updated_at;

-- Tier 2a: строки списка секретов — только enc_row, БЕЗ enc_index/enc_payload.
-- name: ListSecretRows :many
SELECT s.id, s.type, s.version, s.enc_row
FROM secrets s
JOIN vaults v ON v.id = s.vault_id
WHERE s.vault_id = $1 AND v.user_id = $2 AND NOT s.deleted
ORDER BY s.created_at;

-- Tier 2b: индексные блобы для фонового поиска.
-- name: ListSecretIndex :many
SELECT s.id, s.version, s.enc_index
FROM secrets s
JOIN vaults v ON v.id = s.vault_id
WHERE s.vault_id = $1 AND v.user_id = $2 AND NOT s.deleted
ORDER BY s.created_at;

-- Tier 3: чувствительный payload одного секрета — грузится лениво, только при просмотре.
-- name: GetSecretPayload :one
SELECT s.id, s.type, s.version, s.enc_payload
FROM secrets s
JOIN vaults v ON v.id = s.vault_id
WHERE s.id = $1 AND v.user_id = $2 AND NOT s.deleted;

-- GetSecretForUpdate — полная строка секрета с блокировкой (SELECT ... FOR UPDATE) для
-- оптимистичной блокировки внутри транзакции. Возвращает и удалённые (deleted) строки,
-- чтобы отличить «не найдено» от «уже удалено». blob_ref/blob_size нужны usecase/blob
-- (DownloadChunked/AttachBlob type-проверка) для type=binary секретов.
-- name: GetSecretForUpdate :one
SELECT s.id, s.vault_id, s.type, s.enc_row, s.enc_index, s.enc_payload, s.blob_ref, s.blob_size, s.version, s.deleted
FROM secrets s
JOIN vaults v ON v.id = s.vault_id
WHERE s.id = $1 AND v.user_id = $2
FOR UPDATE OF s;

-- UpdateSecretFields — применяет новые шифротексты и инкрементирует версию.
-- Вызывается только после проверки base_version под блокировкой строки.
-- name: UpdateSecretFields :one
UPDATE secrets
SET enc_row = $2, enc_index = $3, enc_payload = $4, version = version + 1, updated_at = now()
WHERE id = $1
RETURNING version;

-- SoftDeleteSecret — помечает секрет удалённым (soft-delete) и инкрементирует версию.
-- name: SoftDeleteSecret :one
UPDATE secrets
SET deleted = true, version = version + 1, updated_at = now()
WHERE id = $1
RETURNING version;

-- BumpVaultVersion — инкрементирует версию папки. Вызывается при любом изменении секретов
-- внутри папки, чтобы CheckFreshness увидел изменение и другие устройства подтянули данные.
-- name: BumpVaultVersion :exec
UPDATE vaults
SET version = version + 1, updated_at = now()
WHERE id = $1;

-- AttachBlob — прописывает blob_ref/blob_size секрету type=binary ПОСЛЕ того как клиент успешно
-- залил зашифрованный блоб в MinIO.
UPDATE secrets
SET blob_ref = $2, blob_size = $3, version = version + 1, updated_at = now()
WHERE id = $1
RETURNING version;
