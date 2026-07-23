-- vaults: локальный кеш метаданных папок + sync-состояние.
CREATE TABLE IF NOT EXISTS vaults (
    id                TEXT PRIMARY KEY,
    wrapped_vault_key BLOB NOT NULL,
    enc_name          BLOB NOT NULL,
    version           INTEGER NOT NULL,
    synced_version    INTEGER NOT NULL DEFAULT 0,
    deleted           INTEGER NOT NULL DEFAULT 0
);
