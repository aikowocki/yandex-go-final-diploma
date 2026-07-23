CREATE TABLE vaults (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wrapped_vault_key bytea NOT NULL,
    enc_name          bytea NOT NULL,
    version           bigint NOT NULL DEFAULT 1,
    deleted           boolean NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON vaults (user_id);

COMMENT ON TABLE vaults IS
    'Ваулты(Папки) пользователя.';

COMMENT ON COLUMN vaults.wrapped_vault_key IS
    'VaultKey, обёрнутый (AEAD) под MasterKey пользователя.';

COMMENT ON COLUMN vaults.enc_name IS
    'Имя папки, зашифрованное AEAD под VaultKey. Сервер имени не знает.';

COMMENT ON COLUMN vaults.version IS
    'Версия папки для дельта-синхронизации. Инкрементируется при изменениях.';

COMMENT ON COLUMN vaults.deleted IS
    'Soft-delete: строка не удаляется физически, чтобы удаление можно было синхронизировать между устройствами.';
