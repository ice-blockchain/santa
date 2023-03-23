-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- task_progress
box.execute([[CREATE TABLE IF NOT EXISTS task_progress (
                        completed_tasks             STRING,
                        pseudo_completed_tasks      STRING,
                        user_id                     STRING NOT NULL PRIMARY KEY,
                        twitter_user_handle         STRING,
                        telegram_user_handle        STRING,
                        friends_invited             UNSIGNED NOT NULL DEFAULT 0,
                        username_set                BOOLEAN,
                        profile_picture_set         BOOLEAN,
                        mining_started              BOOLEAN
                    )
                     WITH ENGINE = 'memtx';]])
--************************************************************************************************************************************
-- referrals
box.execute([[CREATE TABLE IF NOT EXISTS referrals (
                     user_id        STRING NOT NULL PRIMARY KEY,
                     referred_by    STRING NOT NULL
                 )
                  WITH ENGINE = 'memtx';]])
box.execute([[CREATE INDEX IF NOT EXISTS referrals_referred_by_ix ON referrals (referred_by);]])