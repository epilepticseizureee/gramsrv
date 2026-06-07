-- 0069_privacy_and_profile_photo_kinds: account privacy rules, fallback photos,
-- and viewer-private contact profile photos.

CREATE TABLE IF NOT EXISTS account_privacy_rules (
    owner_user_id BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    privacy_key   TEXT        NOT NULL,
    rules         JSONB       NOT NULL DEFAULT '[]'::jsonb,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_user_id, privacy_key)
);

ALTER TABLE profile_photos
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'profile';

ALTER TABLE profile_photos
    DROP CONSTRAINT IF EXISTS profile_photos_pkey;

ALTER TABLE profile_photos
    ADD CONSTRAINT profile_photos_pkey PRIMARY KEY (owner_peer_type, owner_peer_id, kind, photo_id);

ALTER TABLE profile_photos
    DROP CONSTRAINT IF EXISTS profile_photos_kind_check;

ALTER TABLE profile_photos
    ADD CONSTRAINT profile_photos_kind_check CHECK (kind IN ('profile', 'fallback'));

DROP INDEX IF EXISTS profile_photos_current_idx;
CREATE INDEX IF NOT EXISTS profile_photos_current_idx
    ON profile_photos (owner_peer_type, owner_peer_id, kind, sort_order DESC)
    WHERE active;

ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS personal_photo_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS personal_photo_date INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS contacts_personal_photo_idx
    ON contacts (user_id, contact_user_id)
    WHERE personal_photo_id <> 0;

