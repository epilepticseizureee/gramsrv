DROP INDEX IF EXISTS authorizations_user_hash_idx;

ALTER TABLE account_passwords
    DROP COLUMN IF EXISTS recovery_code_expires_at,
    DROP COLUMN IF EXISTS recovery_code,
    DROP COLUMN IF EXISTS recovery_email,
    DROP COLUMN IF EXISTS srp_b,
    DROP COLUMN IF EXISTS srp_b_secret,
    DROP COLUMN IF EXISTS srp_verifier,
    DROP COLUMN IF EXISTS srp_id,
    DROP COLUMN IF EXISTS current_algo_p,
    DROP COLUMN IF EXISTS current_algo_g,
    DROP COLUMN IF EXISTS current_algo_salt2,
    DROP COLUMN IF EXISTS current_algo_salt1;
