-- name: CreateUser :one
INSERT INTO users (login, auth_hash)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserByLogin :one
SELECT * FROM users
WHERE login = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateUserEncKDF :exec
UPDATE users
SET enc_kdf_salt = $2,
    enc_kdf_params = $3,
    updated_at = now()
WHERE id = $1;
