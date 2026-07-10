-- outbox: очередь оффлайн-изменений.
CREATE TABLE IF NOT EXISTS outbox (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    op           TEXT NOT NULL,
    entity       TEXT NOT NULL,
    entity_id    TEXT NOT NULL,
    base_version INTEGER,
    payload      BLOB,
    created_at   TEXT NOT NULL
);
