CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    login             citext NOT NULL UNIQUE,
    auth_hash         text NOT NULL,
    enc_kdf_salt      bytea,
    enc_kdf_params    jsonb,
    totp_secret       bytea,
    totp_enabled      boolean NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE users IS
    'Пользователи GophKeeper';

COMMENT ON COLUMN users.auth_hash IS
    'PHC-строка Argon2id(LoginCredential)';

COMMENT ON COLUMN users.enc_kdf_salt IS
    'Соль Argon2id для вывода MasterSeed из EncryptionPassphrase на клиенте.';

COMMENT ON COLUMN users.enc_kdf_params IS
    'Параметры Argon2id (crypto.Params, JSON), использованные для вывода MasterSeed из '
    'EncryptionPassphrase.';

COMMENT ON COLUMN users.totp_secret IS
    'Секрет TOTP для двухфакторной аутентификации.';

COMMENT ON COLUMN users.totp_enabled IS
    'Флаг, включена ли двухфакторная аутентификация для пользователя.';
