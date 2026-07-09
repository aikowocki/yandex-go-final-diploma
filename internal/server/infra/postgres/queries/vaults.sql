-- name: CreateVault :one
INSERT INTO vaults (user_id, wrapped_vault_key, enc_name)
VALUES ($1, $2, $3)
RETURNING id, version, created_at, updated_at;

-- Tier 1: список ваултов пользователя (id + обёрнутый ключ + зашифрованное имя + версия).
-- name: ListVaultsByUser :many
SELECT id, wrapped_vault_key, enc_name, version
FROM vaults
WHERE user_id = $1 AND NOT deleted
ORDER BY created_at;

-- name: GetVaultByID :one
SELECT id, user_id, wrapped_vault_key, enc_name, version
FROM vaults
WHERE id = $1 AND NOT deleted;
