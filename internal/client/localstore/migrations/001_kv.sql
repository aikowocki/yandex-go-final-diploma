-- kv: произвольные настройки/сессия (НЕ MasterKey).
CREATE TABLE IF NOT EXISTS kv (
    k TEXT PRIMARY KEY,
    v BLOB
);
