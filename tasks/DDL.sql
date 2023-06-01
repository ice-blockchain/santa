-- SPDX-License-Identifier: ice License 1.0
--************************************************************************************************************************************
-- task_progress
CREATE TABLE IF NOT EXISTS task_progress (
                        friends_invited             BIGINT NOT NULL DEFAULT 0,
                        mining_started              BOOLEAN DEFAULT FALSE,
                        username_set                BOOLEAN DEFAULT FALSE,
                        profile_picture_set         BOOLEAN DEFAULT FALSE,
                        completed_tasks             TEXT[],
                        pseudo_completed_tasks      TEXT[],
                        user_id                     TEXT NOT NULL PRIMARY KEY,
                        twitter_user_handle         TEXT,
                        telegram_user_handle        TEXT
                    );