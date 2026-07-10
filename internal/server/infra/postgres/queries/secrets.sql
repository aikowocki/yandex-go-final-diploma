-- name: CreateSecret :one
INSERT INTO secrets (vault_id, type, enc_row, enc_index, enc_payload)
VALUES ($1, $2, $3, $4, $5)
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
