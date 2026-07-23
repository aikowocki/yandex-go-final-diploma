-- name: CreateVault :one
INSERT INTO vaults (user_id, wrapped_vault_key, enc_name)
VALUES ($1, $2, $3)
RETURNING id, version, created_at, updated_at;

-- Tier 1: список папок пользователя (id + обёрнутый ключ + зашифрованное имя + версия).
-- name: ListVaultsByUser :many
SELECT id, wrapped_vault_key, enc_name, version
FROM vaults
WHERE user_id = $1 AND NOT deleted
ORDER BY created_at;

-- name: GetVaultByID :one
SELECT id, user_id, wrapped_vault_key, enc_name, version
FROM vaults
WHERE id = $1 AND NOT deleted;

-- VaultBelongsToUser — быстрая проверка владения папкой (для CreateSecret, где join на INSERT недоступен).
-- name: VaultBelongsToUser :one
SELECT EXISTS (
    SELECT 1 FROM vaults WHERE id = $1 AND user_id = $2 AND NOT deleted
);

-- CheckFreshness — лёгкий запрос версий всех папок пользователя (для клиентского sync).
-- name: CheckVaultFreshness :many
SELECT id, version
FROM vaults
WHERE user_id = $1 AND NOT deleted
ORDER BY id;
