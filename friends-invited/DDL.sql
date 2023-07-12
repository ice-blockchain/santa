-- SPDX-License-Identifier: ice License 1.0

CREATE TABLE IF NOT EXISTS referrals (
                                         processed_at   TIMESTAMP NOT NULL,
                                         deleted BOOLEAN DEFAULT false NOT NULL,
                                         user_id        TEXT NOT NULL,
                                         referred_by    TEXT NOT NULL,
                                         primary key (user_id, referred_by, deleted)
);
CREATE INDEX IF NOT EXISTS referrals_processed_at_ix ON referrals (processed_at);

CREATE TABLE IF NOT EXISTS friends_invited (
                                         invited_count  BIGINT NOT NULL DEFAULT 0,
                                         user_id        TEXT NOT NULL PRIMARY KEY
) WITH (fillfactor = 70);


DO $$ BEGIN
    ALTER TABLE referrals
        ADD COLUMN IF NOT EXISTS deleted BOOLEAN DEFAULT false NOT NULL,
        DROP CONSTRAINT IF EXISTS referrals_pkey;
    if NOT exists (select constraint_name from information_schema.table_constraints where table_name = 'referrals' and constraint_type = 'PRIMARY KEY') then
        ALTER TABLE referrals
            ADD CONSTRAINT referrals_id_refby_deleted_pkey PRIMARY KEY(user_id, referred_by, deleted);
    end if;
END $$;