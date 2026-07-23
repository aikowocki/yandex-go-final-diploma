CREATE TABLE recovery_codes (
    id              bigserial PRIMARY KEY,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_id         text NOT NULL,
    enc_master_key  bytea NOT NULL,
    used            boolean NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_recovery_codes_user_code ON recovery_codes(user_id, code_id);
