-- name: CreateSecret :one
INSERT INTO secrets (vault_id, type, enc_row, enc_index, enc_payload)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, version, created_at, updated_at;

-- Tier 2a: строки списка секретов — только enc_row, БЕЗ enc_index/enc_payload.
-- name: ListSecretRows :many
SELECT id, type, version, enc_row
FROM secrets
WHERE vault_id = $1 AND NOT deleted
ORDER BY created_at;

-- Tier 2b: индексные блобы для фонового расширенного поиска — отдельный запрос, не общий SELECT *.
-- name: ListSecretIndex :many
SELECT id, version, enc_index
FROM secrets
WHERE vault_id = $1 AND NOT deleted
ORDER BY created_at;

-- Tier 3: чувствительный payload одного секрета — грузится лениво, только при просмотре.
-- name: GetSecretPayload :one
SELECT id, type, version, enc_payload
FROM secrets
WHERE id = $1 AND NOT deleted;
