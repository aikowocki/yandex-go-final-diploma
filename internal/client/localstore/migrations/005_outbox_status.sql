-- Статус outbox-записи: pending (по умолчанию) или conflict (требует явного разрешения).
ALTER TABLE outbox ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';

CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox (status, id);
