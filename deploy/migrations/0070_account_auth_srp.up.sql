-- 0070_account_auth_srp: durable SRP 2FA and authorization management fields.

ALTER TABLE account_passwords
    ADD COLUMN IF NOT EXISTS current_algo_salt1 BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN IF NOT EXISTS current_algo_salt2 BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN IF NOT EXISTS current_algo_g INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS current_algo_p BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN IF NOT EXISTS srp_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS srp_verifier BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN IF NOT EXISTS srp_b_secret BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN IF NOT EXISTS srp_b BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN IF NOT EXISTS recovery_email VARCHAR(256) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS recovery_code VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS recovery_code_expires_at TIMESTAMPTZ;

UPDATE authorizations
SET hash = auth_key_id
WHERE hash = 0;

CREATE UNIQUE INDEX IF NOT EXISTS authorizations_user_hash_idx
    ON authorizations (user_id, hash);
