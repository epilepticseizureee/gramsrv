DROP INDEX IF EXISTS contacts_personal_photo_idx;

ALTER TABLE contacts
    DROP COLUMN IF EXISTS personal_photo_date,
    DROP COLUMN IF EXISTS personal_photo_id;

DROP INDEX IF EXISTS profile_photos_current_idx;

ALTER TABLE profile_photos
    DROP CONSTRAINT IF EXISTS profile_photos_kind_check;

ALTER TABLE profile_photos
    DROP CONSTRAINT IF EXISTS profile_photos_pkey;

ALTER TABLE profile_photos
    ADD CONSTRAINT profile_photos_pkey PRIMARY KEY (owner_peer_type, owner_peer_id, photo_id);

ALTER TABLE profile_photos
    DROP COLUMN IF EXISTS kind;

CREATE INDEX IF NOT EXISTS profile_photos_current_idx
    ON profile_photos (owner_peer_type, owner_peer_id, sort_order DESC)
    WHERE active;

DROP TABLE IF EXISTS account_privacy_rules;

