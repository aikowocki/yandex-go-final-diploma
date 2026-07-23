-- sync_enabled: пользователь может выбрать, какие папки синхронизировать фоново/при sync.
-- По умолчанию включено (1) — обратная совместимость с уже существующими локальными кешами.
ALTER TABLE vaults ADD COLUMN sync_enabled INTEGER NOT NULL DEFAULT 1;
