ALTER TABLE users ADD COLUMN enc_master_key bytea;

COMMENT ON COLUMN users.enc_master_key IS
    'Случайный MasterKey, обёрнутый ключом KEK (derived from EncryptionPassphrase через '
    'Argon2id+HKDF). Позволяет менять пароль без перешифровки vault keys.';
