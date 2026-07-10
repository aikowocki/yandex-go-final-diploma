-- secrets: локальный кеш секретов по тирам (enc_index/enc_payload лениво).
CREATE TABLE IF NOT EXISTS secrets (
    id             TEXT PRIMARY KEY,
    vault_id       TEXT NOT NULL,
    type           INTEGER NOT NULL,
    enc_row        BLOB NOT NULL,
    enc_index      BLOB,
    enc_payload    BLOB,
    version        INTEGER NOT NULL,
    index_loaded   INTEGER NOT NULL DEFAULT 0,
    payload_loaded INTEGER NOT NULL DEFAULT 0,
    dirty          INTEGER NOT NULL DEFAULT 0,
    deleted        INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS secrets_vault ON secrets(vault_id);
