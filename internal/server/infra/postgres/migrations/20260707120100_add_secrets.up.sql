CREATE TABLE secrets (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    vault_id    uuid NOT NULL REFERENCES vaults(id) ON DELETE CASCADE,
    type        smallint NOT NULL,
    enc_row     bytea NOT NULL,
    enc_index   bytea NOT NULL,
    enc_payload bytea,
    blob_ref    text,
    blob_size   bigint,
    version     bigint NOT NULL DEFAULT 1,
    deleted     boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON secrets (vault_id);
CREATE INDEX ON secrets (vault_id, type);
CREATE INDEX ON secrets (vault_id, version);

COMMENT ON TABLE secrets IS
    'Секреты внутри ваултов.';

COMMENT ON COLUMN secrets.type IS
    'Тип секрета (ОТКРЫТО, не шифруется): 1=login_password, 2=text, 3=binary, 4=bank_card, 5=totp.';

COMMENT ON COLUMN secrets.enc_row IS
    'Tier 2a: поля строки списка, видимые всегда (title/tags/uri/username), AEAD под VaultKey.';

COMMENT ON COLUMN secrets.enc_index IS
    'Tier 2b: расширенный searchable-индекс (note/custom_fields), AEAD под VaultKey. Догружается в фоне.';

COMMENT ON COLUMN secrets.enc_payload IS
    'Tier 3: чувствительное тело (пароль/PAN/CVV/секрет TOTP), AEAD под VaultKey.';

COMMENT ON COLUMN secrets.blob_ref IS
    'Ключ объекта в MinIO для крупных бинарей.';

COMMENT ON COLUMN secrets.deleted IS
    'Soft-delete для синхронизации удалений между устройствами.';
