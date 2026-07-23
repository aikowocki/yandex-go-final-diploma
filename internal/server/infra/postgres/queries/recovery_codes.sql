-- name: StoreRecoveryCode :exec
INSERT INTO recovery_codes (user_id, code_id, enc_master_key)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, code_id) DO UPDATE SET
    enc_master_key = excluded.enc_master_key,
    used = false;

-- name: GetRecoveryBlob :one
SELECT enc_master_key
FROM recovery_codes
WHERE user_id = $1 AND code_id = $2 AND used = false;

-- name: MarkRecoveryCodeUsed :exec
UPDATE recovery_codes
SET used = true
WHERE user_id = $1 AND code_id = $2;

-- name: DeleteUserRecoveryCodes :exec
DELETE FROM recovery_codes
WHERE user_id = $1;
