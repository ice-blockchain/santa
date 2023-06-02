-- SPDX-License-Identifier: ice License 1.0

CREATE TABLE IF NOT EXISTS referrals (
                                         processed_at   TIMESTAMP NOT NULL,
                                         user_id        TEXT NOT NULL,
                                         referred_by    TEXT NOT NULL,
                                         PRIMARY KEY(user_id, referred_by, processed_at)
);
CREATE INDEX IF NOT EXISTS referrals_processed_at_ix ON referrals (processed_at);

CREATE TABLE IF NOT EXISTS friends_invited (
                                         invited_count  BIGINT NOT NULL DEFAULT 0,
                                         user_id        TEXT NOT NULL PRIMARY KEY
) WITH (fillfactor = 70);